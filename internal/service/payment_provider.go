package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
)

type PaymentProvider interface {
	Name() string
	CreateOrder(ctx context.Context, req PaymentOrderCreateRequest) (*PaymentProviderOrder, error)
	VerifyPaymentSignature(req PaymentSignatureVerification) error
	VerifyWebhookSignature(payload []byte, signature string) error
}

type PaymentOrderCreateRequest struct {
	AttemptID   uuid.UUID
	TenantID    uuid.UUID
	StudentID   uuid.UUID
	AmountPaise int64
	Currency    string
	Receipt     string
	Notes       map[string]string
}

type PaymentProviderOrder struct {
	OrderID     string
	AmountPaise int64
	Currency    string
	Status      string
	CheckoutURL string
	Metadata    map[string]any
}

type PaymentSignatureVerification struct {
	OrderID   string
	PaymentID string
	Signature string
}

type RazorpayPaymentProvider struct {
	keyID         string
	keySecret     string
	webhookSecret string
	baseURL       string
	client        *http.Client
	maxAttempts   int
	retryBackoff  time.Duration
}

func NewRazorpayPaymentProvider(keyID, keySecret, webhookSecret, baseURL string, client *http.Client) *RazorpayPaymentProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.razorpay.com/v1"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else if client.Timeout == 0 {
		clientCopy := *client
		clientCopy.Timeout = 10 * time.Second
		client = &clientCopy
	}
	return &RazorpayPaymentProvider{
		keyID:         strings.TrimSpace(keyID),
		keySecret:     strings.TrimSpace(keySecret),
		webhookSecret: strings.TrimSpace(webhookSecret),
		baseURL:       strings.TrimRight(baseURL, "/"),
		client:        client,
		maxAttempts:   3,
		retryBackoff:  200 * time.Millisecond,
	}
}

func (p *RazorpayPaymentProvider) Name() string {
	return "razorpay"
}

func (p *RazorpayPaymentProvider) CreateOrder(ctx context.Context, req PaymentOrderCreateRequest) (*PaymentProviderOrder, error) {
	if p.keyID == "" || p.keySecret == "" {
		return nil, apperror.New("PAYMENT_PROVIDER_NOT_CONFIGURED", "razorpay credentials are not configured", http.StatusServiceUnavailable)
	}
	body := map[string]any{
		"amount":   req.AmountPaise,
		"currency": req.Currency,
		"receipt":  req.Receipt,
		"notes":    req.Notes,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal razorpay order: %w", err)
	}
	var respBody []byte
	var statusCode int
	for attempt := 1; attempt <= p.maxAttempts; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/orders", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create razorpay request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.SetBasicAuth(p.keyID, p.keySecret)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			if attempt < p.maxAttempts && ctx.Err() == nil {
				if waitErr := sleepWithContext(ctx, retryDelay(p.retryBackoff, attempt)); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, fmt.Errorf("create razorpay order: %w", err)
		}

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		closeErr := resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read razorpay response: %w", err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close razorpay response: %w", closeErr)
		}
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			break
		}
		if !shouldRetryPaymentProviderStatus(statusCode) || attempt == p.maxAttempts {
			return nil, apperror.New("PAYMENT_PROVIDER_ERROR", "razorpay order creation failed", http.StatusBadGateway)
		}
		if waitErr := sleepWithContext(ctx, retryDelay(p.retryBackoff, attempt)); waitErr != nil {
			return nil, waitErr
		}
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, apperror.New("PAYMENT_PROVIDER_ERROR", "razorpay order creation failed", http.StatusBadGateway)
	}

	var parsed struct {
		ID       string         `json:"id"`
		Amount   int64          `json:"amount"`
		Currency string         `json:"currency"`
		Status   string         `json:"status"`
		Notes    map[string]any `json:"notes"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode razorpay order: %w", err)
	}
	if strings.TrimSpace(parsed.ID) == "" {
		return nil, apperror.New("PAYMENT_PROVIDER_ERROR", "razorpay order response did not include an order id", http.StatusBadGateway)
	}
	return &PaymentProviderOrder{
		OrderID:     parsed.ID,
		AmountPaise: parsed.Amount,
		Currency:    parsed.Currency,
		Status:      defaultString(parsed.Status, "created"),
		Metadata:    map[string]any{"provider_notes": parsed.Notes},
	}, nil
}

func shouldRetryPaymentProviderStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func retryDelay(base time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	if attempt <= 1 {
		return base
	}
	return time.Duration(attempt) * base
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (p *RazorpayPaymentProvider) VerifyPaymentSignature(req PaymentSignatureVerification) error {
	if p.keySecret == "" {
		return apperror.New("PAYMENT_PROVIDER_NOT_CONFIGURED", "razorpay key secret is not configured", http.StatusServiceUnavailable)
	}
	body := req.OrderID + "|" + req.PaymentID
	if !verifyHMACSHA256(body, req.Signature, p.keySecret) {
		return apperror.New("PAYMENT_SIGNATURE_INVALID", "payment signature verification failed", http.StatusBadRequest)
	}
	return nil
}

func (p *RazorpayPaymentProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	if p.webhookSecret == "" {
		return apperror.New("PAYMENT_WEBHOOK_NOT_CONFIGURED", "razorpay webhook secret is not configured", http.StatusServiceUnavailable)
	}
	if !verifyHMACSHA256Bytes(payload, signature, p.webhookSecret) {
		return apperror.New("PAYMENT_WEBHOOK_SIGNATURE_INVALID", "webhook signature verification failed", http.StatusBadRequest)
	}
	return nil
}

type FakePaymentProvider struct {
	name          string
	signingSecret string
}

func NewFakePaymentProvider(name, signingSecret string) *FakePaymentProvider {
	if strings.TrimSpace(name) == "" {
		name = "fake"
	}
	if strings.TrimSpace(signingSecret) == "" {
		signingSecret = "test_payment_secret"
	}
	return &FakePaymentProvider{name: strings.TrimSpace(name), signingSecret: strings.TrimSpace(signingSecret)}
}

func (p *FakePaymentProvider) Name() string {
	return p.name
}

func (p *FakePaymentProvider) CreateOrder(_ context.Context, req PaymentOrderCreateRequest) (*PaymentProviderOrder, error) {
	orderID := "order_" + strings.ReplaceAll(req.AttemptID.String(), "-", "")
	return &PaymentProviderOrder{
		OrderID:     orderID,
		AmountPaise: req.AmountPaise,
		Currency:    req.Currency,
		Status:      "created",
		CheckoutURL: "https://checkout.test.eduwallet.local/" + orderID,
		Metadata:    map[string]any{"fake": true},
	}, nil
}

func (p *FakePaymentProvider) VerifyPaymentSignature(req PaymentSignatureVerification) error {
	body := req.OrderID + "|" + req.PaymentID
	if !verifyHMACSHA256(body, req.Signature, p.signingSecret) {
		return apperror.New("PAYMENT_SIGNATURE_INVALID", "payment signature verification failed", http.StatusBadRequest)
	}
	return nil
}

func (p *FakePaymentProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	if !verifyHMACSHA256Bytes(payload, signature, p.signingSecret) {
		return apperror.New("PAYMENT_WEBHOOK_SIGNATURE_INVALID", "webhook signature verification failed", http.StatusBadRequest)
	}
	return nil
}

func verifyHMACSHA256(body, signature, secret string) bool {
	return verifyHMACSHA256Bytes([]byte(body), signature, secret)
}

func verifyHMACSHA256Bytes(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return false
	}
	return hmac.Equal(expected, provided)
}
