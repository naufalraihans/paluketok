// Test E2E skema tahap dinamis: usulan -> konfirmasi -> run -> skema tersimpan.
package httpapi

import (
	"net/http"
	"testing"
	"time"
)

// TestE2E_Stages_PratinjauTidakPotongKredit: GET /v1/projects/{id}/stages
// mengembalikan skema yang SAH dan tidak menyentuh kredit sama sekali — ini
// yang membuat user bisa melihat dulu sebelum memutuskan.
func TestE2E_Stages_PratinjauTidakPotongKredit(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Pratinjau Tahap", "brief": "kasir warung", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	code, before := e.do(t, "GET", "/v1/me", tok, nil)
	if code != 200 {
		t.Fatalf("me: got %d %v", code, before)
	}
	kreditAwal := int(before["credits"].(float64))

	code, res := e.do(t, "GET", "/v1/projects/"+slug+"/stages", tok, nil)
	if code != 200 {
		t.Fatalf("pratinjau tahap: got %d %v", code, res)
	}
	stages, ok := res["stages"].([]any)
	if !ok || len(stages) < 2 {
		t.Fatalf("pratinjau harus mengembalikan >= 2 tahap: %v", res)
	}
	// Bentuk tiap tahap harus lengkap: klien menampilkan key, label, dan format.
	for i, raw := range stages {
		s, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("tahap %d bukan objek: %v", i, raw)
		}
		if s["key"] == "" || s["key"] == nil {
			t.Fatalf("tahap %d tanpa key: %v", i, s)
		}
		if s["label"] == "" || s["label"] == nil {
			t.Fatalf("tahap %d tanpa label: %v", i, s)
		}
		switch s["output_format"] {
		case "markdown", "json", "dbml":
		default:
			t.Fatalf("tahap %d format tidak dikenal: %v", i, s["output_format"])
		}
	}

	// Pratinjau TIDAK boleh memotong kredit: run belum dimulai.
	_, after := e.do(t, "GET", "/v1/me", tok, nil)
	if got := int(after["credits"].(float64)); got != kreditAwal {
		t.Fatalf("pratinjau memotong kredit: %d -> %d", kreditAwal, got)
	}
}

// TestE2E_Stages_SkemaDisimpanDanDipakaiWorker: skema yang dikirim saat membuat
// run tersimpan, terbaca lagi lewat GET /v1/runs/{id}/stages, dan jumlah tahap run
// mengikuti skema itu (bukan selalu enam).
func TestE2E_Stages_SkemaDisimpanDanDipakaiWorker(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Skema Tersimpan", "brief": "b", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	// Tiga tahap, bukan enam: jumlah tahap harus ikut skema.
	body := map[string]any{"stages": []any{
		map[string]any{"key": "prd", "label": "PRD", "intent": "rumuskan masalah",
			"output_format": "markdown"},
		map[string]any{"key": "erd", "label": "ERD", "intent": "skema data",
			"output_format": "dbml", "depends_on": []string{"prd"}},
		map[string]any{"key": "rancangan-ui", "label": "Rancangan UI", "intent": "susun layar",
			"output_format": "markdown", "depends_on": []string{"prd"}},
	}}
	code, run := e.do(t, "POST", "/v1/projects/"+slug+"/runs", tok, body)
	if code != 202 {
		t.Fatalf("buat run: got %d %v", code, run)
	}
	rid, _ := run["id"].(string)
	if rid == "" {
		t.Fatalf("run tanpa id: %v", run)
	}

	// Respons pembuatan run langsung membawa skemanya.
	if st, ok := run["stages"].([]any); !ok || len(st) != 3 {
		t.Fatalf("respons create run harus membawa 3 tahap: %v", run["stages"])
	}
	if run["total"].(float64) != 3 {
		t.Fatalf("total run harus ikut skema (3), dapat %v", run["total"])
	}

	// Skema yang tersimpan sama dengan yang dikirim, urut position.
	code, got := e.do(t, "GET", "/v1/runs/"+rid+"/stages", tok, nil)
	if code != 200 {
		t.Fatalf("baca skema run: got %d %v", code, got)
	}
	list, _ := got["stages"].([]any)
	if len(list) != 3 {
		t.Fatalf("skema tersimpan harus 3 tahap, dapat %d: %v", len(list), got)
	}
	if key := list[2].(map[string]any)["key"]; key != "rancangan-ui" {
		t.Fatalf("urutan skema tidak sesuai kiriman: %v", key)
	}
	if fmt := list[1].(map[string]any)["output_format"]; fmt != "dbml" {
		t.Fatalf("format tahap tidak tersimpan: %v", fmt)
	}

	// Worker menjalankan run sampai selesai dengan TOTAL dari skema.
	waitFor(t, 5*time.Second, func() bool {
		c, rr := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
		return c == http.StatusOK && rr["status"] == "done"
	}, "run done")
	_, fin := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
	if fin["total"].(float64) != 3 {
		t.Fatalf("total setelah selesai harus 3, dapat %v", fin["total"])
	}
	// progress adalah INDEKS tahap yang terakhir dikerjakan (0-based), jadi run
	// 3 tahap selesai dengan progress 2. Total yang menentukan jumlah tahap.
	if fin["progress"].(float64) != 2 {
		t.Fatalf("progress run 3 tahap harus 2 (0-based), dapat %v", fin["progress"])
	}
	// Semua tahap skema harus menghasilkan artefak, termasuk key yang bebas
	// ("rancangan-ui"): bukan cuma enam kind bawaan.
	_, arts := e.do(t, "GET", "/v1/runs/"+rid+"/artifacts", tok, nil)
	artList := arts["artifacts"].([]any)
	if len(artList) != 3 {
		t.Fatalf("harus ada 3 artefak, dapat %d: %v", len(artList), arts)
	}
	kinds := map[string]bool{}
	for _, a := range artList {
		kinds[a.(map[string]any)["kind"].(string)] = true
	}
	if !kinds["rancangan-ui"] {
		t.Fatalf("artefak key bebas tidak dibuat: %v", kinds)
	}
}

// TestE2E_Stages_SkemaTidakSahDitolak400: skema dari klien diperiksa SEBELUM
// run dibuat, jadi kredit tidak pernah terpotong untuk usulan yang tidak sah.
func TestE2E_Stages_SkemaTidakSahDitolak400(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Skema Busuk", "brief": "b", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	_, me := e.do(t, "GET", "/v1/me", tok, nil)
	sebelum := int(me["credits"].(float64))

	cases := map[string]any{
		"format tidak dikenal": map[string]any{"stages": []any{
			map[string]any{"key": "prd", "label": "PRD", "output_format": "pdf"},
			map[string]any{"key": "erd", "label": "ERD", "output_format": "dbml"},
		}},
		"key tidak sah": map[string]any{"stages": []any{
			map[string]any{"key": "PRD!", "label": "PRD", "output_format": "markdown"},
			map[string]any{"key": "erd", "label": "ERD", "output_format": "dbml"},
		}},
		"key dobel": map[string]any{"stages": []any{
			map[string]any{"key": "prd", "label": "a", "output_format": "markdown"},
			map[string]any{"key": "prd", "label": "b", "output_format": "markdown"},
		}},
		"cuma satu tahap": map[string]any{"stages": []any{
			map[string]any{"key": "prd", "label": "PRD", "output_format": "markdown"},
		}},
		"depends_on asing": map[string]any{"stages": []any{
			map[string]any{"key": "prd", "label": "PRD", "output_format": "markdown"},
			map[string]any{"key": "erd", "label": "ERD", "output_format": "dbml",
				"depends_on": []string{"tidak-ada"}},
		}},
	}
	for nama, body := range cases {
		code, r := e.do(t, "POST", "/v1/projects/"+slug+"/runs", tok, body)
		if code != 400 {
			t.Fatalf("%s: harus 400, dapat %d %v", nama, code, r)
		}
	}

	// Tidak ada satu pun percobaan di atas yang boleh memotong kredit.
	_, after := e.do(t, "GET", "/v1/me", tok, nil)
	if got := int(after["credits"].(float64)); got != sebelum {
		t.Fatalf("usulan tidak sah memotong kredit: %d -> %d", sebelum, got)
	}
}

// TestE2E_Stages_RunLamaTanpaSkemaTetapJalan: run tanpa skema (klien tak
// mengirim tahap, judge tidak balas pilihan) tetap memakai enam tahap awal dan
// tetap selesai. Ini jalur cadangan yang menjaga kompatibilitas run lama.
func TestE2E_Stages_RunLamaTanpaSkemaTetapJalan(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Tanpa Skema", "brief": "b", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	code, run := e.do(t, "POST", "/v1/projects/"+slug+"/runs", tok, nil)
	if code != 202 {
		t.Fatalf("buat run: got %d %v", code, run)
	}
	rid, _ := run["id"].(string)

	// Tahap dibalas apa pun keadaannya: klien tidak perlu menebak null.
	code, got := e.do(t, "GET", "/v1/runs/"+rid+"/stages", tok, nil)
	if code != 200 {
		t.Fatalf("baca skema: got %d %v", code, got)
	}
	if list, _ := got["stages"].([]any); len(list) == 0 {
		t.Fatalf("run tanpa skema harus dibalas susunan awal, dapat kosong")
	}

	waitFor(t, 5*time.Second, func() bool {
		c, rr := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
		return c == http.StatusOK && rr["status"] == "done"
	}, "run tanpa skema tetap selesai")
}
