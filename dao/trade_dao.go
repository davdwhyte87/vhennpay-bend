package dao

import (
	"vhennpay-bend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: move to instance DAO value
// TODO: refactor all DAOs to make them DRY

// InsertTrade ...
func (dao *OrderDAO) InsertTrade(trade models.BuyTrade) error {
	collection := dao.db.Collection("buy_trade")
	obj, _ := bson.Marshal(trade)
	_, err := collection.InsertOne(dao.ctx, obj)

	return err
}

// FindTradeByID retrieves a buy trade by its id
func (dao *OrderDAO) FindTradeByID(id string) (models.BuyTrade, error) {
	var trade models.BuyTrade

	collection := dao.db.Collection("buy_trade")
	docID, _ := primitive.ObjectIDFromHex(id)
	err := collection.FindOne(dao.ctx, bson.M{"_id": docID}).Decode(&trade)
	return trade, err
}

// UpdateTrade an existing order
func (dao *OrderDAO) UpdateTrade(trade models.BuyTrade) error {
	collection := dao.db.Collection("buy_trade")
	_, err := collection.UpdateOne(dao.ctx, bson.M{"_id": trade.ID}, bson.M{"$set": trade})
	return err
}

// PoolTradesByTime ...
func (dao *OrderDAO) PoolTradesByTime(interval time.Time, field, status string) ([]models.BuyTrade, error) {
	var trades []models.BuyTrade
	collection := dao.db.Collection("buy_trade")

	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(dao.ctx, bson.M{
		field: bson.M{
			"$ne":  time.Time{}.UTC(),
			"$lte": interval,
		},
		"status": status,
	}, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &trades)
	defer cursor.Close(dao.ctx)

	return trades, err
}

// QueryTrades takes a bson.M filters map and applies the query on the orders collection
func (dao *OrderDAO) QueryTrades(filter bson.M) ([]models.BuyTrade, error) {
	var trades []models.BuyTrade
	collection := dao.db.Collection("buy_trade")

	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(dao.ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &trades)

	return trades, err
}

// QueryTradeMessages ...
func (dao *OrderDAO) QueryTradeMessages(id primitive.ObjectID) ([]models.TradeChat, error) {
	var messages []models.TradeChat

	collection := dao.db.Collection("trade_chat")
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	filter := bson.M{
		"trade_id": id,
	}

	cursor, err := collection.Find(dao.ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &messages)
	return messages, err
}
