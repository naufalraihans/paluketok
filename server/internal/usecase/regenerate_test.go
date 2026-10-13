// Test regenerate (workflow.md 29, planner.md bagian 7, auth_billing.md 3.4):
// sukses = versi naik + hilir stale + charge 1; syarat gagal 422; refund saat
// generate gagal. Semua fake, tanpa LLM live.
package usecase

import (
	"context"
	"errors"
	"testing"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

const (
	uidReg   = entity.ID("0197a23e-4444-7000-8000-000000000001")
	runIDReg = entity.ID("0197a23e-4444-7000-8000-000000000002")
	artIDReg = entity.ID("0197a23e-4444-7000-8000-000000000003")
	projIDRg = entity.ID("0197a23e-4444-7000-8000-000000000004")
)

type fakeRegenRepo struct {
	port.ArtifactRepo
	art   entity.Artifact
	stale []entity.Stage
}

func (f *fakeRegenRepo) GetByID(_ context.Context, id entity.ID) (*entity.Artifact, error) {
	if id != artIDReg {
		return nil, port.ErrNotFound
	}
	a := f.art
	return &a, nil
}
func (f *fakeRegenRepo) InsertVersion(_ context.Context, _ *entity.ArtifactVersion) error {
	return nil
}
func (f *fakeRegenRepo) Update(_ context.Context, a *entity.Artifact) error { f.art = *a; return nil }
func (f *fakeRegenRepo) MarkDownstreamStale(_ context.Context, _ entity.ID, after entity.Stage) error {
	f.stale = append(f.stale, after)
	return nil
}

type fakeRegenLedger struct {
	charged int
	refund  int
}

func (l *fakeRegenLedger) ChargeForRegenerate(_ context.Context, _ entity.ID, _ entity.ID, charged int) (entity.ID, error) {
	l.charged += charged
	return "0197a23e-4444-7000-8000-00000000000c", nil
}
func (l *fakeRegenLedger) RefundCharge(_ context.Context, _ entity.ID, _ entity.ID, credits int) error {
	l.refund += credits
	return nil
}
func (l *fakeRegenLedger) ChargeForRun(_ context.Context, _ entity.ID, _ *entity.PlanRun) error {
	return nil
}
func (l *fakeRegenLedger) RefundRun(_ context.Context, _ *entity.PlanRun, _ int) error { return nil }
func (l *fakeRegenLedger) Grant(_ context.Context, _ entity.ID, _ int, _ entity.ID, _ string) error {
	return nil
}
func (l *fakeRegenLedger) SettleTopUp(_ context.Context, _ *entity.Order) error { return nil }

type fakeRegenRuns struct {
	running bool
	status  entity.RunStatus
}

func (r *fakeRegenRuns) GetByID(_ context.Context, id entity.ID) (*entity.PlanRun, error) {
	st := r.status
	if st == "" {
		st = entity.RunDone
	}
	return &entity.PlanRun{ID: id, ProjectID: projIDRg, UserID: uidReg, Status: st}, nil
}
func (r *fakeRegenRuns) HasOtherRunning(_ context.Context, _, _ entity.ID) (bool, error) {
	return r.running, nil
}
func (r *fakeRegenRuns) HasRunning(_ context.Context, _ entity.ID) (bool, error) { return false, nil }
func (r *fakeRegenRuns) Update(_ context.Context, _ *entity.PlanRun) error       { return nil }
func (r *fakeRegenRuns) ListByProject(_ context.Context, _ entity.ID) ([]entity.PlanRun, error) {
	return nil, nil
}
func (r *fakeRegenRuns) ListAllRuns(_ context.Context, _ string, _ entity.ID, _ port.Page) ([]entity.PlanRun, error) {
	return nil, nil
}

// regenDeps susun RunUsecase dengan fake; regenGenerator hasil tetap/gagal.
func regenUsecase(t *testing.T, talwir error, content string) (*RunUsecase, *fakeRegenRepo, *fakeRegenLedger, *fakeRegenRuns) {
	t.Helper()
	arts := &fakeRegenRepo{art: entity.Artifact{ID: artIDReg, RunID: runIDReg,
		UserID: uidReg, Kind: entity.StagePRD, Content: "lama", Version: 1, Status: entity.ArtifactDraft}}
	ledger := &fakeRegenLedger{}
	runs := &fakeRegenRuns{}
	d := Deps{
		Artifacts: arts, Runs: runs, Ledger: ledger,
		Clock: &fakeClock{}, IDs: &seqIDs{},
	}
	uc := NewRunUsecase(d)
	uc.SetRegenerator(fakeRegenerator{err: talwir, content: content})
	return uc, arts, ledger, runs
}

type fakeRegenerator struct {
	err     error
	content string
}

func (f fakeRegenerator) RegenerateStage(_ context.Context, _ *entity.Artifact) (string, error) {
	return f.content, f.err
}

// --- sukses: charge 1, versi naik, hilir stale, hasil baru disimpan ---

func TestRegenerate_Sukses(t *testing.T) {
	uc, arts, ledger, _ := regenUsecase(t, nil, "isi baru")
	art, err := uc.RegenerateArtifact(context.Background(),
		entity.Actor{UserID: uidReg}, artIDReg)
	if err != nil {
		t.Fatal(err)
	}
	if ledger.charged != 1 {
		t.Fatalf("charge ingin 1, dapat %d", ledger.charged)
	}
	if ledger.refund != 0 {
		t.Fatalf("tidak seharusnya refund: %d", ledger.refund)
	}
	if art.Version != 2 || art.Content != "isi baru" {
		t.Fatalf("versi/isi salah: v%d %q", art.Version, art.Content)
	}
	if len(arts.stale) != 1 || arts.stale[0] != entity.StagePRD {
		t.Fatalf("hilir stale tidak ditandai: %v", arts.stale)
	}
}

// --- run belum done / ada run running => 422 ---

func TestRegenerate_SyaratTolak(t *testing.T) {
	uc2, _, ledger2, runs2 := regenUsecase(t, nil, "x")
	runs2.status = entity.RunFailed
	if _, err := uc2.RegenerateArtifact(context.Background(), entity.Actor{UserID: uidReg}, artIDReg); err == nil {
		t.Fatal("run failed harus ditolak")
	}
	if ledger2.charged != 0 {
		t.Fatal("ditolak tidak boleh mencharge")
	}
	uc3, _, ledger3, runs3 := regenUsecase(t, nil, "x")
	runs3.running = true
	if _, err := uc3.RegenerateArtifact(context.Background(), entity.Actor{UserID: uidReg}, artIDReg); err == nil {
		t.Fatal("ada run running harus ditolak")
	}
	if ledger3.charged != 0 {
		t.Fatal("ditolak tidak boleh mencharge")
	}
}

// --- generate gagal => refund 1, artifact tak berubah ---

func TestRegenerate_GagalRefund(t *testing.T) {
	uc, arts, ledger, _ := regenUsecase(t, errors.New("llm down"), "")
	if _, err := uc.RegenerateArtifact(context.Background(), entity.Actor{UserID: uidReg}, artIDReg); err == nil {
		t.Fatal("harus gagal")
	}
	if ledger.refund != 1 {
		t.Fatalf("refun ingin 1, dapat %d", ledger.refund)
	}
	if ledger.charged != 1 {
		t.Fatalf("charge ingin 1, dapat %d", ledger.charged)
	}
	if arts.art.Content != "lama" || arts.art.Version != 1 {
		t.Fatalf("artifact berubah saat gagal: v%d %q", arts.art.Version, arts.art.Content)
	}
}
