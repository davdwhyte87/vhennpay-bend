package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/mailgun/mailgun-go/v3"
	"gopkg.in/gomail.v2"
)

// EmailData represents the data format for emails
type EmailData struct {
	Title       string
	ContentData interface{}
	EmailTo     string
	Template    string
}

// SendEmail ...
func SendEmail(data EmailData) error {
	template, err := template.ParseFiles("utils/email_template/" + data.Template)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = template.Execute(&buf, data.ContentData)

	if err != nil {
		return err
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_PRIVATE_KEY"))
	message := mg.NewMessage(
		fmt.Sprintf("Dils <%s>", os.Getenv("MAIL_FROM")),
		data.Title,
		"Sent from Dils",
		data.EmailTo,
	)
	message.SetHtml(buf.String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Send the message with a 10 second timeout
	_, _, err = mg.Send(ctx, message)

	if err != nil {
		return err
	}

	return nil
}

func sendGoMail(data EmailData) error {
	template, err := template.ParseFiles("utils/email_template/" + data.Template)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = template.Execute(&buf, data.ContentData)

	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("MAIL_FROM"))
	m.SetHeader("To", data.EmailTo)
	m.SetHeader("Subject", data.Title)
	m.SetBody("text/html", buf.String())

	d := gomail.NewDialer("smtp.gmail.com", 587, os.Getenv("EMAIL_SENDER"), os.Getenv("EMAIL_SENDER_PASS"))
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return (err)
	}

	return nil
}
