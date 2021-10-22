package order

import (
	"vhennpay-bend/models"
	"vhennpay-bend/utils"
	"vhennpay-bend/utils/notifications"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetBuyTrade ...
func (s *Service) GetBuyTrade(w http.ResponseWriter, r *http.Request) {
	tradeID := mux.Vars(r)["id"]

	if tradeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		log.Printf("view_trade: failed to retrieve trade: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   trade,
	})
}

// CreateBuyTrade registers a buy intent to a sell Trade (Order)
func (s *Service) CreateBuyTrade(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBuyTradeReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		log.Printf("buy_intent: error decoding req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))

	order, err := s.dao.FindByID(req.OrderID)
	if err != nil {
		log.Printf("view_order: failed to retrieve order: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	// confirm buyer hasn't initiated and early trade on the current order
	bid, _ := primitive.ObjectIDFromHex(userID.(string))
	q := bson.M{
		"buyer_id": bid,
		"order_id": order.ID,
		"status":   models.TradeInProgress,
	}

	ctrades, err := s.dao.QueryTrades(q)
	if err != nil {
		log.Printf("retrieve_ctrades: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Order not available at this time")
		return
	}

	if len(ctrades) > 0 {
		utils.RespondWithError(w, http.StatusNotFound, "Operation not allowed on order")
		return
	}

	if order.Status != models.OrderPending {
		utils.RespondWithError(w, http.StatusNotFound, "Order not available at this time, order "+order.Status)
		return
	}

	if userID.(string) == order.CreatedBy.Hex() {
		utils.RespondWithError(w, http.StatusNotFound, "Operation not allowed on order")
		return
	}

	if order.AmountSold > 0 && req.Amount > order.AmountSold {
		utils.RespondWithError(w, http.StatusNotFound, "Order has limited funds available")
		return
	}

	if req.Amount > order.Amount || (order.AmountLeft > 0 && req.Amount > order.AmountLeft) {
		utils.RespondWithError(w, http.StatusNotFound, "Order has limited funds available")
		return
	}

	now := time.Now().UTC()

	buyerID, _ := primitive.ObjectIDFromHex(userID.(string))

	trade := models.BuyTrade{
		ID:          primitive.NewObjectID(),
		SellerID:    order.CreatedBy,
		BuyerID:     buyerID,
		OrderID:     order.ID,
		BuyerWallet: req.WalletID,
		Amount:      req.Amount,
		LockTime:    now,
		Status:      models.TradeInProgress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.dao.InsertTrade(trade); err != nil {
		log.Printf("failed to create buy_trade: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	// dispatch notification
	go s.notifiable.SendOrderIntentNotification(trade, trade.BuyerID.Hex(), trade.SellerID.Hex())

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Data:    trade,
		Code:    http.StatusCreated,
		Message: "Buy trade has been initiated",
	})
}

// GetTrades ...
func (s *Service) GetTrades(w http.ResponseWriter, r *http.Request) {
	query := make(bson.M)
	v := r.URL.Query()
	status := v.Get("status")

	userID := r.Context().Value(models.ContextKey("user_id"))
	puserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		log.Printf("get_user_trades: failed to get primitive ID: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No orders found with query")
		return
	}

	query["buyer_id"] = puserID
	if status != "" {
		query["status"] = status
	}

	trades, err := s.dao.QueryTrades(query)
	if err != nil {
		log.Printf("user_trades: failed to retrieve trades: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "No trades found with query")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   trades,
	})
}

// ConfirmTrade ...
func (s *Service) ConfirmTrade(w http.ResponseWriter, r *http.Request) {
	var userIDKey = models.ContextKey("user_id")
	orderCreator := r.Context().Value(userIDKey)
	tradeID := mux.Vars(r)["id"]

	if tradeID == "" {
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		log.Printf("confirm_trade: failed to retrieve trade: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	if trade.Status == models.TradeProcessed {
		utils.RespondWithOk(w, "Trade already marked as confirmed")
		return
	}

	if trade.Status != models.TradeInProgress && trade.Status != models.TradeProcessed {
		utils.RespondWithError(w, http.StatusUnauthorized, "Trade operation not allowed")
		return
	}

	order, err := s.dao.FindByID(trade.OrderID.Hex())
	if err != nil {
		log.Printf("confirm_trade: failed to retrieve order: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	user, err := s.factoryDAO.FactoryFindUser("user", trade.SellerID)
	if err != nil {
		log.Printf("get_user: failed to retrieve user: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Seller not found")
		return
	}

	// validate confirmation triggerer
	if orderCreator.(string) != order.CreatedBy.Hex() {
		utils.RespondWithError(w, http.StatusUnauthorized, "Trade operation not allowed")
		return
	}

	if trade.Confirmed {
		utils.RespondWithOk(w, "Trade is already marked as confirmed")
		return
	}

	now := time.Now().UTC()
	// update trade
	trade.Confirmed = true
	trade.ProcessedAt = time.Now()
	trade.Status = models.TradeProcessed
	trade.UpdatedAt = now

	// update order
	order.AmountSold = order.AmountSold + trade.Amount
	order.AmountLeft = order.Amount - order.AmountSold
	order.UpdatedAt = now

	// update user (seller) attributes
	user.NumTransactions++

	// release fund
	err = s.processConfirmedOrder(trade)
	if err != nil {
		log.Printf("escrow_release: failed to release deposit: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while confirming trade: "+err.Error())
		return
	}

	// update user
	if err := s.factoryDAO.Update("user", user.ID, user); err != nil {
		log.Printf("update_user: failed to update user: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while confirming trade")
		return
	}

	// persit trade changes
	if err := s.dao.UpdateTrade(trade); err != nil {
		log.Printf("confirm_trade: failed to update trade: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while confirming trade")
		return
	}

	// persit order changes
	if err := s.dao.Update(order); err != nil {
		log.Printf("confirm_order: failed to update order: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "An Error occurred while confirming order")
		return
	}

	// notify
	go s.notifiable.SendOrderConfirmedNotification(trade, trade.BuyerID.Hex())

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Order has been marked as confirmed",
	})
}

// MarkTradePaid ...
func (s *Service) MarkTradePaid(w http.ResponseWriter, r *http.Request) {
	tradeID := mux.Vars(r)["id"]
	if tradeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))
	// retrieve trade
	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	user, err := s.factoryDAO.FactoryFindUser("user", trade.BuyerID)
	if err != nil {
		log.Printf("get_user: failed to retrieve user: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "Buyer info not found")
		return
	}

	if trade.MarkPaid {
		utils.RespondWithOk(w, "Trade already marked as paid")
		return
	}

	if trade.BuyerID.Hex() != userID.(string) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Trade not available to user")
		return
	}

	trade.MarkPaid = true
	trade.UpdatedAt = time.Now().UTC()

	if err := s.dao.UpdateTrade(trade); err != nil {
		log.Printf("failed to update trade %v: %v", trade.ID.Hex(), err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	// notify seller
	subject := "[Action Needed] Order marked paid"
	message := fmt.Sprintf("Order for %fQC has been marked as paid by @%s", trade.Amount, user.Username)
	data := notifications.GenericEmailData{
		Content: message,
	}

	go s.notifiable.SendGenericNotification(trade.SellerID.Hex(), subject, data)

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Trade has been marked paid",
	})
}

// CancelTrade ...
func (s *Service) CancelTrade(w http.ResponseWriter, r *http.Request) {
	tradeID := mux.Vars(r)["id"]
	if tradeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))
	// retrieve trade
	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	if trade.Status == models.TradeCancelled {
		utils.RespondWithOk(w, "Trade already marked cancelled")
		return
	}

	if trade.BuyerID.Hex() != userID.(string) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Trade not available to user")
		return
	}

	trade.Status = models.TradeCancelled
	if err := s.dao.UpdateTrade(trade); err != nil {
		log.Printf("failed to update trade %v: %v", trade.ID.Hex(), err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	utils.RespondWithJSON(w, http.StatusAccepted, utils.Response{
		Status:  "success",
		Code:    http.StatusAccepted,
		Message: "Trade has been cancelled",
	})
}

func (s *Service) processConfirmedOrder(trade models.BuyTrade) error {
	log.Printf("releasing funds for BuyTrade: %s", trade.ID.Hex())

	err := s.escrow.ReleaseDeposit(trade, trade.BuyerWallet)
	if err != nil {
		return err
	}

	return err
}

// NewMessage records a new message on a buy trade
func (s *Service) NewMessage(w http.ResponseWriter, r *http.Request) {
	var req models.NewMessageReq
	err := utils.DecodeReq(r, &req)
	if err != nil {
		log.Printf("buy_intent: error decoding req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data sent")
		return
	}

	if req.Message == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Message is empty")
		return
	}

	tradeID := mux.Vars(r)["id"]
	if tradeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	userID := r.Context().Value(models.ContextKey("user_id"))
	// retrieve trade
	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	// guard
	uid := userID.(string)
	log.Println("uid:", uid)
	if uid != trade.SellerID.Hex() && uid != trade.BuyerID.Hex() {
		utils.RespondWithError(w, http.StatusUnauthorized, "Trade not available to user")
		return
	}

	puid, _ := primitive.ObjectIDFromHex(uid)
	now := time.Now().UTC()
	msg := models.TradeChat{
		ID:        primitive.NewObjectID(),
		TradeID:   trade.ID,
		UserID:    puid,
		Message:   req.Message,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var msgType string
	if uid == trade.SellerID.Hex() && trade.BuyerID != primitive.NilObjectID {
		// msg from seller
		msgType = "msg_from:seller"
	}

	if uid == trade.BuyerID.Hex() {
		msgType = "msg_from:buyer"
	}

	if msgType == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "Operation not allowed on trade")
		return
	}

	if err := s.factoryDAO.Insert("trade_chat", msg); err != nil {
		log.Printf("err create_trade_chat: %+v", err)
		utils.RespondWithError(w, http.StatusNotFound, "An Error occurred")
		return
	}

	// dispatch notification
	var (
		subject string
		to      string
		nData   notifications.GenericEmailData
	)
	switch msgType {
	case "msg_from:seller":
		subject = "New message from seller"
		to = trade.BuyerID.Hex()
	case "msg_from:buyer":
		subject = "Trade message from buyer"
		to = trade.SellerID.Hex()
	}
	nData = notifications.GenericEmailData{
		Formatted: false,
		Content:   msg.Message,
	}
	go s.notifiable.SendGenericNotification(to, subject, nData)

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Code:    http.StatusCreated,
		Data:    msg,
		Message: "Trade chat sent",
	})
}

// GetTradeMessages ...
func (s *Service) GetTradeMessages(w http.ResponseWriter, r *http.Request) {
	tradeID := mux.Vars(r)["id"]
	if tradeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid trade ID")
		return
	}

	// retrieve trade
	trade, err := s.dao.FindTradeByID(tradeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Trade not found")
		return
	}

	messages, err := s.dao.QueryTradeMessages(trade.ID)
	if err != nil {
		log.Printf("err_q_trade_messages: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error retrieving trade chat")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.Response{
		Status: "success",
		Code:   http.StatusOK,
		Data:   messages,
	})
}

// AutoCancellationJob pools the orders collection and auto cancel orders not
// processed within set timeframe
func (s *Service) AutoCancellationJob() {
	log.Println("starting order auto cancellation job")
	now := time.Now().UTC()
	interval := now.Add(time.Minute * -10)

	for {
		log.Println("Checking for idle trades...")

		trades, err := s.dao.PoolTradesByTime(interval, "lock_time", models.TradeInProgress)
		if err != nil {
			log.Printf("error pooling orders: %v", err)
			return
		}

		for _, trade := range trades {
			log.Printf("auto cancelling Order #%s", trade.ID.Hex())
			// switch status to cancelled
			trade.Status = models.TradeCancelled
			trade.CancelReason = models.AutoCancellation
			trade.UpdatedAt = time.Now().UTC()

			// update order
			if err := s.dao.UpdateTrade(trade); err != nil {
				log.Printf("failed to update trade cancellation status: %v", err)
				return
			}
		}

		time.Sleep(time.Minute * 1)
	}
}
