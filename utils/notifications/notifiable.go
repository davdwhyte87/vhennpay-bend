package notifications

import (
	"context"
	"vhennpay-bend/dao"
	"vhennpay-bend/models"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// Notifiable defines the functionality of a notification object
type Notifiable interface {
	// Dispatches a push notification to currently configured message server (FCM)
	PushNotification(recipientToken, title, message string) error
	SendOrderCreatedNotification(order models.SellOrder, userid string)
	SendOrderIntentNotification(trade models.BuyTrade, buyerid, sellerid string)
	SendOrderConfirmedNotification(trade models.BuyTrade, buyerid string)
	SendGenericNotification(userid, subject string, data GenericEmailData)
}

type notifiable struct {
	app        *firebase.App
	factoryDAO *dao.FactoryDAO
}

// NewNotifiable returns a new Notifiable implementation with access to all
// notifiable objects (email, fcm)
func NewNotifiable(dao *dao.FactoryDAO) (Notifiable, error) {
	serviceAccountKeyPath := os.Getenv("SERVICE_ACCOUNT_KEY_PATH")
	opt := option.WithCredentialsFile(serviceAccountKeyPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, err
	}

	return &notifiable{app: app, factoryDAO: dao}, nil
}
