package callbacks

import (
	"context"
	"vhennpay-bend/models"
	"vhennpay-bend/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/plutov/paypal/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConfirmPaypalPayment confirms a PayPal orderID and validates the given order
// has a matching amount.
// On confirmation, the value in Quicoins will be transferred to the wallet set
// in the payload
func (s *Service) ConfirmPaypalPayment(w http.ResponseWriter, r *http.Request) {
	var (
		base         = paypal.APIBaseSandBox
		payload      map[string]interface{}
		clientID     = os.Getenv("PAYPAL_CLIENT_ID")
		clientSecret = os.Getenv("PAYPAL_CLIENT_SECRET")
	)
	err := utils.DecodeReq(r, &payload)
	if err != nil {
		log.Printf("err decoding req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data detected")
		return
	}

	if os.Getenv("ENV") != "dev" {
		base = paypal.APIBaseLive
	}

	c, err := paypal.NewClient(clientID, clientSecret, base)
	if err != nil {
		log.Printf("err validating client req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data detected")
		return
	}

	accessToken, err := c.GetAccessToken(context.Background())
	if err != nil {
		log.Printf("err validating accessToken req: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data detected")
		return
	}
	c.SetAccessToken(accessToken.Token)

	payment := payload["payment"].(map[string]interface{})
	orderID := payment["id"]
	walletAddress := payload["walletAddress"]

	purchaseUnits := payload["payment"].(map[string]interface{})["purchase_units"]
	purchaseUnit := purchaseUnits.([]interface{})
	amountUnit := purchaseUnit

	// confirm order hasn't been processed
	paypalOrder, err := s.factoryDAO.Query("ico_trade", bson.M{"order_id": orderID.(string)}, false)
	if err != nil {
		log.Printf("err retrieving paypalOrder: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error processing request")
		return
	}

	if len(paypalOrder.([]bson.M)) > 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Payment already processed")
		return
	}

	order, err := c.GetOrder(context.Background(), orderID.(string))
	if err != nil {
		log.Printf("err retrieving order: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error validating order")
		return
	}

	amountSet := amountUnit[0].(map[string]interface{})["amount"]
	amount := amountSet.(map[string]interface{})["value"]
	if order.PurchaseUnits[0].Amount.Value != amount.(string) {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid order detected")
		return
	}

	// save payment data
	a, _ := strconv.ParseFloat(amount.(string), 64)
	payloadByts, _ := json.Marshal(payload)
	orderData := models.PayPalPayment{
		ID:                 primitive.NewObjectID(),
		OrderID:            orderID.(string),
		WalletAddress:      walletAddress.(string),
		Amount:             a,
		TransactionPayload: string(payloadByts),
		CreatedAt:          time.Now().UTC(),
	}

	if err := s.factoryDAO.Insert("ico_trade", orderData); err != nil {
		log.Printf("err inserting orderData: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request data detected")
		return
	}

	err = s.fundWallet(walletAddress.(string), a)
	if err != nil {
		log.Printf("err releasing funds to ico_trade wallet: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Error validating order")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, utils.Response{
		Status:  "success",
		Code:    http.StatusCreated,
		Message: "Order confirmed",
	})
}

func (s *Service) fundWallet(wallet string, amount float64) error {
	var priceData map[string]interface{}
	resp, err := http.Get(os.Getenv("LID_SERVER_ADDR") + "/api/v1/price")
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&priceData)
	if err != nil {
		return err
	}

	// retrieve rate of Quicoins based on confirmed amount
	currentPrice := priceData["data"].(map[string]interface{})["current_price"]
	amountToSend := amount / currentPrice.(float64)

	// generate transfer payload
	transferEndpoint := os.Getenv("LID_SERVER_ADDR") + "/api/v1/wallet/transfer"
	icoTradeWallet := os.Getenv("ICO_WALLET")
	icoTradeWalletSecret := os.Getenv("ICO_WALLET_SECRET")
	payload := map[string]string{
		"sender_address":     icoTradeWallet,
		"reciever_address":   wallet,
		"amount":             strconv.FormatFloat(amountToSend, 'f', -1, 64),
		"sender_private_key": icoTradeWalletSecret,
	}

	resp, err = utils.PostToChain(transferEndpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var d map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&d)
		if err != nil {
			log.Printf("failed to post ico_trade_tx to chain: %v", err)
			return errors.New("Failed to read transaction data from the QUI chain")
		}

		return fmt.Errorf("Error posting ico_trade_tx to chain, resp: %v", d)
	}

	return nil
}
