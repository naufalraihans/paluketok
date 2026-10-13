package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Kode error mengikuti plan/openapi.md bagian Error.
const (
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeValidation         = "validation_error"
	CodeInsufficientCredit = "insufficient_credits"
	CodeRateLimited        = "rate_limited"
	CodeUnavailable        = "unavailable"
	CodeInternal           = "internal"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// StatusFor memetakan kode error ke status HTTP. Kode tak dikenal -> 500.
func StatusFor(code string) int {
	switch code {
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidation:
		return http.StatusUnprocessableEntity
	case CodeInsufficientCredit:
		return http.StatusPaymentRequired
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// JSON menulis payload sukses.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("gagal encode response", "err", err)
	}
}

// Fail menulis error dalam bentuk {"error":{"code","message"}}.
// Pesan internal tidak pernah diteruskan ke klien: detail masuk log.
func Fail(w http.ResponseWriter, code, message string) {
	JSON(w, StatusFor(code), errorEnvelope{Error: ErrorBody{Code: code, Message: message}})
}

// FailInternal mencatat sebab sebenarnya ke log, klien hanya dapat pesan generik.
func FailInternal(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("error internal", "err", err, "method", r.Method, "path", r.URL.Path)
	Fail(w, CodeInternal, "Terjadi kesalahan")
}
