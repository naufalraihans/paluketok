// Harness E2E (bagian 3): tipe pendukung memStore + penyusun server penuh.
package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"paluketok/server/internal/adapter/judge"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
	"paluketok/server/internal/worker"
)

// Rute notifikasi Midtrans untuk harness. Tidak ada segmen rahasia: keaslian
// dibuktikan signature SHA512, dan harness memakai ServerKey test (e2eServerKey).
const e2eWebhookPath = "/v1/webhooks/midtrans"

// memIDs IDGenerator sekuensial (uuid-shaped, lolos entity.ValidID).
type memIDs struct {
	mu sync.Mutex
	n  int
}

func (g *memIDs) New() entity.ID {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return entity.ID(fmt.Sprintf("0197a23e-8888-7000-8000-%012d", g.n))
}

// memClock jam tetap yang maju per panggilan.
type memClock struct {
	mu sync.Mutex
	t  time.Time
}

func newMemClock() *memClock {
	return &memClock{t: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
}

func (c *memClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(time.Second)
	return c.t
}

// memSettings port.AISettingsRepo in-memory.
type memSettings struct {
	mu sync.Mutex
	m  map[string]*port.AISetting
}

func (s *memSettings) List(context.Context) ([]port.AISetting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []port.AISetting
	for _, v := range s.m {
		out = append(out, *v)
	}
	return out, nil
}

func (s *memSettings) Get(_ context.Context, slot string) (*port.AISetting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[slot]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *memSettings) Upsert(_ context.Context, v *port.AISetting, _ entity.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *v
	s.m[v.Slot] = &cp
	return nil
}

// memProjectsSource worker.ProjectsSource.
type memProjectsSource struct{ s *memStore }

func (m memProjectsSource) GetByID(ctx context.Context, id entity.ID) (*entity.Project, error) {
	return memProjects(m).GetByID(ctx, id)
}

// e2eEnv seluruh rangkaian untuk satu test.
type e2eEnv struct {
	t      *testing.T
	srv    *httptest.Server
	store  *memStore
	llm    *mockLLM
	fix    *jwtFixture
	pool   *worker.Pool
	clk    *memClock
	repos  e2eRepos
	notify string
	pay    *memPayment
}

type e2eRepos struct {
	profiles  memProfiles
	projects  memProjects
	runs      memRuns
	artifacts memArtifacts
	decisions memDecisions
	orders    memOrders
	ledger    memLedger
	messages  memMessages
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newE2E merangkai server + worker pool asli di atas memStore.
func newE2E(t *testing.T) *e2eEnv {
	t.Helper()
	st := newMemStore()
	st.seedProfile(uidE2E, 5)
	adm := st.seedProfile(uidE2EAdmin, 0)
	adm.IsAdmin = true

	fix := newJWTFix(t)
	llmMock := &mockLLM{failOnce: map[entity.Stage]bool{}}
	jc := judge.New(jevMock(t).URL, "e2e-key")

	repos := e2eRepos{
		profiles: memProfiles{st}, projects: memProjects{st}, runs: memRuns{st},
		artifacts: memArtifacts{st}, decisions: memDecisions{st},
		orders: memOrders{st}, ledger: memLedger{st}, messages: memMessages{st},
	}
	clk := newMemClock()
	ids := &memIDs{}
	settings := &memSettings{m: map[string]*port.AISetting{}}

	// Adapter Midtrans palsu: CreateOrder mengembalikan link Snap, dan
	// VerifyTransaction meniru GET /v2/{order_id}/status (sumber kebenaran
	// status transaksi saat webhook diproses).
	payment := newMemPayment()

	base := usecase.Deps{
		Profiles: repos.profiles, Projects: repos.projects, Runs: repos.runs,
		Artifacts: repos.artifacts, Decisions: repos.decisions, Orders: repos.orders,
		Ledger: repos.ledger, Verifier: payment, Notifications: payment,
		Payment: payment, Signatures: payment,
		// Chat konteks proyek: pakai LLM palsu yang sama dengan worker.
		Messages: repos.messages, LLM: llmMock,
		Clock: clk, IDs: ids,
	}

	pipe := worker.New(llmMock, jc, memRunStore{st}, memProjectsSource{st},
		memArtifacts{st}, memLedger{st}, memDecisions{st}, memPlanStages{st},
		ids, clk.Now, discardLog())
	pool, err := worker.NewPool(pipe, memRunStore{st}, ids, clk.Now, discardLog())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pool.Start(ctx, pool.Warmup(ctx))

	runUC := usecase.NewRunUsecase(base)
	runUC.SetRegenerator(pipe)
	// Skema tahap: pipeline yang menyusun usulannya (judge) sekaligus yang
	// menyimpannya; satu objek dua peran.
	runUC.SetStageProposer(pipe, memPlanStages{st})

	projUC := usecase.NewProjectUsecase(base)
	chatUC := usecase.NewChatUsecase(base)
	chatUC.SetResolver(projUC)
	h := &Handlers{
		Projects:  projUC,
		Runs:      runUC,
		Credits:   usecase.NewCreditUsecase(base),
		Orders:    usecase.NewOrderUsecase(base),
		Admin:     usecase.NewAdminUsecase(base, settings),
		Chat:      chatUC,
		Submitter: pool,
	}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h, Verifier: fix.verif}))
	t.Cleanup(srv.Close)

	return &e2eEnv{t: t, srv: srv, store: st, llm: llmMock, fix: fix, pool: pool,
		clk: clk, repos: repos, notify: e2eWebhookPath, pay: payment}
}
