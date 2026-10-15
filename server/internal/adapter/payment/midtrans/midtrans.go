// Package midtrans client Midtrans Snap untuk port.Payment,
// port.SignatureVerifier, port.TransactionVerifier, dan
// port.NotificationParser.
//
// Tanpa auto-retry; satu http.Client dengan timeout 20 detik.
//
// Dua lingkungan, dipilih lewat base URL dari .env:
//
//	Sandbox: https://app.sandbox.midtrans.com  (API: https://api.sandbox.midtrans.com)
//	Produksi: https://app.midtrans.com           (API: https://api.midtrans.com)
//
// Autentikasi: HTTP Basic, username = Server Key, password kosong
// (Authorization: Basic base64(ServerKey + ":")).
//
// Keamanan webhook: Midtrans MENGIRIM signature_key,
// SHA512(order_id + status_code + gross_amount + ServerKey). Jadi notifikasi
// bisa dibuktikan asli TANPA memanggil balik API. Verifikasi balik tetap
// disediakan sebagai lapis kedua (status terkini + anti-replay).
//
// Referensi:
//   - https://docs.midtrans.com/docs/snap-snap-integration-guide
//   - https://docs.midtrans.com/docs/https-notification-webhooks
//   - https://docs.midtrans.com/docs/get-status-api-requests
package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"paluketok/server/internal/port"
)

// Timeout http.Client Midtrans (plan/structure.md baris 70).
const Timeout = 20 * time.Second

// errMidtrans menandai error dari Midtrans (bukan network): status != 2xx.
type errMidtrans struct {
	Status int
	Body   string
}

func (e *errMidtrans) Error() string {
	return fmt.Sprintf("midtrans: http %d: %s", e.Status, e.Body)
}

// Client Midtrans Snap.
//
// authBaseURL dipakai halaman Snap (app.*.midtrans.com),
// apiBaseURL dipakai Get Status API (api.*.midtrans.com). Keduanya dikirim
// terpisah karena Midtrans memakai host berbeda untuk keduanya.
type Client struct {
	authBaseURL string
	apiBaseURL  string
	serverKey   string
	http        *http.Client
}

// New membuat client. authBaseURL dan apiBaseURL tanpa trailing slash.
// apiBaseURL boleh kosong; kalau kosong diturunkan dari authBaseURL dengan
// mengganti host app. -> api.
func New(authBaseURL, apiBaseURL, serverKey string) *Client {
	auth := strings.TrimRight(authBaseURL, "/")
	api := strings.TrimRight(apiBaseURL, "/")
	if api == "" {
		api = strings.Replace(auth, "//app.", "//api.", 1)
	}
	return &Client{
		authBaseURL: auth,
		apiBaseURL:  api,
		serverKey:   serverKey,
		http:        &http.Client{Timeout: Timeout},
	}
}

// ============ Payment (buat token Snap) ============

// snapReq body POST /snap/v1/transactions.
type snapReq struct {
	TransactionDetails snapTxDetails   `json:"transaction_details"`
	ItemDetails        []snapItem      `json:"item_details,omitempty"`
	CustomerDetails    *snapCustomer   `json:"customer_details,omitempty"`
	CreditCard         *snapCreditCard `json:"credit_card,omitempty"`
	Callbacks          *snapCallbacks  `json:"callbacks,omitempty"`
	Expiry             *snapExpiry     `json:"expiry,omitempty"`
}

type snapTxDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int    `json:"gross_amount"`
}

type snapItem struct {
	ID       string `json:"id,omitempty"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type snapCustomer struct {
	FirstName string `json:"first_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

// CreditCard.Secure=true memaksa 3DS pada transaksi kartu.
type snapCreditCard struct {
	Secure bool `json:"secure"`
}

type snapCallbacks struct {
	Finish string `json:"finish,omitempty"`
}

type snapExpiry struct {
	Unit     string `json:"unit"` // "minute" | "hour" | "day"
	Duration int    `json:"duration"`
}

// snapResp balasan 201: {token, redirect_url}.
type snapResp struct {
	Token       string   `json:"token"`
	RedirectURL string   `json:"redirect_url"`
	ErrorMsgs   []string `json:"error_messages"`
}

// CreateOrder POST /snap/v1/transactions -> Snap token + redirect_url.
//
// item_details dijumlahkan harus SAMA dengan gross_amount, kalau tidak Midtrans
// menolak dengan 4xx. Jadi item dikirim hanya bila jumlahnya cocok; kalau tidak
// cocok, item dibuang dan Midtrans hanya menerima total (aman, karena
// gross_amount tetap benar).
func (c *Client) CreateOrder(ctx context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	if req.InvoiceNumber == "" || req.AmountIDR <= 0 {
		return nil, port.ErrValidation
	}
	body := snapReq{
		TransactionDetails: snapTxDetails{
			OrderID:     req.InvoiceNumber,
			GrossAmount: req.AmountIDR,
		},
		CreditCard: &snapCreditCard{Secure: true},
	}
	if items, ok := snapItems(req.Items, req.AmountIDR); ok {
		body.ItemDetails = items
	}
	if req.Email != "" {
		body.CustomerDetails = &snapCustomer{Email: req.Email}
	}
	if !req.ExpiresAt.IsZero() {
		if d := time.Until(req.ExpiresAt); d > time.Minute {
			body.Expiry = &snapExpiry{Unit: "minute", Duration: int(d.Minutes())}
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	respRaw, err := c.do(ctx, c.authBaseURL, http.MethodPost, "/snap/v1/transactions", raw)
	if err != nil {
		return nil, err
	}
	var sr snapResp
	if err := json.Unmarshal(respRaw, &sr); err != nil {
		return nil, err
	}
	if sr.RedirectURL == "" {
		return nil, fmt.Errorf("midtrans: tanpa redirect_url: %s", string(respRaw))
	}
	return &port.PaymentResponse{
		PaymentURL: sr.RedirectURL,
		ExternalID: sr.Token,
		Raw:        respRaw,
	}, nil
}

// snapItems menerjemahkan line items, tapi hanya kalau totalnya persis sama
// dengan gross_amount. Midtrans menolak create bila tidak cocok.
func snapItems(items []port.LineItem, gross int) ([]snapItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	out := make([]snapItem, 0, len(items))
	sum := 0
	for _, it := range items {
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		out = append(out, snapItem{
			Price:    it.Price,
			Quantity: qty,
			Name:     truncate(it.Name, 50),
		})
		sum += it.Price * qty
	}
	if sum != gross {
		return nil, false
	}
	return out, true
}

// ============ SignatureVerifier (keaslian webhook) ============

// VerifySignature menghitung SHA512(order_id + status_code + gross_amount +
// ServerKey) dan membandingkannya dengan signature_key dari notifikasi.
//
// ServerKey TIDAK pernah keluar dari sini: parameter SHA-nya cuma string biasa
// yang habis dipakai, tidak disimpan dan tidak dicatat.
func (c *Client) VerifySignature(in port.SignatureInput) error {
	if in.OrderID == "" || in.StatusCode == "" || in.Signature == "" {
		return port.ErrValidation
	}
	mac := sha512.New()
	// Tulis berurutan tanpa pemisah — persis seperti spesifikasi Midtrans.
	io.WriteString(mac, in.OrderID)
	io.WriteString(mac, in.StatusCode)
	io.WriteString(mac, in.GrossAmt)
	io.WriteString(mac, c.serverKey)
	want := hex.EncodeToString(mac.Sum(nil))

	// Constant-time: cegah kebocoran waktu saat menebak signature.
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(in.Signature)), []byte(want)) != 1 {
		return fmt.Errorf("%w: signature_key tidak cocok", port.ErrForbidden)
	}
	return nil
}

// ============ TransactionVerifier (status terkini) ============

// statusResp GET /v2/{order_id}/status.
type statusResp struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"` // string, mis. "25000.00"
	StatusCode        string `json:"status_code"`
	MerchantID        string `json:"merchant_id"`
	ExpiryTime        string `json:"expiry_time"` // "2020-01-09 18:27:19"
	StatusCodeInt     int    `json:"-"`
}

// VerifyTransaction membaca status transaksi langsung dari Midtrans. Dipakai
// sebagai lapis kedua setelah signature, dan sebagai sumber kebenaran bila
// notifikasi tidak pernah datang.
//
// order_id adalah kunci utama; Midtrans juga menerima transaction_id di posisi
// URL yang sama. in.TransactionID dipakai bila in.OrderID kosong.
func (c *Client) VerifyTransaction(ctx context.Context, in port.VerifyInput) (*port.VerifiedTransaction, error) {
	key := in.OrderID
	if key == "" {
		key = in.TransactionID
	}
	if key == "" {
		return nil, port.ErrValidation
	}
	respRaw, err := c.do(ctx, c.apiBaseURL, http.MethodGet, "/v2/"+urlPathEscape(key)+"/status", nil)
	if err != nil {
		return nil, err
	}
	var sr statusResp
	if err := json.Unmarshal(respRaw, &sr); err != nil {
		return nil, err
	}

	out := &port.VerifiedTransaction{
		OrderID:       sr.OrderID,
		TransactionID: sr.TransactionID,
		Status:        sr.TransactionStatus,
		Method:        sr.PaymentType,
		MerchantID:    sr.MerchantID,
		FraudStatus:   sr.FraudStatus,
	}
	// gross_amount datang sebagai string ("25000.00"); ambil bagian bulatnya.
	if v, err := strconv.ParseFloat(sr.GrossAmount, 64); err == nil {
		out.AmountIDR = int(v)
	}
	// Sukses = settlement, atau capture yang tidak ditolak FDS.
	out.Paid = sr.TransactionStatus == "settlement" ||
		(sr.TransactionStatus == "capture" && sr.FraudStatus != "deny")
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", sr.ExpiryTime, time.Local); err == nil {
		out.ExpiresAt = &t
	}
	out.Raw = respRaw
	return out, nil
}

// ============ NotificationParser (webhook masuk) ============

// hookBody payload notifikasi Midtrans. Field bertambah seiring waktu; hanya
// yang dipakai yang dideklarasikan.
type hookBody struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	MerchantID        string `json:"merchant_id"`
}

// Parse membaca body notifikasi. TIDAK memverifikasi — signature_key-nya
// dikembalikan lewat RawBody + field supaya usecase memanggil VerifySignature
// secara terpisah (satu tanggung jawab per fungsi).
//
// Payload UJI dari dashboard Midtrans (tombol "Test notification URL") hanya
// berisi {"test": true} / contoh tanpa order_id. Itu bukan pembayaran: balas
// ErrTestNotification supaya handler menjawab 200 tanpa memproses apa pun —
// kalau ditolak 422, dashboard menandai endpoint gagal padahal sehat.
func (c *Client) Parse(raw []byte) (*port.VerifiedNotification, error) {
	// probe memisahkan payload uji dari notifikasi. *bool supaya tipe salah
	// (mis. "test":"ya") TIDAK lolos sebagai payload uji — hanya bool sejati.
	var probe struct {
		Test *bool `json:"test"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && probe.Test != nil {
		return nil, fmt.Errorf("%w: payload uji dashboard Midtrans", port.ErrTestNotification)
	}

	var hb hookBody
	if err := json.Unmarshal(raw, &hb); err != nil {
		return nil, fmt.Errorf("%w: body bukan JSON", port.ErrValidation)
	}
	if hb.OrderID == "" || hb.StatusCode == "" || hb.SignatureKey == "" {
		return nil, fmt.Errorf("%w: notifikasi tanpa order_id/status_code/signature_key", port.ErrValidation)
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

// SignatureKey menarik signature_key + status_code + gross_amount dari body
// mentah untuk diserahkan ke VerifySignature. Dipisah dari Parse supaya
// usecase tidak perlu tahu bentuk JSON Midtrans.
func SignatureKey(raw []byte) (port.SignatureInput, error) {
	var hb hookBody
	if err := json.Unmarshal(raw, &hb); err != nil {
		return port.SignatureInput{}, fmt.Errorf("%w: body bukan JSON", port.ErrValidation)
	}
	return port.SignatureInput{
		OrderID:    hb.OrderID,
		StatusCode: hb.StatusCode,
		GrossAmt:   hb.GrossAmount,
		Signature:  hb.SignatureKey,
		RawBody:    raw,
	}, nil
}

// ============ internal ============

// do memanggil Midtrans dengan Basic auth (username ServerKey, password kosong).
// 404 dipetakan ke ErrNotFound supaya usecase bisa membedakan "order tidak ada
// di gateway" dari gangguan gateway biasa.
func (c *Client) do(ctx context.Context, base, method, path string, body []byte) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+
		base64.StdEncoding.EncodeToString([]byte(c.serverKey+":")))
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
		e := &errMidtrans{Status: resp.StatusCode, Body: string(raw)}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", port.ErrNotFound, e.Error())
		}
		return nil, e
	}
	return raw, nil
}

// urlPathEscape meng-escape order_id untuk dipakai sebagai segmen path.
// order_id Midtrans hanya boleh alfanumerik + - _ ~ . jadi ini hampir selalu
// no-op; ada untuk jaga-jaga.
func urlPathEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "/", "%2F"), " ", "%20")
}

// truncate memotong nama item ke n karakter (batas Midtrans 50).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

var _ port.Payment = (*Client)(nil)
var _ port.SignatureVerifier = (*Client)(nil)
var _ port.TransactionVerifier = (*Client)(nil)
var _ port.NotificationParser = (*Client)(nil)
