package dao

import (
	"context"
	"vhennpay-bend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OrderDAO represents an order DAO
type OrderDAO struct {
	ctx        context.Context
	db         *mongo.Database
	Collection *mongo.Collection
}

// NewOrderDAO returns a new OrderDAO
func NewOrderDAO(ctx context.Context, db *mongo.Database) *OrderDAO {
	return &OrderDAO{
		ctx:        context.TODO(),
		db:         db,
		Collection: db.Collection("orders"),
	}
}

// Insert an order into database
func (dao *OrderDAO) Insert(order models.SellOrder) error {
	obj, _ := bson.Marshal(order)
	_, err := dao.Collection.InsertOne(dao.ctx, obj)
	return err
}

// Query takes a bson.M filters map and applies the query on the orders collection
func (dao *OrderDAO) Query(filter bson.M) ([]models.SellOrder, error) {
	var orders []models.SellOrder

	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	cursor, err := dao.Collection.Find(dao.ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &orders)

	return orders, err
}

// PipelineAll ...
func (dao *OrderDAO) PipelineAll(query bson.M) (interface{}, error) {
	var orders []bson.M

	matches := bson.M{
		"$match": query,
	}

	sort := bson.M{
		"$sort": bson.M{"created_at": -1},
	}

	lookup := bson.M{
		"$lookup": bson.M{
			"from":         "user",
			"localField":   "created_by",
			"foreignField": "_id",
			"as":           "user_data",
		},
	}
	unwind := bson.M{
		"$unwind": "$user_data",
	}

	project := bson.M{
		"$project": bson.M{
			"user_data.password":  0,
			"user_data.confirmed": 0,
			"user_data.fcm_token": 0,
			"user_data.passcode":  0,
		},
	}

	pipeline := []bson.M{matches, sort, lookup, unwind, project}
	cursor, err := dao.Collection.Aggregate(dao.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &orders)

	return orders, err
}

// PipelineSingle ...
func (dao *OrderDAO) PipelineSingle(id string, query bson.M) (interface{}, error) {
	var orders []bson.M

	var matches = make(bson.M)
	if query != nil {
		matches = bson.M{
			"$match": query,
		}
	} else {
		docID, _ := primitive.ObjectIDFromHex(id)
		matches = bson.M{
			"$match": bson.M{"_id": docID},
		}
	}

	lookup := bson.M{
		"$lookup": bson.M{
			"from":         "user",
			"localField":   "created_by",
			"foreignField": "_id",
			"as":           "user_data",
		},
	}
	unwind := bson.M{
		"$unwind": "$user_data",
	}

	project := bson.M{
		"$project": bson.M{
			"user_data.password":  0,
			"user_data.confirmed": 0,
			"user_data.fcm_token": 0,
			"user_data.passcode":  0,
		},
	}

	pipeline := []bson.M{matches, lookup, unwind, project}
	cursor, err := dao.Collection.Aggregate(dao.ctx, pipeline)
	if err != nil {
		return models.SellOrder{}, err
	}

	err = cursor.All(dao.ctx, &orders)

	if len(orders) < 1 {
		return models.SellOrder{}, err
	}

	return orders[0], err
}

// PoolQueryByTime ...
func (dao *OrderDAO) PoolQueryByTime(interval time.Time, field, status string) ([]models.SellOrder, error) {
	var orders []models.SellOrder

	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	cursor, err := dao.Collection.Find(dao.ctx, bson.M{
		field: bson.M{
			"$exists": true,
			"$lte":    interval,
		},
		"status": status,
	}, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &orders)

	return orders, err
}

// FindByID retrieves an order by its id
func (dao *OrderDAO) FindByID(id string) (models.SellOrder, error) {
	var order models.SellOrder
	docID, _ := primitive.ObjectIDFromHex(id)
	err := dao.Collection.FindOne(dao.ctx, bson.M{"_id": docID}).Decode(&order)
	return order, err
}

// Update an existing order
func (dao *OrderDAO) Update(order models.SellOrder) error {
	docID, _ := primitive.ObjectIDFromHex(order.ID.Hex())
	_, err := dao.Collection.UpdateOne(dao.ctx, bson.M{"_id": docID}, bson.M{"$set": order})
	return err
}
