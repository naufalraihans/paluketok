package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Paket topup. Server-side, harga dari klien diabaikan
// (auth_billing.md 2.2 "Paket harga ditetapkan di kode konfigurasi").
type Package struct {
	ID      string
	Label   string // teks tampil untuk tagihan; ID tetap lowercase
	Credits int
	Amount  int // IDR
}

// Packages adalah tabel paket awal (auth_billing.md 2.2: Starter 10 kredit, 25.000).
var Packages = map[string]Package{
	"starter": {ID: "starter", Label: "Starter", Credits: 10, Amount: 25000},
}

// orderTTL adalah masa berlaku tagihan (auth_billing.md 2.3: expired 1 jam).
// Dihitung di server lalu dikirim sebagai expiredAt ke gateway.
const orderTTL = time.Hour

// invoiceChars adalah alfabet invoice (check orders: ^[A-Za-z0-9-]{1,30}$).
const invoiceChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

// OrderUsecase menangani topup: buat order, status, webhook, expire.
type OrderUsecase struct {
	deps Deps
}

func NewOrderUsecase(d Deps) *OrderUsecase { return &OrderUsecase{deps: d} }

// newInvoiceNumber membuat "PK-<random>" unik, maks 30 karakter
// (auth_billing.md 2.3), dengan retry tabrakan yang sangat kecil.
func newInvoiceNumber(taken func(string) bool) (string, error) {
	buf := make([]byte, 8)
	for i := 0; i < 5; i++ {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		s := "PK-"
		for _, b := range buf {
			s += string(invoiceChars[int(b)%len(invoiceChars)])
		}
		if !taken(s) {
			return s, nil
		}
	}
	return "", errors.New("invoice number tidak kunjung unik")
}

// CreateOrder membuat order pending lalu memanggil gateway pembayaran
// (auth_billing.md 2.3).
func (u *OrderUsecase) CreateOrder(ctx context.Context, actor entity.Actor, packageID string) (*entity.Order, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	pkg, ok := Packages[strings.ToLower(strings.TrimSpace(packageID))]
	if !ok {
		return nil, ErrValidation
	}

	// Sudah ada order pending untuk paket yang sama? Tetap buat baru: klien
	// boleh punya beberapa order pending sekaligus (tidak ada aturan satu
	// pending di plan). Idempotensi ketat ada di webhook (transaction_id).

	inv, err := newInvoiceNumber(func(s string) bool {
		_, err := u.deps.Orders.GetByInvoiceNumber(ctx, s)
		return err == nil
	})
	if err != nil {
		return nil, wrap(err)
	}

	now := u.deps.Clock.Now()
	expiresAt := now.Add(orderTTL)
	o := &entity.Order{
		ID:             u.deps.IDs.New(),
		UserID:         actor.UserID,
		PackageID:      pkg.ID,
		InvoiceNumber:  inv,
		AmountIDR:      pkg.Amount,
		CreditsGranted: pkg.Credits,
		Status:         entity.OrderPending,
		RequestID:      string(u.deps.IDs.New()),
		ExpiredDate:    &expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := u.deps.Orders.Create(ctx, o); err != nil {
		return nil, wrap(err)
	}

	email := ""
	if actor.Email != "" {
		email = actor.Email
	}
	resp, err := u.deps.Payment.CreateOrder(ctx, port.PaymentRequest{
		InvoiceNumber: o.InvoiceNumber,
		RequestID:     o.RequestID,
		AmountIDR:     o.AmountIDR,
		Description:   "PaluKetok " + pkg.Label + " credits",
		Email:         email,
		ExpiresAt:     expiresAt,
		Items: []port.LineItem{{
			Name:     "PaluKetok " + pkg.Label + " credits",
			Price:    pkg.Amount,
			Quantity: 1,
		}},
	})
	if err != nil {
		return nil, wrap(err)
	}
	url := resp.PaymentURL
	o.PaymentURL = &url
	// Simpan Snap token sebagai jejak audit (bisa dibuka lagi di dashboard
	// Midtrans). Kunci transaksi tetap order_id kita, jadi tidak ada kolom
	// tambahan yang dibutuhkan untuk verifikasi webhook.
	if resp.ExternalID != "" {
		tok := resp.ExternalID
		o.GatewayToken = &tok
	}
	if err := u.deps.Orders.Update(ctx, o); err != nil {
		return nil, wrap(err)
	}
	return o, nil
}

// GetOrderPolicy adalah hasil pengecekan order saat dibaca.
type GetOrderPolicy int

const (
	OrderUntouched GetOrderPolicy = iota
	OrderMarkedExpire
)

// GetOrder membaca order milik actor. Bila masih pending dan lewat
// expired_date, tanyakan ke gateway tanpa webhook (auth_billing.md 2.4 jalan
// kedua). Late settlement (security.md 6): expire yang ternyata dibayar pindah
// ke settlement hanya bila verifikasi gateway mengonfirmasi lunas.
func (u *OrderUsecase) GetOrder(ctx context.Context, actor entity.Actor, id entity.ID) (*entity.Order, GetOrderPolicy, error) {
	if actor.UserID == "" {
		return nil, OrderUntouched, ErrForbidden
	}
	if !entity.ValidID(string(id)) {
		return nil, OrderUntouched, ErrNotFound
	}
	o, err := u.deps.Orders.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, OrderUntouched, ErrNotFound
		}
		return nil, OrderUntouched, wrap(err)
	}
	if o.UserID != actor.UserID && !actor.IsAdmin {
		return nil, OrderUntouched, ErrForbidden
	}
	if o.Status == entity.OrderPending && o.ExpiredDate != nil &&
		u.deps.Clock.Now().After(*o.ExpiredDate) && u.deps.Verifier != nil {

		tx, err := u.deps.Verifier.VerifyTransaction(ctx, port.VerifyInput{OrderID: o.InvoiceNumber})
		switch {
		case err != nil:
			// Gateway gagal/dead: order tetap pending, dibaca apa adanya
			// (jalan expire utama tetap webhook).
		case tx.Paid && tx.AmountIDR == o.AmountIDR:
			if err := u.deps.Ledger.SettleTopUp(ctx, o); err != nil {
				return nil, OrderUntouched, wrap(err)
			}
			fresh, err := u.deps.Orders.GetByID(ctx, o.ID)
			if err != nil {
				return nil, OrderUntouched, wrap(err)
			}
			return fresh, OrderUntouched, nil
		case !tx.Paid:
			o.Status = entity.OrderExpire
			o.UpdatedAt = u.deps.Clock.Now()
			if err := u.deps.Orders.Update(ctx, o); err != nil {
				return nil, OrderUntouched, wrap(err)
			}
			return o, OrderMarkedExpire, nil
		}
	}
	return o, OrderUntouched, nil
}

// ProcessWebhookResult adalah hasil pemrosesan webhook.
type ProcessWebhookResult struct {
	OK               bool
	AlreadyProcessed bool
	Settled          bool
	Expired          bool
	Ignored          bool
}

// ParseWebhook membaca body notifikasi jadi port.VerifiedNotification. Dipisah
// dari ProcessWebhook supaya handler bisa menolak body rusak (422) sebelum
// menyentuh apa pun.
func (u *OrderUsecase) ParseWebhook(raw []byte) (*port.VerifiedNotification, error) {
	if u == nil || u.deps.Notifications == nil {
		return nil, ErrUnavailable
	}
	n, err := u.deps.Notifications.Parse(raw)
	if err != nil {
		// Payload uji dashboard diteruskan apa adanya: handler butuh mengenalinya
		// untuk menjawab 200, bukan 422.
		if errors.Is(err, port.ErrTestNotification) {
			return nil, port.ErrTestNotification
		}
		return nil, asValidation(err)
	}
	if n == nil || n.OrderID == "" {
		return nil, ErrValidation
	}
	return n, nil
}

// ProcessWebhook memproses notifikasi Midtrans.
//
// Dua gerbang, keduanya harus lulus:
//
//  1. Signature: SHA512(order_id+status_code+gross_amount+ServerKey) dihitung
//     ulang dan dibandingkan dengan signature_key. ServerKey hanya diketahui
//     Midtrans dan kita, jadi notifikasi palsu gugur di sini.
//  2. Verifikasi balik: status TERKINI dibaca dari API Midtrans. Menutup celah
//     replay (notifikasi lama dikirim ulang) dan memastikan nominalnya benar.
//
// Order dicari lewat order_id (= invoice_number kita), bukan id transaksi
// terpisah: Midtrans memakai order_id kita sebagai kunci transaksi.
func (u *OrderUsecase) ProcessWebhook(ctx context.Context, n port.VerifiedNotification) (*ProcessWebhookResult, error) {
	if u.deps.Signatures == nil || u.deps.Verifier == nil {
		return nil, ErrUnavailable
	}
	if n.OrderID == "" {
		return nil, ErrValidation
	}

	// Gerbang 1: keaslian notifikasi.
	if err := u.deps.Signatures.VerifySignature(port.SignatureInput{
		OrderID:    n.OrderID,
		StatusCode: n.StatusCode,
		GrossAmt:   n.GrossAmount,
		Signature:  n.Signature,
		RawBody:    n.RawBody,
	}); err != nil {
		return nil, err
	}

	o, err := u.deps.Orders.GetByInvoiceNumber(ctx, n.OrderID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			// Order tidak dikenal BUKAN kesalahan pemanggil. Midtrans mengirim
			// notifikasi contoh dengan order_id karangan saat tombol "Test
			// notification URL" ditekan; order yang dihapus juga jatuh ke sini.
			// Balas 404 membuat Midtrans mengulang terus dan dashboard menandai
			// endpoint gagal. Dianggap "diabaikan" supaya berhenti, tanpa
			// menyentuh state apa pun. Keaslian sudah dijaga Gerbang 1 di atas,
			// jadi ini tidak bisa dipakai menandai order asli lunas.
			return &ProcessWebhookResult{Ignored: true}, nil
		}
		return nil, wrap(err)
	}

	// Gerbang 2: status terkini dari Midtrans.
	tx, err := u.deps.Verifier.VerifyTransaction(ctx, port.VerifyInput{OrderID: n.OrderID})
	if err != nil {
		// Order tidak ada di Midtrans: notifikasi tidak bisa dipertanggungjawabkan.
		if errors.Is(err, port.ErrNotFound) {
			return &ProcessWebhookResult{Ignored: true}, nil
		}
		return nil, wrap(err)
	}
	if tx == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	// order_id yang dikembalikan Midtrans harus sama dengan yang kita tanyakan.
	if tx.OrderID != "" && tx.OrderID != o.InvoiceNumber {
		return nil, ErrNotFound
	}
	// Nominal harus persis sama dengan harga paket yang kita tetapkan.
	if tx.AmountIDR != o.AmountIDR {
		return nil, ErrValidation
	}

	if tx.Paid {
		if o.Status == entity.OrderSettlement {
			return &ProcessWebhookResult{OK: true, AlreadyProcessed: true, Settled: true}, nil
		}
		if err := u.deps.Ledger.SettleTopUp(ctx, o); err != nil {
			return nil, wrap(err)
		}
		return &ProcessWebhookResult{OK: true, Settled: true}, nil
	}

	// Belum lunas. Turunkan pending -> expire hanya pada status kedaluwarsa.
	// Dari settlement tidak boleh kembali (auth_billing.md 2.3).
	if tx.Status == "expire" || tx.Status == "cancel" || tx.Status == "deny" || tx.Status == "failure" {
		if o.Status != entity.OrderPending {
			return &ProcessWebhookResult{OK: true, Ignored: true}, nil
		}
		o.Status = entity.OrderExpire
		o.UpdatedAt = u.deps.Clock.Now()
		if err := u.deps.Orders.Update(ctx, o); err != nil {
			return nil, wrap(err)
		}
		return &ProcessWebhookResult{OK: true, Expired: true}, nil
	}
	// Status lain (pending, authorize, refund): tanpa perubahan data.
	return &ProcessWebhookResult{OK: true, Ignored: true}, nil
}
