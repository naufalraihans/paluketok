package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/port"
)

// serverKey dipakai semua test. Nilai harus sama saat memverifikasi signature.
const testServerKey = "SB-Mid-server-RAHASIA"

// authHeader menghitung header Basic yang diharapkan: base64(serverKey + ":").
func authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(testServerKey+":"))
}

// ---- SignatureVerifier ----

// Rumus Midtrans: SHA512(order_id + status_code + gross_amount + ServerKey).
func sigOf(orderID, statusCode, gross string) string {
	h := sha512.New()
	io.WriteString(h, orderID)
	io.WriteString(h, statusCode)
	io.WriteString(h, gross)
	io.WriteString(h, testServerKey)
	return hex.EncodeToString(h.Sum(nil))
}

func TestVerifySignature_Cocok(t *testing.T) {
	c := New("https://x", "", testServerKey)
	order, code, gross := "PK-1", "200", "25000.00"
	err := c.VerifySignature(port.SignatureInput{
		OrderID: order, StatusCode: code, GrossAmt: gross, Signature: sigOf(order, code, gross),
	})
	if err != nil {
		t.Fatalf("signature sah harus lolos, dapat %v", err)
	}
}

func TestVerifySignature_DitolakBilaSalah(t *testing.T) {
	c := New("https://x", "", testServerKey)
	err := c.VerifySignature(port.SignatureInput{
		OrderID: "PK-1", StatusCode: "200", GrossAmt: "25000.00", Signature: "deadbeef",
	})
	if !errors.Is(err, port.ErrForbidden) {
		t.Fatalf("signature palsu harus ErrForbidden, dapat %v", err)
	}
}

// Nominal yang diubah penyerang harus membatalkan signature (inilah inti
// keamananya: gross_amount ikut ditandatangani).
func TestVerifySignature_NominalDiubahGagal(t *testing.T) {
	c := New("https://x", "", testServerKey)
	// signature dihitung untuk 25000, tapi notifikasi mengaku 1000.
	err := c.VerifySignature(port.SignatureInput{
		OrderID: "PK-1", StatusCode: "200", GrossAmt: "1000.00",
		Signature: sigOf("PK-1", "200", "25000.00"),
	})
	if !errors.Is(err, port.ErrForbidden) {
		t.Fatalf("nominal diubah harus ErrForbidden, dapat %v", err)
	}
}

// Tanpa ServerKey di sisi penyerang, signature tidak bisa dipalsukan.
func TestVerifySignature_ServerKeySalahGagal(t *testing.T) {
	c := New("https://x", "", "SB-Mid-server-LAIN")
	err := c.VerifySignature(port.SignatureInput{
		OrderID: "PK-1", StatusCode: "200", GrossAmt: "25000.00",
		Signature: sigOf("PK-1", "200", "25000.00"),
	})
	if !errors.Is(err, port.ErrForbidden) {
		t.Fatalf("ServerKey lain harus gagal, dapat %v", err)
	}
}

func TestVerifySignature_FieldKosong(t *testing.T) {
	c := New("https://x", "", testServerKey)
	if err := c.VerifySignature(port.SignatureInput{OrderID: "PK-1"}); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("field kosong harus ErrValidation, dapat %v", err)
	}
}

// ---- NotificationParser ----

func TestParse_AmbilFieldPenting(t *testing.T) {
	body := []byte(`{
      "transaction_status":"settlement","order_id":"PK-1","status_code":"200",
      "signature_key":"abc","gross_amount":"25000.00",
      "transaction_id":"57d5293c-e65f-4a29-95e4-5959c3fa335b",
      "fraud_status":"accept","payment_type":"bank_transfer"}`)
	n, err := New("", "", "").Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if n.OrderID != "PK-1" || n.StatusCode != "200" || n.GrossAmount != "25000.00" {
		t.Fatalf("field tidak terbaca: %+v", n)
	}
	if n.Status != "settlement" || n.RawBody == nil {
		t.Fatalf("status/rawbody salah: %+v", n)
	}
}

func TestParse_BodyRusakError(t *testing.T) {
	if _, err := New("", "", "").Parse([]byte("bukan json")); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("body rusak harus ErrValidation, dapat %v", err)
	}
}

// Notifikasi tanpa signature tidak boleh lolos parse: itu bukan dari Midtrans.
func TestParse_TanpaSignatureDitolak(t *testing.T) {
	body := []byte(`{"order_id":"PK-1","status_code":"200","transaction_status":"settlement"}`)
	if _, err := New("", "", "").Parse(body); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("tanpa signature_key harus ErrValidation, dapat %v", err)
	}
}

func TestSignatureKey_DariBodyMentah(t *testing.T) {
	body := []byte(`{"order_id":"PK-9","status_code":"200","gross_amount":"10000.00","signature_key":"xyz"}`)
	si, err := SignatureKey(body)
	if err != nil {
		t.Fatal(err)
	}
	if si.OrderID != "PK-9" || si.Signature != "xyz" || si.GrossAmt != "10000.00" {
		t.Fatalf("SignatureInput salah: %+v", si)
	}
}

// ---- Payment (Snap) ----

func TestCreateOrder_PakaiBasicAuthDanBalasRedirect(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"token":"tok-123","redirect_url":"https://app.sandbox.midtrans.com/snap/v2/vtweb/tok-123"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.URL, testServerKey)
	res, err := c.CreateOrder(context.Background(), port.PaymentRequest{
		InvoiceNumber: "PK-777", AmountIDR: 25000, Email: "n@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != authHeader() {
		t.Fatalf("Authorization salah: %q", gotAuth)
	}
	if gotPath != "/snap/v1/transactions" {
		t.Fatalf("path salah: %q", gotPath)
	}
	if res.PaymentURL == "" || res.ExternalID != "tok-123" {
		t.Fatalf("respons salah: %+v", res)
	}
	td, _ := gotBody["transaction_details"].(map[string]any)
	if td["order_id"] != "PK-777" {
		t.Fatalf("order_id harus InvoiceNumber, dapat %v", td["order_id"])
	}
	// gross_amount JSON number, bukan string.
	if v, ok := td["gross_amount"].(float64); !ok || int(v) != 25000 {
		t.Fatalf("gross_amount salah: %v (%T)", td["gross_amount"], td["gross_amount"])
	}
	// 3DS wajib untuk transaksi kartu.
	cc, _ := gotBody["credit_card"].(map[string]any)
	if cc["secure"] != true {
		t.Fatalf("credit_card.secure harus true, dapat %v", cc["secure"])
	}
}

// item_details hanya dikirim bila jumlahnya persis sama dengan gross_amount;
// Midtrans menolak 4xx kalau beda.
func TestCreateOrder_ItemDikirimBilaJumlahCocok(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"token":"t","redirect_url":"https://x/y"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.URL, testServerKey)
	_, err := c.CreateOrder(context.Background(), port.PaymentRequest{
		InvoiceNumber: "PK-1", AmountIDR: 30000,
		Items: []port.LineItem{{Name: "Kredit 30", Price: 10000, Quantity: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	items, ok := gotBody["item_details"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("item_details harus terkirim, dapat %v", gotBody["item_details"])
	}
}

func TestCreateOrder_ItemDibuangBilaJumlahBeda(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"token":"t","redirect_url":"https://x/y"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.URL, testServerKey)
	_, err := c.CreateOrder(context.Background(), port.PaymentRequest{
		InvoiceNumber: "PK-1", AmountIDR: 25000,
		Items: []port.LineItem{{Name: "Kredit 30", Price: 10000, Quantity: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ada := gotBody["item_details"]; ada {
		t.Fatalf("item_details harus dibuang saat jumlah beda, dapat %v", gotBody["item_details"])
	}
}

func TestCreateOrder_TanpaRedirectURLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"token":"tok"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.URL, testServerKey)
	if _, err := c.CreateOrder(context.Background(), port.PaymentRequest{
		InvoiceNumber: "PK-1", AmountIDR: 1000,
	}); err == nil {
		t.Fatal("tanpa redirect_url harus error")
	}
}

func TestCreateOrder_InvoiceKosongAtauNominalNol(t *testing.T) {
	c := New("http://x", "", testServerKey)
	if _, err := c.CreateOrder(context.Background(), port.PaymentRequest{AmountIDR: 1000}); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("invoice kosong harus ErrValidation, dapat %v", err)
	}
	if _, err := c.CreateOrder(context.Background(), port.PaymentRequest{InvoiceNumber: "PK-1"}); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("nominal 0 harus ErrValidation, dapat %v", err)
	}
}

// 401 dari Midtrans ("Access denied") harus muncul sebagai errMidtrans, bukan
// diam-diam jadi sukses.
func TestCreateOrder_401JadiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error_messages":["Access denied due to unauthorized transaction"]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.URL, "salah")
	if _, err := c.CreateOrder(context.Background(), port.PaymentRequest{
		InvoiceNumber: "PK-1", AmountIDR: 1000,
	}); err == nil {
		t.Fatal("401 harus error")
	}
}

// ---- TransactionVerifier ----

func TestVerifyTransaction_SettlementPaid(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{
          "order_id":"PK-1","transaction_id":"57d5293c","transaction_status":"settlement",
          "fraud_status":"accept","payment_type":"bank_transfer",
          "gross_amount":"25000.00","merchant_id":"M004123",
          "expiry_time":"2020-01-09 18:27:19"}`)
	}))
	defer srv.Close()

	c := New("", srv.URL, testServerKey) // apiBaseURL eksplisit
	tx, err := c.VerifyTransaction(context.Background(), port.VerifyInput{OrderID: "PK-1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v2/PK-1/status" {
		t.Fatalf("path salah: %q", gotPath)
	}
	if !tx.Paid || tx.Status != "settlement" || tx.AmountIDR != 25000 {
		t.Fatalf("hasil salah: %+v", tx)
	}
	if tx.OrderID != "PK-1" || tx.MerchantID != "M004123" {
		t.Fatalf("order/merchant salah: %+v", tx)
	}
	if tx.ExpiresAt == nil {
		t.Fatal("expiry_time harus terbaca")
	}
}

// capture + fraud accept = lunas; capture + fraud deny = tidak lunas.
func TestVerifyTransaction_CaptureTergantungFraud(t *testing.T) {
	bacaan := func(fraud string) *port.VerifiedTransaction {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, `{"order_id":"PK-1","transaction_status":"capture","fraud_status":"`+fraud+`","gross_amount":"1000.00"}`)
		}))
		defer srv.Close()
		tx, err := New("", srv.URL, testServerKey).VerifyTransaction(context.Background(), port.VerifyInput{OrderID: "PK-1"})
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	if !bacaan("accept").Paid {
		t.Fatal("capture+accept harus Paid")
	}
	if bacaan("deny").Paid {
		t.Fatal("capture+deny harus tidak Paid")
	}
}

func TestVerifyTransaction_ExpireTidakPaid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"order_id":"PK-1","transaction_status":"expire","gross_amount":"1000.00"}`)
	}))
	defer srv.Close()

	tx, err := New("", srv.URL, testServerKey).VerifyTransaction(context.Background(), port.VerifyInput{OrderID: "PK-1"})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Paid {
		t.Fatal("expire harus tidak Paid")
	}
}

// 404 = order tidak ada di Midtrans -> ErrNotFound, supaya usecase bisa
// memutuskan webhook itu tidak sah.
func TestVerifyTransaction_404JadiErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"status_code":"404","status_message":"Transaction doesn't exist."}`)
	}))
	defer srv.Close()

	_, err := New("", srv.URL, testServerKey).VerifyTransaction(context.Background(), port.VerifyInput{OrderID: "PK-karangan"})
	if !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("404 harus ErrNotFound, dapat %v", err)
	}
}

func TestVerifyTransaction_KeyKosong(t *testing.T) {
	if _, err := New("", "http://x", testServerKey).VerifyTransaction(context.Background(), port.VerifyInput{}); !errors.Is(err, port.ErrValidation) {
		t.Fatalf("key kosong harus ErrValidation, dapat %v", err)
	}
}

// transaction_id boleh dipakai sebagai ganti order_id.
func TestVerifyTransaction_PakaiTransactionID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{"order_id":"PK-1","transaction_id":"tx-9","transaction_status":"pending","gross_amount":"1000.00"}`)
	}))
	defer srv.Close()

	if _, err := New("", srv.URL, testServerKey).VerifyTransaction(context.Background(),
		port.VerifyInput{TransactionID: "tx-9"}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v2/tx-9/status" {
		t.Fatalf("path salah: %q", gotPath)
	}
}

// ---- New: turunkan apiBaseURL dari authBaseURL ----

func TestNew_TurunkanHostApiDariApp(t *testing.T) {
	c := New("https://app.sandbox.midtrans.com/", "", testServerKey)
	if c.apiBaseURL != "https://api.sandbox.midtrans.com" {
		t.Fatalf("apiBaseURL turunan salah: %q", c.apiBaseURL)
	}
	if c.authBaseURL != "https://app.sandbox.midtrans.com" {
		t.Fatalf("authBaseURL harus tanpa trailing slash: %q", c.authBaseURL)
	}
}

func TestNew_ApiBaseEksplisitMenang(t *testing.T) {
	c := New("https://app.midtrans.com", "https://api.midtrans.com/", testServerKey)
	if c.apiBaseURL != "https://api.midtrans.com" {
		t.Fatalf("apiBaseURL eksplisit harus dipakai: %q", c.apiBaseURL)
	}
}

// Request tanpa body (GET) tidak boleh mengirim body.
func TestDo_GetTanpaBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if len(b) != 0 {
			t.Errorf("GET tidak boleh berbody, dapat %q", string(b))
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Errorf("Authorization harus Basic, dapat %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, `{"order_id":"PK-1","transaction_status":"pending","gross_amount":"0"}`)
	}))
	defer srv.Close()

	if _, err := New("", srv.URL, testServerKey).VerifyTransaction(context.Background(),
		port.VerifyInput{OrderID: "PK-1"}); err != nil {
		t.Fatal(err)
	}
}

// TestParse_PayloadUjiDashboard: tombol "Test notification URL" Midtrans
// mengirim {"test": true} tanpa order_id. Harus dikenali sebagai payload uji
// (bukan ErrValidation), supaya handler bisa menjawab 200 dan dashboard hijau.
func TestParse_PayloadUjiDashboard(t *testing.T) {
	c := New("https://app.sandbox.midtrans.com", "https://api.sandbox.midtrans.com", "SB-Mid-server-x")

	for _, body := range []string{
		`{"test": true}`,
		`{"test":true,"status":"ok"}`,
	} {
		_, err := c.Parse([]byte(body))
		if !errors.Is(err, port.ErrTestNotification) {
			t.Fatalf("body %s: mau ErrTestNotification, dapat %v", body, err)
		}
	}
}

// TestParse_SampahTetapDitolak: pengenalan payload uji tidak boleh melonggarkan
// penolakan body yang benar-benar tidak sah.
func TestParse_SampahTetapDitolak(t *testing.T) {
	c := New("https://app.sandbox.midtrans.com", "https://api.sandbox.midtrans.com", "SB-Mid-server-x")

	cases := map[string]string{
		"bukan JSON":          `ini bukan json`,
		"tanpa signature_key": `{"order_id":"PK-1","status_code":"200","gross_amount":"25000.00"}`,
		"tanpa order_id":      `{"status_code":"200","signature_key":"abc","gross_amount":"25000.00"}`,
		"test bukan bool":     `{"test":"ya"}`,
	}
	for name, body := range cases {
		_, err := c.Parse([]byte(body))
		if !errors.Is(err, port.ErrValidation) {
			t.Fatalf("%s: mau ErrValidation, dapat %v", name, err)
		}
	}
}

var _ = time.Second
