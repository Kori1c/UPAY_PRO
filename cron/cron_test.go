package cron

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"upay_pro/dto"
)

func TestSendAsyncPostAcceptsCreatedOKResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	result, err := sendAsyncPost(server.URL, dto.PaymentNotification_request{
		TradeID:  "trade-1",
		OrderID:  "order-1",
		Amount:   10,
		Status:   2,
		Signature: "signature",
	})

	if err != nil {
		t.Fatalf("expected callback to be accepted, got error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected ok result, got %q", result)
	}
}
