package user

import (
	"context"
	"vhennpay-bend/dao"
	"vhennpay-bend/models"
	"vhennpay-bend/utils"
	"vhennpay-bend/utils/notifications"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service represents the User Service
type Service struct {
	dao        *dao.UserDAO
	factoryDAO *dao.FactoryDAO
	notifiable notifications.Notifiable
}

// NewUserService returns a user service object
func NewUserService(dao *dao.UserDAO, factoryDAO *dao.FactoryDAO) *Service {
	notifiable, err := notifications.NewNotifiable(factoryDAO)
	if err != nil {
		log.Fatalf("notifiable_init: %v", err)
		return nil
	}
	return &Service{
		dao:        dao,
		factoryDAO: factoryDAO,
		notifiable: notifiable,
	}
}

// SignupUser creates a new user account
func (s *Service) SignupUser(w http.ResponseWriter, r *http.Request) {
	var (
		user models.User
		req  models.CreateUserReq
	)
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	// verify user doesn't exist
	count, err := s.dao.Collection.CountDocuments(context.TODO(), bson.D{
		{Key: "email", Value: strings.ToLower(req.Email)},
	})

	if err != nil {
		log.Printf("failed to retrieve user (%s) err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	if count > 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "This user account already exists")
		return
	}

	// update remaining fields
	now := time.Now()
	user.ID = primitive.NewObjectID()
	user.Username = req.Username
	user.Email = strings.ToLower(req.Email)
	user.Confirmed = false
	user.CreatedAt = now
	user.UpdatedAt = now
	user.PassCode = utils.GenPasscode()

	// hash password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "An Error occurred while processing request")
		return
	}
	user.Password = hash

	// save user to db
	if err := s.dao.Insert(user); err != nil {
		log.Printf("failed to insert new user (%s) err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusInternalServerError, "An Error occurred while processing request")
		return
	}

	go sendVerificationEmail(user, false)

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status: "success",
		Code:   http.StatusCreated,
		Data:   user,
	})
}

// ResendOTP resends a new OTP for user signup
func (s *Service) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	// retrive user
	email := strings.ToLower(req.Email)
	user, err := s.dao.FindByEmail(email)
	if err != nil {
		log.Printf("failed to retrieve user object for %s, err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	// regenerate passcode
	user.PassCode = utils.GenPasscode()

	err = s.dao.Update(user)
	if err != nil {
		log.Printf("failed to update user (%s) err, %v", req.Email, err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while verifying account")
		return
	}

	go sendVerificationEmail(user, false)

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "OTP has been resent",
	})
}

// Signin authenticates a user account
func (s *Service) Signin(w http.ResponseWriter, r *http.Request) {
	var (
		req models.LoginReq
	)
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "This is an invalid request data sent")
		return
	}

	// retrive user
	email := strings.ToLower(req.Email)
	user, err := s.dao.FindByEmail(email)
	if err != nil {
		log.Printf("failed to retrieve user object for %s, err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	// validate password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		utils.RespondWithError(w, http.StatusForbidden, "Invalid credentials")
		return
	}

	// generate jwt
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = jwt.MapClaims{
		"exp":       time.Now().Add(time.Hour * 72).Unix(),
		"email":     user.Email,
		"id":        user.ID,
		"user_name": user.Username,
		"is_active": user.Confirmed,
	}
	signedString, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		log.Println("error generating token", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "An Error occurred")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"token":  signedString,
	})
}

// UpdateFCMToken updates the fcm token attached to a user
func (s *Service) UpdateFCMToken(w http.ResponseWriter, r *http.Request) {
	var req models.FCMTokenReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	if req.Token == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))
	user, err := s.dao.FindByID(userID.(string))
	if err != nil {
		log.Printf("failed to retrieve user with id %s err: %v", userID.(string), err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	user.FCMToken = req.Token

	err = s.dao.Update(user)
	if err != nil {
		log.Printf("failed to update user (%s) err, %v", userID.(string), err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while verifying account")
		return
	}

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "FCM token updated",
	})
}

// ConfirmAccount verifies a user account with pass code generated at signup
func (s *Service) ConfirmAccount(w http.ResponseWriter, r *http.Request) {
	var req models.ConfirmAccountReq

	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// retrieve user account
	email := strings.ToLower(req.Email)
	user, err := s.dao.FindByEmail(email)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "No user account found")
		return
	}

	// verify code
	if user.PassCode != req.Code {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid verification code sent")
		return
	}
	user.Confirmed = true

	// update user status
	err = s.dao.Update(user)
	if err != nil {
		log.Printf("failed to update user (%s) err, %v", req.Email, err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while verifying account")
		return
	}

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Account has been activated",
	})
}

// RequestPasswordReset sends a new password reset code
func (s *Service) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetReq

	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	code := utils.GenPasscode()

	email := strings.ToLower(req.Email)
	user, err := s.dao.FindByEmail(email)
	if err != nil {
		log.Printf("failed to retrieve user (%s), err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}
	user.PassCode = code

	err = s.dao.Update(user)
	if err != nil {
		log.Printf("failed to update user (%s), err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "Error generating password reset code")
		return
	}

	// mail new code
	sendVerificationEmail(user, true)
	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Code:    http.StatusCreated,
		Message: "Account reset code has been sent!",
	})
}

// ResetPassword resets a user password
func (s *Service) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordReset

	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// retrieve user
	email := strings.ToLower(req.Email)
	user, err := s.dao.FindByEmail(email)
	if err != nil {
		log.Printf("failed to retrieve user (%s), err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	// check passcode validaity
	if req.Code != user.PassCode {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid reset code")
		return
	}

	// hash new password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("failed to hash password for %s, err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while processing request")
		return
	}

	user.Password = hash

	err = s.dao.Update(user)
	if err != nil {
		log.Printf("failed to update user (%s), err: %v", req.Email, err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Password Reset successful!",
	})
}

// RetrieveUser ...
func (s *Service) RetrieveUser(w http.ResponseWriter, r *http.Request) {

}

func sendVerificationEmail(user models.User, isOTPResend bool) {
	email := utils.EmailData{}
	email.EmailTo = user.Email
	email.ContentData = map[string]interface{}{
		"Name": user.Username,
		"Code": user.PassCode,
	}
	email.Template = "activation.html"
	email.Title = "[DILS]: Welcome"
	if isOTPResend {
		email.Template = "resend_activation.html"
		email.Title = "[DILS]: Reset Password"
	}
	err := utils.SendGoMail(email)
	if err != nil {
		log.Printf("[send_verification]: failed to send mail: %v\n", err)
		return
	}
	log.Println("Verification sent to", user.Email)
}

// AddPaymentOption creates a new payment option for user
func (s *Service) AddPaymentOption(w http.ResponseWriter, r *http.Request) {
	var req models.PaymentOptionReq

	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// TODO: request validation
	var (
		paymentOption interface{}
		selectedOpt   string
		userIDKey     = models.ContextKey("user_id")
	)
	id := r.Context().Value(userIDKey)

	userid, err := primitive.ObjectIDFromHex(id.(string))
	if err != nil {
		log.Printf("failed to parse userid: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	now := time.Now()
	switch req.Type {
	case models.Bank:
		selectedOpt = "bank_payment_option"
		paymentOption = models.BankOption{
			ID:            primitive.NewObjectID(),
			UserID:        userid,
			AccountName:   req.AccountName,
			AccountNumber: req.AccountNumber,
			BankName:      req.BankName,
			SortCode:      req.SortCode,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	case models.PayPal:
		selectedOpt = "paypal_payment_option"
		paymentOption = models.PayPalOption{
			ID:        primitive.NewObjectID(),
			UserID:    userid,
			Email:     strings.ToLower(req.Email),
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	if err := s.factoryDAO.Insert(selectedOpt, paymentOption); err != nil {
		log.Printf("failed to insert new paymentOption (%s) err: %v", id, err)
		utils.RespondWithError(w, http.StatusInternalServerError, "An Error occurred while processing request")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status: "success",
		Code:   http.StatusCreated,
		Data:   paymentOption,
	})
}

// RetrievePaymentOption for user
func (s *Service) RetrievePaymentOption(w http.ResponseWriter, r *http.Request) {
	var req models.GetPaymentOptionReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	var userIDKey = models.ContextKey("user_id")
	userID := r.Context().Value(userIDKey)

	// TODO: validate userid?
	options, err := s.factoryDAO.FindPaymentOpt(userID.(string), req.Type)
	if err != nil {
		log.Printf("failed to retrieve user payment options: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving payment options")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   options,
	})
}

// Notifications ...
func (s *Service) Notifications(w http.ResponseWriter, r *http.Request) {
	var userIDKey = models.ContextKey("user_id")
	userID := r.Context().Value(userIDKey)

	id, _ := primitive.ObjectIDFromHex(userID.(string))
	// build filter
	filter := bson.M{
		"user_id": id,
	}

	notifications, err := s.factoryDAO.QueryNotifications(filter)
	if err != nil {
		log.Printf("failed to retrieve user notifications: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving notifications")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   notifications,
	})
}

// RateUser ...
func (s *Service) RateUser(w http.ResponseWriter, r *http.Request) {
	var req models.RateUserReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))

	user, err := s.dao.FindByID(userID.(string))
	if err != nil {
		log.Printf("failed to retreive user: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	switch req.Type {
	case models.PositiveRating:
		user.PositiveRatings++
	case models.NegativeRating:
		user.NegativeRatings++
	default:
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid rating type")
		return
	}

	if err := s.dao.Update(user); err != nil {
		log.Printf("failed to update user: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot update user rating at this time")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "User ratings updated successfully",
	})
}

// AddWallet ...
func (s *Service) AddWallet(w http.ResponseWriter, r *http.Request) {
	var req models.UserWalletReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))
	puid, _ := primitive.ObjectIDFromHex(userID.(string))

	// check user wallet entry doesn't exist
	wallet, err := s.factoryDAO.Query("user_wallet", bson.M{
		"user_id": puid,
		"address": req.Address,
	})
	if err != nil {
		log.Printf("failed to retrieve user_wallet: %+v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot create user wallet")
		return
	}

	wl := wallet.([]bson.M)

	if len(wl) > 0 {
		utils.RespondWithJSON(w, http.StatusOK, utils.Response{
			Status: "success",
			Code:   http.StatusOK,
			Data:   wl[0],
		})
		return
	}

	// create new entry
	userWallet := models.UserWallet{
		ID:        primitive.NewObjectID(),
		UserID:    puid,
		Address:   req.Address,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.factoryDAO.Insert("user_wallet", userWallet); err != nil {
		log.Printf("failed to create user_wallet: %+v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot create user wallet")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status: "success",
		Code:   http.StatusCreated,
		Data:   userWallet,
	})
}

// GetWallets ...
func (s *Service) GetWallets(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.ContextKey("user_id"))
	puid, _ := primitive.ObjectIDFromHex(userID.(string))

	// check user wallet entry doesn't exist
	wallets, err := s.factoryDAO.Query("user_wallet", bson.M{
		"user_id": puid,
	})
	if err != nil {
		log.Printf("failed to retrieve user_wallet: %+v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot retrieve user wallet")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   wallets,
	})
}

// DeleteWallet ...
func (s *Service) DeleteWallet(w http.ResponseWriter, r *http.Request) {
	walletID := mux.Vars(r)["id"]

	userID := r.Context().Value(models.ContextKey("user_id"))
	puid, _ := primitive.ObjectIDFromHex(userID.(string))
	pwid, _ := primitive.ObjectIDFromHex(walletID)

	wallets, err := s.factoryDAO.Query("user_wallet", bson.M{
		"_id":     pwid,
		"user_id": puid,
	})

	if err != nil {
		log.Printf("failed to retrieve user_wallet: %+v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot retrieve user wallet")
		return
	}

	if len(wallets.([]bson.M)) < 1 {
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot retrieve user wallet")
		return
	}

	// drop wallet document
	if err := s.factoryDAO.Remove("user_wallet", bson.M{"_id": pwid}); err != nil {
		log.Printf("failed to delete user wallet: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot delete user wallet")
		return
	}

	utils.RespondWithOk(w, "User wallet deleted")
}

// Currencies ...
func (s *Service) Currencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := s.factoryDAO.Query("currencies", bson.M{})
	if err != nil {
		log.Printf("failed to retrieve currencies: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving currency list")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   currencies,
	})
}

// NewSupportChat ...
func (s *Service) NewSupportChat(w http.ResponseWriter, r *http.Request) {
	var req models.NewSupportChatReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))

	if req.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Message is empty")
		return
	}

	user, err := s.dao.FindByID(userID.(string))
	if err != nil {
		log.Printf("failed to retrieve user with id %s err: %v", userID.(string), err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	msg := models.SupportChat{
		ID:         primitive.NewObjectID(),
		UserID:     user.ID,
		Username:   user.Username,
		Message:    req.Message,
		SentByUser: req.SentByUser,
		CreatedAt:  time.Now().UTC(),
	}
	// TODO: update support staff user
	if err := s.replyChat("6051e375008d7358b139fe19", msg, true); err != nil {
		log.Printf("failed to create support chat entry: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Unable to send response at this time")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Code:    http.StatusCreated,
		Data:    msg,
		Message: "Support chat sent",
	})
}

// ReplySupportChat ...
func (s *Service) ReplySupportChat(w http.ResponseWriter, r *http.Request) {
	var req models.NewSupportChatReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	userID := mux.Vars(r)["userId"]

	user, err := s.dao.FindByID(userID)
	if err != nil {
		log.Printf("failed to retrieve user with id %s err: %v", userID, err)
		utils.RespondWithError(w, http.StatusNotFound, "User account not found")
		return
	}

	msg := models.SupportChat{
		ID:         primitive.NewObjectID(),
		UserID:     user.ID,
		Username:   "DILS-Support",
		Message:    req.Message,
		SentByUser: false,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.replyChat(user.ID.Hex(), msg, false); err != nil {
		log.Printf("failed to send support chat response: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Unable to send response at this time")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Code:    http.StatusCreated,
		Data:    msg,
		Message: "Support chat reply sent",
	})
}

// GetAllChats returns a list of users who have initiated a support chat
func (s *Service) GetAllChats(w http.ResponseWriter, r *http.Request) {

	chats, err := s.factoryDAO.GetSupportChatUsers()
	if err != nil {
		log.Printf("err_q_support_messages: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving support chats")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   chats,
	})
}

// GetSupportChats ...
func (s *Service) GetSupportChats(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(models.ContextKey("user_id"))
	uid, _ := primitive.ObjectIDFromHex(userID.(string))

	messages, err := s.getUserChats(uid)
	if err != nil {
		log.Printf("err_q_support_messages: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving support chat")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   messages,
	})
}

// GetUserChats ...
func (s *Service) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	uid, _ := primitive.ObjectIDFromHex(userID)

	messages, err := s.getUserChats(uid)
	if err != nil {
		log.Printf("err_q_support_messages: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving support chat")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   messages,
	})
}

// GetChatUser ...
func (s *Service) GetChatUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]

	user, err := s.dao.FindByID(userID)
	if err != nil {
		log.Printf("err_user_not_found: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving chat user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   user,
	})
}

func (s *Service) getUserChats(id primitive.ObjectID) (interface{}, error) {
	messages, err := s.factoryDAO.Query("support_chat", bson.M{
		"user_id": id,
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Service) replyChat(to string, msg models.SupportChat, fromUser bool) error {
	subject := "New Support Chat from " + msg.Username

	if err := s.factoryDAO.Insert("support_chat", msg); err != nil {
		return err
	}

	if !fromUser {
		subject = "New response to your support chat"
	}
	// notify user
	nData := notifications.GenericEmailData{
		Formatted: false,
		Content:   msg.Message,
	}

	go s.notifiable.SendGenericNotification(to, subject, nData)
	return nil
}
