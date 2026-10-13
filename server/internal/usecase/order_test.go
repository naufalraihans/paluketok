package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

func seedOrder(t *testing.T, deps Deps, f *fakes, status entity.OrderStatus) *entity.Order {
	t.Helper()
	seedProfile(f.profiles, userA, 0)
	o := &entity.Order{
		ID: deps.IDs.New(), UserID: userA, PackageID: "starter",
		InvoiceNumber: "PK-seed", AmountIDR: 25000, CreditsGranted: 10,
		Status: status, RequestID: string(deps.IDs.New()),
		CreatedAt: deps.Clock.Now(), UpdatedAt: deps.Clock.Now(),
	}
	if err := f.orders.Create(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	return o
}

func dokuNotif(inv string, amount int, tx, order string) *port.Notification {
	return &port.Notification{InvoiceNumber: inv, AmountIDR: amount, Transaction: tx, OrderStatus: order}
}

// ===== 13d: buat order =====

func TestCreateOrder_Sukses(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	uc := NewOrderUsecase(deps)

	o, err := uc.CreateOrder(context.Background(), entity.Actor{UserID: userA}, "starter")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if o.Status != entity.OrderPending || o.AmountIDR != 25000 || o.CreditsGranted != 10 {
		t.Fatalf("order salah: %+v", o)
	}
	// harga dari konfigurasi server, bukan klien (auth_billing 2.2)
	if o.InvoiceNumber == "" || !strings.HasPrefix(o.InvoiceNumber, "PK-") {
		t.Fatalf("invoice harus PK-..., dapat %q", o.InvoiceNumber)
	}
	if len(o.InvoiceNumber) > 30 {
		t.Fatalf("invoice maks 30 char, dapat %d", len(o.InvoiceNumber))
	}
	if o.PaymentURL == nil || *o.PaymentURL != "https://pay.doku/x" {
		t.Fatalf("payment_url harus dari DOKU: %v", o.PaymentURL)
	}
	// expired_date 20261014100000 WIB = 03:00 UTC
	if o.ExpiredDate == nil || !o.ExpiredDate.Equal(time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("expired_date salah parse: %v", o.ExpiredDate)
	}
	if f.payment.lastReq.RequestID != o.RequestID || f.payment.lastReq.InvoiceNumber != o.InvoiceNumber {
		t.Fatalf("request ke DOKU tidak pakai invoice/request_id order")
	}
}

func TestCreateOrder_PaketDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	uc := NewOrderUsecase(deps)

	// package_id tidak dikenal -> 422 (bukan taking harga dari klien)
	if _, err := uc.CreateOrder(context.Background(), entity.Actor{UserID: userA}, "paket-mahal-1-perak"); !errors.Is(err, ErrValidation) {
		t.Fatalf("paket asing harus 422, dapat %v", err)
	}
	if len(f.orders.byID) != 0 {
		t.Fatalf("tidak boleh ada order tercipta")
	}
	// actor kosong -> 403
	if _, err := uc.CreateOrder(context.Background(), entity.Actor{}, "starter"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("actor kosong harus 403, dapat %v", err)
	}
}

// ===== 13d: webhook =====

func TestWebhook_SignatureSalah401(t *testing.T) {
	deps, _, f := newTestCore()
	seedOrder(t, deps, f, entity.OrderPending)
	f.signature.err = errors.New("bad signature")
	uc := NewOrderUsecase(deps)

	_, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("signature salah harus 401, dapat %v", err)
	}
	// parser tidak boleh terpanggil sebelum signature valid
	if len(f.signature.ops) != 1 {
		t.Fatalf("signature harus diverifikasi sekali")
	}
}

func TestWebhook_InvoiceTidakDikenal404(t *testing.T) {
	deps, _, f := newTestCore()
	seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif("PK-unknown", 25000, "SUCCESS", "")
	uc := NewOrderUsecase(deps)

	_, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("{}")})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("invoice asing harus 404, dapat %v", err)
	}
}

func TestWebhook_RequestIDBeda404(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "SUCCESS", "")
	f.parser.out.RequestID = "request-id-lain"
	uc := NewOrderUsecase(deps)

	_, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("{}")})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("request_id beda dari notifikasi kita harus 404, dapat %v", err)
	}
}

func TestWebhook_SUCCESS_SettleSekali(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "SUCCESS", "")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	res, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res.Settled {
		t.Fatalf("webhook success: %v %+v", err, res)
	}
	if got := f.profiles.byID[userA].Credits; got != 10 {
		t.Fatalf("kredit harus +10, dapat %d", got)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderSettlement || stored.PaidAt == nil {
		t.Fatalf("order harus settlement: %+v", stored)
	}
	// ledger: +10 reason topup ref_type order (auth_billing 2.3 step 3)
	row := f.ledger.ledger[len(f.ledger.ledger)-1]
	if row.Delta != 10 || row.Reason != entity.ReasonTopUp || row.RefType != entity.RefOrder || *row.RefID != o.ID {
		t.Fatalf("ledger topup salah: %+v", row)
	}

	// replay webhook DOKU (+12 jam) -> 200 tanpa perubahan data
	f.profiles.byID[userA].Credits = 999 // penanda: kalau settle lagi nilainya berubah
	res2, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res2.OK || !res2.AlreadyProcessed {
		t.Fatalf("replay harus OK already-processed: %v %+v", err, res2)
	}
	if f.profiles.byID[userA].Credits != 999 {
		t.Fatalf("replay tidak boleh menambah kredit")
	}
}

// Guard security.md 6 langkah 9a: nominal beda = 422.
func TestWebhook_NominalBedaDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 99999, "SUCCESS", "")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	_, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("nominal beda harus 422, dapat %v", err)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending {
		t.Fatalf("status tidak boleh berubah: %q", stored.Status)
	}
}

// FAILED diabaikan: bukan status akhir (security.md 6 langkah 9b).
func TestWebhook_FAILED_Diabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "FAILED", "")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	res, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res.OK || !res.Ignored {
		t.Fatalf("FAILED harus OK+ignored: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("FAILED tidak boleh mengubah apa pun: %+v", stored)
	}
}

// ORDER_EXPIRED dari pending -> expire, tanpa kredit (9c).
func TestWebhook_EXPIRED_DariPending(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "", "ORDER_EXPIRED")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	res, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res.Expired {
		t.Fatalf("expired: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderExpire || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("order harus expire tanpa kredit: %+v credits=%d", stored, f.profiles.byID[userA].Credits)
	}
}

// ORDER_EXPIRED datang setelah settlement: diabaikan, transisi satu arah (2.3).
func TestWebhook_EXPIRED_SetelahSettlementDiabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderSettlement)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "", "ORDER_EXPIRED")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	res, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res.Ignored {
		t.Fatalf("expired di atas settlement harus ignored: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderSettlement {
		t.Fatalf("settlement tidak boleh turun: %q", stored.Status)
	}
}

// ORDER_RECOVERED tidak diaktifkan v1: balas 200 tanpa perubahan (9d).
func TestWebhook_RECOVERED_Diabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.parser.out = dokuNotif(o.InvoiceNumber, 25000, "", "ORDER_RECOVERED")
	f.parser.out.RequestID = o.RequestID
	uc := NewOrderUsecase(deps)

	res, err := uc.ProcessWebhook(context.Background(), port.WebhookInput{RawBody: []byte("x")})
	if err != nil || !res.OK || !res.Ignored {
		t.Fatalf("recovered harus 200 ignored: %v %+v", err, res)
	}
}

// ===== 13d: expire lewat GET (jalan kedua, auth_billing 2.4) =====

func TestGetOrder_PendingLewatExpire_TanyaDOKU(t *testing.T) {
	deps, clock, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	exp := clock.Now().Add(-time.Hour)
	o.ExpiredDate = &exp
	_ = f.orders.Update(context.Background(), o)

	f.payment.statuses = map[string]*port.PaymentStatus{
		o.InvoiceNumber: {InvoiceNumber: o.InvoiceNumber, OrderStatus: "ORDER_EXPIRED"},
	}
	uc := NewOrderUsecase(deps)

	got, policy, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userA}, o.ID)
	if err != nil || policy != OrderMarkedExpire {
		t.Fatalf("get order expired: %v %+v", err, policy)
	}
	if got.Status != entity.OrderExpire || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("harus expire tanpa kredit: %+v", got)
	}
}

// Late settlement (security.md 6): expire yang DOKU bilang SUCCESS -> settle.
func TestGetOrder_LateSettlement(t *testing.T) {
	deps, clock, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	exp := clock.Now().Add(-time.Hour)
	o.ExpiredDate = &exp
	_ = f.orders.Update(context.Background(), o)

	f.payment.statuses = map[string]*port.PaymentStatus{
		o.InvoiceNumber: {InvoiceNumber: o.InvoiceNumber, Transaction: "SUCCESS", AmountIDR: 25000},
	}
	uc := NewOrderUsecase(deps)

	got, _, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userA}, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entity.OrderSettlement {
		t.Fatalf("late settlement harus pindah ke settlement: %+v", got)
	}
	if f.profiles.byID[userA].Credits != 10 {
		t.Fatalf("late settlement harus +10 kredit, dapat %d", f.profiles.byID[userA].Credits)
	}
}

func TestGetOrder_OwnerCheckDanPendingTidakPanggilDOKU(t *testing.T) {
	deps, _, f := newTestCore()
	o := seedOrder(t, deps, f, entity.OrderPending)
	uc := NewOrderUsecase(deps)

	// user lain: 403
	if _, _, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userB}, o.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("user lain harus 403, dapat %v", err)
	}
	// belum lewat expired_date: tidak memanggil DOKU
	if f.payment.statuses != nil {
		t.Fatalf("DOKU tidak boleh dipanggil untuk order pending fresh")
	}
	// id non-UUID
	if _, _, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userA}, entity.ID("xx")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id non-UUID harus 404, dapat %v", err)
	}
}

func TestParseDOKUExpiry(t *testing.T) {
	if parseDOKUExpiry("20261014100000") == nil {
		t.Fatal("format valid tidak boleh nil")
	}
	if parseDOKUExpiry("rusak") != nil {
		t.Fatal("format rusak harus nil")
	}
}
