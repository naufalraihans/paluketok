// Mock port.Payment + port.SignatureVerifier + port.TransactionVerifier +
// port.NotificationParser untuk E2E: tidak memanggil Midtrans sungguhan.
//
// Signature ditiru dengan cara yang SAMA seperti adapter asli: SHA512 atas
// order_id + status_code + gross_amount + serverKey. Jadi test E2E benar-benar
// melatih jalur verifikasi tanda tangan, bukan melewatinya.
package httpapi

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"paluketok/server/internal/port"
)

// e2eServerKey = ServerKey palsu untuk test. Sama dengan yang dipegang
// memPayment dan yang dipakai helper e2eSign.
const e2eServerKey = "SB-Mid-server-e2e-rahasia"

// memPayment menyimpan transaksi yang "ada" di sisi gateway palsu. Test
// mendaftarkan transaksi lewat setTx sebelum menembak webhook.
type memPayment struct {
	mu sync.Mutex
	// tx: order_id -> hasil verifikasi. Order tidak terdaftar = 404,
	// meniru notifikasi untuk transaksi yang tidak ada di Midtrans.
	tx map[string]*port.VerifiedTransaction
}

func newMemPayment() *memPayment {
	return &memPayment{tx: map[string]*port.VerifiedTransaction{}}
}

// setTx mendaftarkan transaksi seperti yang akan dikembalikan Get Status API.
func (m *memPayment) setTx(orderID string, amount int, status string, paid bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tx[orderID] = &port.VerifiedTransaction{
		OrderID:   orderID,
		AmountIDR: amount,
		Status:    status,
		Method:    "qris",
		Paid:      paid,
	}
}

func (m *memPayment) CreateOrder(_ context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	token := "snap-tok-" + req.InvoiceNumber
	return &port.PaymentResponse{
		PaymentURL: "https://app.sandbox.midtrans.com/snap/v2/vtweb/" + token,
		ExternalID: token,
	}, nil
}

// VerifyTransaction meniru GET /v2/{order_id}/status. Order tak dikenal ->
// port.ErrNotFound, sama seperti 404 Midtrans.
func (m *memPayment) VerifyTransaction(_ context.Context, in port.VerifyInput) (*port.VerifiedTransaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.tx[in.OrderID]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *tx
	return &cp, nil
}

// VerifySignature menghitung ulang SHA512 seperti adapter Midtrans asli.
func (m *memPayment) VerifySignature(in port.SignatureInput) error {
	if in.OrderID == "" || in.StatusCode == "" || in.Signature == "" {
		return port.ErrValidation
	}
	if in.Signature != e2eSign(in.OrderID, in.StatusCode, in.GrossAmt) {
		return port.ErrForbidden
	}
	return nil
}

// Parse meniru adapter Midtrans: payload uji dashboard ({"test":true}) dikenali
// lewat ErrTestNotification; body tanpa signature_key ditolak ErrValidation.
func (m *memPayment) Parse(raw []byte) (*port.VerifiedNotification, error) {
	var probe struct {
		Test *bool `json:"test"`
	}
	_ = json.Unmarshal(raw, &probe)
	if probe.Test != nil {
		return nil, port.ErrTestNotification
	}

	var hb struct {
		OrderID           string `json:"order_id"`
		TransactionID     string `json:"transaction_id"`
		TransactionStatus string `json:"transaction_status"`
		FraudStatus       string `json:"fraud_status"`
		StatusCode        string `json:"status_code"`
		SignatureKey      string `json:"signature_key"`
		GrossAmount       string `json:"gross_amount"`
	}
	if err := json.Unmarshal(raw, &hb); err != nil {
		return nil, port.ErrValidation
	}
	if hb.OrderID == "" || hb.StatusCode == "" || hb.SignatureKey == "" {
		return nil, port.ErrValidation
	}
	return &port.VerifiedNotification{
		OrderID:       hb.OrderID,
		TransactionID: hb.TransactionID,
		StatusCode:    hb.StatusCode,
		GrossAmount:   hb.GrossAmount,
		Status:        hb.TransactionStatus,
		FraudStatus:   hb.FraudStatus,
		Signature:     hb.SignatureKey,
		Event:         hb.TransactionStatus,
		RawBody:       raw,
	}, nil
}

// e2eSign menghitung signature_key palsu dengan rumus Midtrans.
func e2eSign(orderID, statusCode, gross string) string {
	return e2eSignWith(orderID, statusCode, gross, e2eServerKey)
}

// e2eSignWith mengizinkan ServerKey lain, untuk menguji tanda tangan penyerang.
func e2eSignWith(orderID, statusCode, gross, serverKey string) string {
	h := sha512.New()
	io.WriteString(h, orderID)
	io.WriteString(h, statusCode)
	io.WriteString(h, gross)
	io.WriteString(h, serverKey)
	return hex.EncodeToString(h.Sum(nil))
}

// e2eNotifBody menyusun body notifikasi Midtrans yang signature-nya SAH.
func e2eNotifBody(orderID, gross, status string) []byte {
	const code = "200"
	body := map[string]any{
		"order_id":           orderID,
		"transaction_id":     "tx-" + orderID,
		"transaction_status": status,
		"status_code":        code,
		"gross_amount":       gross,
		"signature_key":      e2eSign(orderID, code, gross),
		"payment_type":       "qris",
	}
	b, _ := json.Marshal(body)
	return b
}

var _ port.Payment = (*memPayment)(nil)
var _ port.SignatureVerifier = (*memPayment)(nil)
var _ port.TransactionVerifier = (*memPayment)(nil)
var _ port.NotificationParser = (*memPayment)(nil)

var _ = errors.Is
