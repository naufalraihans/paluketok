// Webhook DOKU: POST /v1/webhooks/doku. Tanpa JWT; signature HMAC-SHA256 satu-
// satunya gerbang (plan/security.md bagian 6). Urutan: body mentah 16 KB ->
// header lengkap -> verify (hmac.Equal di adapter) -> parse -> usecase.
package httpapi

import (
	"io"
	"net/http"

	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
	"paluketok/server/pkg/response"
)

const webhookMaxBody = 16 << 10 // security.md bagian 6 langkah 1 dan bagian 8

// WebhookDOKU memproses callback DOKU. Selalu balas 2xx cepat untuk kasus
// yang sudah dikenal (signature valid), 4xx hanya untuk request yang tidak
// bisa dipercaya ATAU nominal tidak cocok (tolak, plans security.md langkah 7).
func (h *Handlers) WebhookDOKU(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, webhookMaxBody+1))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"error": response.ErrorBody{Code: response.CodeValidation, Message: "body tidak terbaca"},
		})
		return
	}
	if len(raw) > webhookMaxBody {
		response.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{
			"error": response.ErrorBody{Code: response.CodeValidation, Message: "body terlalu besar"},
		})
		return
	}

	in := port.WebhookInput{
		ClientID:         r.Header.Get("Client-Id"),
		RequestID:        r.Header.Get("Request-Id"),
		RequestTimestamp: r.Header.Get("Request-Timestamp"),
		Signature:        r.Header.Get("Signature"),
		RequestTarget:    h.NotifyPath,
		RawBody:          raw,
	}
	res, err := h.Orders.ProcessWebhook(r.Context(), in)
	if err != nil {
		switch err {
		case usecase.ErrUnauthorized, usecase.ErrNotFound:
			// signature salah / invoice asing: 401/404 tanpa perubahan data.
			writeErr(w, r, err)
		default:
			writeErr(w, r, err)
		}
		return
	}
	// Replay / ignored / settled / expired: 200. DOKU hanya butuh tahu sudah diterima.
	response.JSON(w, http.StatusOK, map[string]any{
		"received": true, "settled": res.Settled, "expired": res.Expired,
		"already_processed": res.AlreadyProcessed,
	})
}
