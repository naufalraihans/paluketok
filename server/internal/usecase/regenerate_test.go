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
	art      entity.Artifact
	stale    []entity.Stage
	versions []entity.ArtifactVersion
}

func (f *fakeRegenRepo) GetByID(_ context.Context, id entity.ID) (*entity.Artifact, error) {
	if id != artIDReg {
		return nil, port.ErrNotFound
	}
	a := f.art
	return &a, nil
}
func (f *fakeRegenRepo) InsertVersion(_ context.Context, v *entity.ArtifactVersion) error {
	// unik (artifact_id, version) ala uq_artifact_versions: duplikat = 23505.
	for _, ex := range f.versions {
		if ex.ArtifactID == v.ArtifactID && ex.Version == v.Version {
			return port.ErrConflict
		}
	}
	f.versions = append(f.versions, *v)
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

// --- 23505: pipeline sudah menulis v1 ke history lewat Upsert (guard
// not exists, artifact_upsert.go). Regenerate yang mengirim versi LAMA (=1)
// lagi TIDAK BOLEH membuat usecase gagal: versi lama cukup sekali di history
// (erd.md "Aturan versi"). Di DB asli ini 23505 uq_artifact_versions ---
func TestRegenerate_VersiLamaSudahDiHistory_TetapSukses(t *testing.T) {
	uc, arts, ledger, _ := regenUsecase(t, nil, "isi baru")
	// kondisi asli: pipeline run sudah menulis history versi 1.
	runAsal := runIDReg
	arts.versions = append(arts.versions, entity.ArtifactVersion{
		ID: "0197a23e-4444-7000-8000-0000000000aa", ArtifactID: artIDReg,
		Version: 1, Content: "lama", CreatedByRunID: &runAsal,
	})
	art, err := uc.RegenerateArtifact(context.Background(),
		entity.Actor{UserID: uidReg}, artIDReg)
	if err != nil {
		t.Fatalf("regenerate harus sukses walau v1 sudah di history: %v", err)
	}
	if ledger.refund != 0 {
		t.Fatalf("tidak seharusnya refund: %d", ledger.refund)
	}
	if art.Version != 2 || art.Content != "isi baru" {
		t.Fatalf("versi/isi salah: v%d %q", art.Version, art.Content)
	}
	if arts.art.Version != 2 || arts.art.Content != "isi baru" {
		t.Fatalf("artifact tersimpan salah: v%d %q", arts.art.Version, arts.art.Content)
	}
	// history harus tetap TEPAT SEKALI per versi: v1 tanpa duplikat.
	n := 0
	for _, v := range arts.versions {
		if v.Version == 1 {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("history v1 ingin 1 baris, dapat %d", n)
	}
}

// --- regenerate dua kali berturut: history v1,v2 tanpa duplikat/loncat,
// artifacts.version = 3 (erd.md "Aturan versi") ---
func TestRegenerate_DuaKali_HistoryBerurutan(t *testing.T) {
	uc, arts, _, _ := regenUsecase(t, nil, "isi v2")
	// kondisi asli: history v1 sudah ditulis pipeline.
	runAsal := runIDReg
	arts.versions = append(arts.versions, entity.ArtifactVersion{
		ID: "0197a23e-4444-7000-8000-0000000000aa", ArtifactID: artIDReg,
		Version: 1, Content: "lama", CreatedByRunID: &runAsal,
	})
	if _, err := uc.RegenerateArtifact(context.Background(), entity.Actor{UserID: uidReg}, artIDReg); err != nil {
		t.Fatalf("regenerate 1: %v", err)
	}
	uc.SetRegenerator(fakeRegenerator{content: "isi v3"})
	if _, err := uc.RegenerateArtifact(context.Background(), entity.Actor{UserID: uidReg}, artIDReg); err != nil {
		t.Fatalf("regenerate 2: %v", err)
	}
	if arts.art.Version != 3 {
		t.Fatalf("artifacts.version ingin 3, dapat %d", arts.art.Version)
	}
	versi := map[int]bool{}
	for _, v := range arts.versions {
		if v.ArtifactID != artIDReg {
			continue
		}
		if versi[v.Version] {
			t.Fatalf("versi %d dobel di history", v.Version)
		}
		versi[v.Version] = true
	}
	if !versi[1] || !versi[2] {
		t.Fatalf("history harus berisi v1 dan v2, dapat %v", versi)
	}
	if versi[3] {
		t.Fatal("versi 3 belum lama, tidak boleh masuk history")
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
