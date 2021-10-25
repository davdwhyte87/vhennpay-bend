package main

import (
	"context"
	"vhennpay-bend/api/callbacks"
	"vhennpay-bend/api/order"
	"vhennpay-bend/api/user"
	"vhennpay-bend/dao"
	"vhennpay-bend/models"
	"vhennpay-bend/utils"
	"vhennpay-bend/utils/escrow"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	userDAO          *dao.UserDAO
	factoryDAO       *dao.FactoryDAO
	orderDAO         *dao.OrderDAO
	userService      *user.Service
	orderService     *order.Service
	callbacksService *callbacks.Service
	jwtSecret        string
	dbname           = "dils"
)

func main() {
	// env := os.Getenv("ENV")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
		// if env != "" && env != "DEPLOY-DEV" {
		// 	log.Println("No .env file found")
		// 	return
		// }
	}

	println(os.Getenv("EMAIL_SENDER"))
	jwtSecret = os.Getenv("SECRET")

	client, err := initDatabase()
	if err != nil {
		log.Fatalf("failed to initialize database, err: %v", err)
		return
	}

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

	initServices(client.Database(dbname))

	r := initRoutes()
	r.Use(func(next http.Handler) http.Handler {
		return handlers.LoggingHandler(os.Stdout, next)
	})

	// background services
	go orderService.AutoCancellationJob()

	port := os.Getenv("PORT")
	log.Println("Running server on port", port)

	header := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"})
	origins := handlers.AllowedOrigins([]string{"*"})

	h := handlers.CORS(header, methods, origins)
	if err := http.ListenAndServe(":"+port, h(r)); err != nil {
		log.Fatal(err)
	}
}

func initRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "dilis-backend"}`))
	})
	v1 := r.PathPrefix("/api/v1").Subrouter()
	userRouter := v1.PathPrefix("/user").Subrouter()
	ordersRouter := v1.PathPrefix("/orders").Subrouter()
	tradesRouter := v1.PathPrefix("/trades").Subrouter()
	supportRouter := v1.PathPrefix("/support").Subrouter()
	callbacksRouter := v1.PathPrefix("/callbacks").Subrouter()

	//utils
	v1.HandleFunc("/currencies", userService.Currencies).Methods("GET")
	callbacksRouter.HandleFunc("/paypal-confirm",
		callbacksService.ConfirmPaypalPayment).Methods("POST")

	// support
	supportRouter.HandleFunc("/chats/users", userService.GetAllChats).Methods("GET")
	supportRouter.HandleFunc("/chats/users/{userId}", userService.GetChatUser).Methods("GET")
	supportRouter.HandleFunc("/chats/r/{userId}", userService.GetUserChats).Methods("GET")
	supportRouter.HandleFunc("/chats/r/{userId}", userService.ReplySupportChat).Methods("POST")
	supportRouter.HandleFunc("/chats", useAuth(userService.GetSupportChats)).Methods("GET")
	supportRouter.HandleFunc("/chats", useAuth(userService.NewSupportChat)).Methods("POST")

	// Orders
	ordersRouter.HandleFunc("", useAuth(orderService.GetUserOrders)).Methods("GET")
	ordersRouter.HandleFunc("/create", useAuth(orderService.CreateSellOrder)).Methods("POST")
	ordersRouter.HandleFunc("/pending", useAuth(orderService.GetPendingOrders)).Methods("GET")
	ordersRouter.HandleFunc("/{id}", useAuth(orderService.ViewOrder)).Methods("GET")
	ordersRouter.HandleFunc("/{id}/cancel", useAuth(orderService.CancelOrder)).Methods("PUT")
	ordersRouter.HandleFunc("/{id}/trades", useAuth(orderService.ViewOrderTrades)).Methods("GET")

	// Trades
	tradesRouter.HandleFunc("", useAuth(orderService.GetTrades)).Methods("GET")
	tradesRouter.HandleFunc("/create", useAuth(orderService.CreateBuyTrade)).Methods("POST")
	tradesRouter.HandleFunc("/{id}/confirm", useAuth(orderService.ConfirmTrade)).Methods("PUT")
	tradesRouter.HandleFunc("/{id}/paid", useAuth(orderService.MarkTradePaid)).Methods("PUT")
	tradesRouter.HandleFunc("/{id}/cancel", useAuth(orderService.CancelTrade)).Methods("PUT")
	tradesRouter.HandleFunc("/{id}/messages", useAuth(orderService.NewMessage)).Methods("POST")
	tradesRouter.HandleFunc("/{id}/messages", useAuth(orderService.GetTradeMessages)).Methods("GET")
	tradesRouter.HandleFunc("/{id}", useAuth(orderService.GetBuyTrade)).Methods("GET")

	// Users
	userRouter.HandleFunc("/signup", userService.SignupUser).Methods("POST")
	userRouter.HandleFunc("/signin", userService.Signin).Methods("POST")
	userRouter.HandleFunc("/signup/resend-otp", userService.ResendOTP).Methods("POST")
	userRouter.HandleFunc("/confirm", userService.ConfirmAccount).Methods("POST")
	userRouter.HandleFunc("/fcm-token", useAuth(userService.UpdateFCMToken)).Methods("POST")
	userRouter.HandleFunc("/passwords/request", userService.RequestPasswordReset).Methods("POST")
	userRouter.HandleFunc("/passwords/reset", userService.ResetPassword).Methods("POST")
	userRouter.HandleFunc("/payment-options", useAuth(userService.AddPaymentOption)).Methods("POST")
	userRouter.HandleFunc("/payment-options/info", useAuth(userService.RetrievePaymentOption)).Methods("POST")
	userRouter.HandleFunc("/{id}/rate", useAuth(userService.RateUser)).Methods("POST")

	userRouter.HandleFunc("/notifications", useAuth(userService.Notifications)).Methods("GET")
	userRouter.HandleFunc("/wallets", useAuth(userService.AddWallet)).Methods("POST")
	userRouter.HandleFunc("/wallets", useAuth(userService.GetWallets)).Methods("GET")
	userRouter.HandleFunc("/wallets/{id}", useAuth(userService.DeleteWallet)).Methods("DELETE")

	return r
}

func initDatabase() (*mongo.Client, error) {
	dbURI := os.Getenv("MONGO_URI")
	dbUser := os.Getenv("MONGO_USER")
	dbPass := os.Getenv("MONGO_PASS")

	if dbPass == "" {
		return nil, errors.New("MONGO_SERVER pass not set")
	}

	client, ctx, err := dao.Initialize(dbURI, dbUser, dbPass, dbname)
	if err != nil {
		return nil, err
	}

	initCollections(ctx, client)

	return client, nil
}

func initCollections(ctx context.Context, client *mongo.Client) {
	db := client.Database(dbname)
	userDAO = dao.NewUserDAO(ctx, db)
	factoryDAO = dao.NewFactoryDAO(ctx, db)
	orderDAO = dao.NewOrderDAO(ctx, db)
}

func initServices(db *mongo.Database) {
	userService = user.NewUserService(userDAO, factoryDAO)
	escrowSrv := escrow.InitEscrow(db)
	orderService = order.NewOrderService(orderDAO, escrowSrv, factoryDAO)
	callbacksService = callbacks.NewCallbacksService(factoryDAO)
}

// useAuth validates a token for protected routes
func useAuth(nextHandler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "You are not authorized")
			return
		}
		token, err := jwt.Parse(authorizationHeader, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(jwtSecret), nil
		})
		if err != nil {
			log.Printf("auth parse err: %v", err)
			utils.RespondWithError(w, http.StatusUnauthorized, "You are not authorized")
			return
		}

		var userIDKey = models.ContextKey("user_id")
		var userEmailKey = models.ContextKey("user_email")

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			var id, email string
			id, ok = claims["id"].(string)
			if !ok {
				utils.RespondWithError(w, http.StatusUnauthorized, "Error converting claim to string")
				return
			}
			email, ok = claims["email"].(string)
			if !ok {
				utils.RespondWithError(w, http.StatusUnauthorized, "Error converting claim to string")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, id)
			rctx := context.WithValue(ctx, userEmailKey, email)

			nextHandler.ServeHTTP(w, r.WithContext(rctx))
			return
		}

		utils.RespondWithError(w, http.StatusUnauthorized, "An authorized error occurred")
	})
}
