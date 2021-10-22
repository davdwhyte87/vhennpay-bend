package dao

import (
	"vhennpay-bend/models"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FindNotificationByID ...
func (dao *FactoryDAO) FindNotificationByID(id string) (interface{}, error) {
	var notification bson.M

	collection, ok := dao.Collections["notifications"]
	if !ok {
		return nil, errors.New("invalid collection type")
	}

	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	err = collection.FindOne(dao.ctx, bson.M{"_id": docID}).Decode(&notification)
	return notification, err
}

// QueryNotifications ...
func (dao *FactoryDAO) QueryNotifications(filter bson.M) ([]models.Notification, error) {
	var notifications []models.Notification
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	collection, ok := dao.Collections["notifications"]
	if !ok {
		return nil, errors.New("invalid collection type")
	}

	cursor, err := collection.Find(dao.ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &notifications)

	return notifications, err
}
