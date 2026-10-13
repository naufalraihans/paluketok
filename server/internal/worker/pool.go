// Worker pool (plan/workflow.md 22, plan/security.md bagian 8 & 11):
// pool 4 goroutine, antrean in-memory 20 slot, penuh = 503 (ErrUnavailable).
// Klaim run lewat DB (FOR UPDATE SKIP LOCKED) - dua worker tak pernah dapat
// run sama. Start: run 'queued' diulang; run 'running' ditandai failed
// server_restart + refund (idempoten). Shutdown: drain dengan batas waktu.
package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Angka awal plan/security.md bagian 8 dan planner.md bagian 8.
const (
	PoolSize      = 4
	QueueSize     = 20
	DrainTimeout  = 25 * time.Second
	ReasonRestart = "server_restart"
)

// ErrQueueFull dibalas Submit saat antrean penuh (dipetakan 503 oleh handler).
var ErrQueueFull = errors.New("antrean worker penuh")

// Pool kumpulan worker tetap.
type Pool struct {
	pipeline *Pipeline
	runs     RunStore
	queue    chan entity.ID
	log      *slog.Logger
	ids      IDGen
	clock    func() time.Time
	wg       sync.WaitGroup
	queueMu  sync.Mutex
	stopping bool
}

// NewPool membangun pool, melakukan restart recovery di DB, dan mengisi
// antrean dari run queued yang tertinggal.
func NewPool(p *Pipeline, runs RunStore, ids IDGen, clock func() time.Time, log *slog.Logger) (*Pool, error) {
	pl := &Pool{pipeline: p, runs: runs, queue: make(chan entity.ID, QueueSize),
		log: log, ids: ids, clock: clock}

	// Restart recovery (security.md bagian 11): running sisa proses mati
	// -> failed + refund; queued -> masuk antrean lagi.
	n, err := runs.RequeueRunning(context.Background(), clock())
	if err != nil {
		return nil, err
	}
	if n > 0 {
		log.Warn("restart recovery", "run_ditandai_failed", n)
	}
	return pl, nil
}

// Start menyalakan pool goroutine dan memuat run queued ke antrean.
func (pl *Pool) Start(ctx context.Context, warm []port.QueuedRun) {
	for _, q := range warm {
		pl.enqueue(q.ID)
	}
	for i := 0; i < PoolSize; i++ {
		pl.wg.Add(1)
		go pl.loop(ctx)
	}
}

// Warmup mengambil daftar queued dari DB (dipanggil main setelah pool dibuat).
func (pl *Pool) Warmup(ctx context.Context) []port.QueuedRun {
	q, err := pl.runs.ListQueued(ctx)
	if err != nil {
		pl.log.Warn("muat run queued gagal", "err", err)
		return nil
	}
	return q
}

// Submit menyodok run baru ke antrean. Penuh = ErrQueueFull (503).
// Run masih 'queued' di DB; worker mengklaimnya nanti secara atomik.
func (pl *Pool) Submit(runID entity.ID) error {
	pl.queueMu.Lock()
	if pl.stopping {
		pl.queueMu.Unlock()
		return ErrQueueFull
	}
	pl.queueMu.Unlock()
	return pl.enqueue(runID)
}

// enqueue non-blocking; penuh balik ErrQueueFull.
func (pl *Pool) enqueue(id entity.ID) error {
	select {
	case pl.queue <- id:
		return nil
	default:
		return ErrQueueFull
	}
}

// loop siklus kerja satu worker: antrean -> klaim (atomik) -> pipeline.
func (pl *Pool) loop(ctx context.Context) {
	defer pl.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-pl.queue:
			if !ok {
				return
			}
			pl.runOne(ctx, id)
		}
	}
}

// runOne menjalankan satu run; panic ditangkap -> run failed + refund,
// proses Tetap hidup (security.md bagian 10).
func (pl *Pool) runOne(ctx context.Context, id entity.ID) {
	defer func() {
		if rec := recover(); rec != nil {
			pl.log.Error("panic worker", "run", id, "panic", rec)
			if run, err := pl.runs.GetFull(ctx, id); err == nil {
				_ = pl.pipeline.failRun(ctx, run, "panic_worker")
			}
		}
	}()
	if err := pl.pipeline.ProcessID(ctx, id); err != nil {
		pl.log.Error("run gagal diproses", "run", id, "err", err)
	}
}

// Stop drain: tunggu antrean kosong dan worker selesai sampai batas waktu.
// Sisa antrean tetap 'queued' di DB - diambil lagi proses berikutnya.
func (pl *Pool) Stop() {
	pl.queueMu.Lock()
	pl.stopping = true
	pl.queueMu.Unlock()

	deadline := time.After(DrainTimeout)
	for {
		select {
		case <-deadline:
			pl.log.Warn("drain habis waktu", "sisa_antrean", len(pl.queue))
			return
		default:
		}
		pl.queueMu.Lock()
		busy := len(pl.queue) > 0
		pl.queueMu.Unlock()
		if !busy {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	done := make(chan struct{})
	go func() { pl.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-deadline2(pl):
	}
	// Sisa run 'running' di sini memang milik worker yang masih berjalan
	// dalam WaitGroup; jika masih ada, biarkan - recovery start berikutnya
	// yang menandai (security.md: start menandai running sisa).
}

func deadline2(*Pool) <-chan time.Time { return time.After(DrainTimeout) }
