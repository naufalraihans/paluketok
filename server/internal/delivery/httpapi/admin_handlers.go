// Handler admin (openapi.md bagian admin) + GET/POST artifact + AI slot.
// Semua berada di grup Admin (middleware admin 403) kecuali dinamis.
package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
	"paluketok/server/pkg/response"
)

// Handlers field tambahan untuk admin (setup di router; nil-safe di test lama).
// Admin usecase dipegang Handlers.Admin.

// --- GET /v1/admin/projects ---
func (h *Handlers) AdminListProjects(w http.ResponseWriter, r *http.Request) {
	page := pageFrom(r)
	list, err := h.Admin.ListProjects(r.Context(), actorFrom(r), page)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, projectDTO(&list[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"projects": out, "limit": page.Limit, "offset": page.Offset})
}

// --- GET /v1/admin/runs?status=&user_id= ---
func (h *Handlers) AdminListRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	userID := entity.ID(q.Get("user_id"))
	if userID != "" && !entity.ValidID(string(userID)) {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"error": response.ErrorBody{Code: response.CodeValidation, Message: "user_id tidak valid"}})
		return
	}
	list, err := h.Admin.ListRuns(r.Context(), actorFrom(r), status, userID, pageFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, runDTO(&list[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"runs": out})
}

// --- PATCH /v1/admin/users/{id} (plan_tier saja; is_admin tak tersentuh) ---
func (h *Handlers) AdminPatchUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		PlanTier string `json:"plan_tier"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"error": response.ErrorBody{Code: response.CodeValidation, Message: "body tidak valid"}})
		return
	}
	if err := h.Admin.SetPlanTier(r.Context(), actorFrom(r), id, entity.PlanTier(body.PlanTier)); err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"id": id, "plan_tier": body.PlanTier})
}

// --- POST /v1/admin/users/{id}/credits (grant manual) ---
func (h *Handlers) AdminGrantCredits(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Credits        int    `json:"credits"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"error": response.ErrorBody{Code: response.CodeValidation, Message: "body tidak valid"}})
		return
	}
	err := h.Credits.Grant(r.Context(), actorFrom(r), id, usecase.GrantInput{
		Credits: body.Credits, IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"id": id, "credits_granted": body.Credits})
}

// --- GET /v1/admin/decisions ---
func (h *Handlers) AdminListDecisions(w http.ResponseWriter, r *http.Request) {
	list, err := h.Admin.ListDecisions(r.Context(), actorFrom(r), pageFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, decisionDTOOf(&list[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"decisions": out})
}

// --- GET /v1/admin/settings/ai ---
func (h *Handlers) AdminListAISettings(w http.ResponseWriter, r *http.Request) {
	list, err := h.Admin.ListAISettings(r.Context(), actorFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	type slotDTO struct {
		Slot            string `json:"slot"`
		Provider        string `json:"provider"`
		BaseURL         string `json:"base_url"`
		Model           string `json:"model"`
		APIKey          string `json:"api_key"`
		MaxInputTokens  int    `json:"max_input_tokens"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		IsActive        bool   `json:"is_active"`
	}
	out := make([]slotDTO, 0, len(list))
	for _, s := range list {
		out = append(out, slotDTO{Slot: s.Slot, Provider: s.Provider, BaseURL: s.BaseURL,
			Model: s.Model, APIKey: s.APIKey, MaxInputTokens: s.MaxInputTokens,
			MaxOutputTokens: s.MaxOutputTokens, IsActive: s.IsActive})
	}
	response.JSON(w, http.StatusOK, map[string]any{"slots": out})
}

// --- PUT /v1/admin/settings/ai/{slot} ---
func (h *Handlers) AdminSetAISetting(w http.ResponseWriter, r *http.Request) {
	slot := r.PathValue("slot")
	if slot == "" {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"error": response.ErrorBody{Code: response.CodeNotFound, Message: "slot tidak ada"},
		})
		return
	}
	var body struct {
		Provider        string `json:"provider"`
		BaseURL         string `json:"base_url"`
		Model           string `json:"model"`
		APIKey          string `json:"api_key"`
		MaxInputTokens  int    `json:"max_input_tokens"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		IsActive        bool   `json:"is_active"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"error": response.ErrorBody{Code: response.CodeValidation, Message: "body tidak valid"}})
		return
	}
	s := &port.AISetting{Slot: slot, Provider: body.Provider, BaseURL: body.BaseURL,
		Model: body.Model, APIKey: body.APIKey, MaxInputTokens: body.MaxInputTokens,
		MaxOutputTokens: body.MaxOutputTokens, IsActive: body.IsActive}
	if err := h.Admin.SetAISetting(r.Context(), actorFrom(r), s); err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"slot": slot, "is_active": s.IsActive})
}

// --- POST /v1/admin/settings/ai/test (uji koneksi, tanpa menyimpan) ---
func (h *Handlers) AdminTestAISetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slot    string `json:"slot"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&body); err != nil {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"error": response.ErrorBody{Code: response.CodeValidation, Message: "body tidak valid"}})
		return
	}
	ok, latency, err := h.Admin.TestAISetting(r.Context(), actorFrom(r), port.AISetting{
		Slot: body.Slot, BaseURL: body.BaseURL, Model: body.Model, APIKey: body.APIKey,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	msg := "Koneksi berhasil. Model menjawab dalam " + latency.Round(time.Millisecond).String() + "."
	if !ok {
		msg = "Server terjangkau, tapi model tidak menjawab seperti yang diharapkan."
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"ok": ok, "message": msg, "latency_ms": latency.Milliseconds(),
	})
}

// --- GET /v1/artifacts/{id} (owner; openapi.md baris 20) ---
func (h *Handlers) GetArtifact(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	art, err := h.Runs.GetArtifact(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, artifactDTO(art))
}

// --- POST /v1/artifacts/{id}/regenerate (openapi.md baris 21) ---
func (h *Handlers) RegenerateArtifact(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	art, err := h.Runs.RegenerateArtifact(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusAccepted, artifactDTO(art))
}

// decisionDTOOf tanpa state/criteria mentah (aturan openapi.md).
func decisionDTOOf(d *entity.Decision) map[string]any {
	return map[string]any{
		"id": d.ID, "run_id": d.RunID, "stage": d.Stage, "question": d.Question,
		"gate": d.Gate, "round": d.Round, "choice": d.Choice, "confidence": d.Confidence,
		"jev_decision_id": d.JEVDecisionID, "created_at": d.CreatedAt,
	}
}
