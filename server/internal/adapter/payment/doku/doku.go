// Package doku client DOKU Checkout untuk port.Payment, port.SignatureVerifier,
// dan port.NotificationParser (plan/candidates/doku-arch.md).
// Tanpa auto-retry; satu http.Client dengan timeout 20 detik.
package doku

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"paluketok/server/internal/port"
)

// Timeout http.Client DOKU (plan/structure.md baris 70).
const Timeout = 20 * time.Second

// errDoku menandai error dari DOKU (bukan network): status != 2xx.
type errDoku struct {
	Status int
	Body   string
}

func (e *errDoku) Error() string {
	return fmt.Sprintf("doku: http %d: %s", e.Status, e.Body)
}

// asNotFound dipakai usecase untuk membaca 409 create sebagai "order sudah ada".
func IsAlreadyExists(err error) bool {
	var e *errDoku
	return errors.As(err, &e) && e.Status == http.StatusConflict
}

// Client DOKU. Base URL dari .env (mis. https://api-sandbox.doku.com).
type Client struct {
	baseURL    string
	clientID   string
	secretKey  string
	notifyPath string
	http       *http.Client
}

// New membuat client. url memanggil tanpa trailing slash.
func New(baseURL, clientID, secretKey, notificationPath string) *Client {
	return &Client{
		baseURL:    baseURL,
		clientID:   clientID,
		secretKey:  secretKey,
		notifyPath: notificationPath,
		http:       &http.Client{Timeout: Timeout},
	}
}

// ============ signature komponen (security.md bagian 6) ============

// signatureComponents membangun string komponen sesuai docs DOKU:
// tiap baris diawali NAMA komponen, "Nama:nilai". 5 baris saat POST (ada
// Digest dari body), 4 baris saat GET tanpa body (TANPA baris Digest,
// docs signature-from-api-get-method). requestTarget adalah PATH terdaftar
// di back office (DOKU_NOTIFICATION_PATH) untuk webhook; untuk API keluar
// adalah path API DOKU yang dipakai.
func signatureComponents(clientID, requestID, timestamp, requestTarget string, rawBody []byte) string {
	s := "Client-Id:" + clientID +
		"\nRequest-Id:" + requestID +
		"\nRequest-Timestamp:" + timestamp +
		"\nRequest-Target:" + requestTarget
	if rawBody != nil {
		digest := base64.StdEncoding.EncodeToString(hashSHA256(rawBody))
		s += "\nDigest:" + digest
	}
	return s
}

func hashSHA256(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

// signRequest menghasilkan nilai header Signature untuk request keluar.
func signRequest(secret, clientID, requestID, timestamp, requestTarget string, rawBody []byte) string {
	component := signatureComponents(clientID, requestID, timestamp, requestTarget, rawBody)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(component))
	return "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ============ SignatureVerifier (webhook masuk) ============

// Verify memeriksa signature webhook DOKU dengan hmac.Equal (constant time).
// 5 komponen, Digest dari body MENTAH; Request-Target dari konfigurasi
// (path terdaftar), bukan dari header (dikutip dari plan/security.md bag.6).
func (c *Client) Verify(in port.WebhookInput) error {
	if in.ClientID != c.clientID {
		return port.ErrForbidden
	}
	component := signatureComponents(in.ClientID, in.RequestID, in.RequestTimestamp, c.notifyPath, in.RawBody)
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(component))
	expect := "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expect), []byte(in.Signature)) {
		return port.ErrForbidden
	}
	return nil
}

// ============ NotificationParser ============

// notifBody bentuk json longgar body webhook DOKU (field boleh bertambah).
type notifBody struct {
	Order struct {
		Amount        int    `json:"amount"`
		InvoiceNumber string `json:"invoice_number"`
		RequestID     string `json:"request_id"`
		Status        string `json:"status"` // ORDER_EXPIRED dll.
	} `json:"order"`
	Transaction map[string]any `json:"transaction"`
}

// Parse mengubah body webhook ke port.Notification:
// transaction.status dipakai untuk SUCCESS/FAILED, order.status untuk
// ORDER_EXPIRED/RECOVERED (security.md bagian 6 langkah 9).
func (c *Client) Parse(raw []byte) (*port.Notification, error) {
	var nb notifBody
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&nb); err != nil {
		return nil, err
	}
	n := &port.Notification{
		InvoiceNumber: nb.Order.InvoiceNumber,
		RequestID:     nb.Order.RequestID,
		AmountIDR:     nb.Order.Amount,
		OrderStatus:   nb.Order.Status,
	}
	// transaction.status ada di transaction.status
	if nb.Transaction != nil {
		if s, ok := nb.Transaction["status"].(string); ok {
			n.Transaction = s
		}
	}
	return n, nil
}

// ============ Payment (checkout + status) ============

// CreateOrder POST /checkout/v1/payment. Merespons 409 bila request_id sudah
// dikenal DOKU (auth_billing.md 2.3: dibaca "sudah ada", usecase lanjut).
func (c *Client) CreateOrder(ctx context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	lineItems := make([]map[string]any, 0, len(req.Items))
	for _, it := range req.Items {
		lineItems = append(lineItems, map[string]any{
			"name":     it.Name,
			"price":    it.Price,
			"quantity": it.Quantity,
		})
	}
	body := map[string]any{
		"order": map[string]any{
			"amount":         req.AmountIDR,
			"invoice_number": req.InvoiceNumber,
			"line_items":     lineItems,
		},
		"payment": map[string]any{
			"payment_due_date": 60, // menit: 1 jam (auth_billing 2.3 expired 1 jam)
		},
		"customer": map[string]any{}, // boleh kosong
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	path := "/checkout/v1/payment"
	headers, err := c.signedHeaders(http.MethodPost, path, req.RequestID, raw)
	if err != nil {
		return nil, err
	}
	respRaw, err := c.do(ctx, http.MethodPost, c.baseURL+path, headers, raw)
	if err != nil {
		return nil, err
	}

	var dr struct {
		Message  []string `json:"message"`
		Response struct {
			Order struct {
				InvoiceNumber string `json:"invoice_number"`
				Amount        string `json:"amount"`
			} `json:"order"`
			Payment struct {
				URL         string `json:"url"`          // checkout page
				ExpiredDate string `json:"expired_date"` // yyyyMMddHHmmss UTC+7 dari DOKU
				TokenID     string `json:"token_id"`
			} `json:"payment"`
		} `json:"response"`
	}
	if err := json.Unmarshal(respRaw, &dr); err != nil {
		return nil, err
	}
	expired := dr.Response.Payment.ExpiredDate
	if expired == "" {
		// ponytail: fallback hitung sendiri; DOKU normalnya selalu kirim expired_date.
		expired = c.computeExpired(time.Now())
	}
	if dr.Response.Payment.URL == "" {
		return nil, fmt.Errorf("doku: tanpa payment.url: %s", string(respRaw))
	}
	return &port.PaymentResponse{
		PaymentURL:  dr.Response.Payment.URL,
		ExpiredDate: expired,
		Raw:         respRaw,
	}, nil
}

// CheckStatus GET /orders/v1/status/{invoiceNumber}. Tanpa retry.
func (c *Client) CheckStatus(ctx context.Context, invoiceNumber string) (*port.PaymentStatus, error) {
	path := "/orders/v1/status/" + invoiceNumber
	// GET: 4 baris komponen, TANPA baris Digest (body nil); docs
	// signature-from-api-get-method.
	headers, err := c.signedHeaders(http.MethodGet, path, newRequestID(), nil)
	if err != nil {
		return nil, err
	}
	respRaw, err := c.do(ctx, http.MethodGet, c.baseURL+path, headers, nil)
	if err != nil {
		return nil, err
	}
	var st struct {
		Order struct {
			InvoiceNumber string `json:"invoice_number"`
			Amount        int    `json:"amount"`
			Status        string `json:"status"` // ORDER_EXPIRED dll?
			Transaction   []struct {
				Status string `json:"status"` // SUCCESS, FAILED...
			} `json:"transaction"`
		} `json:"order"`
	}
	if err := json.Unmarshal(respRaw, &st); err != nil {
		return nil, err
	}
	out := &port.PaymentStatus{
		InvoiceNumber: st.Order.InvoiceNumber,
		OrderStatus:   st.Order.Status,
		AmountIDR:     st.Order.Amount,
		Raw:           respRaw,
	}
	// transaction berupa list di status API DOKU; ambil status terakhir
	for _, t := range st.Order.Transaction {
		out.Transaction = t.Status
	}
	return out, nil
}

// ============ internal ============

// signedHeaders menghasilkan semua header DOKU (doku-arch bagian 2.3).
// Request-Timestamp UTC ISO8601 Z penuh: 2006-01-02T15:04:05Z.
func (c *Client) signedHeaders(method, requestTarget, requestID string, body []byte) (map[string]string, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	sig := signRequest(c.secretKey, c.clientID, requestID, timestamp, requestTarget, body)
	return map[string]string{
		"Client-Id":         c.clientID,
		"Request-Id":        requestID,
		"Request-Timestamp": timestamp,
		"Signature":         sig,
	}, nil
}

func (c *Client) do(ctx context.Context, method, url string, headers map[string]string, body []byte) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, &errDoku{Status: resp.StatusCode, Body: string(raw)}
	}
	return raw, nil
}

// computeExpired membuat yyyyMMddHHmmss UTC+7 dari satu waktu. Durasi
// expired (1 jam, auth_billing.md 2.3) dihitung usecase; adapter konversi
// zona dan format saja.
func (c *Client) computeExpired(base time.Time) string {
	return base.In(time.FixedZone("WIB", 7*3600)).Format("20060102150405")
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

var _ port.Payment = (*Client)(nil)
var _ port.SignatureVerifier = (*Client)(nil)
var _ port.NotificationParser = (*Client)(nil)
