package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

func TestCreateProject_Sukses(t *testing.T) {
	deps, clock, profiles, projects, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	uc := NewProjectUsecase(deps)

	p, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{
		Name:  "  PaluKetok Demo  ",
		Brief: "Website toko online sederhana",
		Stack: entity.Stack{FE: "svelte", BE: "go", DB: "supabase", Payment: "doku"},
	})
	if err != nil {
		t.Fatalf("CreateProject gagal: %v", err)
	}
	if p.Name != "PaluKetok Demo" {
		t.Errorf("nama tidak ter-trim: %q", p.Name)
	}
	if p.OwnerID != userA {
		t.Errorf("owner salah: %q", p.OwnerID)
	}
	if p.Status != entity.ProjectDraft {
		t.Errorf("status awal harus draft, dapat %q", p.Status)
	}
	if !p.CreatedAt.Equal(clock.Now()) {
		t.Errorf("created_at bukan dari Clock: %v", p.CreatedAt)
	}
	// Entry urut: entity 5, port
	if len(projects.calls) != 1 || projects.calls[0] != "create" {
		t.Errorf("repo tidak dipanggil sekali: %v", projects.calls)
	}
}

func TestCreateProject_TolakNamaKosongDanBriefKosong(t *testing.T) {
	deps, _, profiles, _, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	uc := NewProjectUsecase(deps)

	cases := []struct {
		nama    string
		in      CreateProjectInput
		harapIs error
	}{
		{"nama kosong", CreateProjectInput{Name: "   ", Brief: "ada"}, ErrValidation},
		{"nama kebab", CreateProjectInput{Name: "a\x00b", Brief: "ada"}, ErrValidation},
		{"nama 121 char", CreateProjectInput{Name: strings.Repeat("x", 121), Brief: "ada"}, ErrValidation},
		{"brief kosong", CreateProjectInput{Name: "ok", Brief: ""}, ErrValidation},
		{"brief 8001 char", CreateProjectInput{Name: "ok", Brief: strings.Repeat("y", 8001)}, ErrValidation},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			_, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, c.in)
			if !errors.Is(err, c.harapIs) {
				t.Fatalf("harap %v, dapat %v", c.harapIs, err)
			}
		})
	}
	// batas harus inklusif: 120 char nama dan 8000 char brief diterima
	if _, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{
		Name: strings.Repeat("x", 120), Brief: strings.Repeat("y", 8000),
	}); err != nil {
		t.Fatalf("batas 120/8000 harus diterima, dapat: %v", err)
	}
}

func TestCreateProject_TolakActorKosong(t *testing.T) {
	deps, _, _, _, _, _ := newTestDeps()
	uc := NewProjectUsecase(deps)
	_, err := uc.CreateProject(context.Background(), entity.Actor{}, CreateProjectInput{Name: "x", Brief: "y"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("harap ErrForbidden, dapat %v", err)
	}
}

func TestGetProject_OwnerCheck(t *testing.T) {
	deps, _, profiles, _, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	seedProfile(profiles, userB, 3)
	uc := NewProjectUsecase(deps)

	p, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{Name: "milik A", Brief: "b"})
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// pemilik: boleh
	if _, err := uc.GetProject(context.Background(), entity.Actor{UserID: userA}, p.ID); err != nil {
		t.Fatalf("pemilik harus boleh, dapat %v", err)
	}
	// user lain: 403, bukan 404
	if _, err := uc.GetProject(context.Background(), entity.Actor{UserID: userB}, p.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("user lain harus ErrForbidden, dapat %v", err)
	}
	// admin: boleh baca semua (plan/roles.md)
	if _, err := uc.GetProject(context.Background(), entity.Actor{UserID: admin, IsAdmin: true}, p.ID); err != nil {
		t.Fatalf("admin harus boleh, dapat %v", err)
	}
	// id bukan UUID: 404 tanpa query (plan/security.md bagian 4)
	if _, err := uc.GetProject(context.Background(), entity.Actor{UserID: userA}, "bukan-uuid"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id non-UUID harus ErrNotFound, dapat %v", err)
	}
	// UUID tidak ada: 404
	if _, err := uc.GetProject(context.Background(), entity.Actor{UserID: userA}, entity.ID("99999999-9999-4999-8999-999999999999")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id tak ada harus ErrNotFound, dapat %v", err)
	}
}

func TestListProjects_HanyaMilikActorDanPaging(t *testing.T) {
	deps, _, profiles, _, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	seedProfile(profiles, userB, 3)
	uc := NewProjectUsecase(deps)

	for i := 0; i < 3; i++ {
		if _, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{Name: "A", Brief: "b"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userB}, CreateProjectInput{Name: "B", Brief: "b"}); err != nil {
		t.Fatal(err)
	}

	list, err := uc.ListProjects(context.Background(), entity.Actor{UserID: userA}, port.Page{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("A harus lihat 3 project, dapat %d", len(list))
	}
	for _, p := range list {
		if p.OwnerID != userA {
			t.Errorf("list A memuat project user lain: %q", p.OwnerID)
		}
	}

	// limit di luar batas di-clamp ke 50, bukan diteruskan apa adanya
	page2, err := uc.ListProjects(context.Background(), entity.Actor{UserID: userA}, port.Page{Limit: 9999, Offset: -5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 3 {
		t.Errorf("clamp limit gagal: %d", len(page2))
	}
}

func TestUpdateProject_OwnerCheckDanValidasi(t *testing.T) {
	deps, _, profiles, _, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	seedProfile(profiles, userB, 3)
	uc := NewProjectUsecase(deps)

	p, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{Name: "awal", Brief: "b"})
	if err != nil {
		t.Fatal(err)
	}

	// user lain tidak boleh mengubah
	namaBaru := "diubah B"
	if _, err := uc.UpdateProject(context.Background(), entity.Actor{UserID: userB}, p.ID, UpdateProjectInput{Name: &namaBaru}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("B harus ErrForbidden, dapat %v", err)
	}

	// pemilik boleh
	got, err := uc.UpdateProject(context.Background(), entity.Actor{UserID: userA}, p.ID, UpdateProjectInput{Name: &namaBaru})
	if err != nil {
		t.Fatalf("update owner: %v", err)
	}
	if got.Name != "diubah B" {
		t.Errorf("nama tidak berubah: %q", got.Name)
	}

	// status harus enum
	statusSalah := entity.ProjectStatus("selesai")
	if _, err := uc.UpdateProject(context.Background(), entity.Actor{UserID: userA}, p.ID, UpdateProjectInput{Status: &statusSalah}); !errors.Is(err, ErrValidation) {
		t.Fatalf("status salah harus ErrValidation, dapat %v", err)
	}

	// riwayat: created_at tidak berubah, updated_at ikut Clock
	if !got.CreatedAt.Equal(p.CreatedAt) {
		t.Errorf("created_at berubah saat update")
	}
	if !got.UpdatedAt.Equal(deps.Clock.Now()) {
		t.Errorf("updated_at bukan dari Clock: %v", got.UpdatedAt)
	}
}

func TestArchiveProject_SetStatusBukanHapus(t *testing.T) {
	deps, _, profiles, projects, _, _ := newTestDeps()
	seedProfile(profiles, userA, 3)
	seedProfile(profiles, userB, 3)
	uc := NewProjectUsecase(deps)

	p, err := uc.CreateProject(context.Background(), entity.Actor{UserID: userA}, CreateProjectInput{Name: "arsip", Brief: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.ArchiveProject(context.Background(), entity.Actor{UserID: userB}, p.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("B harus ErrForbidden, dapat %v", err)
	}
	if err := uc.ArchiveProject(context.Background(), entity.Actor{UserID: userA}, p.ID); err != nil {
		t.Fatalf("arsip owner: %v", err)
	}

	got, err := projects.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("baris harus masih ada setelah arsip (bukan hapus): %v", err)
	}
	if got.Status != entity.ProjectArchived {
		t.Errorf("status harus archived, dapat %q", got.Status)
	}
	// tidak ada panggilan delete: fake hanya punya create/update/archive
	for _, c := range projects.calls {
		if c == "delete" {
			t.Errorf("archive tidak boleh menghapus baris")
		}
	}
}
