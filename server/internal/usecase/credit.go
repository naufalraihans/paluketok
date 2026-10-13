package usecase

import (
	"context"
	"errors"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// CreditUsecase menangani saldo dan ledger. Semua perubahan lewat
// LedgerRepo (FOR UPDATE dan transaksi ada di repository, security.md 7).
type CreditUsecase struct {
	deps Deps
}

func NewCreditUsecase(d Deps) *CreditUsecase {
	return &CreditUsecase{deps: d}
}

// GrantInput adalah input POST /v1/admin/users/{id}/credits (openapi.md).
type GrantInput struct {
	Credits        int
	IdempotencyKey string
}

// Grant menambah kredit oleh admin. Syarat: actor admin, credits > 0,
// idempotency_key tidak kosong (kunci idempotensi uq_ledger_one_grant_per_key).
func (u *CreditUsecase) Grant(ctx context.Context, actor entity.Actor, userID entity.ID, in GrantInput) error {
	if !actor.IsAdmin {
		return ErrForbidden
	}
	if in.Credits <= 0 {
		return ErrValidation
	}
	if in.IdempotencyKey == "" {
		return ErrValidation
	}
	if !entity.ValidID(string(userID)) {
		return ErrNotFound
	}
	if _, err := u.deps.Profiles.GetByID(ctx, userID); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return wrap(err)
	}
	if err := u.deps.Ledger.Grant(ctx, userID, in.Credits, actor.UserID, in.IdempotencyKey); err != nil {
		return wrap(err)
	}
	return nil
}

// Me mengembalikan profil actor untuk GET /v1/me (auth_billing.md 1.4).
// Token version sudah diverifikasi middleware; usecase hanya membaca.
func (u *CreditUsecase) Me(ctx context.Context, actor entity.Actor) (*entity.Profile, error) {
	if actor.UserID == "" {
		return nil, ErrForbidden
	}
	p, err := u.deps.Profiles.GetByID(ctx, actor.UserID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	return p, nil
}

// RefundRun me-refund run yang gagal (auth_billing.md 3.2). Refund kedua untuk
// run yang sama diharuskan no-op oleh repository (unique index), bukan error.
func (u *CreditUsecase) RefundRun(ctx context.Context, actor entity.Actor, runID entity.ID) error {
	if actor.UserID == "" {
		return ErrForbidden
	}
	if !entity.ValidID(string(runID)) {
		return ErrNotFound
	}
	run, err := u.deps.Runs.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return ErrNotFound
		}
		return wrap(err)
	}
	if run.UserID != actor.UserID && !actor.IsAdmin {
		return ErrForbidden
	}
	if run.Status != entity.RunFailed {
		return ErrConflict
	}
	if run.CreditsCharged <= 0 {
		return ErrValidation
	}
	return wrap(u.deps.Ledger.RefundRun(ctx, run, run.CreditsCharged))
}
