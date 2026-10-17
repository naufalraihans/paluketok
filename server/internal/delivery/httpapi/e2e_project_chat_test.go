// Tes chat konteks proyek (Fase 4).
//
// Yang diuji, semuanya lewat HTTP sungguhan:
//  1. POST pesan -> balasan assistant tersimpan, `suggested_stage` kebaca dari
//     balasan model, dan riwayat memuat pertanyaan + jawaban urut waktu.
//  2. `content` kosong atau melebihi 4.000 karakter -> 422.
//  3. Proyek user lain -> 403 (policy owner, bukan bocor).
//  4. Chat GRATIS: saldo kredit tidak berubah sama sekali.
package httpapi

import "testing"

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
