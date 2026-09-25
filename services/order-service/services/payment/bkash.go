package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"kather_baksho/models"
)

type BKashProvider struct {
	appKey      string
	appSecret   string
	username    string
	password    string
	baseURL     string
	idToken     string
	tokenExpiry time.Time
	mu          sync.Mutex
	client      *http.Client
}

func NewBKashProvider() *BKashProvider {
	baseURL := os.Getenv("BKASH_BASE_URL")
	if baseURL == "" {
		baseURL = "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
	}
	return &BKashProvider{
		appKey:    os.Getenv("BKASH_APP_KEY"),
		appSecret: os.Getenv("BKASH_APP_SECRET"),
		username:  os.Getenv("BKASH_USERNAME"),
		password:  os.Getenv("BKASH_PASSWORD"),
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (b *BKashProvider) Name() string {
	return "bkash"
}

// grantToken retrieves an id_token from bKash PGW.
func (b *BKashProvider) grantToken() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.idToken != "" && time.Now().Before(b.tokenExpiry) {
		return b.idToken, nil
	}

	payload := map[string]string{
		"app_key":    b.appKey,
		"app_secret": b.appSecret,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", b.baseURL+"/tokenized/checkout/token/grant", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("username", b.username)
	req.Header.Set("password", b.password)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("bkash grant token network error: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		IDToken     string `json:"id_token"`
		ExpiresIn   int    `json:"expires_in"`
		StatusCode  string `json:"statusCode"`
		StatusMsg   string `json:"statusMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("bkash token decode failed: %w", err)
	}

	if result.IDToken == "" {
		return "", fmt.Errorf("bkash token error: %s - %s", result.StatusCode, result.StatusMsg)
	}

	b.idToken = result.IDToken
	b.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return b.idToken, nil
}

func (b *BKashProvider) Initiate(order *models.Order, returnURL string) (*PaymentSession, error) {
	// If credentials missing, provide realistic sandbox simulation
	if b.appKey == "" || b.username == "" {
		simPaymentID := fmt.Sprintf("BKASH-SIM-%d-%d", order.ID, time.Now().Unix())
		log.Printf("[bKash] Credentials missing; running in Sandbox Simulation mode for Order #%d", order.ID)
		return &PaymentSession{
			SessionID:     simPaymentID,
			OrderID:       order.ID,
			Amount:        order.TotalPrice,
			Currency:      "BDT",
			PaymentMethod: "bkash",
			GatewayURL:    fmt.Sprintf("/api/payments/bkash/simulate?paymentID=%s&orderID=%d&amount=%.2f", simPaymentID, order.ID, order.TotalPrice),
		}, nil
	}

	token, err := b.grantToken()
	if err != nil {
		return nil, fmt.Errorf("bkash token failed: %w", err)
	}

	invoiceNum := fmt.Sprintf("INV-%d-%d", order.ID, time.Now().Unix())
	createPayload := map[string]interface{}{
		"mode":                  "0011",
		"payerReference":        fmt.Sprintf("USER-%d", order.UserID),
		"callbackURL":           returnURL,
		"amount":                fmt.Sprintf("%.2f", order.TotalPrice),
		"currency":              "BDT",
		"intent":                "sale",
		"merchantInvoiceNumber": invoiceNum,
	}
	body, _ := json.Marshal(createPayload)

	req, err := http.NewRequest("POST", b.baseURL+"/tokenized/checkout/create", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("X-APP-Key", b.appKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bkash create payment request failed: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		PaymentID     string `json:"paymentID"`
		BKashURL      string `json:"bkashURL"`
		StatusCode    string `json:"statusCode"`
		StatusMessage string `json:"statusMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if res.PaymentID == "" {
		return nil, fmt.Errorf("bkash create payment failed: %s (%s)", res.StatusMessage, res.StatusCode)
	}

	return &PaymentSession{
		SessionID:     res.PaymentID,
		OrderID:       order.ID,
		Amount:        order.TotalPrice,
		Currency:      "BDT",
		PaymentMethod: "bkash",
		GatewayURL:    res.BKashURL,
		ProviderData:  res,
	}, nil
}

func (b *BKashProvider) Verify(payload map[string]interface{}) (*PaymentResult, error) {
	paymentID, _ := payload["paymentID"].(string)
	if paymentID == "" {
		paymentID, _ = payload["payment_id"].(string)
	}
	if paymentID == "" {
		return nil, fmt.Errorf("missing paymentID in bkash verification")
	}

	// If simulation payment ID
	if len(paymentID) >= 9 && paymentID[:9] == "BKASH-SIM" {
		status, _ := payload["status"].(string)
		if status == "" {
			status = "SUCCESS"
		}
		return &PaymentResult{
			SessionID:     paymentID,
			TransactionID: fmt.Sprintf("TRX-BKASH-%d", time.Now().UnixNano()),
			Status:        status,
			PaymentMethod: "bkash",
		}, nil
	}

	token, err := b.grantToken()
	if err != nil {
		return nil, err
	}

	reqBody, _ := json.Marshal(map[string]string{"paymentID": paymentID})
	req, _ := http.NewRequest("POST", b.baseURL+"/tokenized/checkout/execute", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("X-APP-Key", b.appKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var execRes struct {
		PaymentID         string `json:"paymentID"`
		TrxID             string `json:"trxID"`
		TransactionStatus string `json:"transactionStatus"`
		Amount            string `json:"amount"`
		Currency          string `json:"currency"`
		StatusCode        string `json:"statusCode"`
		StatusMessage     string `json:"statusMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return nil, err
	}

	status := "FAILED"
	if execRes.TransactionStatus == "Completed" || execRes.StatusCode == "0000" {
		status = "SUCCESS"
	}

	return &PaymentResult{
		SessionID:       execRes.PaymentID,
		TransactionID:   execRes.TrxID,
		Status:          status,
		PaymentMethod:   "bkash",
		GatewayResponse: execRes.StatusMessage,
	}, nil
}

func (b *BKashProvider) Query(paymentID string) (*PaymentResult, error) {
	if b.appKey == "" {
		return &PaymentResult{
			SessionID:     paymentID,
			TransactionID: fmt.Sprintf("TRX-BKASH-SIM-%d", time.Now().Unix()),
			Status:        "SUCCESS",
			PaymentMethod: "bkash",
		}, nil
	}

	token, err := b.grantToken()
	if err != nil {
		return nil, err
	}

	reqBody, _ := json.Marshal(map[string]string{"paymentID": paymentID})
	req, _ := http.NewRequest("POST", b.baseURL+"/tokenized/checkout/payment/status", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("X-APP-Key", b.appKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var qRes struct {
		PaymentID         string `json:"paymentID"`
		TrxID             string `json:"trxID"`
		TransactionStatus string `json:"transactionStatus"`
		StatusCode        string `json:"statusCode"`
	}
	json.NewDecoder(resp.Body).Decode(&qRes)

	status := "FAILED"
	if qRes.TransactionStatus == "Completed" || qRes.StatusCode == "0000" {
		status = "SUCCESS"
	}

	return &PaymentResult{
		SessionID:     qRes.PaymentID,
		TransactionID: qRes.TrxID,
		Status:        status,
		PaymentMethod: "bkash",
	}, nil
}

func (b *BKashProvider) Refund(paymentID string, amount float64, reason string) error {
	token, err := b.grantToken()
	if err != nil {
		return err
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"paymentID": paymentID,
		"amount":    fmt.Sprintf("%.2f", amount),
		"reason":    reason,
		"sku":       "REFUND",
	})
	req, _ := http.NewRequest("POST", b.baseURL+"/tokenized/checkout/payment/refund", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("X-APP-Key", b.appKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
