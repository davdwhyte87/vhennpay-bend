package escrow

import (
	"bytes"
	"context"
	"vhennpay-bend/models"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TODO: extract db calls to DAO methods

// Lid network API endpoints
var (
	transferEndpoint string
)

// Escrow represents the escrow service
type Escrow struct {
	db *mongo.Database
}

// InitEscrow ...
func InitEscrow(db *mongo.Database) *Escrow {
	base := fmt.Sprintf("%s/api/v1", os.Getenv("LID_SERVER_ADDR"))
	transferEndpoint = base + "/wallet/transfer"
	log.Println(transferEndpoint)
	return &Escrow{db}
}

// NewDeposit deposits N amount in escrow for Order O
func (e *Escrow) NewDeposit(orderID, userID primitive.ObjectID, amount float64, walletID, walletSecret string) error {
	escrow := models.EscrowDeposit{
		ID:           primitive.NewObjectID(),
		OrderID:      orderID,
		UserID:       userID,
		SourceWallet: walletID,
		Amount:       amount,
		CreatedAt:    time.Now().UTC(),
	}

	// post transaction to chain
	escrowWallet := os.Getenv("ESCROW_WALLET")
	payload := map[string]string{
		"sender_address":     walletID,
		"reciever_address":   escrowWallet,
		"amount":             strconv.FormatFloat(amount, 'f', -1, 64),
		"sender_private_key": walletSecret,
	}
	resp, err := postToChain(transferEndpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var d map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&d)
		if err != nil {
			log.Printf("failed to post tx to chain: %v", err)
			return errors.New("Failed to read transaction data from the QUI chain")
		}
		log.Printf("error posting to chain, resp: %v", d)
		return errors.New("Error posting transaction to QUI chain: " + d["error"].(string))
	}

	b, _ := bson.Marshal(escrow)
	_, err = e.db.Collection("escrow").InsertOne(context.TODO(), b)
	return err
}

// ReverseDeposit reverses the deposited amount in escrow back to user's source
// wallet
// Actions like cancel trade will trigger this method
func (e *Escrow) ReverseDeposit(order models.SellOrder) error {
	var escrow models.EscrowDeposit
	escrowWallet := os.Getenv("ESCROW_WALLET")
	escrowWalletSecret := os.Getenv("ESCROW_WALLET_SECRET")

	// retrieve deposit
	err := e.db.Collection("escrow").FindOne(context.TODO(), bson.M{
		"order_id": order.ID,
		"user_id":  order.CreatedBy,
	}).Decode(&escrow)
	if err != nil {
		return err
	}

	// confirm escrow hasn't been released
	if escrow.Released {
		return errors.New("Operation not allowed, escrow has already been released")
	}

	// build payload and transfer amount
	reversalAmount := escrow.Amount - escrow.ReleasedAmount

	payload := map[string]string{
		"sender_address":     escrowWallet,
		"reciever_address":   order.WalletID,
		"amount":             strconv.FormatFloat(reversalAmount, 'f', -1, 64),
		"sender_private_key": escrowWalletSecret,
	}

	resp, err := postToChain(transferEndpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var d map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&d)
		if err != nil {
			log.Printf("failed to post tx to chain: %v", err)
			return errors.New("Failed to read transaction data from the QUI chain")
		}
		log.Printf("error posting to chain, resp: %v", d)
		return errors.New("Error posting transaction to QUI chain: " + d["error"].(string))
	}

	now := time.Now().UTC()
	escrow.Released = true
	escrow.UpdatedAt = now

	_, err = e.db.Collection("escrow").
		UpdateOne(context.TODO(), bson.M{"_id": escrow.ID}, bson.M{"$set": escrow})
	if err != nil {
		return err
	}

	// create escrow release for reversal
	escrowRelease := models.EscrowRelease{
		ID:            primitive.NewObjectID(),
		ParentID:      escrow.ID,
		Recipient:     order.CreatedBy,
		Amount:        reversalAmount,
		WalletAddress: order.WalletID,
		CreatedAt:     now,
	}

	b, _ := bson.Marshal(escrowRelease)
	_, err = e.db.Collection("escrow_release").InsertOne(context.TODO(), b)

	return err
}

// ReleaseDeposit releases an escrowed amount to the trade receipient
func (e *Escrow) ReleaseDeposit(trade models.BuyTrade, receipient string) error {
	var escrow models.EscrowDeposit
	escrowWallet := os.Getenv("ESCROW_WALLET")
	escrowWalletSecret := os.Getenv("ESCROW_WALLET_SECRET")

	// retrieve deposit
	err := e.db.Collection("escrow").FindOne(context.TODO(), bson.M{
		"order_id": trade.OrderID,
		"user_id":  trade.SellerID,
	}).Decode(&escrow)
	if err != nil {
		return err
	}

	// skip if already released
	if escrow.Released {
		return errors.New("Escrow already released")
	}

	if (escrow.Amount - escrow.ReleasedAmount) < trade.Amount {
		return errors.New("Deposit in escrow not enough to cover transaction")
	}

	payload := map[string]string{
		"sender_address":     escrowWallet,
		"reciever_address":   receipient,
		"amount":             strconv.FormatFloat(trade.Amount, 'f', -1, 64),
		"sender_private_key": escrowWalletSecret,
	}

	resp, err := postToChain(transferEndpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var d map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&d)
		if err != nil {
			log.Printf("failed to post tx to chain: %v", err)
			return errors.New("Failed to read transaction data from the QUI chain")
		}
		return fmt.Errorf("Error posting to chain, resp: %v", d)
	}

	b, _ := ioutil.ReadAll(resp.Body)
	log.Printf("escrow: tx posted to chain, resp: %v", string(b))

	// mark as released
	escrow.ReleasedAmount = escrow.ReleasedAmount + trade.Amount
	if (escrow.Amount - escrow.ReleasedAmount) == 0 {
		escrow.Released = true
	}
	now := time.Now().UTC()
	escrow.UpdatedAt = now

	_, err = e.db.Collection("escrow").UpdateOne(context.TODO(), bson.M{"_id": escrow.ID}, bson.M{"$set": escrow})
	if err != nil {
		return err
	}

	// create escrow release for trade
	escrowRelease := models.EscrowRelease{
		ID:            primitive.NewObjectID(),
		ParentID:      escrow.ID,
		TradeID:       trade.ID,
		Recipient:     trade.BuyerID,
		Amount:        trade.Amount,
		WalletAddress: receipient,
		CreatedAt:     now,
	}

	b, _ = bson.Marshal(escrowRelease)
	_, err = e.db.Collection("escrow_release").InsertOne(context.TODO(), b)

	return err
}

// GenCurrencyList ...
func (e *Escrow) GenCurrencyList() {
	dump := `
{
  "AUD": "Australian Dollar",
  "CHF": "Swiss Franc",
  "EUR": "Euro",
  "GBP": "British Pound Sterling",
  "GHS": "Ghanaian Cedi",
  "JPY": "Japanese Yen",
  "NGN": "Nigerian Naira",
  "SGD": "Singapore Dollar",
  "USD": "United States Dollar"
}
	`
	var d map[string]string
	_ = json.Unmarshal([]byte(dump), &d)
	for k, v := range d {
		go func(k, s string) {
			var d = map[string]string{
				"name":  s,
				"value": k,
			}
			b, _ := bson.Marshal(d)
			_, err := e.db.Collection("currencies").InsertOne(context.Background(), b)
			if err != nil {
				log.Printf("create_cur_err: %v", err)
			}
		}(k, v)
	}

}

func postToChain(endpoint string, payload interface{}) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	return client.Do(req)
}
