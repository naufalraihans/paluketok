package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"paluketok/server/internal/delivery/middleware"
	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
	"paluketok/server/internal/usecase"
)

// F11 langkah 25: pageFrom default/cap dan admin runs tanpa user_id.

func TestPageFrom_DefaultDanCap(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		limit  int
		offset int
	}{
		{"tanpa param", "", 20, 0},
		{"limit saja", "limit=5", 5, 0},
		{"cap 100", "limit=5000", 100, 0},
		{"limit 0", "limit=0", 20, 0},
		{"limit aneh", "limit=abc", 20, 0},
		{"limit negatif", "limit=-3", 20, 0},
		{"offset aneh", "offset=-7", 20, 0},
		{"offset huruf", "offset=x", 20, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := "/v1/admin/runs"
			if c.query != "" {
				url += "?" + c.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			p := pageFrom(req)
			if p.Limit != c.limit || p.Offset != c.offset {
				t.Fatalf("query %q: ingin Page{%d,%d}, dapat Page{%d,%d}",
					c.query, c.limit, c.offset, p.Limit, p.Offset)
			}
		})
	}
}

// fakeRunsRepo rekam argumen terakhir ListRuns, balik list kosong.
type fakeRunsRepo struct {
	status string
	userID entity.ID
	page   port.Page
}

func (f *fakeRunsRepo) ListAllRuns(_ context.Context, status string, userID entity.ID, page port.Page) ([]entity.PlanRun, error) {
	f.status, f.userID, f.page = status, userID, page
	return []entity.PlanRun{}, nil
}

func (f *fakeRunsRepo) GetByID(context.Context, entity.ID) (*entity.PlanRun, error) {
	return nil, port.ErrNotFound
}

func (f *fakeRunsRepo) ListByProject(context.Context, entity.ID) ([]entity.PlanRun, error) {
	return nil, nil
}

func (f *fakeRunsRepo) HasOtherRunning(context.Context, entity.ID, entity.ID) (bool, error) {
	return false, nil
}

func (f *fakeRunsRepo) HasRunning(context.Context, entity.ID) (bool, error) {
	return false, nil
}

func (f *fakeRunsRepo) Update(context.Context, *entity.PlanRun) error { return nil }

// newRunsHandler pasang repo rekam lewat Deps saat konstruksi.
func newRunsHandler(repo port.RunRepo) *Handlers {
	admin := usecase.NewAdminUsecase(usecase.Deps{Runs: repo}, nil)
	return &Handlers{Admin: admin}
}

func withAdmin(req *http.Request) *http.Request {
	actor := entity.Actor{UserID: "a4f40000-0000-4000-8000-00000000d0ad", IsAdmin: true}
	return req.WithContext(middleware.WithActor(req.Context(), actor))
}

func doAdminRuns(query string) (*httptest.ResponseRecorder, *fakeRunsRepo) {
	repo := &fakeRunsRepo{}
	h := newRunsHandler(repo)
	req := withAdmin(httptest.NewRequest(http.MethodGet, "/v1/admin/runs"+query, nil))
	rec := httptest.NewRecorder()
	h.AdminListRuns(rec, req)
	return rec, repo
}

// Tanpa user_id: handler balik 200 (bukan 500 dari SQLSTATE 22P02),
// repo menerima userID kosong + page default.
func TestAdminListRuns_TanpaUserID_OK(t *testing.T) {
	rec, repo := doAdminRuns("")
	if rec.Code != http.StatusOK {
		t.Fatalf("ingin 200, dapat %d: %s", rec.Code, rec.Body.String())
	}
	if repo.userID != "" {
		t.Fatalf("userID harus kosong, dapat %q", repo.userID)
	}
	if repo.page.Limit != 20 || repo.page.Offset != 0 {
		t.Fatalf("page default salah: %+v", repo.page)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body bukan JSON: %v", err)
	}
	if _, ok := body["runs"]; !ok {
		t.Fatalf("body harus punya kunci runs: %s", rec.Body.String())
	}
}

// user_id valid diteruskan; user_id busuk ditolak 422; limit diteruskan.
func TestAdminListRuns_UserIDValidDanBusuk(t *testing.T) {
	uid := "a4f40000-0000-4000-8000-00000000d001"
	rec, repo := doAdminRuns("?user_id=" + uid + "&status=succeeded&limit=7")
	if rec.Code != http.StatusOK {
		t.Fatalf("valid ingin 200, dapat %d", rec.Code)
	}
	if string(repo.userID) != uid || repo.status != "succeeded" || repo.page.Limit != 7 {
		t.Fatalf("argumen salah: user=%q status=%q page=%+v", repo.userID, repo.status, repo.page)
	}

	rec2, _ := doAdminRuns("?user_id=bukan-uuid")
	if rec2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("user_id busuk ingin 422, dapat %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "user_id tidak valid") {
		t.Fatalf("pesan salah: %s", rec2.Body.String())
	}
}
