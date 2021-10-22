package notifications

import (
	"vhennpay-bend/utils"
)

// GenericOrderData represents the OrderCreated email notification data
type GenericOrderData struct {
	OrderID string
	Amount  float64
	Name    string
	Seller  string
}

// GenericEmailData ...
type GenericEmailData struct {
	Formatted bool
	Content   string
}

// BuyIntentData ...
type BuyIntentData struct {
	Name          string
	OrderID       string
	Amount        float64
	BuyerUsername string
}

// SendOrderCreatedMail ...
func SendOrderCreatedMail(to string, data GenericOrderData) error {
	subject := "New Order Created"
	err := send(to, subject, "order_created.html", data)
	return err
}

// SendGenericMail ...
func SendGenericMail(to, subject string, data GenericEmailData) error {
	return send(to, subject, "generic.html", data)
}

// SendOrderConfirmedMail ...
func SendOrderConfirmedMail(to string, data GenericOrderData) error {
	subject := "Order Confirmed"
	err := send(to, subject, "order_confirmed.html", data)

	return err
}

// SendBuyIntentMail ...
func SendBuyIntentMail(to string, data BuyIntentData) error {
	subject := "Posted trade has a buyer"
	err := send(to, subject, "trade_registered.html", data)

	return err
}

func send(to, subject, temp string, data interface{}) error {
	payload := utils.EmailData{
		Title:       subject,
		ContentData: data,
		Template:    temp,
		EmailTo:     to,
	}

	return utils.SendEmail(payload)
}
