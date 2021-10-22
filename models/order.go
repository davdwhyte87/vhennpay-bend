package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Order statuses
const (
	OrderPending   = "pending"
	OrderCancelled = "cancelled"
	OrderCompleted = "completed"
)

// Trade statuses
const (
	TradeInProgress = "in-progress"
	TradePending    = "pending"
	TradeProcessed  = "processed"
	TradeCancelled  = "cancelled"
)

// CancelReason ...
type CancelReason uint

// Cancel reasons
const (
	ManualCancellation CancelReason = iota
	AutoCancellation
)

// SellOrder ...
type SellOrder struct {
	ID                primitive.ObjectID `json:"id" bson:"_id"`
	CreatedBy         primitive.ObjectID `json:"created_by" bson:"created_by"`
	ExRate            float64            `json:"ex_rate" bson:"ex_rate"`
	Amount            float64            `json:"amount" bson:"amount"`
	AmountSold        float64            `json:"amount_sold" bson:"amount_sold"`
	AmountLeft        float64            `json:"amount_left" bson:"amount_left"`
	Currency          string             `json:"currency" bson:"currency"`
	PhoneNumber       string             `json:"phone_number" bson:"phone_number"`
	WalletID          string             `json:"wallet_id" bson:"wallet_id"`
	PaymentOption     int32              `json:"payment_option" bson:"payment_option"`
	PaymentOptionID   primitive.ObjectID `json:"payment_option_id" bson:"payment_option_id"`
	PaymentOptionData interface{}        `json:"payment_option_data" bson:"-"`
	Note              string             `json:"note" bson:"note"`
	Status            string             `json:"status" bson:"status"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
}

// BuyTrade represents an initiated sell trade
type BuyTrade struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	SellerID    primitive.ObjectID `json:"seller_id" bson:"seller_id"`
	BuyerID     primitive.ObjectID `json:"buyer_id" bson:"buyer_id"`
	OrderID     primitive.ObjectID `json:"order_id" bson:"order_id"`
	BuyerWallet string             `json:"buyer_wallet" bson:"buyer_wallet"`
	Amount      float64            `json:"amount" bson:"amount"`
	Confirmed   bool               `json:"confirmed" bson:"confirmed"`
	MarkPaid    bool               `json:"mark_paid" bson:"mark_paid"`
	Rating      uint               `json:"rating" bson:"rating"`
	// LockTime to indicate when order was shown interest (buy/sell interest-action)
	LockTime     time.Time `json:"lock_time" bson:"lock_time"`
	Status       string    `json:"status" bson:"status"`
	ProcessedAt  time.Time `json:"processed_at" bson:"processed_at"`
	CancelReason `json:"cancel_reason" bson:"cancel_reason"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

// NewMessageReq ...
type NewMessageReq struct {
	Message string
}

// SellOrderReq ...
type SellOrderReq struct {
	ExRate           float64 `json:"ex_rate"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	PhoneNumber      string  `json:"phone_number"`
	WalletID         string  `json:"wallet_id" `
	PaymentOption    int32   `json:"payment_option"`
	PaymentOptionID  string  `json:"payment_option_id"`
	WalletPrivateKey string  `json:"wallet_private_key"`
	Note             string  `json:"note"`
}

// CancelOrderReq ...
type CancelOrderReq struct {
	Reason CancelReason `json:"reason"`
}

// CreateBuyTradeReq represents the request payload to buy from a sell trade
type CreateBuyTradeReq struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	WalletID string  `json:"wallet_id"`
}
