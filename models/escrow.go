package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EscrowDeposit represents a summary of an order token deposit to escrow
type EscrowDeposit struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	OrderID        primitive.ObjectID `json:"order_id" bson:"order_id"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	SourceWallet   string             `json:"source_wallet" bson:"source_wallet"`
	Amount         float64            `json:"amount" bson:"amount"`
	ReleasedAmount float64            `json:"released_amount" bson:"released_amount"`
	Released       bool               `json:"released" bson:"released"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

// EscrowRelease represents an escrow fund release to a trade
type EscrowRelease struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	ParentID      primitive.ObjectID `json:"parent_id" bson:"parent_id"`
	TradeID       primitive.ObjectID `json:"trade_id" bson:"trade_id"`
	Recipient     primitive.ObjectID `json:"recipient" bson:"recipient"`
	Amount        float64            `json:"amount" bson:"amount"`
	WalletAddress string             `json:"wallet_address" bson:"wallet_address"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}
