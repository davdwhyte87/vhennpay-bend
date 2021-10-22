package dao

import (
	"vhennpay-bend/models"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FindPaymentOptByID retrieves a payment option matching id
func (dao *FactoryDAO) FindPaymentOptByID(id string, optType models.PaymentOption) (interface{}, error) {
	var option bson.M

	opt, err := getPaymentOptionFromType(optType)
	if err != nil {
		return nil, err
	}

	collection, ok := dao.Collections[opt]
	if !ok {
		return nil, errors.New("invalid collection type")
	}

	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	err = collection.FindOne(dao.ctx, bson.M{"_id": docID}).Decode(&option)
	return option, err
}

// FindPaymentOpt retrieves a payment matching userid & type
func (dao *FactoryDAO) FindPaymentOpt(id string, optType models.PaymentOption) (interface{}, error) {
	var options []bson.M

	opt, err := getPaymentOptionFromType(optType)
	if err != nil {
		return nil, err
	}
	collection, ok := dao.Collections[opt]
	if !ok {
		return nil, errors.New("invalid collection type")
	}

	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	cursor, err := collection.Find(dao.ctx, bson.M{
		"user_id": docID,
	})
	if err != nil {
		return nil, err
	}
	err = cursor.All(dao.ctx, &options)

	return options, err
}

func getPaymentOptionFromType(optType models.PaymentOption) (string, error) {
	switch optType {
	case models.Bank:
		return "bank_payment_option", nil
	case models.PayPal:
		return "paypal_payment_option", nil
	default:
		return "", errors.New("invalid payment option type")
	}
}
