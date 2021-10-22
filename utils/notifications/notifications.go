package notifications

import (
	"context"
	"vhennpay-bend/models"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func cErr(tag string, err error) {
	if err != nil {
		log.Printf("%s: %v", tag, err)
		return
	}
}

// SendGenericNotification ...
func (n *notifiable) SendGenericNotification(userid, subject string, data GenericEmailData) {
	user, err := n.getUser(userid)
	cErr("rtv_user", err)

	err = SendGenericMail(user.Email, subject, data)
	cErr("err_send_generic_mail", err)

	if data.Formatted {
		return
	}

	err = n.PushNotification(user.FCMToken, subject, data.Content)
	cErr("err_send_generic_PN", err)

	// TODO: persit generic notification?
}

// SendOrderCreatedNotification ...
func (n *notifiable) SendOrderCreatedNotification(order models.SellOrder, userid string) {
	user, err := n.getUser(userid)
	cErr("rtv_user", err)

	data := GenericOrderData{OrderID: order.ID.Hex(), Amount: order.Amount, Name: user.Username}
	message := fmt.Sprintf(orderCreatedMsg, order.ID.Hex())

	err = SendOrderCreatedMail(user.Email, data)
	cErr("err_send_order_created_mail: %v", err)

	err = n.PushNotification(user.FCMToken, orderCreatedTitle, message)
	cErr("err_order_created_PN", err)

	// store notification object
	n.persit(orderCreatedTitle, message, order.ID.Hex(), userid, models.AInfo, models.OrderN)
}

// SendOrderIntentNotification ...
func (n *notifiable) SendOrderIntentNotification(trade models.BuyTrade, buyerid, sellerid string) {
	buyer, err := n.getUser(buyerid)
	cErr("rtv_buyer", err)

	seller, err := n.getUser(sellerid)
	cErr("rtv_seller", err)

	message := fmt.Sprintf(orderNewIntentMsg, buyer.Username, trade.Amount)

	data := BuyIntentData{Name: seller.Username, OrderID: trade.OrderID.Hex(), BuyerUsername: buyer.Username, Amount: trade.Amount}
	err = SendBuyIntentMail(seller.Email, data)
	cErr("err_send_buyintent_mail", err)

	err = n.PushNotification(seller.FCMToken, orderNewIntentTitle, message)
	cErr("err_buyintent_PN", err)

	// store notification object
	n.persit(orderNewIntentTitle, message, trade.OrderID.Hex(), sellerid, models.AInfo, models.OrderN)
}

// SendOrderConfirmedNotification ...
func (n *notifiable) SendOrderConfirmedNotification(trade models.BuyTrade, buyerid string) {
	buyer, err := n.getUser(buyerid)
	cErr("rtv_buyer", err)
	seller, err := n.getUser(trade.SellerID.Hex())
	cErr("rtv_seller", err)

	message := fmt.Sprintf("Your trade for %fQC has been marked as confirmed", trade.Amount)
	data := GenericOrderData{Name: buyer.Username, Amount: trade.Amount, Seller: seller.Username}
	err = SendOrderConfirmedMail(buyer.Email, data)
	cErr("err_order_confirmed_mail", err)

	err = n.PushNotification(buyer.FCMToken, orderConfirmedTitle, message)
	cErr("err_order_confirmed_PN", err)

	// store notification object
	n.persit(orderConfirmedTitle, message, trade.OrderID.Hex(), buyerid, models.ACompleted, models.OrderN)
}

// private

func (n *notifiable) getUser(id string) (models.User, error) {
	var user models.User

	collection, ok := n.factoryDAO.Collections["user"]
	if !ok {
		return user, errors.New("User collection not retrieable from factoryDAO")
	}

	docID, _ := primitive.ObjectIDFromHex(id)
	err := collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&user)

	return user, err
}

func (n *notifiable) persit(
	title,
	message,
	orderID,
	userID string,
	action models.NotificationActionType,
	_type models.NotificationType,
) {

	pOrderID, _ := primitive.ObjectIDFromHex(orderID)
	pUserID, _ := primitive.ObjectIDFromHex(userID)

	notification := models.Notification{
		ID:        primitive.NewObjectID(),
		Title:     title,
		OrderID:   pOrderID,
		UserID:    pUserID,
		Action:    action,
		Type:      _type,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}

	err := n.factoryDAO.Insert("notifications", notification)
	cErr("err_persist_notification", err)
}
