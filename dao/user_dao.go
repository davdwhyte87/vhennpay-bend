package dao

import (
	"context"
	"vhennpay-bend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserDAO represents a user DAO
type UserDAO struct {
	ctx        context.Context
	db         *mongo.Database
	Collection *mongo.Collection
}

// NewUserDAO returns a configured UserDAO
func NewUserDAO(ctx context.Context, db *mongo.Database) *UserDAO {
	return &UserDAO{
		ctx:        context.TODO(),
		db:         db,
		Collection: db.Collection("user"),
	}
}

// FindAll get list of users
func (dao *UserDAO) FindAll() ([]models.User, error) {
	var users []models.User
	cursor, err := dao.Collection.Find(dao.ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	err = cursor.All(dao.ctx, &users)
	return users, err
}

// FindByID ... get a user by its id
func (dao *UserDAO) FindByID(id string) (models.User, error) {
	var user models.User
	docID, _ := primitive.ObjectIDFromHex(id)
	err := dao.Collection.FindOne(dao.ctx, bson.M{"_id": docID}).Decode(&user)
	return user, err
}

// FindByEmail ... get a user by the email
func (dao *UserDAO) FindByEmail(email string) (models.User, error) {
	var user models.User
	err := dao.Collection.FindOne(dao.ctx, bson.M{"email": email}).Decode(&user)
	return user, err
}

// Insert a user into database
func (dao *UserDAO) Insert(user models.User) error {
	userb, _ := bson.Marshal(user)
	_, err := dao.Collection.InsertOne(dao.ctx, userb)
	return err
}

// Update an existing user
func (dao *UserDAO) Update(user models.User) error {
	docID, _ := primitive.ObjectIDFromHex(user.ID.Hex())
	_, err := dao.Collection.UpdateOne(dao.ctx, bson.M{"_id": docID}, bson.M{"$set": user})
	return err
}
