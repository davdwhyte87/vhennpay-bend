package dao

import (
	"context"
	"vhennpay-bend/models"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FactoryDAO represents a dao for scalfolding and accessing collections
type FactoryDAO struct {
	ctx         context.Context
	db          *mongo.Database
	Collections map[string]*mongo.Collection
}

// NewFactoryDAO returns a new FactoryDAO
func NewFactoryDAO(ctx context.Context, db *mongo.Database) *FactoryDAO {
	// TODO: move to separate DAOs
	collections := []string{
		"ico_trade",
		"trade_chat",
		"support_chat",
		"currencies",
		"notifications",
		"user",
		"bank_payment_option",
		"paypal_payment_option",
		"escrow_deposits",
		"user_wallet",
	}
	dao := &FactoryDAO{
		ctx:         context.TODO(),
		db:          db,
		Collections: make(map[string]*mongo.Collection),
	}

	for _, opt := range collections {
		dao.Add(opt)
	}

	return dao
}

// Add collection to list
func (dao *FactoryDAO) Add(key string) {
	c := dao.db.Collection(key)
	dao.Collections[key] = c
}

// Insert a collection into database
func (dao *FactoryDAO) Insert(key string, obj interface{}) error {
	collection, ok := dao.Collections[key]
	if !ok {
		return errors.New("Invalid collection")
	}
	c, _ := bson.Marshal(obj)
	_, err := collection.InsertOne(dao.ctx, c)
	return err
}

// Update ...
func (dao *FactoryDAO) Update(key string, id primitive.ObjectID, obj interface{}) error {
	collection, ok := dao.Collections[key]
	if !ok {
		return errors.New("Invalid collection")
	}
	_, err := collection.UpdateOne(dao.ctx, bson.M{"_id": id}, bson.M{"$set": obj})
	return err
}

// FactoryFindUser ...
func (dao *FactoryDAO) FactoryFindUser(key string, id primitive.ObjectID) (models.User, error) {
	var obj models.User

	collection, ok := dao.Collections[key]
	if !ok {
		return obj, errors.New("Invalid collection")
	}

	err := collection.FindOne(dao.ctx, bson.M{"_id": id}).Decode(&obj)
	return obj, err
}

// Query ...
func (dao *FactoryDAO) Query(ckey string, filter bson.M, sort ...bool) (interface{}, error) {
	var data []bson.M

	opts := options.Find()
	if len(sort) > 0 {
		opts.SetSort(bson.M{"created_at": -1})
	}

	collection, ok := dao.Collections[ckey]
	if !ok {
		return nil, errors.New("Invalid collection")
	}
	cursor, err := collection.Find(dao.ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &data)

	return data, err
}

// Remove ...
func (dao *FactoryDAO) Remove(ckey string, filter bson.M) error {
	collection, ok := dao.Collections[ckey]
	if !ok {
		return errors.New("Invalid collection")
	}

	_, err := collection.DeleteOne(dao.ctx, filter)
	if err != nil {
		return err
	}

	return nil
}
