package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents an app user
type User struct {
	ID              primitive.ObjectID `json:"id" bson:"_id"`
	Username        string             `json:"username" bson:"username"`
	Email           string             `json:"email" bson:"email"`
	Password        string             `json:"-" bson:"password"`
	FCMToken        string             `json:"fcm_token" bson:"fcm_token"`
	Confirmed       bool               `json:"confirmed" bson:"confirmed"`
	PassCode        int                `json:"-" bson:"pass_code"`
	PositiveRatings int                `json:"positive_ratings" bson:"positive_ratings"`
	NegativeRatings int                `json:"negative_ratings" bson:"negative_ratings"`
	NumTransactions int                `json:"num_transactions" bson:"num_transactions"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

// UserWallet ...
type UserWallet struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Address   string             `json:"address" bson:"address"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// UserWalletReq ...
type UserWalletReq struct {
	Address string `json:"address"`
}

// CreateUserReq represents the request model for signup
type CreateUserReq struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginReq represents the login request
type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ConfirmAccountReq represents a confirm account request
type ConfirmAccountReq struct {
	Email string `json:"email"`
	Code  int    `json:"code"`
}

// PasswordResetReq ...
type PasswordResetReq struct {
	Email string `json:"email"`
}

// PasswordReset represents a password request request
type PasswordReset struct {
	Email    string `json:"email"`
	Code     int    `json:"code"`
	Password string `json:"password"`
	//ConfirmPassword string `json:"confirm_password"`
}

// FCMTokenReq ...
type FCMTokenReq struct {
	Token string `json:"token"`
}

// RatingType ...
type RatingType uint

// Rating types
const (
	PositiveRating RatingType = iota
	NegativeRating
)

// RateUserReq ...
type RateUserReq struct {
	Type RatingType `json:"type"`
}

// NewSupportChatReq ...
type NewSupportChatReq struct {
	Message    string `json:"message"`
	SentByUser bool   `json:"sent_by_user"`
}
