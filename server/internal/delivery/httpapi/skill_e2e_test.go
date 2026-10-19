package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// Tes E2E endpoint admin skill (migrasi 0009), lewat router dan JWT sah seperti
// jalur produksi. Yang diuji: siapa boleh menulis, bentuk balasan, dan pesan
// error yang sampai ke pengguna.

// Admin menambah skill, lalu daftarnya memuat skill itu.
func TestE2E_Skill_AdminTambahDanDaftar(t *testing.T) {
	e := newE2E(t)
	adm := e.fix.token(t, uidE2EAdmin)

	code, dibuat := e.do(t, http.MethodPost, "/v1/admin/skills", adm, map[string]any{
		"nama": "cek-keamanan", "ringkasan": "periksa ancaman", "isi": "# Aturan\n\n- periksa izin"})
	if code != http.StatusCreated {
		t.Fatalf("tambah skill: got %d %v", code, dibuat)
	}
	if dibuat["nama"] != "cek-keamanan" {
		t.Fatalf("nama tidak dikembalikan: %v", dibuat["nama"])
	}

	code, list := e.do(t, http.MethodGet, "/v1/admin/skills", adm, nil)
	if code != http.StatusOK {
		t.Fatalf("daftar skill: got %d %v", code, list)
	}
	skills, _ := list["skills"].([]any)
	if len(skills) != 1 {
		t.Fatalf("harus ada 1 skill, dapat %d: %v", len(skills), list)
	}
	first, _ := skills[0].(map[string]any)
	if first["nama"] != "cek-keamanan" {
		t.Fatalf("skill tidak muncul: %v", first)
	}
	if isi, _ := first["isi"].(string); !strings.Contains(isi, "periksa izin") {
		t.Fatalf("isi skill tidak utuh: %q", isi)
	}
}

// Pengguna biasa DITOLAK. Skill global, jadi ini batas keamanan yang penting.
func TestE2E_Skill_BukanAdminDitolak(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	code, body := e.do(t, http.MethodPost, "/v1/admin/skills", tok,
		map[string]any{"nama": "nakal", "isi": "coba-coba"})
	if code != http.StatusForbidden {
		t.Fatalf("pengguna biasa harus 403, dapat %d: %v", code, body)
	}
	// Pesan error tidak boleh menggemakan isi kiriman.
	if s, _ := body["error"].(map[string]any); s != nil {
		if msg, _ := s["message"].(string); strings.Contains(msg, "coba-coba") {
			t.Fatal("pesan error tidak boleh memuat isi kiriman")
		}
	}
	// Dan yang penting: skill-nya benar-benar tidak tersimpan.
	if _, list := e.do(t, http.MethodGet, "/v1/admin/skills", e.fix.token(t, uidE2EAdmin), nil); list != nil {
		if arr, _ := list["skills"].([]any); len(arr) != 0 {
			t.Fatalf("skill nakal tersimpan: %v", arr)
		}
	}
}

// Nama duplikat = 409, bukan 500: itu tabrakan yang bisa dipahami pengguna.
func TestE2E_Skill_NamaDuplikat409(t *testing.T) {
	e := newE2E(t)
	adm := e.fix.token(t, uidE2EAdmin)
	in := map[string]any{"nama": "ganda", "isi": "isi pertama"}

	if code, body := e.do(t, http.MethodPost, "/v1/admin/skills", adm, in); code != http.StatusCreated {
		t.Fatalf("tambah pertama: got %d %v", code, body)
	}
	code, body := e.do(t, http.MethodPost, "/v1/admin/skills", adm, in)
	if code != http.StatusConflict {
		t.Fatalf("duplikat harus 409, dapat %d: %v", code, body)
	}
}

// Nama tidak valid dan isi kosong = 422, dicek sebelum menyentuh DB.
func TestE2E_Skill_ValidasiInput(t *testing.T) {
	e := newE2E(t)
	adm := e.fix.token(t, uidE2EAdmin)

	kasus := []map[string]any{
		{"nama": "Cek Keamanan", "isi": "isi"},            // huruf besar + spasi
		{"nama": "cek_keamanan", "isi": "isi"},            // garis bawah tidak bisa dipanggil
		{"nama": "a", "isi": ""},                          // isi kosong
		{"nama": "-awalan", "isi": "isi"},                 // tanda hubung di awal
		{"nama": "ok", "isi": strings.Repeat("x", 20001)}, // melebihi batas
	}
	for _, in := range kasus {
		code, body := e.do(t, http.MethodPost, "/v1/admin/skills", adm, in)
		if code != http.StatusUnprocessableEntity {
			t.Errorf("input %v harus 422, dapat %d: %v", in, code, body)
		}
	}
}

// Ubah isi; nama TIDAK berubah walau dikirim; hapus mengembalikan 204 dan
// hapus kedua 404.
func TestE2E_Skill_UbahDanHapus(t *testing.T) {
	e := newE2E(t)
	adm := e.fix.token(t, uidE2EAdmin)

	code, dibuat := e.do(t, http.MethodPost, "/v1/admin/skills", adm,
		map[string]any{"nama": "ubah-nama", "ringkasan": "lama", "isi": "isi lama"})
	if code != http.StatusCreated {
		t.Fatalf("tambah: got %d %v", code, dibuat)
	}
	id, _ := dibuat["id"].(string)
	if id == "" {
		t.Fatalf("id tidak terbaca: %v", dibuat)
	}

	// Nama di body sengaja berbeda: harus DIABAIKAN, karena nama adalah cara
	// skill dipanggil dan menggantinya memutus percakapan yang menyebutnya.
	code, ubah := e.do(t, http.MethodPut, "/v1/admin/skills/"+id, adm,
		map[string]any{"nama": "nama-baru", "ringkasan": "baru", "isi": "isi baru"})
	if code != http.StatusOK {
		t.Fatalf("ubah: got %d %v", code, ubah)
	}
	if ubah["nama"] != "ubah-nama" {
		t.Fatalf("nama tidak boleh berubah: %v", ubah["nama"])
	}
	if isi, _ := ubah["isi"].(string); isi != "isi baru" {
		t.Fatalf("isi tidak berubah: %q", isi)
	}

	if code, body := e.do(t, http.MethodDelete, "/v1/admin/skills/"+id, adm, nil); code != http.StatusNoContent {
		t.Fatalf("hapus: got %d %v", code, body)
	}
	// Hapus lagi: 404, membuktikan barisnya hilang, bukan diam-diam gagal.
	if code, _ := e.do(t, http.MethodDelete, "/v1/admin/skills/"+id, adm, nil); code != http.StatusNotFound {
		t.Fatalf("hapus kedua harus 404, dapat %d", code)
	}
}

// /judge di chat lewat HTTP: balasan tersimpan sebagai pesan assistant dan
// memuat keputusan judge.
func TestE2E_PerintahJudgeLewatChat(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	code, p := e.do(t, http.MethodPost, "/v1/projects", tok, map[string]any{
		"name": "Uji Judge", "brief": "proyek untuk menguji perintah judge",
		"stack": map[string]string{"be": "go"}})
	if code != http.StatusCreated {
		t.Fatalf("buat proyek: got %d %v", code, p)
	}
	slug, _ := p["slug"].(string)
	if slug == "" {
		t.Fatal("proyek harus punya slug")
	}

	code, reply := e.do(t, http.MethodPost, "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "/judge [gas, tunda] lanjut migrasi ini?"})
	if code != http.StatusOK {
		t.Fatalf("kirim perintah: got %d %v", code, reply)
	}
	content, _ := reply["content"].(string)
	if !strings.Contains(content, "Keputusan") {
		t.Fatalf("balasan tidak memuat keputusan judge: %q", content)
	}
	if reply["role"] != "assistant" {
		t.Fatalf("balasan harus pesan assistant: %v", reply["role"])
	}

	// Tersimpan di riwayat: percakapan memuat perintah user dan balasannya.
	code, hist := e.do(t, http.MethodGet, "/v1/projects/"+slug+"/messages", tok, nil)
	if code != http.StatusOK {
		t.Fatalf("riwayat: got %d %v", code, hist)
	}
	arr, _ := hist["messages"].([]any)
	if len(arr) < 2 {
		t.Fatalf("harus ada 2 pesan (perintah + balasan), dapat %d: %v", len(arr), hist)
	}
}

// /help dan daftar skill bekerja lewat chat, dan teks biasa tidak tersentuh.
func TestE2E_PerintahHelpDanBiasa(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	adm := e.fix.token(t, uidE2EAdmin)

	_, p := e.do(t, http.MethodPost, "/v1/projects", tok, map[string]any{
		"name": "Uji Help", "brief": "bantuan", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	// /help menjawab sendiri, tanpa memanggil LLM planner.
	code, reply := e.do(t, http.MethodPost, "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "/help"})
	if code != http.StatusOK {
		t.Fatalf("/help: got %d %v", code, reply)
	}
	if c, _ := reply["content"].(string); !strings.Contains(c, "/judge") {
		t.Fatalf("/help tidak menyebut perintah: %q", c)
	}

	// /skill mendaftar skill yang terpasang (admin sudah menambah satu).
	if code, body := e.do(t, http.MethodPost, "/v1/admin/skills", adm,
		map[string]any{"nama": "daftar-uji", "ringkasan": "ringkas", "isi": "isi"}); code != http.StatusCreated {
		t.Fatalf("siapkan skill: got %d %v", code, body)
	}
	code, reply = e.do(t, http.MethodPost, "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "/skill"})
	if code != http.StatusOK {
		t.Fatalf("/skill: got %d %v", code, reply)
	}
	if c, _ := reply["content"].(string); !strings.Contains(c, "daftar-uji") {
		t.Fatalf("/skill tidak memuat skill terpasang: %q", c)
	}

	// Teks tanpa garis miring tetap dijawab planner (LLM palsu).
	code, reply = e.do(t, http.MethodPost, "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "hmmm, kamu yakin?"})
	if code != http.StatusOK {
		t.Fatalf("teks biasa: got %d %v", code, reply)
	}
	if c, _ := reply["content"].(string); c == "" {
		t.Fatal("planner tidak menjawab")
	}
}
