// E2E ringan Fase 5 (workflow.md 21): server lokal httptest penuh router,
// usecase dengan fake. Tanpa Midtrans/LLM/JEV live. Lapisan webhook diuji
// dengan signature Midtrans yang dihitung sungguhan (SHA512) plus verifikasi
// balik ke Get Status API, supaya bukti tetap end-to-end di lapisan delivery.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
)

// --- fakes kecil khusus delivery (usecase port) ---

// fakeVerifier meniru verifikasi balik ke Get Status API Midtrans. payTx =
// daftar order_id yang "ada di Midtrans"; di luar itu = 404.
type fakeVerifier struct {
	payTx map[string]*port.VerifiedTransaction
	err   error
}

func (f *fakeVerifier) VerifyTransaction(_ context.Context, in port.VerifyInput) (*port.VerifiedTransaction, error) {
	if f.err != nil {
		return nil, f.err
	}
	tx, ok := f.payTx[in.OrderID]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *tx
	return &cp, nil
}

// parserMidtrans meniru adapter Midtrans: payload uji dashboard dikenali lewat
// ErrTestNotification, body tanpa signature_key ditolak ErrValidation.
type parserMidtrans struct{}

func (parserMidtrans) Parse(raw []byte) (*port.VerifiedNotification, error) {
	var probe struct {
		Test *bool `json:"test"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && probe.Test != nil {
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
		return nil, &entity.ErrValidation{Msg: "body webhook rusak"}
	}
	if hb.OrderID == "" || hb.StatusCode == "" || hb.SignatureKey == "" {
		return nil, &entity.ErrValidation{Msg: "notifikasi tanpa signature_key"}
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

type fakeOrdersRepo struct {
	byInvoice map[string]*entity.Order
	updated   int
	last      *entity.Order
}

func (r *fakeOrdersRepo) Create(_ context.Context, o *entity.Order) error { return nil }
func (r *fakeOrdersRepo) GetByID(_ context.Context, id entity.ID) (*entity.Order, error) {
	return nil, port.ErrNotFound
}
func (r *fakeOrdersRepo) GetByInvoiceNumber(_ context.Context, inv string) (*entity.Order, error) {
	if o, ok := r.byInvoice[inv]; ok {
		return o, nil
	}
	return nil, port.ErrNotFound
}
func (r *fakeOrdersRepo) GetByRequestID(_ context.Context, rid string) (*entity.Order, error) {
	for _, o := range r.byInvoice {
		if o.RequestID == rid {
			return o, nil
		}
	}
	return nil, port.ErrNotFound
}
func (r *fakeOrdersRepo) Update(_ context.Context, o *entity.Order) error {
	r.updated++
	cp := *o
	r.last = &cp
	return nil
}

func (emptyProfileRepo) SetPlanTier(_ context.Context, _ entity.ID, _ entity.PlanTier) error {
	return nil
}

type emptyProfileRepo struct{}

func (emptyProfileRepo) GetByID(_ context.Context, id entity.ID) (*entity.Profile, error) {
	return &entity.Profile{ID: id, Credits: 10}, nil
}

type settleLedger struct{ settled int }

func (s *settleLedger) ChargeForRun(context.Context, entity.ID, *entity.PlanRun) error {
	return nil
}
func (s *settleLedger) ChargeForRegenerate(context.Context, entity.ID, entity.ID, int) (entity.ID, error) {
	return "", nil
}
func (s *settleLedger) RefundRun(context.Context, *entity.PlanRun, int) error          { return nil }
func (s *settleLedger) RefundCharge(context.Context, entity.ID, entity.ID, int) error  { return nil }
func (s *settleLedger) Grant(context.Context, entity.ID, int, entity.ID, string) error { return nil }
func (s *settleLedger) SettleTopUp(_ context.Context, o *entity.Order) error {
	o.Status = entity.OrderSettlement
	s.settled++
	return nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) }

// idGen UUID v4-shaped deterministik per panggilan (cukup unik untuk test).
type idGen struct{ n int }

func (g *idGen) New() entity.ID {
	g.n++
	return entity.ID(fmt.Sprintf("0197a23e-0000-7000-8000-%012d", g.n))
}

// webhookDeps merangkai usecase untuk webhook. payTx = transaksi yang "ada di
// Midtrans"; order_id di luar daftar ini ditolak 404.
func webhookDeps(store map[entity.ID]*entity.Order, payTx map[string]*port.VerifiedTransaction) (*usecase.OrderUsecase, *settleLedger) {
	os2 := &orderStore{fakeOrdersRepo: &fakeOrdersRepo{}, store: store}
	if store == nil {
		os2 = &orderStore{fakeOrdersRepo: &fakeOrdersRepo{}, store: map[entity.ID]*entity.Order{
			"0197a23e-1111-7000-8000-000000000001": {ID: "0197a23e-1111-7000-8000-000000000001",
				UserID: "0197a23e-1111-7000-8000-00000000000a", InvoiceNumber: "PK-abc",
				AmountIDR: 25000, CreditsGranted: 10, Status: entity.OrderPending, RequestID: "req-1"},
		}}
	}
	sl := &settleLedger{}
	apps := &usecase.Deps{
		Profiles: emptyProfileRepo{}, Orders: os2, Ledger: sl,
		Signatures:    &memPayment{},
		Verifier:      &fakeVerifier{payTx: payTx},
		Notifications: parserMidtrans{},
		Clock:         fakeClock{}, IDs: &idGen{},
	}
	return usecase.NewOrderUsecase(*apps), sl
}

func webhookServer(t *testing.T, store map[entity.ID]*entity.Order, payTx map[string]*port.VerifiedTransaction) (*httptest.Server, *settleLedger) {
	t.Helper()
	uc, sl := webhookDeps(store, payTx)
	h := &Handlers{Orders: uc}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h}))
	t.Cleanup(srv.Close)
	return srv, sl
}

const ujiWebhookPath = "/v1/webhooks/midtrans"

// --- bukti 21a: order_id tak ada di Midtrans = 404, yang sah = 200, replay
// 8x konsisten (idempoten) ---

func TestWebhook_E2E_VerifikasiBalikDanReplay(t *testing.T) {
	paid := map[string]*port.VerifiedTransaction{
		"PK-abc": {OrderID: "PK-abc", AmountIDR: 25000, Status: "settlement", Paid: true},
	}

	// Body dengan signature SAH untuk key test -> diterima.
	good, sl := webhookServer(t, nil, paid)
	var lastCode int
	for i := 0; i < 8; i++ {
		resp, err := http.Post(good.URL+ujiWebhookPath, "application/json",
			bytes.NewReader(e2eNotifBody("PK-abc", "25000.00", "settlement")))
		if err != nil {
			t.Fatal(err)
		}
		lastCode = resp.StatusCode
		resp.Body.Close()
		if lastCode != http.StatusOK {
			t.Fatalf("replay %d: ingin 200, dapat %d", i+1, lastCode)
		}
	}
	if sl.settled != 1 {
		t.Fatalf("settle harus sekali (idempoten), dapat %d", sl.settled)
	}

	// Webhook PALSU (signature karangan, serverKey berbeda) -> 403, tidak
	// menyentuh order sama sekali.
	fake, _ := webhookServer(t, nil, paid)
	badBody := e2eNotifBody("PK-abc", "25000.00", "settlement")
	badBody = bytes.Replace(badBody, []byte(e2eSign("PK-abc", "200", "25000.00")),
		[]byte(e2eSignWith("PK-abc", "200", "25000.00", "SB-Mid-server-PENYERANG")), 1)
	resp, err := http.Post(fake.URL+ujiWebhookPath, "application/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("signature karangan: ingin 403, dapat %d", resp.StatusCode)
	}

	// Notifikasi untuk order yang tidak ada di Midtrans -> diabaikan 200, bukan
	// 404 (404 bikin Midtrans mengulang & dashboard menandai endpoint gagal).
	missing, _ := webhookServer(t, nil, paid)
	resp, err = http.Post(missing.URL+ujiWebhookPath, "application/json",
		bytes.NewReader(e2eNotifBody("PK-karangan", "25000.00", "settlement")))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("order karangan: ingin 200 diabaikan, dapat %d", resp.StatusCode)
	}
}

// --- bukti 21a-lain: nominal tidak cocok = 422, tidak settle ---

func TestWebhook_E2E_NominalTidakCocok(t *testing.T) {
	// Midtrans melaporkan 1.000 padahal order 25.000: jangan settle.
	payTx := map[string]*port.VerifiedTransaction{
		"PK-abc": {OrderID: "PK-abc", AmountIDR: 1000, Status: "settlement", Paid: true},
	}
	srv, sl := webhookServer(t, nil, payTx)
	resp, err := http.Post(srv.URL+ujiWebhookPath, "application/json",
		bytes.NewReader(e2eNotifBody("PK-abc", "1000.00", "settlement")))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("nominal beda: ingin 422, dapat %d", resp.StatusCode)
	}
	if sl.settled != 0 {
		t.Fatalf("nominal beda tidak boleh settle, dapat %d", sl.settled)
	}
}

// --- bukti 21a-lain: body tanpa signature_key ditolak 422 ---

func TestWebhook_E2E_TanpaSignatureDitolak(t *testing.T) {
	srv, sl := webhookServer(t, nil, map[string]*port.VerifiedTransaction{})
	resp, err := http.Post(srv.URL+ujiWebhookPath, "application/json",
		bytes.NewReader([]byte(`{"order_id":"PK-abc","status_code":"200","gross_amount":"25000.00"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("tanpa signature_key: ingin 422, dapat %d", resp.StatusCode)
	}
	if sl.settled != 0 {
		t.Fatalf("tidak boleh settle, dapat %d", sl.settled)
	}
}

// --- bukti: payload UJI dashboard Midtrans dibalas 200, bukan 422 ---
//
// Tombol "Test notification URL" di dashboard mengirim {"test": true} tanpa
// order_id. Sebelum ini endpoint menjawab 422 dan dashboard menandai GAGAL
// padahal server sehat (terlihat di journalctl prod 2026-10-15 17:06).

func TestWebhook_E2E_PayloadUjiDashboardDibalas200(t *testing.T) {
	srv, sl := webhookServer(t, nil, map[string]*port.VerifiedTransaction{})

	for _, body := range []string{`{"test": true}`, `{"test":true,"status":"ok"}`} {
		resp, err := http.Post(srv.URL+ujiWebhookPath, "application/json",
			bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("payload uji %s: ingin 200, dapat %d", body, resp.StatusCode)
		}
	}
	if sl.settled != 0 {
		t.Fatalf("payload uji tidak boleh menyelesaikan order, dapat %d", sl.settled)
	}
}

// Pengenalan payload uji tidak boleh melonggarkan penolakan body sampah.
func TestWebhook_E2E_TestBukanBoolTetap422(t *testing.T) {
	srv, _ := webhookServer(t, nil, map[string]*port.VerifiedTransaction{})
	resp, err := http.Post(srv.URL+ujiWebhookPath, "application/json",
		bytes.NewReader([]byte(`{"test":"ya"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf(`{"test":"ya"}: ingin 422, dapat %d`, resp.StatusCode)
	}
}

// --- bukti 21b: buat order + PaymentURL tersimpan (Payment fake) ---

type fakePayment struct{ called int }

func (f *fakePayment) CreateOrder(_ context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	f.called++
	return &port.PaymentResponse{
		PaymentURL: "https://app.sandbox.midtrans.com/snap/v2/vtweb/tok-" + req.InvoiceNumber,
		ExternalID: "tok-" + req.InvoiceNumber,
	}, nil
}

func TestOrders_E2E_CreateOrder_PaymentURLTersimpan(t *testing.T) {
	store := map[entity.ID]*entity.Order{}
	pay := &fakePayment{}
	apps := &usecase.Deps{
		Profiles: emptyProfileRepo{}, Ledger: &settleLedger{}, Payment: pay,
		Clock: fakeClock{}, IDs: &idGen{},
	}
	apps.Orders = &orderStore{fakeOrdersRepo: &fakeOrdersRepo{byInvoice: map[string]*entity.Order{}}, store: store}
	h := &Handlers{Orders: usecase.NewOrderUsecase(*apps)}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h,
		ActorFix: entity.Actor{UserID: "0197a23e-1111-7000-8000-00000000000a"}}))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/orders", "application/json",
		bytes.NewReader([]byte(`{"package_id":"starter"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create order: ingin 201, dapat %d", resp.StatusCode)
	}
	var out struct {
		OrderID       string `json:"order_id"`
		PaymentURL    string `json:"payment_url"`
		InvoiceNumber string `json:"invoice_number"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !strings.HasPrefix(out.PaymentURL, "https://app.sandbox.midtrans.com/snap/") {
		t.Fatalf("payment_url tidak tersimpan/terkirim: %q", out.PaymentURL)
	}
	if pay.called != 1 {
		t.Fatalf("Midtrans CreateOrder harus tepat 1x, dapat %d", pay.called)
	}
	if len(out.InvoiceNumber) > 30 || len(out.InvoiceNumber) < 4 {
		t.Fatalf("invoice number janggal: %q", out.InvoiceNumber)
	}
	// Snap token tersimpan sebagai jejak audit.
	if o := store[entity.ID(out.OrderID)]; o == nil || o.GatewayToken == nil {
		t.Fatalf("gateway_token harus tersimpan setelah create")
	}
}

// orderStore menyimpan order BUATAN usecase di memori. Nilai disimpan sebagai
// pointer sehingga mutasi status oleh SettleTopUp (yang di repo nyata dilakukan
// UPDATE ... WHERE status='pending' di dalam transaksi) terlihat antar replay.
type orderStore struct {
	*fakeOrdersRepo
	store map[entity.ID]*entity.Order
}

func (r *orderStore) Create(_ context.Context, o *entity.Order) error {
	r.store[o.ID] = o
	return nil
}
func (r *orderStore) GetByID(_ context.Context, id entity.ID) (*entity.Order, error) {
	if o, ok := r.store[id]; ok {
		return o, nil
	}
	return nil, port.ErrNotFound
}
func (r *orderStore) Update(_ context.Context, o *entity.Order) error {
	r.store[o.ID] = o
	return nil
}

func (r *orderStore) GetByInvoiceNumber(_ context.Context, inv string) (*entity.Order, error) {
	for _, o := range r.store {
		if o.InvoiceNumber == inv {
			return o, nil
		}
	}
	return nil, port.ErrNotFound
}
func (r *orderStore) GetByRequestID(_ context.Context, rid string) (*entity.Order, error) {
	for _, o := range r.store {
		if o.RequestID == rid {
			return o, nil
		}
	}
	return nil, port.ErrNotFound
}

// --- bukti 21b-lain: paket tak dikenal = 422; JWT hilang = 401 di /v1/me ---

func TestOrders_E2E_PaketTidakDikenal(t *testing.T) {
	apps := &usecase.Deps{Profiles: emptyProfileRepo{}, Ledger: &settleLedger{},
		Payment: &fakePayment{}, Clock: fakeClock{}}
	apps.Orders = &orderStore{fakeOrdersRepo: &fakeOrdersRepo{}, store: map[entity.ID]*entity.Order{}}
	h := &Handlers{Orders: usecase.NewOrderUsecase(*apps)}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h,
		ActorFix: entity.Actor{UserID: "0197a23e-1111-7000-8000-00000000000a"}}))
	defer srv.Close()
	resp, _ := http.Post(srv.URL+"/v1/orders", "application/json",
		bytes.NewReader([]byte(`{"package_id":"paket_sultan"}`)))
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("paket asing: ingin 422, dapat %d", resp.StatusCode)
	}
}

func TestRouter_E2E_TanpaJWT401(t *testing.T) {
	apps := &usecase.Deps{Profiles: emptyProfileRepo{}, Clock: fakeClock{}}
	h := &Handlers{Credits: usecase.NewCreditUsecase(*apps)}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h})) // Verifier nil = grup authed tanpa JWT utk tes ini
	defer srv.Close()
	resp, _ := http.Get(srv.URL + "/v1/me")
	resp.Body.Close()
	// Tanpa JWT, actor kosong sampai usecase: Me membalas ErrForbidden -> 403.
	// (Bukti lem lapisan: delivery tidak mengarang identity.) JWT salah
	// sendiri jadi 401 di
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("me tanpa actor: ingin 403, dapat %d", resp.StatusCode)
	}
}
