package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// seedOrder menyiapkan satu order pending (atau status lain) milik userA,
// lengkap dengan transaksi yang terdaftar di sisi gateway palsu.
//
// Midtrans memakai invoice_number kita sebagai order_id, jadi kuncinya adalah
// InvoiceNumber — bukan id transaksi terpisah.
func seedOrder(t *testing.T, deps Deps, f *fakes, status entity.OrderStatus) *entity.Order {
	t.Helper()
	o := &entity.Order{
		ID: deps.IDs.New(), UserID: userA, PackageID: "starter",
		InvoiceNumber: "PK-seed", AmountIDR: 25000, CreditsGranted: 10,
		Status: status, RequestID: string(deps.IDs.New()),
		CreatedAt: deps.Clock.Now(), UpdatedAt: deps.Clock.Now(),
	}
	if err := f.orders.Create(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	// Transaksi terdaftar di sisi gateway: verifikasi balik akan menemukannya.
	f.verifier.tx[o.InvoiceNumber] = &port.VerifiedTransaction{
		OrderID: o.InvoiceNumber, AmountIDR: o.AmountIDR,
		Status: "settlement", Paid: true,
	}
	return o
}

// midtransParser meniru adapter Midtrans: baca field yang dipakai, dan body
// tanpa signature_key / rusak jadi ErrValidation. Ini definisi yang dipakai tes
// webhook di sini (adapter asli punya tes sendiri di adapter/payment/midtrans).
type midtransParser struct{}

func (midtransParser) Parse(raw []byte) (*port.VerifiedNotification, error) {
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
		return nil, ErrValidation
	}
	if hb.OrderID == "" || hb.StatusCode == "" || hb.SignatureKey == "" {
		return nil, ErrValidation
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

// notifBody menyusun body notifikasi Midtrans dengan signature SAH untuk
// ServerKey yang dipegang fakeSigner. Ini yang membuat tes melatih jalur
// verifikasi tanda tangan sungguhan, bukan melewatinya.
func notifBody(t *testing.T, f *fakes, orderID, gross, status string) []byte {
	t.Helper()
	return f.signer.signBody(orderID, gross, status)
}

// notif membaca body jadi VerifiedNotification lewat parser yang sama dengan
// produksi (midtransParser), supaya tes tidak memasang jalur pintas.
func notif(t *testing.T, raw []byte) port.VerifiedNotification {
	t.Helper()
	n, err := midtransParser{}.Parse(raw)
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	return *n
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
	if o.AmountIDR != 25000 || o.CreditsGranted != 10 || o.Status != entity.OrderPending {
		t.Fatalf("order janggal: %+v", o)
	}
	if o.PaymentURL == nil || !strings.HasPrefix(*o.PaymentURL, "https://") {
		t.Fatalf("payment_url harus diisi gateway: %+v", o.PaymentURL)
	}
	if f.payment.calls != 1 {
		t.Fatalf("gateway harus dipanggil sekali, dapat %d", f.payment.calls)
	}
	stored, err := f.orders.GetByID(context.Background(), o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PaymentURL == nil || *stored.PaymentURL != *o.PaymentURL {
		t.Fatalf("payment_url harus tersimpan: %+v", stored.PaymentURL)
	}
}

func TestCreateOrder_PaketDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	uc := NewOrderUsecase(deps)

	if _, err := uc.CreateOrder(context.Background(), entity.Actor{UserID: userA}, "paket_sultan"); !errors.Is(err, ErrValidation) {
		t.Fatalf("paket asing harus 422, dapat %v", err)
	}
	if f.payment.calls != 0 {
		t.Fatalf("paket asing tidak boleh memanggil gateway, dapat %d", f.payment.calls)
	}
}

// ===== 13d: webhook (Midtrans: signature + verifikasi balik) =====

// Inti keamanan Midtrans, lapis 1: signature harus cocok. Body dengan
// signature_key karangan (ServerKey penyerang) ditolak SEBELUM order dicari.
func TestWebhook_SignatureSalahDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	deps.Notifications = midtransParser{}
	uc := NewOrderUsecase(deps)

	// Signature dihitung dengan ServerKey penyerang.
	const code = "200"
	attacker := sha512Hex(o.InvoiceNumber + code + "25000.00" + "SB-Mid-server-PENYERANG")
	body, _ := json.Marshal(map[string]any{
		"order_id": o.InvoiceNumber, "status_code": code, "gross_amount": "25000.00",
		"transaction_status": "settlement", "signature_key": attacker,
	})
	in := notif(t, body)

	if _, err := uc.ProcessWebhook(context.Background(), in); !errors.Is(err, port.ErrForbidden) {
		t.Fatalf("signature salah harus 403, dapat %v", err)
	}
	// Order tidak boleh tersentuh, dan gateway tidak perlu dipanggil.
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending {
		t.Fatalf("status tidak boleh berubah: %q", stored.Status)
	}
	if f.profiles.byID[userA].Credits != 0 {
		t.Fatal("signature salah tidak boleh menambah kredit")
	}
}

// Lapis 2: signature sah, tapi Midtrans tidak mengenal order itu (transaksi
// dihapus / data uji) -> 404, walau body terlihat sempurna.
func TestWebhook_OrderTakDikenalGateway404(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	// Gateway tidak mengenal order ini.
	delete(f.verifier.tx, o.InvoiceNumber)
	deps.Notifications = midtransParser{}
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "settlement"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil {
		t.Fatalf("order tak dikenal gateway harus diabaikan (200), dapat %v", err)
	}
	if !res.Ignored {
		t.Fatalf("order tak dikenal gateway harus Ignored, dapat %+v", res)
	}
	// Verifikasi balik HARUS benar-benar dicoba — kalau tidak, kita cuma
	// mempercayai body webhook.
	if len(f.verifier.ops) != 1 || f.verifier.ops[0] != "verify:"+o.InvoiceNumber {
		t.Fatalf("verifikasi balik harus dipanggil sekali, dapat %v", f.verifier.ops)
	}
	if f.profiles.byID[userA].Credits != 0 {
		t.Fatal("webhook gagal verifikasi tidak boleh menambah kredit")
	}
}

// order_id yang tidak terkait order mana pun diabaikan (200), bukan 404.
//
// 404 membuat Midtrans mengulang notifikasi terus-menerus dan dashboard
// menandai endpoint GAGAL — terlihat di journalctl prod 2026-10-15 17:13 ketika
// tombol "Test notification URL" mengirim order_id karangan. Signature tetap
// diverifikasi lebih dulu (gerbang pertama), lalu order tidak ditemukan.
func TestWebhook_OrderIDTanpaOrderDiabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	deps.Notifications = midtransParser{}
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, "PK-tidak-ada", "25000.00", "settlement"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil {
		t.Fatalf("order tidak ada harus diabaikan (200), dapat %v", err)
	}
	if !res.Ignored || res.Settled {
		t.Fatalf("order tidak ada harus Ignored tanpa Settled, dapat %+v", res)
	}
	if len(f.verifier.ops) != 0 {
		t.Fatalf("tanpa order tidak boleh memanggil gateway, dapat %v", f.verifier.ops)
	}
}

// Body rusak / tanpa signature_key = 422 di ParseWebhook, tanpa menyentuh apa pun.
func TestWebhook_TanpaSignature422(t *testing.T) {
	deps, _, f := newTestCore()
	seedOrder(t, deps, f, entity.OrderPending)
	deps.Notifications = midtransParser{}
	uc := NewOrderUsecase(deps)

	for _, body := range []string{
		`{"order_id":"PK-seed","status_code":"200","gross_amount":"25000.00"}`, // tanpa signature_key
		`bukan-json`, // body rusak
	} {
		if _, err := uc.ParseWebhook([]byte(body)); !errors.Is(err, ErrValidation) {
			t.Fatalf("body %q harus 422, dapat %v", body, err)
		}
	}
	if len(f.verifier.ops) != 0 {
		t.Fatalf("body rusak tidak boleh memanggil gateway, dapat %v", f.verifier.ops)
	}
}

// Midtrans mengembalikan order_id lain dari yang kita tanyakan -> 404.
func TestWebhook_OrderIDBukanMilikOrder404(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	// Gateway bilang transaksi ini untuk order lain.
	f.verifier.tx[o.InvoiceNumber].OrderID = "PK-orang-lain"
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "settlement"))
	if _, err := uc.ProcessWebhook(context.Background(), in); !errors.Is(err, ErrNotFound) {
		t.Fatalf("order_id bukan milik order harus 404, dapat %v", err)
	}
}

func TestWebhook_Lunas_SettleSekali(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "settlement"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res.Settled {
		t.Fatalf("webhook lunas: %v %+v", err, res)
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

	// Replay: Midtrans mengulang notifikasi. Harus idempoten.
	f.profiles.byID[userA].Credits = 999 // penanda: kalau settle lagi nilainya berubah
	res2, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res2.OK || !res2.AlreadyProcessed {
		t.Fatalf("replay harus OK already-processed: %v %+v", err, res2)
	}
	if f.profiles.byID[userA].Credits != 999 {
		t.Fatal("replay tidak boleh menambah kredit")
	}
}

// capture + fraud accept juga berarti lunas (kartu kredit Midtrans).
func TestWebhook_CaptureFraudAccept_Settle(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	tx := f.verifier.tx[o.InvoiceNumber]
	tx.Paid = true
	tx.Status = "capture"
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "capture"))
	in.FraudStatus = "accept"
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res.Settled {
		t.Fatalf("capture/accept harus settle: %v %+v", err, res)
	}
	if f.profiles.byID[userA].Credits != 10 {
		t.Fatalf("kredit harus +10, dapat %d", f.profiles.byID[userA].Credits)
	}
}

// Guard security.md 6 langkah 9a: nominal beda = 422.
func TestWebhook_NominalBedaDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.verifier.tx[o.InvoiceNumber].AmountIDR = 99999
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "99999.00", "settlement"))
	if _, err := uc.ProcessWebhook(context.Background(), in); !errors.Is(err, ErrValidation) {
		t.Fatalf("nominal beda harus 422, dapat %v", err)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending {
		t.Fatalf("status tidak boleh berubah: %q", stored.Status)
	}
}

// Belum lunas dan bukan status pembatalan: tidak ada perubahan data.
func TestWebhook_BelumLunas_Diabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.verifier.tx[o.InvoiceNumber].Paid = false
	f.verifier.tx[o.InvoiceNumber].Status = "pending"
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "pending"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res.OK || !res.Ignored {
		t.Fatalf("belum lunas harus OK+ignored: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("tidak boleh mengubah apa pun: %+v", stored)
	}
}

// Transaksi kedaluwarsa dari pending -> expire, tanpa kredit.
func TestWebhook_Expired_DariPending(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.verifier.tx[o.InvoiceNumber].Paid = false
	f.verifier.tx[o.InvoiceNumber].Status = "expire"
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "expire"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res.Expired {
		t.Fatalf("expire: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderExpire || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("order harus expire tanpa kredit: %+v credits=%d", stored, f.profiles.byID[userA].Credits)
	}
}

// cancel / deny / failure sama-sama menurunkan pending -> expire.
func TestWebhook_CancelDenyFailure_JadiExpire(t *testing.T) {
	for _, status := range []string{"cancel", "deny", "failure"} {
		t.Run(status, func(t *testing.T) {
			deps, _, f := newTestCore()
			o := seedOrder(t, deps, f, entity.OrderPending)
			f.verifier.tx[o.InvoiceNumber].Paid = false
			f.verifier.tx[o.InvoiceNumber].Status = status
			uc := NewOrderUsecase(deps)

			in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", status))
			res, err := uc.ProcessWebhook(context.Background(), in)
			if err != nil || !res.Expired {
				t.Fatalf("%s harus jadi expire: %v %+v", status, err, res)
			}
			stored, _ := f.orders.GetByID(context.Background(), o.ID)
			if stored.Status != entity.OrderExpire {
				t.Fatalf("order harus expire: %q", stored.Status)
			}
		})
	}
}

// Expire datang setelah settlement: diabaikan, transisi satu arah (2.3).
func TestWebhook_Expired_SetelahSettlementDiabaikan(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderSettlement)
	f.verifier.tx[o.InvoiceNumber].Paid = false
	f.verifier.tx[o.InvoiceNumber].Status = "expire"
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "expire"))
	res, err := uc.ProcessWebhook(context.Background(), in)
	if err != nil || !res.Ignored {
		t.Fatalf("expire di atas settlement harus ignored: %v %+v", err, res)
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderSettlement {
		t.Fatalf("settlement tidak boleh turun: %q", stored.Status)
	}
}

// Gateway tidak bisa dihubungi: error diteruskan (Midtrans akan retry), bukan
// settle buta.
func TestWebhook_GatewayMatiTidakSettle(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	f.verifier.err = port.ErrUnavailable
	uc := NewOrderUsecase(deps)

	in := notif(t, notifBody(t, f, o.InvoiceNumber, "25000.00", "settlement"))
	if _, err := uc.ProcessWebhook(context.Background(), in); err == nil {
		t.Fatal("gateway mati harus mengembalikan error")
	}
	if f.profiles.byID[userA].Credits != 0 {
		t.Fatal("gateway mati tidak boleh menambah kredit")
	}
	stored, _ := f.orders.GetByID(context.Background(), o.ID)
	if stored.Status != entity.OrderPending {
		t.Fatalf("status harus tetap pending: %q", stored.Status)
	}
}

// ===== 13d: expire lewat GET (jalan kedua, auth_billing 2.4) =====

func TestGetOrder_PendingLewatExpire_JadiExpire(t *testing.T) {
	deps, clock, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	exp := clock.Now().Add(-time.Hour)
	o.ExpiredDate = &exp
	_ = f.orders.Update(context.Background(), o)

	f.verifier.tx[o.InvoiceNumber].Paid = false
	f.verifier.tx[o.InvoiceNumber].Status = "expire"
	uc := NewOrderUsecase(deps)

	got, policy, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userA}, o.ID)
	if err != nil || policy != OrderMarkedExpire {
		t.Fatalf("get order expired: %v %+v", err, policy)
	}
	if got.Status != entity.OrderExpire || f.profiles.byID[userA].Credits != 0 {
		t.Fatalf("harus expire tanpa kredit: %+v", got)
	}
}

// Late settlement (security.md 6): order yang lewat waktu tapi Midtrans bilang
// lunas -> harus settle, kredit masuk.
func TestGetOrder_LateSettlement(t *testing.T) {
	deps, clock, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	exp := clock.Now().Add(-time.Hour)
	o.ExpiredDate = &exp
	_ = f.orders.Update(context.Background(), o)
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

func TestGetOrder_OwnerCheckDanPendingTidakPanggilGateway(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 0)
	o := seedOrder(t, deps, f, entity.OrderPending)
	uc := NewOrderUsecase(deps)

	// user lain: 403
	if _, _, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userB}, o.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("user lain harus 403, dapat %v", err)
	}
	// belum lewat expired_date: tidak memanggil gateway
	if len(f.verifier.ops) != 0 {
		t.Fatalf("gateway tidak boleh dipanggil untuk order pending fresh, dapat %v", f.verifier.ops)
	}
	// id non-UUID
	if _, _, err := uc.GetOrder(context.Background(), entity.Actor{UserID: userA}, entity.ID("xx")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id non-UUID harus 404, dapat %v", err)
	}
}
