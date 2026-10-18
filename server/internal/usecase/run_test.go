package usecase

import (
	"context"
	"errors"
	"testing"

	"paluketok/server/internal/entity"
)

func seedProject(t *testing.T, deps Deps, f *fakes, owner entity.ID) *entity.Project {
	t.Helper()
	seedProfile(f.profiles, owner, 3)
	p, err := NewProjectUsecase(deps).CreateProject(context.Background(), entity.Actor{UserID: owner},
		CreateProjectInput{Name: "proj", Brief: "b", Stack: entity.Stack{FE: "svelte", BE: "go", DB: "supabase", Payment: "midtrans"}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// ===== 13c: runs =====

func TestCreateRun_SuksesPotongKredit(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	uc := NewRunUsecase(deps)

	run, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if run.Status != entity.RunQueued || run.CreditsCharged != 1 || run.Total != entity.TotalStages {
		t.Fatalf("run awal salah: %+v", run)
	}
	if got := f.profiles.byID[userA].Credits; got != 2 {
		t.Fatalf("saldo harus 3-1=2, dapat %d", got)
	}
	// ledger: -1, reason run, ref plan_run (auth_billing 3.1)
	row := f.ledger.ledger[len(f.ledger.ledger)-1]
	if row.Delta != -1 || row.Reason != entity.ReasonRun || row.RefType != entity.RefPlanRun {
		t.Fatalf("ledger charge salah: %+v", row)
	}
}

// 3.1 langkah 2: saldo kurang dari 1 -> 402 insufficient_credits.
func TestCreateRun_SaldoKurang402(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	f.profiles.byID[userA].Credits = 0
	uc := NewRunUsecase(deps)

	_, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	if !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("harus 402, dapat %v", err)
	}
	if len(f.runs.byID) != 0 {
		t.Fatalf("run tidak boleh tercipta saat saldo kurang")
	}
}

// 13c: ada run running di project sama -> tolak 422 (instruksi user, menang
// atas planner.md:129). Run pertama dimatikan statusnya jadi running manual.
func TestCreateRun_DitolakSaatAdaRunning(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	uc := NewRunUsecase(deps)

	first, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	first.Status = entity.RunRunning
	if err := f.runs.Update(context.Background(), first); err != nil {
		t.Fatal(err)
	}

	_, err = uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("harus 422, dapat %v", err)
	}
	// project lain (owner sama) tidak terpengaruh
	p2 := seedProject(t, deps, f, userA)
	if _, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p2.ID, nil); err != nil {
		t.Fatalf("project lain harus jalan: %v", err)
	}
	// run done/failed/queued lama tidak menghalangi
	first.Status = entity.RunDone
	_ = f.runs.Update(context.Background(), first)
	if _, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil); err != nil {
		t.Fatalf("run done tidak boleh menghalangi: %v", err)
	}
}

func TestCreateRun_OwnerCheckDanValidasiID(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	seedProfile(f.profiles, admin, 3)
	uc := NewRunUsecase(deps)

	// user lain: 403
	if _, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userB}, p.ID, nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("user lain harus 403, dapat %v", err)
	}
	// admin boleh untuk project siapa pun (roles.md)
	if _, err := uc.CreateRun(context.Background(), entity.Actor{UserID: admin, IsAdmin: true}, p.ID, nil); err != nil {
		t.Fatalf("admin harus boleh: %v", err)
	}
	// id non-UUID: 404 tanpa query
	if _, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, entity.ID("xx"), nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id non-UUID harus 404, dapat %v", err)
	}
}

func TestGetRun_OwnerCheck(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	uc := NewRunUsecase(deps)
	run, err := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.GetRun(context.Background(), entity.Actor{UserID: userB}, run.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("user lain harus 403, dapat %v", err)
	}
	if _, err := uc.GetRun(context.Background(), entity.Actor{UserID: userA}, run.ID); err != nil {
		t.Fatalf("pemilik harus boleh: %v", err)
	}
}

func TestFailRun_InvalidTransition(t *testing.T) {
	deps, _, f := newTestCore()
	p := seedProject(t, deps, f, userA)
	uc := NewRunUsecase(deps)
	run, _ := uc.CreateRun(context.Background(), entity.Actor{UserID: userA}, p.ID, nil)
	run.Status = entity.RunDone
	_ = f.runs.Update(context.Background(), run)

	if err := uc.FailRun(context.Background(), run.ID, "x"); !errors.Is(err, ErrConflict) {
		t.Fatalf("run done difail harus 409, dapat %v", err)
	}
}
