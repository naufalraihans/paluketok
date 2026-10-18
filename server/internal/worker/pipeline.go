// Pipeline 6 tahap (docs/plan/planner.md bagian 1-5, docs/plan/workflow.md 23):
// per tahap: tulis current_stage di AWAL -> prompt (brief + artefak
// sebelumnya) -> LLM -> validasi (gagal = retry dengan pesan error, maks 2
// percobaan total per planner.md 5 "Coba ulang ... Maks 2 kali") -> simpan
// artifact + versi. Kegagalan tahap setelah retry = run failed + refund sekali.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// stageTimeout per tahap dari docs/plan/structure.md ("timeout per dependensi":
// LLM 180 detik).
const stageTimeout = 180 * time.Second

// maxValidateAttempts: 1 percobaan awal + 1 retry dengan pesan error
// (planner.md bagian 5). workflow.md 23 merangkumnya "gagal dua kali".
const maxValidateAttempts = 2

// Pipeline menjalankan satu run dari queued sampai done/failed.
type Pipeline struct {
	llm       LLM
	judge     Judge
	runs      RunStore
	projects  ProjectsSource
	artifacts ArtifactStore
	ledger    LedgerStore
	decisions DecisionsStore
	stages    PlanStageStore
	ids       IDGen
	clock     func() time.Time
	log       *slog.Logger

	// state per-run (diisi process untuk setiap run):
	brief        string
	plan         []entity.PlanStage // skema tahap run ini
	stackSummary map[string]string
	prev         map[entity.Stage]string // artefak lolos (kind -> content)
	prevDigest   map[entity.Stage]string // ringkasan untuk evidence JEV
}

// IDGen pembuat ID (implementasi produksi: cmd/web idsReal).
type IDGen interface{ New() entity.ID }

// New membuat pipeline.
func New(llm LLM, judge Judge, runs RunStore, projects ProjectsSource,
	artifacts ArtifactStore, ledger LedgerStore, decisions DecisionsStore,
	stages PlanStageStore, ids IDGen, clock func() time.Time, log *slog.Logger) *Pipeline {
	return &Pipeline{llm: llm, judge: judge, runs: runs, projects: projects,
		artifacts: artifacts, ledger: ledger, decisions: decisions,
		stages: stages, ids: ids, clock: clock, log: log}
}

// Process menjalankan SATU klaim: klaim queued -> pipeline -> done/failed.
// Error hanya untuk kegagalan infrastruktur tak terduga; kegagalan pipeline
// ditangani di dalam (failed + refund). Aman dipanggil dari goroutine worker.
func (p *Pipeline) Process(ctx context.Context) error {
	q, err := p.runs.ClaimQueued(ctx)
	if errors.Is(err, port.ErrNotFound) {
		return nil // antrean DB kosong
	} else if err != nil {
		return err
	}
	return p.ProcessID(ctx, q.ID)
}

// ProcessID menjalankan run dengan id yang SUDAH di-claim (status running).
// Dipakai pool antrean in-memory yang sudah tahu id-nya dari Submit.
func (p *Pipeline) ProcessID(ctx context.Context, id entity.ID) error {
	run, err := p.runs.GetFull(ctx, id)
	if err != nil {
		return fmt.Errorf("ambil run %s: %w", id, err)
	}
	// Penjaga satu-running per project (planner.md bagian 8): bila project
	// punya run running lain, kembalikan run ini ke antrean DB.
	if other, err := p.runs.HasOtherRunning(ctx, run.ProjectID, run.ID); err == nil && other {
		return p.runs.SaveProgress(ctx, run.ID, entity.Stage(""), run.Progress, entity.RunQueued)
	}
	return p.process(ctx, run)
}

// process memuat data project lalu mengeksekusi semua tahap.
func (p *Pipeline) process(ctx context.Context, run *entity.PlanRun) error {
	proj, err := p.projects.GetByID(ctx, run.ProjectID)
	if err != nil {
		return p.failRun(ctx, run, "project hilang")
	}
	p.brief = proj.Brief
	p.stackSummary = map[string]string{
		"fe": proj.Stack.FE, "be": proj.Stack.BE, "db": proj.Stack.DB,
	}
	return p.execute(ctx, run)
}

// execute memproses seluruh tahap untuk run yang sudah 'running'.
//
// Urutannya dari plan_stages milik run, bukan entity.Stages: jumlah dan nama
// tahap ditentukan skema. Run lama tanpa baris plan_stages jatuh ke susunan awal
// enam tahap supaya tetap bisa dijalankan ulang.
func (p *Pipeline) execute(ctx context.Context, run *entity.PlanRun) error {
	p.prev = map[entity.Stage]string{}
	p.prevDigest = map[entity.Stage]string{}

	plan, err := p.planStages(ctx, run)
	if err != nil {
		return p.failRun(ctx, run, "db")
	}
	// total = jumlah tahap sebenarnya (tugas Fase B poin 2).
	if err := p.runs.SaveTotal(ctx, run.ID, len(plan)); err == nil {
		run.Total = len(plan)
	}
	p.plan = plan

	for i, st := range plan {
		if err := ctx.Err(); err != nil {
			return p.failRun(ctx, run, "server_restart")
		}
		// a0: tahap SEDANG berjalan ditulis di AWAL (planner.md bagian 1/6).
		if err := p.runs.SaveProgress(ctx, run.ID, st.Key, i, entity.RunRunning); err != nil {
			return p.failRun(ctx, run, "db")
		}

		draft, err := p.generateWithRetry(ctx, run, st)
		if err != nil {
			return p.failRun(ctx, run, err.Error())
		}

		// Simpan artifact + versi (uq_artifacts_run_kind). kind = key skema.
		art := &entity.Artifact{
			ID: p.ids.New(), RunID: run.ID, UserID: run.UserID,
			Kind: st.Key, Content: draft, Version: 1, Status: entity.ArtifactDraft,
			CreatedAt: p.clock(), UpdatedAt: p.clock(),
		}
		if err := p.artifacts.Upsert(ctx, art); err != nil {
			return p.failRun(ctx, run, "db")
		}

		p.prev[st.Key] = draft
		p.prevDigest[st.Key] = truncate(draft, 400)
	}

	return p.doneRun(ctx, run)
}

// planStages membaca skema tahap run. Kosong = run lama tanpa skema, dan
// susunan awal enam tahap dipakai apa adanya.
func (p *Pipeline) planStages(ctx context.Context, run *entity.PlanRun) ([]entity.PlanStage, error) {
	if p.stages == nil {
		return entity.DefaultPlanStages(run.ID), nil
	}
	plan, err := p.stages.ListByRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if len(plan) == 0 {
		return entity.DefaultPlanStages(run.ID), nil
	}
	return plan, nil
}

// generateWithRetry: panggil LLM + validasi. Gagal validasi menyuntikkan
// pesan error ke prompt berikutnya (planner.md bagian 5). Maks 2 percobaan.
func (p *Pipeline) generateWithRetry(ctx context.Context, run *entity.PlanRun, st entity.PlanStage) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxValidateAttempts; attempt++ {
		dctx, cancel := context.WithTimeout(ctx, stageTimeout)
		resp, err := p.llm.Complete(dctx, p.buildPrompt(st, lastErr))
		cancel()
		if err != nil {
			return "", fmt.Errorf("llm %s: %w", st.Key, err)
		}
		run.TokensIn += resp.Usage.InputTokens
		run.TokensOut += resp.Usage.OutputTokens

		if verr := validatePlanStage(st, resp.Content, p.prev); verr != nil {
			lastErr = verr
			continue // retry dengan pesan error ditambahkan
		}
		return resp.Content, nil
	}
	return "", fmt.Errorf("validasi %s gagal %d kali: %v", st.Key, maxValidateAttempts, lastErr)
}

// doneRun menutup run sebagai done.
func (p *Pipeline) doneRun(ctx context.Context, run *entity.PlanRun) error {
	run.Status = entity.RunDone
	run.FinishedAt = timePtr(p.clock())
	if err := p.runs.Finish(ctx, run.ID, entity.RunDone, "", p.clock()); err != nil {
		return err
	}
	p.log.Info("run selesai", "run", run.ID, "progress", run.Progress)
	return nil
}

// failRun menandai failed + refund sekali (idempoten di repo), sesuai
// planner.md bagian 5 dan security.md bagian 10 (panic -> failed, bukan mati).
func (p *Pipeline) failRun(ctx context.Context, run *entity.PlanRun, reason string) error {
	run.Status = entity.RunFailed
	msg := reason
	if len(msg) > 500 {
		msg = msg[:500] // kolom error tidak boleh membengkak
	}
	run.Error = &msg
	run.FinishedAt = timePtr(p.clock())
	if err := p.runs.Finish(ctx, run.ID, entity.RunFailed, msg, p.clock()); err != nil {
		p.log.Error("tandai run gagal", "run", run.ID, "err", err)
	}
	if err := p.ledger.RefundRun(ctx, run, run.CreditsCharged); err != nil {
		p.log.Error("refund run gagal", "run", run.ID, "err", err)
		return err
	}
	p.log.Warn("run gagal", "run", run.ID, "alasan", msg)
	return nil
}

// artifactDigests ringkasan artefak untuk state JEV (state ringkas, bukan
// repository penuh - planner.md bagian 4).
func (p *Pipeline) artifactDigests() map[string]string {
	out := map[string]string{}
	for k, v := range p.prevDigest {
		out[string(k)] = v
	}
	return out
}

// truncate memotong teks aman rune.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func strPtr(s string) *string        { return &s }
func timePtr(t time.Time) *time.Time { return &t }
