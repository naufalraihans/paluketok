// httptest endpoint admin (workflow.md 26, openapi.md bagian admin):
// non-admin 403, admin 200, PATCH user plan_tier, grant credits, settings/ai
// masked + put. Server memakai ActorFix admin/user untuk melewati JWT.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
)

const (
	uidAdmin = "0197a23e-1111-7000-8000-0000000000ad"
	uidUser  = "0197a23e-1111-7000-8000-00000000000a"
)

// fakeAdminDeps menyusun Handlers dengan fake: profile admin+user dan
// settings in-memory.
type fakeAdminDeps struct {
	profiles  *adminProfiles
	settings  *adminSettings
	decisions *adminDecisions
	projects  *adminProjects
	runs      *adminRuns
	ledger    *adminLedger
}

type adminProfiles struct {
	byID map[entity.ID]*entity.Profile
}

func (r *adminProfiles) GetByID(_ context.Context, id entity.ID) (*entity.Profile, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *adminProfiles) SetPlanTier(_ context.Context, id entity.ID, tier entity.PlanTier) error {
	p, ok := r.byID[id]
	if !ok {
		return port.ErrNotFound
	}
	p.PlanTier = tier
	return nil
}

func (r *adminProfiles) IsAdmin(_ context.Context, id entity.ID) (bool, error) {
	p, ok := r.byID[id]
	if !ok {
		return false, port.ErrNotFound
	}
	return p.IsAdmin, nil
}

type adminSettings struct{ slots map[string]*port.AISetting }

func (s *adminSettings) List(_ context.Context) ([]port.AISetting, error) {
	var out []port.AISetting
	for _, v := range s.slots {
		out = append(out, *v)
	}
	return out, nil
}

func (s *adminSettings) Get(_ context.Context, slot string) (*port.AISetting, error) {
	v, ok := s.slots[slot]
	if !ok {
		return nil, port.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *adminSettings) Upsert(_ context.Context, v *port.AISetting, _ entity.ID) error {
	cp := *v
	s.slots[v.Slot] = &cp
	return nil
}

type adminDecisions struct{ list []entity.Decision }

func (d *adminDecisions) Create(_ context.Context, _ *entity.Decision) error { return nil }
func (d *adminDecisions) ListAll(_ context.Context, _ port.Page) ([]entity.Decision, error) {
	return d.list, nil
}
func (d *adminDecisions) ListByRun(_ context.Context, _ entity.ID) ([]entity.Decision, error) {
	return d.list, nil
}
func (d *adminDecisions) NextRound(_ context.Context, _ entity.ID, _ entity.Stage) (int, error) {
	return 1, nil
}

type adminProjects struct{ list []entity.Project }

func (p *adminProjects) Create(_ context.Context, _ *entity.Project) error { return nil }
func (p *adminProjects) GetByID(_ context.Context, _ entity.ID) (*entity.Project, error) {
	return nil, port.ErrNotFound
}
func (p *adminProjects) ListByOwner(_ context.Context, _ entity.ID, _ port.Page) ([]entity.Project, error) {
	return nil, nil
}
func (p *adminProjects) ListAll(_ context.Context, _ port.Page) ([]entity.Project, error) {
	return p.list, nil
}
func (p *adminProjects) Update(_ context.Context, _ *entity.Project) error { return nil }
func (p *adminProjects) Archive(_ context.Context, _ entity.ID, _ entity.ID, _ time.Time) error {
	return nil
}

type adminRuns struct{ list []entity.PlanRun }

func (r *adminRuns) GetByID(_ context.Context, _ entity.ID) (*entity.PlanRun, error) {
	return nil, port.ErrNotFound
}
func (r *adminRuns) Update(_ context.Context, _ *entity.PlanRun) error { return nil }
func (r *adminRuns) ListByProject(_ context.Context, _ entity.ID) ([]entity.PlanRun, error) {
	return nil, nil
}
func (r *adminRuns) ListAllRuns(_ context.Context, _ string, _ entity.ID, _ port.Page) ([]entity.PlanRun, error) {
	return r.list, nil
}
func (r *adminRuns) HasRunning(_ context.Context, _ entity.ID) (bool, error) { return false, nil }
func (r *adminRuns) HasOtherRunning(_ context.Context, _, _ entity.ID) (bool, error) {
	return false, nil
}

type adminLedger struct{ granted int }

func (l *adminLedger) Grant(_ context.Context, _ entity.ID, credits int, _ entity.ID, _ string) error {
	l.granted += credits
	return nil
}
func (l *adminLedger) ChargeForRun(_ context.Context, _ entity.ID, _ *entity.PlanRun) error {
	return nil
}
func (l *adminLedger) ChargeForRegenerate(_ context.Context, _ entity.ID, _ entity.ID, _ int) (entity.ID, error) {
	return "0197a23e-3333-7000-8000-000000000001", nil
}
func (l *adminLedger) RefundRun(_ context.Context, _ *entity.PlanRun, _ int) error { return nil }
func (l *adminLedger) RefundCharge(_ context.Context, _ entity.ID, _ entity.ID, _ int) error {
	return nil
}
func (l *adminLedger) SettleTopUp(_ context.Context, _ *entity.Order) error { return nil }

func adminServer(t *testing.T, d *fakeAdminDeps, actor entity.Actor) (*httptest.Server, *fakeAdminDeps) {
	t.Helper()
	if d == nil {
		d = &fakeAdminDeps{
			profiles: &adminProfiles{byID: map[entity.ID]*entity.Profile{
				uidAdmin: {ID: uidAdmin, IsAdmin: true, PlanTier: entity.TierPro},
				uidUser:  {ID: uidUser, PlanTier: entity.TierFree},
			}},
			settings:  &adminSettings{slots: map[string]*port.AISetting{}},
			decisions: &adminDecisions{},
			projects:  &adminProjects{},
			runs:      &adminRuns{},
			ledger:    &adminLedger{},
		}
	}
	apps := usecase.Deps{
		Profiles: d.profiles, Projects: d.projects, Runs: d.runs,
		Decisions: d.decisions, Ledger: d.ledger,
		Clock: fakeClock{}, IDs: &idGen{},
	}
	h := &Handlers{
		Projects: usecase.NewProjectUsecase(apps),
		Runs:     usecase.NewRunUsecase(apps),
		Credits:  usecase.NewCreditUsecase(apps),
		Orders:   usecase.NewOrderUsecase(apps),
		Admin:    usecase.NewAdminUsecase(apps, d.settings),
	}
	srv := httptest.NewServer(NewRouter(RouterOptions{Handlers: h, ActorFix: actor}))
	t.Cleanup(srv.Close)
	return srv, d
}

func doJSON(t *testing.T, srv *httptest.Server, method, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var rd *bytes.Reader
	if body == nil {
		rd = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, srv.URL+path, rd)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

// Non-admin ditolak 403 di SEMUA endpoint admin.
func TestAdmin_ForbidNonAdmin(t *testing.T) {
	srv, _ := adminServer(t, nil, entity.Actor{UserID: uidUser})
	paths := []string{
		"/v1/admin/projects", "/v1/admin/runs", "/v1/admin/decisions",
		"/v1/admin/settings/ai",
	}
	for _, p := range paths {
		resp, _ := doJSON(t, srv, "GET", p, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s: ingin 403, dapat %d", p, resp.StatusCode)
		}
	}
	resp, _ := doJSON(t, srv, "PATCH", "/v1/admin/users/"+uidUser, map[string]any{"plan_tier": "pro"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("patch user non-admin: ingin 403, dapat %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, srv, "POST", "/v1/admin/users/"+uidUser+"/credits", map[string]any{"credits": 5, "idempotency_key": "k1"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("grant non-admin: ingin 403, dapat %d", resp.StatusCode)
	}
}

// Admin: list projects & runs & decisions, patch tier, grant, settings AI.
func TestAdmin_HappyPath(t *testing.T) {
	srv, d := adminServer(t, nil, entity.Actor{UserID: uidAdmin, IsAdmin: true})

	d.projects.list = []entity.Project{{ID: "0197a23e-2222-7000-8000-000000000001",
		OwnerID: uidUser, Name: "Proj A", Status: entity.ProjectDraft}}
	d.runs.list = []entity.PlanRun{{ID: "0197a23e-2222-7000-8000-000000000002",
		UserID: uidUser, Status: entity.RunDone, Progress: 6}}
	d.decisions.list = []entity.Decision{{ID: "0197a23e-2222-7000-8000-000000000003",
		RunID: "0197a23e-2222-7000-8000-000000000002", UserID: uidUser,
		Stage: entity.StagePRD, Gate: entity.GateExecute, Round: 1,
		State: json.RawMessage(`{"x":1}`), Criteria: json.RawMessage(`{"y":2}`)}}

	resp, out := doJSON(t, srv, "GET", "/v1/admin/projects", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("list projects: %d", resp.StatusCode)
	}
	if arr, ok := out["projects"].([]any); !ok || len(arr) != 1 {
		t.Fatalf("projects harus 1 item: %v", out)
	}

	// Baris kedua: status + isi run tidak dipakai, tapi respons wajib 200.
	respRuns, runsOut := doJSON(t, srv, "GET", "/v1/admin/runs?status=done", nil)
	if respRuns.StatusCode != 200 {
		t.Fatalf("list runs: %d", respRuns.StatusCode)
	}
	if arr, ok := runsOut["runs"].([]any); !ok || len(arr) != 1 {
		t.Fatalf("runs harus 1 item: %v", runsOut)
	}

	// Keputusan admin TIDAK membawa state mentah & criteria (aturan openapi).
	resp, decisions := doJSON(t, srv, "GET", "/v1/admin/decisions", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("list decisions: %d", resp.StatusCode)
	}
	if raw, _ := decisions["decisions"].([]any); len(raw) == 1 {
		first := raw[0].(map[string]any)
		if _, bad := first["state"]; bad {
			t.Fatal("decisions admin membocorkan state mentah")
		}
	}

	// PATCH plan_tier.
	resp, _ = doJSON(t, srv, "PATCH", "/v1/admin/users/"+uidUser, map[string]any{"plan_tier": "pro"})
	if resp.StatusCode != 200 {
		t.Fatalf("patch tier: %d", resp.StatusCode)
	}
	if d.profiles.byID[uidUser].PlanTier != entity.TierPro {
		t.Fatal("plan_tier tidak berubah")
	}

	// Grant kredit idempoten key wajib.
	resp, _ = doJSON(t, srv, "POST", "/v1/admin/users/"+uidUser+"/credits", map[string]any{"credits": 5, "idempotency_key": "bonus-1"})
	if resp.StatusCode != 200 {
		t.Fatalf("grant: %d", resp.StatusCode)
	}
	if d.ledger.granted != 5 {
		t.Fatalf("granted ingin 5, dapat %d", d.ledger.granted)
	}
	resp, _ = doJSON(t, srv, "POST", "/v1/admin/users/"+uidUser+"/credits", map[string]any{"credits": 0})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("grant tanpa key/credits: ingin 422, dapat %d", resp.StatusCode)
	}

	// Settings AI: PUT lalu GET masked.
	resp, _ = doJSON(t, srv, "PUT", "/v1/admin/settings/ai/planner", map[string]any{
		"provider": "9router", "base_url": "https://api.example.com/v1",
		"model": "gpt-x", "api_key": "sk-secret-1234567890",
		"max_input_tokens": 8000, "max_output_tokens": 4000, "is_active": true,
	})
	if resp.StatusCode != 200 {
		t.Fatalf("put settings: %d", resp.StatusCode)
	}
	resp, out = doJSON(t, srv, "GET", "/v1/admin/settings/ai", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get settings: %d", resp.StatusCode)
	}
	slots, _ := out["slots"].([]any)
	if len(slots) != 1 {
		t.Fatalf("slots ingin 1, dapat %d", len(slots))
	}
	key := slots[0].(map[string]any)["api_key"].(string)
	if key == "sk-secret-1234567890" || !bytes.Contains([]byte(key), []byte("...")) {
		t.Fatalf("api_key tidak masked: %q", key)
	}
	// base_url host lokal ditolak.
	resp, _ = doJSON(t, srv, "PUT", "/v1/admin/settings/ai/planner", map[string]any{
		"provider": "x", "base_url": "http://localhost:9999", "model": "m",
		"api_key": "k", "max_input_tokens": 1, "max_output_tokens": 1, "is_active": true,
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("base_url localhost: ingin 422, dapat %d", resp.StatusCode)
	}
}
