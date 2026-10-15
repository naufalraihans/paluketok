// Webhook Midtrans: POST /v1/webhooks/midtrans. Tanpa JWT.
//
// Midtrans MENGIRIM signature_key di body:
//
//	SHA512(order_id + status_code + gross_amount + ServerKey)
//
// ServerKey hanya diketahui Midtrans dan kita, jadi notifikasi palsu tidak bisa
// dibuat tanpa kunci itu. Verifikasi tanda tangan ada di usecase
// (OrderUsecase.ProcessWebhook) karena ia membutuhkan urutan yang sama dengan
// pencarian order dan pencocokan nominal.
//
// Rute tidak memakai segmen rahasia: tanda tangan sudah menjadi buktinya, dan
// URL yang mudah ditebak membuat konfigurasi dashboard Midtrans lebih sederhana.
package httpapi

import (
	"errors"
	"io"
	"net/http"

	"paluketok/server/internal/port"
	"paluketok/server/pkg/response"
)

const webhookMaxBody = 16 << 10 // security.md bagian 6 langkah 1 dan bagian 8

// WebhookMidtrans memproses notifikasi Midtrans. Balas 2xx untuk notifikasi
// yang sah supaya Midtrans berhenti mengulang; 4xx untuk yang tidak bisa
// dipercaya, sehingga salah konfigurasi cepat terlihat di dashboard.
func (h *Handlers) WebhookMidtrans(w http.ResponseWriter, r *http.Request) {
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

	// Parse dulu: body rusak ditolak 422 sebelum menyentuh order mana pun.
	notif, err := h.Orders.ParseWebhook(raw)
	if err != nil {
		// Payload uji dari dashboard ({"test":true}) bukan pembayaran: jawab 200
		// supaya tombol "Test notification URL" hijau, tanpa memproses apa pun.
		if errors.Is(err, port.ErrTestNotification) {
			response.JSON(w, http.StatusOK, map[string]any{"received": true, "test": true})
			return
		}
		writeErr(w, r, err)
		return
	}
	res, err := h.Orders.ProcessWebhook(r.Context(), *notif)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"received": true, "settled": res.Settled, "expired": res.Expired,
		"already_processed": res.AlreadyProcessed, "ignored": res.Ignored,
	})
}
