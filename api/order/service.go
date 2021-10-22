package order

import (
	"vhennpay-bend/dao"
	"vhennpay-bend/models"
	"vhennpay-bend/utils"
	"vhennpay-bend/utils/escrow"
	"vhennpay-bend/utils/notifications"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service represents the User Service
type Service struct {
	dao        *dao.OrderDAO
	escrow     *escrow.Escrow
	factoryDAO *dao.FactoryDAO
	notifiable notifications.Notifiable
}

// NewOrderService returns a new order service
func NewOrderService(dao *dao.OrderDAO, escrow *escrow.Escrow, factoryDAO *dao.FactoryDAO) *Service {
	notifiable, err := notifications.NewNotifiable(factoryDAO)
	if err != nil {
		log.Fatalf("notifiable_init: %v", err)
		return nil
	}
	return &Service{dao: dao, escrow: escrow, factoryDAO: factoryDAO, notifiable: notifiable}
}

// CreateSellOrder creates a new sell order
func (s *Service) CreateSellOrder(w http.ResponseWriter, r *http.Request) {
	var (
		order models.SellOrder
		req   models.SellOrderReq
	)
	err := utils.DecodeReq(r, &req)
	if err != nil {
		log.Printf("error decoding create_order req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))

	// skip validation of components
	// TODO: add struct wide validation
	if req.PhoneNumber == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Phone number is missing from input")
		return
	}

	now := time.Now().UTC()
	paymentOptionID, _ := primitive.ObjectIDFromHex(req.PaymentOptionID)
	uid, _ := primitive.ObjectIDFromHex(userID.(string))
	order.ID = primitive.NewObjectID()
	order.CreatedBy = uid
	order.Amount = req.Amount
	order.AmountLeft = req.Amount
	order.Currency = req.Currency
	order.PhoneNumber = req.PhoneNumber
	order.ExRate = req.ExRate
	order.WalletID = req.WalletID
	order.PaymentOptionID = paymentOptionID
	order.PaymentOption = req.PaymentOption
	order.Note = req.Note
	order.Status = models.OrderPending
	order.CreatedAt = now
	order.UpdatedAt = now

	err = s.escrow.NewDeposit(
		order.ID,
		order.CreatedBy,
		order.Amount,
		req.WalletID,
		req.WalletPrivateKey,
	)
	if err != nil {
		log.Printf("failed to init escrow deposit: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("An error occurred while charging wallet, reason: %v", err.Error()))
		return
	}

	if err := s.dao.Insert(order); err != nil {
		log.Printf("failed to create new sell order: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "An Error occurred while processing request")
		return
	}

	// prepare and dispatch notifications
	go s.notifiable.SendOrderCreatedNotification(order, userID.(string))

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status: "success",
		Code:   http.StatusCreated,
		Data:   order,
	})
}

// ViewOrder ...
func (s *Service) ViewOrder(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["id"]

	if orderID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	orderIn, err := s.dao.PipelineSingle(orderID, nil)
	if err != nil {
		log.Printf("view_order: failed to retrieve order: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Order not found")
		return
	}
	order := orderIn.(bson.M)
	var option interface{}
	if order["payment_option_id"] != primitive.NilObjectID {
		option, err = s.factoryDAO.FindPaymentOptByID(
			order["payment_option_id"].(primitive.ObjectID).Hex(),
			models.PaymentOption(order["payment_option"].(int32)),
		)
		if err != nil {
			log.Printf("view_order: failed to retrieve order payment option: %v", err)
			utils.RespondWithError(w, http.StatusNotFound, "Order not found")
			return
		}

	}
	order["payment_option_data"] = option

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   order,
	})
}

// GetPendingOrders applies filters to retreive pending orders
func (s *Service) GetPendingOrders(w http.ResponseWriter, r *http.Request) {
	query := make(bson.M)

	v := r.URL.Query()
	currency := v.Get("currency")
	amount := v.Get("amount")
	query["status"] = models.OrderPending

	if currency != "" && currency != "any" {
		query["currency"] = bson.M{
			"$regex":   currency,
			"$options": "i",
		}
	}

	if amount != "" {
		a, _ := strconv.ParseFloat(amount, 64)
		query["amount"] = bson.M{
			"$gte": a,
		}
	}

	orders, err := s.dao.PipelineAll(query)
	if err != nil {
		log.Printf("pending_orders: failed to retrieve orders: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No orders found with query")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   orders,
	})
}

// GetUserOrders returns orders assiocated with an authenticated User with
// optional filters
func (s *Service) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	query := make(bson.M)
	v := r.URL.Query()
	status := v.Get("status")

	userID := r.Context().Value(models.ContextKey("user_id"))
	puserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		log.Printf("get_user_orders: failed to get primitive ID: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No orders found with query")
		return
	}
	// prepare query
	query["created_by"] = puserID

	if status != "" {
		query["status"] = status
	}

	orders, err := s.dao.Query(query)
	if err != nil {
		log.Printf("user_orders: failed to retrieve orders: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No orders found with query")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   orders,
	})

}

// ViewOrderTrades ...
func (s *Service) ViewOrderTrades(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["id"]
	if orderID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	porderID, _ := primitive.ObjectIDFromHex(orderID)
	trades, err := s.dao.QueryTrades(bson.M{
		"order_id": porderID,
	})

	if err != nil {
		log.Printf("get_trades: failed to retrieve trade: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No trades found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   trades,
	})
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["id"]
	if orderID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))

	// retrieve order
	order, err := s.dao.FindByID(orderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	// confirm cancellation triggerer
	if order.CreatedBy.Hex() != userID.(string) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Order not available to user")
		return
	}

	order.Status = models.OrderCancelled

	if err := s.dao.Update(order); err != nil {
		log.Printf("failed to update order %v: %v", order.ID.Hex(), err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	err = s.escrow.ReverseDeposit(order)
	if err != nil {
		log.Printf("failed to reverse escrow deposit: %v", err)
	}

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Order has been cancelled",
	})
}
