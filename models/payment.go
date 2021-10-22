package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PaymentOption represents a payment option
type PaymentOption uint

const (
	// Bank payment option
	Bank PaymentOption = iota
	// PayPal payment option
	PayPal
	// Stripe payment option
	Stripe
)

// PaymentOptionReq represents the payment_option create request payload
type PaymentOptionReq struct {
	Type          PaymentOption `json:"type"`
	Email         string        `json:"email"`
	AccountName   string        `json:"account_name"`
	AccountNumber string        `json:"account_number"`
	BankName      string        `json:"bank_name"`
	SortCode      string        `json:"sort_code"`
}

// GetPaymentOptionReq ...
type GetPaymentOptionReq struct {
	Type PaymentOption `json:"type"`
	//UserID string        `json:"user_id"`
}

// PayPalOption ...
type PayPalOption struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Email     string             `json:"email" bson:"email"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// BankOption ...
type BankOption struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	AccountName   string             `json:"account_name" bson:"account_name"`
	AccountNumber string             `json:"account_number" bson:"account_number"`
	BankName      string             `json:"bank_name" bson:"bank_name"`
	SortCode      string             `json:"sort_code" bson:"sort_code"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

// PayPalPayment ...
type PayPalPayment struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id"`
	OrderID            string             `json:"order_id" bson:"order_id"`
	WalletAddress      string             `json:"wallet_address" bson:"wallet_address"`
	Amount             float64            `json:"amount" bson:"amount"`
	TransactionPayload string             `json:"transaction_payload" bson:"transaction_payload"`
	CreatedAt          time.Time          `json:"created_at" bson:"created_at"`
}
