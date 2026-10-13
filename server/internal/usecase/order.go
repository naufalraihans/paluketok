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
	Label   string // teks tampil untuk item DOKU; ID tetap lowercase
	Credits int
	Amount  int // IDR
}

// Packages adalah tabel paket awal (auth_billing.md 2.2: Starter 10 kredit, 25.000).
var Packages = map[string]Package{
	"starter": {ID: "starter", Label: "Starter", Credits: 10, Amount: 25000},
}

// invoiceChars adalah alfabet invoice (check orders: ^[A-Za-z0-9-]{1,30}$).
const invoiceChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

// OrderUsecase menangani topup DOKU: buat order, status, webhook, expire.
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

// CreateOrder membuat order pending lalu memanggil DOKU
// (auth_billing.md 2.3). request_id = UUID dari IDGenerator, jadi retry dengan
// request_id sama diterjemahkan port.ErrConflict menjadi order yang ada.
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
	// pending di plan). Idempotensi ketat ada di webhook (request_id).

	inv, err := newInvoiceNumber(func(s string) bool {
		_, err := u.deps.Orders.GetByInvoiceNumber(ctx, s)
		return err == nil
	})
	if err != nil {
		return nil, wrap(err)
	}

	o := &entity.Order{
		ID:             u.deps.IDs.New(),
		UserID:         actor.UserID,
		PackageID:      pkg.ID,
		InvoiceNumber:  inv,
		AmountIDR:      pkg.Amount,
		CreditsGranted: pkg.Credits,
		Status:         entity.OrderPending,
		RequestID:      string(u.deps.IDs.New()),
		CreatedAt:      u.deps.Clock.Now(),
		UpdatedAt:      u.deps.Clock.Now(),
	}
	if err := u.deps.Orders.Create(ctx, o); err != nil {
		return nil, wrap(err)
	}

	resp, err := u.deps.Payment.CreateOrder(ctx, port.PaymentRequest{
		InvoiceNumber: o.InvoiceNumber,
		RequestID:     o.RequestID,
		AmountIDR:     o.AmountIDR,
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
	exp := parseDOKUExpiry(resp.ExpiredDate)
	o.ExpiredDate = exp
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
// expired_date, tandai expire tanpa kredit (auth_billing.md 2.4 jalan kedua).
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
	// Jalan expire kedua (auth_billing.md 2.4): GET order yang masih pending
	// dan lewat expired_date menanyakan status ke DOKU. Late settlement
	// (security.md 6): expire yang dibayar pindah ke settlement hanya bila
	// status API DOKU mengonfirmasi SUCCESS.
	if o.Status == entity.OrderPending && o.ExpiredDate != nil && u.deps.Clock.Now().After(*o.ExpiredDate) && u.deps.Payment != nil {
		st, err := u.deps.Payment.CheckStatus(ctx, o.InvoiceNumber)
		switch {
		case err != nil:
			// DOKU gagal/dead: order tetap pending, dibaca apa adanya
			// (jalan expire utama tetap webhook).
		case st.Transaction == "SUCCESS" && st.AmountIDR == o.AmountIDR:
			if err := u.deps.Ledger.SettleTopUp(ctx, o); err != nil {
				return nil, OrderUntouched, wrap(err)
			}
			fresh, err := u.deps.Orders.GetByID(ctx, o.ID)
			if err != nil {
				return nil, OrderUntouched, wrap(err)
			}
			return fresh, OrderUntouched, nil
		case st.OrderStatus == "ORDER_EXPIRED":
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

// ProcessWebhookConsumer adalah hasil pemrosesan webhook.
type ProcessWebhookResult struct {
	OK               bool
	AlreadyProcessed bool
	Settled          bool
	Expired          bool
	Ignored          bool
}

// ProcessWebhook memproses notifikasi DOKU sesuai security.md 6 langkah 6-9:
// signature dulu (401), cari order (404), idempotensi request_id (200 no-op),
// lalu cabang: SUCCESS -> settle sekali; FAILED -> diabaikan;
// ORDER_EXPIRED -> expire; ORDER_RECOVERED -> diabaikan (dicatat).
func (u *OrderUsecase) ProcessWebhook(ctx context.Context, in port.WebhookInput) (*ProcessWebhookResult, error) {
	if u.deps.Signature == nil || u.deps.Notifications == nil {
		return nil, ErrUnavailable
	}
	if err := u.deps.Signature.Verify(in); err != nil {
		return nil, ErrUnauthorized
	}
	notif, err := u.deps.Notifications.Parse(in.RawBody)
	if err != nil {
		return nil, asValidation(err)
	}

	o, err := u.deps.Orders.GetByInvoiceNumber(ctx, notif.InvoiceNumber)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	// Cross-check request_id notifikasi (transaction.original_request_id)
	// bila dikirim DOKU: beda order = bukan notifikasi kita (security.md 6
	// langkah 7: tidak dikenal = 404).
	if notif.RequestID != "" && notif.RequestID != o.RequestID {
		return nil, ErrNotFound
	}

	switch {
	case notif.Transaction == "SUCCESS":
		// Cocokkan nominal (security.md 6 langkah 9a) lalu settle sekali.
		if notif.AmountIDR != o.AmountIDR {
			return nil, ErrValidation
		}
		if o.Status == entity.OrderSettlement {
			return &ProcessWebhookResult{OK: true, AlreadyProcessed: true, Settled: true}, nil
		}
		if err := u.deps.Ledger.SettleTopUp(ctx, o); err != nil {
			return nil, wrap(err)
		}
		return &ProcessWebhookResult{OK: true, Settled: true}, nil
	case notif.OrderStatus == "ORDER_EXPIRED":
		// Kedaluwarsa hanya turun dari pending. Dari settlement tidak boleh
		// kembali (auth_billing.md 2.3).
		if o.Status != entity.OrderPending {
			return &ProcessWebhookResult{OK: true, Ignored: true}, nil
		}
		o.Status = entity.OrderExpire
		o.UpdatedAt = u.deps.Clock.Now()
		if err := u.deps.Orders.Update(ctx, o); err != nil {
			return nil, wrap(err)
		}
		return &ProcessWebhookResult{OK: true, Expired: true}, nil
	case notif.Transaction == "FAILED", notif.OrderStatus == "ORDER_RECOVERED":
		// FAILED bukan status akhir (checkout ganti metode), RECOVERED tidak
		// diaktifkan v1 (security.md 6 langkah 9b-9d). Tanpa perubahan data.
		return &ProcessWebhookResult{OK: true, Ignored: true}, nil
	default:
		return &ProcessWebhookResult{OK: true, Ignored: true}, nil
	}
}

// parseDOKUExpiry mengubah "yyyyMMddHHmmss" (UTC+7) menjadi time UTC.
// Nilai rusak dibiarkan nil: order tetap pending, expire lewat webhook.
func parseDOKUExpiry(s string) *time.Time {
	if len(s) != 14 {
		return nil
	}
	loc := time.FixedZone("WIB", 7*3600)
	t, err := time.ParseInLocation("20060102150405", s, loc)
	if err != nil {
		return nil
	}
	u := t.UTC()
	return &u
}
