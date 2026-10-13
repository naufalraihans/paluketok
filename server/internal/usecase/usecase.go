// Package usecase memuat logika bisnis PaluKetok. Package ini tidak boleh
// import pgx, gorm, net/http, atau delivery/* (plan/architecture.md bagian 2).
package usecase

import (
	"errors"
	"fmt"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Deps adalah dependensi bersama semua usecase. Semua lewat port, sehingga
// usecase bisa dites tanpa DB dan tanpa HTTP (plan/architecture.md bagian 4).
type Deps struct {
	Profiles      port.ProfileRepo
	Projects      port.ProjectRepo
	Runs          port.RunRepo
	Artifacts     port.ArtifactRepo
	Decisions     port.DecisionRepo
	Orders        port.OrderRepo
	Ledger        port.LedgerRepo
	Payment       port.Payment
	Signature     port.SignatureVerifier
	Notifications port.NotificationParser
	Clock         port.Clock
	IDs           IDGenerator
}

// IDGenerator membuat UUID tanpa menarik dependensi UUID ke usecase.
// Implementasi produksi ada di cmd/web (composition root).
type IDGenerator interface {
	New() entity.ID
}

// ErrForbidden dibalas 403. Owner check ada di usecase, bukan handler
// (plan/architecture.md bagian 3).
var ErrForbidden = errors.New("forbidden")

// ErrNotFound dibalas 404.
var ErrNotFound = errors.New("not found")

// ErrValidation dibalas 422.
var ErrValidation = errors.New("validation")

// ErrInsufficientCredits dibalas 402 (plan/openapi.md bagian Error).
var ErrInsufficientCredits = errors.New("insufficient credits")

// ErrConflict dibalas 409.
var ErrConflict = errors.New("conflict")

// ErrUnavailable dibalas 503.
var ErrUnavailable = errors.New("unavailable")

// wrap memetakan error dari port ke error usecase. Error tak dikenal
// diteruskan apa adanya supaya handler bisa menjadikannya 500.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, port.ErrForbidden):
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	case errors.Is(err, port.ErrNotFound):
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	case errors.Is(err, port.ErrValidation):
		return fmt.Errorf("%w: %v", ErrValidation, err)
	case errors.Is(err, port.ErrInsufficientCredits):
		return fmt.Errorf("%w: %v", ErrInsufficientCredits, err)
	case errors.Is(err, port.ErrConflict):
		return fmt.Errorf("%w: %v", ErrConflict, err)
	case errors.Is(err, port.ErrUnavailable):
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return err
}

// asValidation membungkus error validasi dari entity agar usecase bisa
// memeriksanya dengan errors.As tanpa bergantung tipe entity.
func asValidation(err error) error {
	var ve *entity.ErrValidation
	if errors.As(err, &ve) {
		return fmt.Errorf("%w: %s", ErrValidation, ve.Msg)
	}
	return err
}

// pageOrDefault menerapkan batas halaman. Nilai dari klien tidak dipercaya.
func pageOrDefault(p port.Page) port.Page {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// ErrUnauthorized dibalas 401, antara lain webhook dengan signature salah.
var ErrUnauthorized = errors.New("unauthorized")
