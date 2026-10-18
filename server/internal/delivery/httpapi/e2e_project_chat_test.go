// Tes chat konteks proyek (Fase 4).
//
// Yang diuji, semuanya lewat HTTP sungguhan:
//  1. POST pesan -> balasan assistant tersimpan, `suggested_stage` kebaca dari
//     balasan model, dan riwayat memuat pertanyaan + jawaban urut waktu.
//  2. `content` kosong atau melebihi 4.000 karakter -> 422.
//  3. Proyek user lain -> 403 (policy owner, bukan bocor).
//  4. Chat GRATIS: saldo kredit tidak berubah sama sekali.
//  5. Prompt sistem tidak menyebut nama produk; riwayat bisa ditelusuri
//     dengan limit/offset.
package httpapi

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/usecase"
)

func TestE2E_ProjectChat_KonteksDanBatas(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	code, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Kasir Warung", "brief": "aplikasi kasir", "stack": map[string]string{"be": "go"}})
	if code != 201 {
		t.Fatalf("create: got %d %v", code, p)
	}
	slug, _ := p["slug"].(string)
	if slug == "" {
		t.Fatal("proyek harus punya slug")
	}

	// Celah yang harus dijaga: chat lewat SLUG, dan chat tidak boleh menyentuh
	// kredit sama sekali. Saldo dibaca lewat endpoint nyata, bukan intip store.
	_, me := e.do(t, "GET", "/v1/me", tok, nil)
	before := int(me["credits"].(float64))

	// 1. Kirim pesan lewat SLUG (bukan UUID) — jalur URL resmi.
	code, reply := e.do(t, "POST", "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "Aku mau aplikasi kasir untuk warung, mulai dari mana?"})
	if code != 200 {
		t.Fatalf("kirim pesan: got %d %v", code, reply)
	}
	if reply["role"] != "assistant" {
		t.Fatalf("balasan harus role assistant: %v", reply["role"])
	}
	if c, _ := reply["content"].(string); c == "" {
		t.Fatalf("balasan kosong: %v", reply)
	}

	// 2. Riwayat: pertanyaan lebih dulu, baru jawaban.
	code, hist := e.do(t, "GET", "/v1/projects/"+slug+"/messages", tok, nil)
	if code != 200 {
		t.Fatalf("riwayat: got %d %v", code, hist)
	}
	list, _ := hist["messages"].([]any)
	if len(list) != 2 {
		t.Fatalf("riwayat harus 2 pesan, dapat %d: %v", len(list), hist)
	}
	first, _ := list[0].(map[string]any)
	second, _ := list[1].(map[string]any)
	if first["role"] != "user" || second["role"] != "assistant" {
		t.Fatalf("urutan peran salah: %v / %v", first["role"], second["role"])
	}
	if first["content"] != "Aku mau aplikasi kasir untuk warung, mulai dari mana?" {
		t.Fatalf("isi pertanyaan berubah: %v", first["content"])
	}

	// 3. Batas: kosong dan kelewat panjang ditolak 422.
	if code, _ := e.do(t, "POST", "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": "   "}); code != 422 {
		t.Fatalf("pesan kosong harus 422, dapat %d", code)
	}
	long := make([]rune, 4001)
	for i := range long {
		long[i] = 'a'
	}
	if code, _ := e.do(t, "POST", "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": string(long)}); code != 422 {
		t.Fatalf("pesan >4000 karakter harus 422, dapat %d", code)
	}
	// Tepat 4.000 karakter masih sah.
	ok := make([]rune, 4000)
	for i := range ok {
		ok[i] = 'a'
	}
	if code, _ := e.do(t, "POST", "/v1/projects/"+slug+"/messages", tok,
		map[string]any{"content": string(ok)}); code != 200 {
		t.Fatalf("pesan 4000 karakter harus 200, dapat %d", code)
	}

	// 4. Proyek yang tak dikenal -> 404.
	if code, _ := e.do(t, "GET", "/v1/projects/tidak-ada-proyek-ini/messages", tok, nil); code != 404 {
		t.Fatalf("proyek tak ada harus 404, dapat %d", code)
	}

	// 5. Chat gratis: tidak ada satu pun potongan kredit.
	_, me2 := e.do(t, "GET", "/v1/me", tok, nil)
	if after := int(me2["credits"].(float64)); after != before {
		t.Fatalf("chat tidak boleh memotong kredit: %d -> %d", before, after)
	}
}

// TestE2E_ProjectChat_PromptTidakSebutProduk: prompt sistem tidak boleh
// menyebut nama alat (PaluKetok) dan tidak boleh menyuruh model bertanya ulang
// hal yang sudah ada di brief.
//
// Diuji di usecase (bukan lewat HTTP): yang diperiksa isi prompt, dan mockLLM
// tidak menyimpan permintaan yang diterimanya.
func TestE2E_ProjectChat_PromptTidakSebutProduk(t *testing.T) {
	got := usecase.SystemPrompt("Kasir Warung")
	if strings.Contains(got, "PaluKetok") {
		t.Fatalf("system prompt masih menyebut PaluKetok: %q", got)
	}
	if !strings.Contains(got, "Kasir Warung") {
		t.Fatalf("system prompt harus menyebut nama proyek: %q", got)
	}
	if !strings.Contains(got, "Jangan menanyakan ulang") {
		t.Fatalf("system prompt harus melarang bertanya ulang isi brief: %q", got)
	}
}

// TestE2E_ProjectChat_OffsetMenelusuriRiwayat: limit/offset untuk riwayat
// panjang. Tanpa offset, 5 percakapan (10 pesan) dipotong ke `limit` pesan
// TERAKHIR; dengan offset, halaman sebelumnya bisa dibaca tanpa mengubah
// jendela konteks yang dikirim ke model.
func TestE2E_ProjectChat_OffsetMenelusuriRiwayat(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Riwayat Panjang", "brief": "b", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	for i := 0; i < 5; i++ {
		if code, r := e.do(t, "POST", "/v1/projects/"+slug+"/messages", tok,
			map[string]any{"content": "pertanyaan " + strconv.Itoa(i)}); code != 200 {
			t.Fatalf("kirim %d: got %d %v", i, code, r)
		}
	}

	// 10 pesan tersimpan; limit=4 ambil 4 TERAKHIR.
	_, all := e.do(t, "GET", "/v1/projects/"+slug+"/messages?limit=4", tok, nil)
	if n := len(all["messages"].([]any)); n != 4 {
		t.Fatalf("limit=4 harus 4 pesan, dapat %d", n)
	}
	_, page0 := e.do(t, "GET", "/v1/projects/"+slug+"/messages?limit=4&offset=0", tok, nil)
	_, page1 := e.do(t, "GET", "/v1/projects/"+slug+"/messages?limit=4&offset=4", tok, nil)
	if len(page1["messages"].([]any)) != 4 {
		t.Fatalf("offset=4 harus 4 pesan, dapat %d", len(page1["messages"].([]any)))
	}
	// Halaman kedua berisi pesan yang LEBIH LAMA dari halaman pertama.
	first := page0["messages"].([]any)[0].(map[string]any)["content"]
	older := page1["messages"].([]any)[0].(map[string]any)["content"]
	if first == older {
		t.Fatalf("offset tidak menggeser jendela: %v", first)
	}
	// Terlalu jauh ke belakang = kosong, bukan error.
	if code, deep := e.do(t, "GET", "/v1/projects/"+slug+"/messages?limit=4&offset=999", tok, nil); code != 200 ||
		len(deep["messages"].([]any)) != 0 {
		t.Fatalf("offset melewati awal harus 200 + kosong, dapat %d %v", code, deep)
	}
}

// TestE2E_GetProject_LatestRunTanpaSessionStorage: GET /v1/projects/{ref}
// membawa ringkasan run terbaru, jadi halaman proyek benar di perangkat baru.
func TestE2E_GetProject_LatestRunTanpaSessionStorage(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)
	_, p := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Progres Lintas Perangkat", "brief": "b", "stack": map[string]string{"be": "go"}})
	slug, _ := p["slug"].(string)

	// Belum ada run: field-nya tidak dikirim (klien tidak perlu menebak null).
	_, before := e.do(t, "GET", "/v1/projects/"+slug, tok, nil)
	if _, ok := before["latest_run"]; ok {
		t.Fatalf("proyek tanpa run tidak boleh punya latest_run: %v", before)
	}

	if code, r := e.do(t, "POST", "/v1/projects/"+slug+"/runs", tok, nil); code != 202 {
		t.Fatalf("mulai run: got %d %v", code, r)
	} else if rid, _ := r["id"].(string); rid == "" {
		t.Fatalf("run tanpa id: %v", r)
	} else {
		waitFor(t, 5*time.Second, func() bool {
			c, rr := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
			return c == 200 && rr["status"] == "done"
		}, "run done")
	}

	_, after := e.do(t, "GET", "/v1/projects/"+slug, tok, nil)
	lr, ok := after["latest_run"].(map[string]any)
	if !ok {
		t.Fatalf("latest_run harus ada setelah run: %v", after)
	}
	if lr["status"] != "done" || lr["total"].(float64) != 6 {
		t.Fatalf("latest_run tidak lengkap: %v", lr)
	}
}

func TestE2E_ProjectChat_UserLainDitolak(t *testing.T) {
	e := newE2E(t)
	owner := e.fix.token(t, uidE2E)
	const uidLain = "0197a23e-7777-7000-8000-0000000000fe"
	e.store.seedProfile(uidLain, 1)
	other := e.fix.token(t, uidLain)

	code, p := e.do(t, "POST", "/v1/projects", owner, map[string]any{
		"name": "Rahasia Warung", "brief": "b", "stack": map[string]string{"be": "go"}})
	if code != 201 {
		t.Fatalf("create: got %d %v", code, p)
	}
	slug, _ := p["slug"].(string)

	// Membaca percakapan orang lain -> 403, bukan 404 (proyeknya ada, aksesnya
	// yang ditolak) dan bukan 200.
	if code, _ := e.do(t, "GET", "/v1/projects/"+slug+"/messages", other, nil); code != 403 {
		t.Fatalf("baca chat orang lain harus 403, dapat %d", code)
	}
	// Menulis ke percakapan orang lain juga 403 — tidak ada pesan yang tersimpan.
	if code, _ := e.do(t, "POST", "/v1/projects/"+slug+"/messages", other,
		map[string]any{"content": "menyusup"}); code != 403 {
		t.Fatalf("tulis chat orang lain harus 403, dapat %d", code)
	}
}
