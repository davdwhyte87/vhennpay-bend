package notifications

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
)

// PushNotification dispatches a push notification to a user token
func (n *notifiable) PushNotification(recipientToken, title, message string) error {
	if recipientToken == "" {
		return nil
	}

	ctx := context.Background()
	client, err := n.app.Messaging(ctx)
	if err != nil {
		return err
	}
	msg := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  message,
		},
		Token: recipientToken,
	}

	response, err := client.Send(ctx, msg)
	if err != nil {
		return err
	}

	log.Printf("Successfully sent message: %v", response)
	return nil
}
