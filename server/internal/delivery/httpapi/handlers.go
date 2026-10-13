// Package httpapi merangkai delivery HTTP Fase 5 (workflow.md 18-21):
// DTO, handler, route. Handler TIDAK memanggil repository/adapter langsung —
// lewat usecase (plan/structure.md aturan dependency).
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"paluketok/server/internal/delivery/middleware"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
	"paluketok/server/pkg/response"
)

// Handlers memegang semua usecase. Route memakai struct ini.
type Handlers struct {
	Projects *usecase.ProjectUsecase
	Runs     *usecase.RunUsecase
	Credits  *usecase.CreditUsecase
	Orders   *usecase.OrderUsecase
	Admin    *usecase.AdminUsecase

	// SubmitOp antrean worker (Fase 6); nil di test worker-less.
	Submitter Submitter
	// NotifyPath: path notifikasi TERDAFTAR di Ba...[truncated]
	// (security.md bagian 6 langkah 4). Dari config, bukan dari header.
	NotifyPath string
}

// --- error mapping: usecase error -> HTTP sesuai openapi.md bagian Error ---

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	log := middleware.LogFrom(r.Context())
	code := response.CodeInternal
	status := http.StatusInternalServerError
	msg := "kesalahan server"
	var target *entity.ErrValidation
	switch {
	case errors.Is(err, usecase.ErrUnauthorized):
		code, status, msg = response.CodeUnauthorized, http.StatusUnauthorized, "tidak sah"
	case errors.Is(err, usecase.ErrForbidden):
		code, status, msg = response.CodeForbidden, http.StatusForbidden, "tidak diizinkan"
	case errors.Is(err, usecase.ErrNotFound):
		code, status, msg = response.CodeNotFound, http.StatusNotFound, "tidak ditemukan"
	case errors.Is(err, usecase.ErrConflict):
		code, status, msg = response.CodeRateLimited, http.StatusConflict, "konflik"
		code = "conflict"
	case errors.Is(err, usecase.ErrInsufficientCredits):
		code, status, msg = response.CodeInsufficientCredit, http.StatusPaymentRequired, "Saldo kurang"
	case errors.Is(err, usecase.ErrValidation):
		code, status, msg = response.CodeValidation, http.StatusUnprocessableEntity, "data tidak valid"
	case errors.Is(err, usecase.ErrUnavailable):
		code, status, msg = response.CodeUnavailable, http.StatusServiceUnavailable, "layanan tidak tersedia"
	case errors.As(err, &target):
		code, status, msg = response.CodeValidation, http.StatusUnprocessableEntity, target.Msg
	}
	// 5xx dilog, 4xx tidak.
	if status >= 500 {
		log.Error("handler error", "err", err)
	} else {
		log.Info("handler tolak", "err", err.Error())
	}
	response.JSON(w, status, map[string]any{
		"error": response.ErrorBody{Code: code, Message: msg},
	})
}

// --- DTO helpers ---

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefStack(s *entity.Stack) entity.Stack {
	if s == nil {
		return entity.Stack{}
	}
	return *s
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": response.ErrorBody{Code: response.CodeValidation, Message: "body harus JSON valid"},
		})
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (entity.ID, bool) {
	v := r.PathValue(name)
	if !entity.ValidID(v) {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"error": response.ErrorBody{Code: response.CodeNotFound, Message: "tidak ditemukan"},
		})
		return "", false
	}
	return entity.ID(v), true
}

func actorFrom(r *http.Request) entity.Actor {
	a, _ := middleware.ActorFrom(r.Context())
	return a
}

func pageFrom(r *http.Request) port.Page {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	return port.Page{Limit: limit, Offset: offset}
}

// --- /v1/me ---

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	p, err := h.Credits.Me(r.Context(), actorFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, meDTO(p))
}

func meDTO(p *entity.Profile) map[string]any {
	d := map[string]any{
		"id":         p.ID,
		"plan_tier":  p.PlanTier,
		"is_admin":   p.IsAdmin,
		"credits":    p.Credits,
		"created_at": p.CreatedAt,
	}
	if p.Username != nil {
		d["username"] = *p.Username
	}
	if p.FullName != nil {
		d["full_name"] = *p.FullName
	}
	if p.AvatarURL != nil {
		d["avatar_url"] = *p.AvatarURL
	}
	return d
}

// --- projects ---

type projectIn struct {
	Name   *string               `json:"name"`
	Brief  *string               `json:"brief"`
	Stack  *entity.Stack         `json:"stack"`
	Status *entity.ProjectStatus `json:"status"`
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var in projectIn
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := h.Projects.CreateProject(r.Context(), actorFrom(r), usecase.CreateProjectInput{
		Name: deref(in.Name), Brief: deref(in.Brief), Stack: derefStack(in.Stack),
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, projectDTO(p))
}

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	ps, err := h.Projects.ListProjects(r.Context(), actorFrom(r), pageFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(ps))
	for i := range ps {
		out = append(out, projectDTO(&ps[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"projects": out})
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.Projects.GetProject(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, projectDTO(p))
}

func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in projectIn
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := h.Projects.UpdateProject(r.Context(), actorFrom(r), id, usecase.UpdateProjectInput{
		Name: in.Name, Brief: in.Brief, Stack: in.Stack, Status: in.Status,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, projectDTO(p))
}

func (h *Handlers) ArchiveProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Projects.ArchiveProject(r.Context(), actorFrom(r), id); err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"status": "archived"})
}

func projectDTO(p *entity.Project) map[string]any {
	return map[string]any{
		"id":         p.ID,
		"owner_id":   p.OwnerID,
		"name":       p.Name,
		"brief":      p.Brief,
		"stack":      p.Stack,
		"status":     p.Status,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}

// --- runs ---

func (h *Handlers) CreateRun(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	run, err := h.Runs.CreateRun(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	// Worker pool (Fase 6): masukkan ke antrean. Gagal antrean = 503
	// (security.md bagian 8) - run tetap 'queued' di DB dan akan diambil
	// worker pada cold start berikutnya.
	if h.Submitter != nil {
		if serr := h.Submitter.Submit(run.ID); serr != nil {
			response.JSON(w, http.StatusServiceUnavailable,
				map[string]any{"error": map[string]string{
					"code": "unavailable", "message": "antrean penuh, coba lagi nanti"}})
			return
		}
	}
	response.JSON(w, http.StatusAccepted, runDTOwithStages(run))
}

func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	run, err := h.Runs.GetRun(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, runDTO(run))
}

func (h *Handlers) ListRunArtifacts(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	as, err := h.Runs.ListRunArtifacts(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(as))
	for i := range as {
		out = append(out, artifactDTO(&as[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"artifacts": out})
}

var stages = []entity.Stage{entity.StagePRD, entity.StageERD, entity.StageOpenAPI,
	entity.StageRoles, entity.StageSecurity, entity.StageStructure}

func runDTO(r *entity.PlanRun) map[string]any {
	d := map[string]any{
		"id":         r.ID,
		"project_id": r.ProjectID,
		"status":     r.Status,
		"progress":   r.Progress,
		"total":      r.Total,
	}
	if r.CurrentStage != nil {
		d["current_stage"] = *r.CurrentStage
	}
	d["escalations"] = r.Escalations
	return d
}

func runDTOwithStages(r *entity.PlanRun) map[string]any {
	d := runDTO(r)
	d["stages"] = stages
	d["credits_charged"] = r.CreditsCharged
	return d
}

func artifactDTO(a *entity.Artifact) map[string]any {
	return map[string]any{
		"id":         a.ID,
		"run_id":     a.RunID,
		"kind":       a.Kind,
		"version":    a.Version,
		"status":     a.Status,
		"content":    a.Content,
		"created_at": a.CreatedAt,
	}
}

// --- decisions ---

func (h *Handlers) ListDecisions(w http.ResponseWriter, r *http.Request) {
	runID := entity.ID(r.URL.Query().Get("run_id"))
	ds, err := h.Runs.ListDecisions(r.Context(), actorFrom(r), runID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	// openapi.md: tanpa state mentah dan criteria internal.
	out := make([]map[string]any, 0, len(ds))
	for _, d := range ds {
		out = append(out, map[string]any{
			"id": d.ID, "run_id": d.RunID, "stage": d.Stage, "question": d.Question,
			"gate": d.Gate, "round": d.Round, "choice": d.Choice,
			"confidence": d.Confidence, "created_at": d.CreatedAt,
		})
	}
	response.JSON(w, http.StatusOK, map[string]any{"decisions": out})
}

// --- orders ---

func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PackageID string `json:"package_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := h.Orders.CreateOrder(r.Context(), actorFrom(r), in.PackageID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, orderDTO(o))
}

func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	o, _, err := h.Orders.GetOrder(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, orderDTO(o))
}

func orderDTO(o *entity.Order) map[string]any {
	d := map[string]any{
		"order_id":        o.ID,
		"invoice_number":  o.InvoiceNumber,
		"package_id":      o.PackageID,
		"amount_idr":      o.AmountIDR,
		"credits_granted": o.CreditsGranted,
		"status":          o.Status,
		"created_at":      o.CreatedAt,
	}
	if o.PaymentURL != nil {
		d["payment_url"] = *o.PaymentURL
	}
	if o.ExpiredDate != nil {
		d["expired_date"] = o.ExpiredDate.Format("20060102150405")
	}
	return d
}
