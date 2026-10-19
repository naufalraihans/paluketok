package httpapi

import (
	"net/http"

	"paluketok/server/internal/usecase"
	"paluketok/server/pkg/response"
)

// Handler skill perancangan (migrasi 0009).
//
// Semua lewat grup admin di router, jadi `is_admin` sudah ditegakkan middleware.
// Usecase memeriksa ulang supaya aturannya tetap berlaku bila handler ini
// dipasang di jalur lain kelak.

type skillReq struct {
	Nama      string `json:"nama"`
	Ringkasan string `json:"ringkasan"`
	Isi       string `json:"isi"`
}

// AdminListSkills (GET /v1/admin/skills).
func (h *Handlers) AdminListSkills(w http.ResponseWriter, r *http.Request) {
	if h.Admin == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	list, err := h.Admin.ListSkills(r.Context(), actorFrom(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	// Objek berisi array, bukan array telanjang, supaya penambahan bidang di
	// kemudian hari tidak merusak pemanggil.
	response.JSON(w, http.StatusOK, map[string]any{"skills": skillDTOList(list)})
}

// AdminCreateSkill (POST /v1/admin/skills).
func (h *Handlers) AdminCreateSkill(w http.ResponseWriter, r *http.Request) {
	if h.Admin == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	var in skillReq
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.Admin.CreateSkill(r.Context(), actorFrom(r), in.Nama, in.Ringkasan, in.Isi)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, skillDTO(s))
}

// AdminUpdateSkill (PUT /v1/admin/skills/{id}).
func (h *Handlers) AdminUpdateSkill(w http.ResponseWriter, r *http.Request) {
	if h.Admin == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	var in skillReq
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.Admin.UpdateSkill(r.Context(), actorFrom(r), r.PathValue("id"), in.Ringkasan, in.Isi)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, skillDTO(s))
}

// AdminDeleteSkill (DELETE /v1/admin/skills/{id}).
func (h *Handlers) AdminDeleteSkill(w http.ResponseWriter, r *http.Request) {
	if h.Admin == nil {
		writeErr(w, r, usecase.ErrUnavailable)
		return
	}
	if err := h.Admin.DeleteSkill(r.Context(), actorFrom(r), r.PathValue("id")); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
