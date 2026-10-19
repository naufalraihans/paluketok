// Package httpapi merangkai delivery HTTP Fase 5 (workflow.md 18-21):
// DTO, handler, route. Handler TIDAK memanggil repository/adapter langsung —
// lewat usecase (docs/plan/structure.md aturan dependency).
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
	// Chat: percakapan perancangan per proyek (Fase 4).
	Chat *usecase.ChatUsecase

	// SubmitOp antrean worker (Fase 6); nil di test worker-less.
	Submitter Submitter
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
	// port.ErrForbidden datang dari adapter gateway (signature webhook tidak
	// cocok). Bukan kesalahan server: pemanggilnya memang tidak diizinkan.
	case errors.Is(err, port.ErrForbidden):
		code, status, msg = response.CodeForbidden, http.StatusForbidden, "tidak diizinkan"
	case errors.Is(err, port.ErrNotFound):
		code, status, msg = response.CodeNotFound, http.StatusNotFound, "tidak ditemukan"
	case errors.Is(err, port.ErrValidation):
		code, status, msg = response.CodeValidation, http.StatusUnprocessableEntity, "data tidak valid"
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

// pathRef menerima UUID ATAU slug dari URL. Dipakai rute proyek yang harus
// menerima dua bentuk: UUID (tautan lama) dan slug (tautan ramah baca).
func pathRef(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	v := r.PathValue(name)
	if v == "" {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"error": response.ErrorBody{Code: response.CodeNotFound, Message: "tidak ditemukan"},
		})
		return "", false
	}
	return v, true
}

func actorFrom(r *http.Request) entity.Actor {
	a, _ := middleware.ActorFrom(r.Context())
	return a
}

// pageFrom baca limit/offset. Absen/aneh → default 20, cap 100, offset >= 0.
// Tanpa ini limit 0 sampai ke SQL (`limit 0`) dan list balik kosong.
func pageFrom(r *http.Request) port.Page {
	const (
		defaultLimit = 20
		maxLimit     = 100
	)
	q := r.URL.Query()
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}
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
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	actor := actorFrom(r)
	p, err := h.Projects.GetProject(r.Context(), actor, ref)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	d := projectDTO(p)
	// Ringkasan run terbaru ikut di sini supaya halaman proyek tidak perlu
	// mengandalkan run id di sessionStorage (proyek dibuka di perangkat lain
	// tetap tahu progresnya). Gagal baca run tidak boleh menjatuhkan proyek.
	if run, rerr := h.Runs.LatestRun(r.Context(), actor, p.ID); rerr == nil && run != nil {
		d["latest_run"] = runDTO(run)
	}
	response.JSON(w, http.StatusOK, d)
}

func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	var in projectIn
	if !decodeJSON(w, r, &in) {
		return
	}
	id, err := h.Projects.ResolveProjectID(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
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
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	id, err := h.Projects.ResolveProjectID(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
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
		"slug":       p.Slug,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}

// --- runs ---

func (h *Handlers) CreateRun(w http.ResponseWriter, r *http.Request) {
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	// URL proyek boleh berbentuk slug; run tetap disimpan dengan UUID proyek.
	id, err := h.Projects.ResolveProjectID(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	// Body opsional: skema tahap yang sudah disetujui di layar konfirmasi.
	// Tanpa body / tanpa "stages", skema disusun judge saat itu juga.
	var body struct {
		Stages []struct {
			Key          string   `json:"key"`
			Label        string   `json:"label"`
			Intent       string   `json:"intent"`
			OutputFormat string   `json:"output_format"`
			DependsOn    []string `json:"depends_on"`
		} `json:"stages"`
	}
	confirmed := []entity.PlanStage(nil)
	if r.Body != nil {
		if derr := json.NewDecoder(r.Body).Decode(&body); derr == nil {
			for _, s := range body.Stages {
				st := entity.PlanStage{
					Key:          entity.Stage(s.Key),
					Label:        s.Label,
					Intent:       s.Intent,
					OutputFormat: entity.OutputFormat(s.OutputFormat),
				}
				for _, d := range s.DependsOn {
					st.DependsOn = append(st.DependsOn, entity.Stage(d))
				}
				confirmed = append(confirmed, st)
			}
			if len(confirmed) > 0 {
				// Ditolak di sini, bukan di worker: lebih baik 400 daripada run
				// jalan dengan skema yang sudah dipastikan tidak sah.
				clean, nerr := entity.NormalizePlanStages(confirmed)
				if nerr != nil {
					response.JSON(w, http.StatusBadRequest,
						map[string]any{"error": map[string]string{
							"code": "validation_error", "message": nerr.Error()}})
					return
				}
				confirmed = clean
			}
		}
	}
	run, err := h.Runs.CreateRun(r.Context(), actorFrom(r), id, confirmed)
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
	plan, _ := h.Runs.PlanStagesOfRun(r.Context(), actorFrom(r), run.ID)
	response.JSON(w, http.StatusAccepted, runDTOwithStages(run, plan))
}

// PreviewStages mengembalikan USULAN skema tahap tanpa membuat run dan tanpa
// memotong kredit: isi layar konfirmasi sebelum user menekan Mulai.
func (h *Handlers) PreviewStages(w http.ResponseWriter, r *http.Request) {
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	id, err := h.Projects.ResolveProjectID(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	stages, note, err := h.Runs.PreviewStages(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := map[string]any{"stages": stagesDTO(stages)}
	if note != "" {
		out["note"] = note
	}
	response.JSON(w, http.StatusOK, out)
}

// RunStages membaca skema tahap satu run (openapi.md GET /v1/runs/{id}/stages).
func (h *Handlers) RunStages(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	stages, err := h.Runs.PlanStagesOfRun(r.Context(), actorFrom(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"stages": stagesDTO(stages)})
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

// ListProjectArtifacts: POST-sidebar progresif (docs/rencana-chat-planning.md B.4).
// Artefak terbaru per kind untuk satu project, tanpa perlu run id dari klien.
func (h *Handlers) ListProjectArtifacts(w http.ResponseWriter, r *http.Request) {
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	as, err := h.Projects.ListProjectArtifacts(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(as))
	for i := range as {
		d := artifactDTO(&as[i])
		// Sidebar tidak butuh isi lengkap; jangan kirim konten ke daftar.
		delete(d, "content")
		out = append(out, d)
	}
	response.JSON(w, http.StatusOK, map[string]any{"artifacts": out})
}

// --- chat konteks proyek (Fase 4) ---

func messageDTO(m *entity.ProjectMessage) map[string]any {
	d := map[string]any{
		"id":         m.ID,
		"project_id": m.ProjectID,
		"role":       m.Role,
		"content":    m.Content,
		"created_at": m.CreatedAt,
	}
	// Kosong -> tidak dikirim, supaya klien tidak perlu menebak null.
	if m.SuggestedStage != "" {
		d["suggested_stage"] = m.SuggestedStage
	}
	if m.Model != "" {
		d["model"] = m.Model
	}
	if m.TokensIn > 0 || m.TokensOut > 0 {
		d["usage"] = map[string]any{"input_tokens": m.TokensIn, "output_tokens": m.TokensOut}
	}
	return d
}

// SendMessage (POST /v1/projects/{id}/messages). Chat GRATIS: tidak ada
// potongan kredit dan tidak ada baris credits_ledger.
//
// `session_id` opsional: kosong = sesi terakhir proyek, dan bila proyek belum
// punya sesi, satu sesi baru dibuat dari pesan pertama.
func (h *Handlers) SendMessage(w http.ResponseWriter, r *http.Request) {
	if h.Chat == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Content   string `json:"content"`
		SessionID string `json:"session_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	m, err := h.Chat.SendMessageInSession(r.Context(), actorFrom(r), ref, body.SessionID, body.Content)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, messageDTO(m))
}

// ListMessages (GET /v1/projects/{id}/messages): riwayat percakapan urut waktu.
//
// limit=/offset= dipakai untuk MENELUSURI riwayat (default 20, cap 100),
// sedangkan konteks yang dikirim ke model tetap 20 pesan terakhir
// (usecase.ChatMaxHistory) — dua angka itu memang berbeda urusannya.
//
// `?session_id=` memfilter ke satu sesi. Tanpa itu seluruh proyek dibaca, dan
// itu mempertahankan perilaku pemanggil lama.
func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request) {
	if h.Chat == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	var (
		ms  []entity.ProjectMessage
		err error
	)
	if sid := r.URL.Query().Get("session_id"); sid != "" {
		ms, err = h.Chat.MessagesInSession(r.Context(), actorFrom(r), ref, sid, pageFrom(r))
	} else {
		ms, err = h.Chat.ListMessages(r.Context(), actorFrom(r), ref, pageFrom(r))
	}
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(ms))
	for i := range ms {
		out = append(out, messageDTO(&ms[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"messages": out})
}

// ListChatSessions (GET /v1/projects/{id}/sessions): daftar sesi proyek.
func (h *Handlers) ListChatSessions(w http.ResponseWriter, r *http.Request) {
	if h.Chat == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	list, err := h.Chat.Sessions(r.Context(), actorFrom(r), ref)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, sessionDTO(&list[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"sessions": out})
}

// CreateChatSession (POST /v1/projects/{id}/sessions): buat sesi baru.
func (h *Handlers) CreateChatSession(w http.ResponseWriter, r *http.Request) {
	if h.Chat == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Judul string `json:"judul"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	s, err := h.Chat.NewSession(r.Context(), actorFrom(r), ref, body.Judul)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, sessionDTO(s))
}

// DeleteChatSession (DELETE /v1/projects/{id}/sessions/{sid}).
//
// Sesi terakhir tidak bisa dihapus: proyek tanpa sesi membuat layar chat
// kosong tanpa jalan kembali.
func (h *Handlers) DeleteChatSession(w http.ResponseWriter, r *http.Request) {
	if h.Chat == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	ref, ok := pathRef(w, r, "id")
	if !ok {
		return
	}
	sid, ok := pathRef(w, r, "sid")
	if !ok {
		return
	}
	if err := h.Chat.DeleteSession(r.Context(), actorFrom(r), ref, sid); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sessionDTO bentuk satu sesi di API.
func sessionDTO(s *entity.ChatSession) map[string]any {
	return map[string]any{
		"id":         s.ID,
		"project_id": s.ProjectID,
		"judul":      s.Judul,
		"created_at": s.CreatedAt,
		"updated_at": s.UpdatedAt,
	}
}

// skillDTO bentuk satu skill di API.
//
// Wajib lewat DTO, bukan entity langsung: entity memakai nama medan Go
// (`Nama`, `Isi`) sehingga JSON-nya jadi huruf besar dan klien membaca
// `undefined`. Semua endpoint lain di paket ini memakai pola yang sama.
func skillDTO(s *entity.Skill) map[string]any {
	return map[string]any{
		"id":         s.ID,
		"nama":       s.Nama,
		"ringkasan":  s.Ringkasan,
		"isi":        s.Isi,
		"created_at": s.CreatedAt,
		"updated_at": s.UpdatedAt,
	}
}

// skillDTOList bentuk daftar skill.
func skillDTOList(list []entity.Skill) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, skillDTO(&list[i]))
	}
	return out
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

// stageDTO bentuk satu tahap di API. `check_spec` dikirim mentah (json.RawMessage)
// supaya klien bisa menampilkan/menyuntingnya tanpa server mendikte bentuknya.
func stageDTO(s *entity.PlanStage) map[string]any {
	d := map[string]any{
		"key":           s.Key,
		"label":         s.Label,
		"position":      s.Position,
		"intent":        s.Intent,
		"output_format": s.OutputFormat,
	}
	if len(s.CheckSpec) > 0 {
		d["check_spec"] = json.RawMessage(s.CheckSpec)
	}
	if len(s.DependsOn) > 0 {
		d["depends_on"] = s.DependsOn
	}
	return d
}

func stagesDTO(list []entity.PlanStage) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, stageDTO(&list[i]))
	}
	return out
}

// runDTOwithStages dipakai saat MEMBUAT run: skema tahapnya ikut supaya klien
// tidak perlu menanyakan lagi.
func runDTOwithStages(r *entity.PlanRun, plan []entity.PlanStage) map[string]any {
	d := runDTO(r)
	d["stages"] = stagesDTO(plan)
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
