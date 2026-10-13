package port

import (
	"context"
)

// LLMMessage adalah satu pesan untuk client OpenAI-compatible.
type LLMMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// LLMRequest adalah satu panggilan generasi. Slot menunjuk baris ai_settings
// (plan/planner.md bagian 9); nilainya dari DB, bukan dari klien.
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
// (plan/structure.md "Sumber tunggal angka timeout adapter").
type LLM interface {
	Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}

// JudgeQuestion adalah satu pertanyaan untuk JEV (plan/jev_ai.md).
type JudgeQuestion struct {
	ID           string
	Type         string // "choice" | "noul" | "score"
	Instructions string
	Criteria     any
}

// JudgeRequest adalah satu panggilan gate JEV. State ringkas, bukan repository
// penuh (plan/planner.md bagian 4).
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
// (plan/planner.md bagian 4, plan/jev_ai.md).
type Judge interface {
	Decide(ctx context.Context, req JudgeRequest) (*JudgeResponse, error)
}

// LineItem adalah satu baris tagihan ke DOKU.
type LineItem struct {
	Name     string
	Price    int
	Quantity int
}

// PaymentRequest adalah permintaan membuat halaman bayar.
type PaymentRequest struct {
	InvoiceNumber string
	RequestID     string
	AmountIDR     int
	Items         []LineItem
}

// PaymentResponse berisi halaman bayar dan sumber kedaluwarsa dari DOKU
// (plan/auth_billing.md bagian 2.3-2.4).
type PaymentResponse struct {
	PaymentURL  string
	ExpiredDate string // format yyyyMMddHHmmss UTC+7, apa adanya dari DOKU
	Raw         []byte
}

// PaymentStatus adalah hasil cek status ke DOKU.
type PaymentStatus struct {
	InvoiceNumber string
	Transaction   string // SUCCESS | FAILED | ...
	OrderStatus   string // ORDER_EXPIRED | ORDER_RECOVERED | ...
	AmountIDR     int
	Raw           []byte
}

// Payment adalah client DOKU Checkout. Timeout 20 detik, tanpa auto-retry
// (plan/security.md bagian 6).
type Payment interface {
	CreateOrder(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
	CheckStatus(ctx context.Context, invoiceNumber string) (*PaymentStatus, error)
}

// WebhookInput adalah notifikasi DOKU yang sudah dibaca mentah. Body mentah
// wajib dipakai apa adanya untuk Digest (plan/security.md bagian 6 langkah 1).
type WebhookInput struct {
	ClientID         string
	RequestID        string
	RequestTimestamp string
	Signature        string
	RequestTarget    string
	RawBody          []byte
}

// SignatureVerifier memverifikasi signature HMAC-SHA256 notifikasi DOKU
// (plan/security.md bagian 6 langkah 2-5).
type SignatureVerifier interface {
	Verify(in WebhookInput) error
}

// Notification adalah isi notifikasi DOKU setelah parse. Status pesanan dan
// status transaksi DIPISAH karena kedaluwarsa ada di order.status, bukan
// transaction.status (plan/security.md bagian 6 langkah 9).
type Notification struct {
	InvoiceNumber string
	RequestID     string
	AmountIDR     int
	Transaction   string // SUCCESS | FAILED | ...
	OrderStatus   string // ORDER_EXPIRED | ORDER_RECOVERED | ...
}

// NotificationParser mem-parse body notifikasi DOKU.
type NotificationParser interface {
	Parse(raw []byte) (*Notification, error)
}

// Jenis error adapter yang harus dikenali usecase. Adapter membungkus error
// transport/SDK-nya ke salah satu ini supaya usecase tidak tahu SDK.
var (
	// ErrInsufficientCredits dipetakan ke 402 (plan/openapi.md bagian Error).
	ErrInsufficientCredits = errKind("insufficient credits")
	// ErrNotFound dipetakan ke 404.
	ErrNotFound = errKind("not found")
	// ErrConflict dipetakan ke 409, mis. unique index kredit.
	ErrConflict = errKind("conflict")
	// ErrUnavailable dipetakan ke 503, mis. key LLM belum diisi admin
	// (plan/openapi.md bagian AI).
	ErrUnavailable = errKind("unavailable")
	// ErrForbidden dipetakan ke 403, mis. owner check gagal.
	ErrForbidden = errKind("forbidden")
	// ErrValidation dipetakan ke 422.
	ErrValidation = errKind("validation")
)

type errKind string

func (e errKind) Error() string { return string(e) }
