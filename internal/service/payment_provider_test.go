package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRazorpayPaymentProviderCreateOrderRetriesTransientFailure(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/orders" {
			t.Fatalf("unexpected path %s", got)
		}
		if atomic.AddInt32(&attempts, 1) == 1 {
			http.Error(w, "temporary provider failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id":       "order_retry_ok",
			"amount":   125000,
			"currency": "INR",
			"status":   "created",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewRazorpayPaymentProvider("key", "secret", "webhook", server.URL, server.Client())
	provider.retryBackoff = time.Millisecond

	order, err := provider.CreateOrder(context.Background(), PaymentOrderCreateRequest{
		AttemptID:   uuid.New(),
		TenantID:    uuid.New(),
		StudentID:   uuid.New(),
		AmountPaise: 125000,
		Currency:    "INR",
		Receipt:     "attempt-1",
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.OrderID != "order_retry_ok" {
		t.Fatalf("unexpected order id %q", order.OrderID)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestRazorpayPaymentProviderAppliesDefaultTimeout(t *testing.T) {
	t.Parallel()

	provider := NewRazorpayPaymentProvider("key", "secret", "webhook", "https://api.example", &http.Client{})

	if provider.client.Timeout != 10*time.Second {
		t.Fatalf("expected default timeout, got %s", provider.client.Timeout)
	}
}
