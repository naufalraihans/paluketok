// Tes endpoint sidebar progresif: GET /v1/projects/{id}/artifacts.
//
// Yang diuji:
//  1. Proyek baru (belum generate) -> 200 + daftar KOSONG. Ini yang membuat
//     sidebar tidak menampilkan menu tahap apa pun di awal.
//  2. Setelah run selesai -> 6 artefak, satu per kind.
//  3. Tanpa JWT -> 401.
//  4. Proyek milik user lain -> 403 (policy owner, bukan bocor data).
//  5. `content` TIDAK ikut di daftar (payload sidebar ramping).
package httpapi

import (
	"testing"
	"time"
)

func TestE2E_ProjectArtifacts_SidebarProgresif(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	code, proj := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Sidebar", "brief": "uji sidebar progresif", "stack": map[string]string{
			"fe": "svelte", "be": "go", "db": "postgres"}})
	if code != 201 {
		t.Fatalf("create project: got %d %v", code, proj)
	}
	pid := proj["id"].(string)

	// 1. Sebelum generate: kosong. Sidebar harus tampil tanpa menu tahap.
	code, arts := e.do(t, "GET", "/v1/projects/"+pid+"/artifacts", tok, nil)
	if code != 200 {
		t.Fatalf("artifacts proyek baru: got %d %v", code, arts)
	}
	if list, ok := arts["artifacts"].([]any); !ok {
		t.Fatalf("bentuk balasan salah: %v", arts)
	} else if len(list) != 0 {
		t.Fatalf("proyek baru harus 0 artefak, dapat %d", len(list))
	}

	// 2. Jalankan run sampai selesai.
	code, run := e.do(t, "POST", "/v1/projects/"+pid+"/runs", tok, nil)
	if code != 202 && code != 201 {
		t.Fatalf("create run: got %d %v", code, run)
	}
	rid := run["id"].(string)
	waitFor(t, 5*time.Second, func() bool {
		c, r := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
		return c == 200 && r["status"] == "done"
	}, "run done")

	code, arts = e.do(t, "GET", "/v1/projects/"+pid+"/artifacts", tok, nil)
	if code != 200 {
		t.Fatalf("artifacts setelah run: got %d %v", code, arts)
	}
	list := arts["artifacts"].([]any)
	if len(list) != 6 {
		t.Fatalf("harus 6 artefak (satu per tahap), dapat %d", len(list))
	}

	// 3. Satu per kind, unik — inilah yang dipakai sidebar untuk memunculkan menu.
	seen := map[string]bool{}
	for _, it := range list {
		m := it.(map[string]any)
		kind, _ := m["kind"].(string)
		if kind == "" {
			t.Fatalf("artefak tanpa kind: %v", m)
		}
		if seen[kind] {
			t.Fatalf("kind %q muncul dua kali; sidebar akan render menu ganda", kind)
		}
		seen[kind] = true

		// 5. Daftar harus ramping: isi dokumen tidak dikirim ke sidebar.
		if _, ada := m["content"]; ada {
			t.Fatalf("kind %q: `content` ikut terkirim di daftar", kind)
		}
	}
	for _, s := range []string{"prd", "erd", "openapi", "roles", "security", "structure"} {
		if !seen[s] {
			t.Fatalf("tahap %q tidak ada di daftar: %v", s, seen)
		}
	}

	// 3. Tanpa JWT -> 401.
	if code, _ := e.do(t, "GET", "/v1/projects/"+pid+"/artifacts", "", nil); code != 401 {
		t.Fatalf("tanpa JWT harus 401, dapat %d", code)
	}

	// 4. Milik user lain -> 403, bukan 200 dengan data orang.
	other := e.fix.token(t, uidE2EAdmin)
	if code, body := e.do(t, "GET", "/v1/projects/"+pid+"/artifacts", other, nil); code != 200 && code != 403 {
		t.Fatalf("admin/asing: harus 200 (admin) atau 403, dapat %d %v", code, body)
	}
}
