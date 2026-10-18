// Fake semua dependensi worker untuk test (tanpa LLM/JEV/DB live).
package worker

import (
	"context"
	"sync"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// fakeRuns menjalankan logika antrean in-memory setara ClaimQueued atomik:
// mutex menjamin dua pemanggil tidak mendapat run sama.
type fakeRuns struct {
	mu     sync.Mutex
	byID   map[entity.ID]*entity.PlanRun
	order  []entity.ID // urutan klaim (queued dulu)
	refund entity.ID   // refund terakhir (untuk hitung)
}

func newFakeRuns() *fakeRuns { return &fakeRuns{byID: map[entity.ID]*entity.PlanRun{}} }

func (f *fakeRuns) add(run *entity.PlanRun) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *run
	f.byID[run.ID] = &cp
	if run.Status == entity.RunQueued {
		f.order = append(f.order, run.ID)
	}
}

func (f *fakeRuns) GetFull(_ context.Context, id entity.ID) (*entity.PlanRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeRuns) ClaimQueued(_ context.Context) (*port.QueuedRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, id := range f.order {
		r := f.byID[id]
		if r.Status == entity.RunQueued {
			r.Status = entity.RunRunning
			f.order = append(f.order[:i], f.order[i+1:]...)
			return &port.QueuedRun{ID: r.ID, ProjectID: r.ProjectID, UserID: r.UserID}, nil
		}
	}
	return nil, port.ErrNotFound
}

func (f *fakeRuns) SaveProgress(_ context.Context, id entity.ID, stage entity.Stage, progress int, status entity.RunStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok {
		return port.ErrNotFound
	}
	if stage != "" {
		s := stage
		r.CurrentStage = &s
	}
	r.Progress = progress
	r.Status = status
	return nil
}

func (f *fakeRuns) SaveTotal(_ context.Context, id entity.ID, total int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok {
		return port.ErrNotFound
	}
	r.Total = total
	return nil
}

func (f *fakeRuns) Finish(_ context.Context, id entity.ID, status entity.RunStatus, runErr string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok {
		return port.ErrNotFound
	}
	r.Status = status
	e := runErr
	r.Error = &e
	return nil
}

func (f *fakeRuns) ListQueued(_ context.Context) ([]port.QueuedRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []port.QueuedRun
	for _, id := range f.order {
		r := f.byID[id]
		if r.Status == entity.RunQueued {
			out = append(out, port.QueuedRun{ID: r.ID, ProjectID: r.ProjectID, UserID: r.UserID})
		}
	}
	return out, nil
}

func (f *fakeRuns) RequeueRunning(_ context.Context, _ time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.byID {
		if r.Status == entity.RunRunning {
			r.Status = entity.RunFailed
			e := ReasonRestart
			r.Error = &e
			f.refund = r.ID
			n++
		}
	}
	return n, nil
}

func (f *fakeRuns) HasOtherRunning(_ context.Context, projectID, excludeID entity.ID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, r := range f.byID {
		if id != excludeID && r.ProjectID == projectID && r.Status == entity.RunRunning {
			return true, nil
		}
	}
	return false, nil
}

// fakeProjects satu project tetap.
type fakeProjects struct{ p *entity.Project }

func (f fakeProjects) GetByID(_ context.Context, _ entity.ID) (*entity.Project, error) {
	return f.p, nil
}

// fakeArtifacts mencatat kind yang tersimpan dan urutannya.
type fakeArtifacts struct {
	mu     sync.Mutex
	saved  []entity.Stage
	perRun map[entity.ID][]*entity.Artifact
}

func (f *fakeArtifacts) Upsert(_ context.Context, a *entity.Artifact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, a.Kind)
	if f.perRun == nil {
		f.perRun = map[entity.ID][]*entity.Artifact{}
	}
	cp := *a
	f.perRun[a.RunID] = append(f.perRun[a.RunID], &cp)
	return nil
}

// ListByRun untuk regenerate (worker/regenerate.go).
func (f *fakeArtifacts) ListByRun(_ context.Context, runID entity.ID) ([]entity.Artifact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []entity.Artifact
	for _, a := range f.perRun[runID] {
		out = append(out, *a)
	}
	return out, nil
}

// fakeLedger menghitung refund; idempoten = tidak menghitung dua kali run sama.
type fakeLedger struct {
	mu     sync.Mutex
	refund map[entity.ID]int
}

func newFakeLedger() *fakeLedger { return &fakeLedger{refund: map[entity.ID]int{}} }

func (f *fakeLedger) RefundRun(_ context.Context, run *entity.PlanRun, credits int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refund[run.ID] += credits // uji idempotensi di kedua sisi; pipeline yang menjamin sekali
	return nil
}

func (f *fakeLedger) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := 0
	for _, v := range f.refund {
		t += v
	}
	return t
}

// fakeDecisions mencatat gate.
type fakeDecisions struct {
	mu    sync.Mutex
	items []entity.Decision
}

func (f *fakeDecisions) Create(_ context.Context, d *entity.Decision) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, *d)
	return nil
}

// fakeLLM: antrean jawaban per panggilan; fail n pertama validasi gagal.
type fakeLLM struct {
	mu    sync.Mutex
	resp  []string // content per panggilan berikutnya
	calls int
}

func (f *fakeLLM) push(content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resp = append(f.resp, content)
}

func (f *fakeLLM) Complete(_ context.Context, _ port.LLMRequest) (*port.LLMResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.resp) == 0 {
		return &port.LLMResponse{Content: "kalimat tanpa seksi"}, nil
	}
	c := f.resp[0]
	f.resp = f.resp[1:]
	return &port.LLMResponse{Content: c, Usage: port.LLMUsage{InputTokens: 10, OutputTokens: 5}}, nil
}

// fakeJudge: confidence per pertanyaan, opsi pertama sebagai pilihan.
type fakeJudge struct {
	mu    sync.Mutex
	score []float64 // antrian skor per panggilan; kosong = 0.99
}

func (f *fakeJudge) push(s float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.score = append(f.score, s)
}

func (f *fakeJudge) Decide(_ context.Context, _ port.JudgeRequest) (*port.JudgeResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sc := 0.99
	if len(f.score) > 0 {
		sc = f.score[0]
		f.score = f.score[1:]
	}
	return &port.JudgeResponse{
		DecisionID: "jev-test-1",
		Answers: map[string]port.JudgeAnswer{
			"q1": {Choice: "rekomen", Confidence: sc, Probabilities: map[string]float64{"rekomen": sc}},
		},
	}, nil
}

// fakePlanStages skema tahap per run. Kosong = run lama, worker pakai bawaan.
type fakePlanStages struct {
	mu    sync.Mutex
	byRun map[entity.ID][]entity.PlanStage
}

func newFakePlanStages() *fakePlanStages {
	return &fakePlanStages{byRun: map[entity.ID][]entity.PlanStage{}}
}

func (f *fakePlanStages) put(runID entity.ID, plan []entity.PlanStage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byRun[runID] = plan
}

func (f *fakePlanStages) ListByRun(_ context.Context, runID entity.ID) ([]entity.PlanStage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.byRun[runID], nil
}

func (f *fakePlanStages) ReplaceByRun(_ context.Context, runID entity.ID, stages []entity.PlanStage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]entity.PlanStage, len(stages))
	copy(cp, stages)
	f.byRun[runID] = cp
	return nil
}

// fakeIDs urut.
type fakeIDs struct{ n int }

func (f *fakeIDs) New() entity.ID {
	f.n++
	return entity.ID("00000000-0000-4000-8000-00000000000" + string(rune('0'+f.n%10)))
}

// valid konten per tahap (lolos validate).
func validContent(stage entity.Stage) string {
	switch stage {
	case entity.StagePRD:
		return "## Tujuan\nbuat x\n## Scope\na\n## Acceptance criteria\nb\n## Di luar scope\nc"
	case entity.StageERD:
		return "Table users {\n  id uuid [pk]\n}\nTable runs {\n  id uuid [pk]\n  user_id uuid\n}\nRef: runs.user_id > users.id"
	case entity.StageOpenAPI:
		return `{"openapi":"3.1.0","paths":{"/v1/me":{"get":{"responses":{"200":{"description":"ok"}}}}}}`
	case entity.StageRoles:
		return "# roles\n| path | admin |\n| /v1/me | ya |"
	case entity.StageSecurity:
		return "Trust boundary: JWT. Validasi input: ketat. Checklist: 1) 2) 3)"
	case entity.StageStructure:
		return "fe/\nbe/\ndb/"
	}
	return "x"
}
