// Tes URL ramah baca: proyek diakses lewat slug, UUID lama tetap hidup.
//
// Yang diuji:
//  1. Proyek baru langsung dapat slug dari namanya ("Kasir Warung" ->
//     "kasir-warung"), bukan UUID.
//  2. Dua proyek dengan nama sama tidak bentrok: yang kedua dapat "nama-2".
//  3. GET /v1/projects/{slug} membalas proyeknya — URL enak dibaca.
//  4. GET /v1/projects/{uuid} MASIH membalas proyek yang sama. Tautan lama
//     yang terlanjur dibagikan tidak boleh mati.
//  5. Proyek user lain lewat slug -> 403, bukan bocor.
package httpapi

import (
	"testing"
)

func TestE2E_ProjectSlug_URLRamahBaca(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	code, p1 := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Kasir Warung", "brief": "aplikasi kasir", "stack": map[string]string{"be": "go"}})
	if code != 201 {
		t.Fatalf("create 1: got %d %v", code, p1)
	}
	// 1. slug diturunkan dari nama, bukan UUID.
	if p1["slug"] != "kasir-warung" {
		t.Fatalf("slug proyek 1: mau 'kasir-warung', dapat %v", p1["slug"])
	}
	id1 := p1["id"].(string)

	// 2. nama sama -> slug kedua diberi akhiran, tidak bentrok.
	code, p2 := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "Kasir Warung", "brief": "kasir lagi", "stack": map[string]string{"be": "go"}})
	if code != 201 {
		t.Fatalf("create 2: got %d %v", code, p2)
	}
	if p2["slug"] != "kasir-warung-2" {
		t.Fatalf("slug proyek 2: mau 'kasir-warung-2', dapat %v", p2["slug"])
	}

	// 3. akses lewat slug.
	code, bySlug := e.do(t, "GET", "/v1/projects/kasir-warung", tok, nil)
	if code != 200 {
		t.Fatalf("GET lewat slug: got %d %v", code, bySlug)
	}
	if bySlug["id"] != id1 {
		t.Fatalf("slug mengembalikan proyek salah: %v vs %v", bySlug["id"], id1)
	}

	// 4. UUID lama tetap jalan (tautan yang sudah dibagikan tidak mati).
	code, byUUID := e.do(t, "GET", "/v1/projects/"+id1, tok, nil)
	if code != 200 {
		t.Fatalf("GET lewat UUID harus tetap 200: got %d %v", code, byUUID)
	}
	if byUUID["id"] != id1 {
		t.Fatalf("UUID mengembalikan proyek salah: %v", byUUID["id"])
	}

	// 4b. slug yang tak ada -> 404, bukan 500.
	if code, _ := e.do(t, "GET", "/v1/projects/tidak-ada-ini", tok, nil); code != 404 {
		t.Fatalf("slug tak ada harus 404, dapat %d", code)
	}

	// 5. user lain (non-admin) lewat slug -> 403.
	const uidLain = "0197a23e-7777-7000-8000-0000000000ff"
	e.store.seedProfile(uidLain, 1)
	other := e.fix.token(t, uidLain)
	if code, _ := e.do(t, "GET", "/v1/projects/kasir-warung", other, nil); code != 403 {
		t.Fatalf("user lain lewat slug harus 403, dapat %d", code)
	}
}

// Dua user boleh punya proyek dengan nama sama -> slug sama per owner.
// Tiap orang harus mendapat proyeknya SENDIRI, bukan 403 karena slug dipakai
// bersama, dan bukan proyek milik orang lain.
func TestE2E_ProjectSlug_SamaBedaOwner(t *testing.T) {
	e := newE2E(t)
	tokA := e.fix.token(t, uidE2E)

	const uidB = "0197a23e-7777-7000-8000-0000000000bb"
	e.store.seedProfile(uidB, 5)
	tokB := e.fix.token(t, uidB)

	body := map[string]any{"name": "Warung Kopi", "brief": "kasir", "stack": map[string]string{"be": "go"}}
	code, a := e.do(t, "POST", "/v1/projects", tokA, body)
	if code != 201 {
		t.Fatalf("create A: %d %v", code, a)
	}
	code, b := e.do(t, "POST", "/v1/projects", tokB, body)
	if code != 201 {
		t.Fatalf("create B: %d %v", code, b)
	}
	// Keduanya dapat slug yang sama: unik per owner, bukan unik global.
	if a["slug"] != "warung-kopi" || b["slug"] != "warung-kopi" {
		t.Fatalf("slug harus sama untuk dua owner: %v vs %v", a["slug"], b["slug"])
	}
	// Masing-masing membuka slug itu dan harus mendapat proyeknya sendiri.
	code, gotA := e.do(t, "GET", "/v1/projects/warung-kopi", tokA, nil)
	if code != 200 || gotA["id"] != a["id"] {
		t.Fatalf("A harus dapat proyek A: code %d id %v (mau %v)", code, gotA["id"], a["id"])
	}
	code, gotB := e.do(t, "GET", "/v1/projects/warung-kopi", tokB, nil)
	if code != 200 || gotB["id"] != b["id"] {
		t.Fatalf("B harus dapat proyek B: code %d id %v (mau %v)", code, gotB["id"], b["id"])
	}
}
