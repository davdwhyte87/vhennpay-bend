package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationType ...
type NotificationType string

// NotificationActionType ...
type NotificationActionType string

// Notification types
const (
	TradeN   NotificationType = "trade"
	OrderN                    = "order"
	AccountN                  = "account"
)

// Notification action types
const (
	AInfo      NotificationActionType = "info"
	APayment   NotificationActionType = "payment"
	ACompleted NotificationActionType = "completed"
)

// Notification represents an actionable/non-actionable notification model
type Notification struct {
	ID        primitive.ObjectID     `json:"id" bson:"_id"`
	Title     string                 `json:"title" bson:"title"`
	OrderID   primitive.ObjectID     `json:"order_id" bson:"order_id"`
	UserID    primitive.ObjectID     `json:"user_id" bson:"user_id"`
	Type      NotificationType       `json:"type" bson:"type"`
	Message   string                 `json:"message" bson:"message"`
	Action    NotificationActionType `json:"action" bson:"action"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
}
