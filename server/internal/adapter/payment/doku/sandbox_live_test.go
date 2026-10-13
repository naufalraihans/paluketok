package doku

// Test sandbox live: SATU create-order. Key sandbox dari server/.env lewat
// env DOKU_CLIENT_ID / DOKU_SECRET_KEY / DOKU_BASE_URL. Skip bila kosong.
// Test ini berbiaya nol (sandbox) tapi menyentuh jaringan — jalur biasa tetap
// unit test.

import (
	"context"
	"os"
	"testing"
	"time"

	"paluketok/server/internal/port"
)

func paymentRequestSandbox() port.PaymentRequest {
	return port.PaymentRequest{
		InvoiceNumber: "PK-uji-f4-" + time.Now().Format("060102150405"),
		RequestID:     newRequestID(),
		AmountIDR:     25000,
		Items: []port.LineItem{{
			Name: "PaluKetok Starter 10 kredit", Price: 25000, Quantity: 1,
		}},
	}
}

func TestSandbox_CreateOrder(t *testing.T) {
	base := os.Getenv("DOKU_BASE_URL")
	cid := os.Getenv("DOKU_CLIENT_ID")
	secret := os.Getenv("DOKU_SECRET_KEY")
	if base == "" || cid == "" || secret == "" {
		t.Skip("env DOKU belum terisi: skip sandbox")
	}
	c := New(base, cid, secret, os.Getenv("DOKU_NOTIFICATION_PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	resp, err := c.CreateOrder(ctx, paymentRequestSandbox())
	if err != nil {
		t.Fatalf("create order sandbox: %v", err)
	}
	if resp.PaymentURL == "" {
		t.Fatalf("payment_url kosong. raw=%s", string(resp.Raw))
	}
	t.Logf("sandbox OK: payment_url=%s expired=%s", resp.PaymentURL, resp.ExpiredDate)
}
