package port

import (
	"context"
	"time"
)

// LLMMessage adalah satu pesan untuk client OpenAI-compatible.
type LLMMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// LLMRequest adalah satu panggilan generasi. Slot menunjuk baris ai_settings
// (docs/plan/planner.md bagian 9); nilainya dari DB, bukan dari klien.
type LLMRequest struct {
	Slot        string
	Messages    []LLMMessage
	MaxTokens   int
	Temperature float64
}

// LLMUsage adalah pemakaian token yang dilaporkan provider, untuk
// plan_runs.tokens_in / tokens_out (erd.dbml).
type LLMUsage struct {
	InputTokens  int
	OutputTokens int
}

// LLMResponse adalah hasil generasi.
type LLMResponse struct {
	Content string
	Model   string
	Usage   LLMUsage
}

// LLM adalah client OpenAI-compatible. Timeout 180 detik ada di adapter
// (docs/plan/structure.md "Sumber tunggal angka timeout adapter").
type LLM interface {
	Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}

// JudgeQuestion adalah satu pertanyaan untuk JEV (docs/plan/jev_ai.md).
type JudgeQuestion struct {
	ID           string
	Type         string // "choice" | "noul" | "score"
	Instructions string
	Criteria     any
}

// JudgeRequest adalah satu panggilan gate JEV. State ringkas, bukan repository
// penuh (docs/plan/planner.md bagian 4).
type JudgeRequest struct {
	State     any
	Questions []JudgeQuestion
}

// JudgeAnswer adalah jawaban JEV per question ID.
type JudgeAnswer struct {
	QuestionID    string
	Choice        string
	Confidence    float64
	Probabilities map[string]float64
}

// JudgeResponse adalah hasil gate JEV. DecisionID dicatat ke
// decisions.jev_decision_id (erd.dbml).
type JudgeResponse struct {
	DecisionID string
	Model      string
	Answers    map[string]JudgeAnswer
	State      any
	Criteria   any
}

// Judge adalah client gateway JEV. Timeout 30 detik, tanpa auto-retry
// (docs/plan/planner.md bagian 4, docs/plan/jev_ai.md).
type Judge interface {
	Decide(ctx context.Context, req JudgeRequest) (*JudgeResponse, error)
}

// LineItem adalah satu baris tagihan. Midtrans menerima item_details, tapi
// bentuknya (id/price/quantity/name) berbeda dari struct ini, jadi adapter yang
// menerjemahkan.
type LineItem struct {
	Name     string
	Price    int
	Quantity int
}

// PaymentRequest adalah permintaan membuat halaman bayar.
//
// Description, Email, dan ExpiresAt ditambahkan saat migrasi ke gateway
// Ringkasan item masuk Description; email dipakai Midtrans untuk
// mengirim bukti bayar.
type PaymentRequest struct {
	InvoiceNumber string
	RequestID     string
	AmountIDR     int
	Items         []LineItem
	Description   string // judul tagihan yang dilihat pembeli
	Email         string
	ExpiresAt     time.Time // zero = biarkan gateway memakai defaultnya
}

// PaymentResponse berisi halaman bayar dari gateway.
//
// Midtrans Snap membalas {token, redirect_url}: PaymentURL = redirect_url (halaman
// bayar Midtrans langsung, tak perlu halaman bayar sendiri), ExternalID = token.
// TransactionID kosong karena Midtrans memakai order_id KITA sebagai kunci
// transaksi — tidak ada id transaksi tambahan yang perlu disimpan.
type PaymentResponse struct {
	PaymentURL    string
	ExternalID    string // Midtrans: Snap token
	TransactionID string // kosong untuk Midtrans (kunci = order_id kita)
	Raw           []byte
}

// Payment adalah client gateway pembayaran. Timeout 20 detik, tanpa auto-retry
// (docs/plan/security.md bagian 6).
type Payment interface {
	CreateOrder(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

// VerifyInput menunjuk transaksi yang harus dibaca ulang dari gateway.
type VerifyInput struct {
	// OrderID adalah order_id yang kita kirim saat create (invoice_number).
	// Midtrans memakai nilai ini sebagai kunci transaksi.
	OrderID string
	// TransactionID opsional: alternatif pencarian lewat id transaksi Midtrans.
	TransactionID string
}

// VerifiedTransaction adalah kebenaran dari gateway, bukan dari webhook.
type VerifiedTransaction struct {
	OrderID       string // = invoice_number kita
	TransactionID string // id transaksi Midtrans (uuid)
	AmountIDR     int
	Status        string // Midtrans: capture | settlement | pending | deny | cancel | expire | failure | refund
	Method        string // Midtrans: payment_type, mis. bank_transfer / qris / credit_card
	MerchantID    string
	Paid          bool // settlement, atau capture yang lolos fraud check
	FraudStatus   string
	ExpiresAt     *time.Time
	Raw           []byte
}

// TransactionVerifier membaca ulang transaksi dari gateway memakai kredensial
// server. Lapis kedua setelah signature: walau signature sudah membuktikan
// notifikasi asli dari gateway, pembacaan ulang memastikan status TERKINI dan
// mencegah replay notifikasi lama.
type TransactionVerifier interface {
	VerifyTransaction(ctx context.Context, in VerifyInput) (*VerifiedTransaction, error)
}

// SignatureInput adalah data yang dibutuhkan untuk memverifikasi keaslian
// notifikasi. Midtrans menandatangani: SHA512(order_id + status_code +
// gross_amount + ServerKey).
type SignatureInput struct {
	OrderID    string
	StatusCode string
	GrossAmt   string
	Signature  string
	RawBody    []byte
}

// SignatureVerifier memverifikasi tanda tangan notifikasi gateway. Midtrans
// menyediakannya, jadi ini gerbang pertama; TransactionVerifier jadi gerbang
// kedua.
type SignatureVerifier interface {
	VerifySignature(in SignatureInput) error
}

// VerifiedNotification adalah notifikasi yang sudah dibaca dari body. Isinya
// BELUM terbukti — pemanggil wajib memverifikasi Signature lewat
// SignatureVerifier sebelum mempercayai OrderID/Status/GrossAmount.
type VerifiedNotification struct {
	OrderID       string // = invoice_number kita
	TransactionID string
	StatusCode    string
	GrossAmount   string
	Status        string // transaction_status
	FraudStatus   string
	Signature     string // signature_key dari body, untuk SignatureVerifier
	Event         string // nama status, untuk pencatatan
	RawBody       []byte
}

// NotificationParser mem-parse body notifikasi menjadi VerifiedNotification.
// Parse TIDAK memverifikasi apa pun — verifikasi dilakukan SignatureVerifier.
type NotificationParser interface {
	Parse(raw []byte) (*VerifiedNotification, error)
}

// Jenis error adapter yang harus dikenali usecase. Adapter membungkus error
// transport/SDK-nya ke salah satu ini supaya usecase tidak tahu SDK.
var (
	// ErrInsufficientCredits dipetakan ke 402 (docs/plan/openapi.md bagian Error).
	ErrInsufficientCredits = errKind("insufficient credits")
	// ErrNotFound dipetakan ke 404.
	ErrNotFound = errKind("not found")
	// ErrConflict dipetakan ke 409, mis. unique index kredit.
	ErrConflict = errKind("conflict")
	// ErrUnavailable dipetakan ke 503, mis. key LLM belum diisi admin
	// (docs/plan/openapi.md bagian AI).
	ErrUnavailable = errKind("unavailable")
	// ErrForbidden dipetakan ke 403, mis. owner check gagal.
	ErrForbidden = errKind("forbidden")
	// ErrValidation dipetakan ke 422.
	ErrValidation = errKind("validation")
	// ErrTestNotification menandai payload UJI dari dashboard gateway (bukan
	// pembayaran). Handler menjawab 200 supaya tombol "Test notification URL"
	// lolos, tanpa memproses apa pun.
	ErrTestNotification = errKind("test notification")
)

type errKind string

func (e errKind) Error() string { return string(e) }
