package usecase

import (
	"context"
	"errors"
	"testing"

	"paluketok/server/internal/entity"
)

// ===== 13b: kredit (grant admin, /v1/me, refund run) =====

func TestGrant_AdminOK(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userB, 0)
	uc := NewCreditUsecase(deps)

	if err := uc.Grant(context.Background(), entity.Actor{UserID: admin, IsAdmin: true}, userB,
		GrantInput{Credits: 5, IdempotencyKey: "kv-1"}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	if got := f.profiles.byID[userB].Credits; got != 5 {
		t.Fatalf("saldo harus 5, dapat %d", got)
	}
	// guard ledger: reason grant, actor_id + idempotency_key terisi (13b).
	row := f.ledger.ledger[len(f.ledger.ledger)-1]
	if row.Reason != entity.ReasonGrant || row.ActorID == nil || *row.ActorID != admin {
		t.Fatalf("ledger grant salah: %+v", row)
	}
	if row.IdempotencyKey == nil || *row.IdempotencyKey != "kv-1" {
		t.Fatalf("idempotency_key tidak tercatat: %+v", row)
	}
}

func TestGrant_ReplayIdempotencyKeyNoOp(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userB, 0)
	uc := NewCreditUsecase(deps)

	in := GrantInput{Credits: 5, IdempotencyKey: "kv-dobel"}
	_ = uc.Grant(context.Background(), entity.Actor{UserID: admin, IsAdmin: true}, userB, in)
	_ = uc.Grant(context.Background(), entity.Actor{UserID: admin, IsAdmin: true}, userB, in)
	if got := f.profiles.byID[userB].Credits; got != 5 {
		t.Fatalf("replay kunci sama tidak boleh menambah: saldo %d, harus 5", got)
	}
}

func TestGrant_BukanAdminDitolak(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userB, 0)
	uc := NewCreditUsecase(deps)

	err := uc.Grant(context.Background(), entity.Actor{UserID: userA}, userB, GrantInput{Credits: 5, IdempotencyKey: "k"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("bukan admin harus ErrForbidden, dapat %v", err)
	}
	if f.profiles.byID[userB].Credits != 0 {
		t.Fatalf("saldo tidak boleh berubah")
	}
}

// Guard 13b: delta 0 / negatif, idempotency_key kosong, target tidak ada.
func TestGrant_Guard(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userB, 0)
	uc := NewCreditUsecase(deps)
	act := entity.Actor{UserID: admin, IsAdmin: true}

	cases := []struct {
		nama string
		in   GrantInput
		id   entity.ID
	}{
		{"delta 0", GrantInput{Credits: 0, IdempotencyKey: "k1"}, userB},
		{"delta negatif", GrantInput{Credits: -3, IdempotencyKey: "k2"}, userB},
		{"kunci kosong", GrantInput{Credits: 5}, userB},
		{"user tidak ada", GrantInput{Credits: 5, IdempotencyKey: "k3"}, entity.ID("99999999-9999-4999-8999-999999999999")},
		{"id bukan uuid", GrantInput{Credits: 5, IdempotencyKey: "k4"}, entity.ID("bukan-uuid")},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			if err := uc.Grant(context.Background(), act, c.id, c.in); !errors.Is(err, ErrValidation) && !errors.Is(err, ErrNotFound) {
				t.Fatalf("harus 422/404, dapat %v", err)
			}
		})
	}
	if len(f.ledger.ledger) != 0 {
		t.Fatalf("tidak boleh ada baris ledger: %d", len(f.ledger.ledger))
	}
}

func TestMe(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 7)
	uc := NewCreditUsecase(deps)

	p, err := uc.Me(context.Background(), entity.Actor{UserID: userA})
	if err != nil || p.Credits != 7 {
		t.Fatalf("me: %v, credits %d", err, p.Credits)
	}
	if _, err := uc.Me(context.Background(), entity.Actor{UserID: entity.ID("99999999-9999-4999-8999-999999999999")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("user tak ada harus 404, dapat %v", err)
	}
	if _, err := uc.Me(context.Background(), entity.Actor{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("actor kosong harus 403, dapat %v", err)
	}
}

func TestRefundRun_SekaliSaja(t *testing.T) {
	deps, _, f := newTestCore()
	seedProfile(f.profiles, userA, 1)
	uc := NewRunUsecase(deps)
	proj, err := NewProjectUsecase(deps).CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{Name: "p", Brief: "b", Stack: entity.Stack{FE: "svelte", BE: "go", DB: "supabase", Payment: "midtrans"}})
	if err != nil {
		t.Fatal(err)
	}
	run, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, proj.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	// run gagal (mis. worker / shutdown) -> refund
	if err := uc.FailRun(context.Background(), run.ID, "server_restart"); err != nil {
		t.Fatalf("fail run: %v", err)
	}
	if got := f.profiles.byID[userA].Credits; got != 1 {
		t.Fatalf("saldo harus balik ke 1, dapat %d", got)
	}
	// refund kedua pada run sama = no-op (uq_ledger_one_refund_per_run)
	if err := uc.FailRun(context.Background(), run.ID, "lagi"); err != nil {
		t.Fatalf("fail ulang harus no-op, dapat %v", err)
	}
	if got := f.profiles.byID[userA].Credits; got != 1 {
		t.Fatalf("refund dobel: saldo %d", got)
	}
}
