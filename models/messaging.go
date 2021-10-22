package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TradeChat ...
type TradeChat struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	TradeID   primitive.ObjectID `json:"trade_id" bson:"trade_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Message   string             `json:"message" bson:"message"`
	ImageURL  string             `json:"image_url"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// SupportChat ...
type SupportChat struct {
	ID         primitive.ObjectID `json:"id" bson:"_id"`
	UserID     primitive.ObjectID `json:"user_id" bson:"user_id"`
	Username   string             `json:"username" bson:"username"`
	Message    string             `json:"message" bson:"message"`
	SentByUser bool               `json:"sent_by_user" bson:"sent_by_user"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}
