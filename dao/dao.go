package dao

import (
	"context"
	
	// "os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// DAO is the base interface for accessing data
type DAO interface {
}

// Initialize a connection
func Initialize(dbURI, user, serverPass, dbname string) (*mongo.Client, context.Context, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client()
	clientOpts.SetAuth(options.Credential{
		AuthMechanism: "SCRAM-SHA-1",
		Username:      user,
		Password:      serverPass,
	})
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://david:.r!kFvZDu5D69$S@vhennpaytest.q7zsi.mongodb.net/vhennpaytest?retryWrites=true&w=majority"))
	if err != nil {
		return nil, nil, err
	}

	// ping primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, err
	}

	return client, ctx, nil
}
