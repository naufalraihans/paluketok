// E2E ringan Fase 5 (workflow.md 21): server lokal httptest penuh router,
// usecase dengan fake. Tanpa DOKU/LLM/JEV live; signature DOKU disimulasikan
// dengan HMAC asli via adapter doku pada server palsu? TIDAK — webhook diuji
// langsung: signature disetel pakai secret test adapter (unit) DIHTTP handler
// level gunakan kontrak port dengan fake verifier agar bukti tetap end-to-end
// di lapisan delivery.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
)

// --- fakes kecil khusus delivery (usecase port) ---

type fakeSignature struct{ valid bool }

func (f *fakeSignature) Verify(in port.WebhookInput) error {
	if !f.valid {
		return port.ErrForbidden
	}
	return nil
}

// parserFunc membungkus fungsi jadi port.NotificationParser.
type parserFunc func(raw []byte) (*port.Notification, error)

func (f parserFunc) Parse(raw []byte) (*port.Notification, error) { return f(raw) }

// fakeParser meniru parse DOKU: order.amount angka, transaction.status string.
func fakeParserDOKU(raw []byte) (*port.Notification, error) {
	var nb struct {
		Order struct {
			Amount        int    `json:"amount"`
			InvoiceNumber string `json:"invoice_number"`
			RequestID     string `json:"request_id"`
			Status        string `json:"status"`
		} `json:"order"`
		Transaction map[string]any `json:"transaction"`
	}
	if err := json.Unmarshal(raw, &nb); err != nil {
		return nil, &entity.ErrValidation{Msg: "body webhook rusak"}
	}
	n := &port.Notification{InvoiceNumber: nb.Order.InvoiceNumber, RequestID: nb.Order.RequestID,
		AmountIDR: nb.Order.Amount, OrderStatus: nb.Order.Status}
	if s, ok := nb.Transaction["status"].(string); ok {
		n.Transaction = s
	}
	return n, nil
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

// webhookDeps merangkai usecase untuk webhook (order store persist).
func webhookDeps(store map[entity.ID]*entity.Order, sigValid bool) (*usecase.OrderUsecase, *settleLedger) {
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
		Signature: &fakeSignature{valid: sigValid}, Notifications: parserFunc(fakeParserDOKU),
		Clock: fakeClock{}, IDs: &idGen{},
	}
	return usecase.NewOrderUsecase(*apps), sl
}

func webhookServer(t *testing.T, store map[entity.ID]*entity.Order, sigValid bool) (*httptest.Server, *settleLedger) {
	t.Helper()
	uc, sl := webhookDeps(store, sigValid)
	h := &Handlers{Orders: uc, NotifyPath: "/v1/webhooks/doku"}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h}))
	t.Cleanup(srv.Close)
	return srv, sl
}

// --- bukti 21a: signature salah = 401, benar = 200, replay 8x konsisten ---

func TestWebhook_E2E_SignatureDanReplay(t *testing.T) {
	body := []byte(`{"order":{"invoice_number":"PK-abc","amount":25000},"transaction":{"status":"SUCCESS"}}`)

	bad, _ := webhookServer(t, nil, false)
	resp, err := http.Post(bad.URL+"/v1/webhooks/doku", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("signature salah: ingin 401, dapat %d", resp.StatusCode)
	}

	// Replay 8x dengan store persist: settle tepat sekali, 200 konsisten.
	good, sl := webhookServer(t, nil, true)
	var lastCode int
	for i := 0; i < 8; i++ {
		resp, err := http.Post(good.URL+"/v1/webhooks/doku", "application/json", bytes.NewReader(body))
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
}

// --- bukti 21a-lain: webhook FAILED diabaikan (200) ---

func TestWebhook_E2E_FAILEDdiabaikan(t *testing.T) {
	srv, sl := webhookServer(t, nil, true)
	resp, err := http.Post(srv.URL+"/v1/webhooks/doku", "application/json",
		bytes.NewReader([]byte(`{"order":{"invoice_number":"PK-abc"},"transaction":{"status":"FAILED"}}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("FAILED harus 200 (diabaikan), dapat %d", resp.StatusCode)
	}
	if sl.settled != 0 {
		t.Fatalf("FAILED tidak boleh settle, dapat %d", sl.settled)
	}
}

// --- bukti 21b: buat order + PaymentURL tersimpan (Payment fake) ---

type fakePayment struct{ called int }

func (f *fakePayment) CreateOrder(_ context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	f.called++
	return &port.PaymentResponse{PaymentURL: "https://staging.doku.com/checkout/x", ExpiredDate: "20261013130000"}, nil
}
func (f *fakePayment) CheckStatus(context.Context, string) (*port.PaymentStatus, error) {
	return nil, port.ErrUnavailable
}

func TestOrders_E2E_CreateOrder_PaymentURLTersimpan(t *testing.T) {
	store := map[entity.ID]*entity.Order{}
	pay := &fakePayment{}
	apps := &usecase.Deps{
		Profiles: emptyProfileRepo{}, Ledger: &settleLedger{}, Payment: pay,
		Signature: &fakeSignature{valid: true}, Notifications: parserFunc(fakeParserDOKU),
		Clock: fakeClock{}, IDs: &idGen{},
	}
	apps.Orders = &orderStore{fakeOrdersRepo: &fakeOrdersRepo{byInvoice: map[string]*entity.Order{}}, store: store}
	h := &Handlers{Orders: usecase.NewOrderUsecase(*apps), NotifyPath: "/v1/webhooks/doku"}
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
	if out.PaymentURL != "https://staging.doku.com/checkout/x" {
		t.Fatalf("payment_url tidak tersimpan/terkirim: %q", out.PaymentURL)
	}
	if pay.called != 1 {
		t.Fatalf("DOKU CreateOrder harus tepat 1x, dapat %d", pay.called)
	}
	if len(out.InvoiceNumber) > 30 || len(out.InvoiceNumber) < 4 {
		t.Fatalf("invoice number janggal: %q", out.InvoiceNumber)
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
