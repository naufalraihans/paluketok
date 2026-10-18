package usecase

import (
	"strings"
	"testing"
)

// TestSystemPrompt_IdentitasKetokMagic menjaga identitas asisten.
//
// Model yang tidak diberi identitas akan mengaku sebagai model AI atau
// menyebut vendornya. Aturannya harus ada di prompt, dan nama produk pemilik
// (PaluKetok) tidak boleh muncul di sana: model bicara tentang proyek pengguna.
func TestSystemPrompt_IdentitasKetokMagic(t *testing.T) {
	p := SystemPrompt("POS Warung")

	if !strings.Contains(p, "Ketok Magic") {
		t.Error("prompt tidak memuat identitas Ketok Magic")
	}
	if !strings.Contains(p, "POS Warung") {
		t.Error("prompt tidak memuat nama proyek pengguna")
	}
	if strings.Contains(p, "PaluKetok") {
		t.Error("prompt menyebut nama produk pemilik (PaluKetok)")
	}
	// Nama model/vendor lain tidak boleh diundang masuk lewat prompt.
	for _, terlarang := range []string{"GPT", "Claude", "Gemini", "OpenAI", "Anthropic"} {
		if strings.Contains(p, terlarang) {
			t.Errorf("prompt menyebut nama model/vendor: %s", terlarang)
		}
	}
}

// TestSystemPrompt_LarangEmDash memastikan aturan gaya tulis dikirim ke model.
func TestSystemPrompt_LarangEmDash(t *testing.T) {
	p := SystemPrompt("POS Warung")
	if !strings.Contains(p, "em dash") {
		t.Error("prompt tidak melarang em dash")
	}
}

// TestBersihkanGaya_EmDashJadiKoma: jaring pengaman kalau model tetap memakai
// em dash walau sudah dilarang.
func TestBersihkanGaya_EmDashJadiKoma(t *testing.T) {
	kasus := []struct {
		nama  string
		masuk string
		harap string
	}{
		{"em dash diapit spasi", "HP Android — laptop", "HP Android, laptop"},
		{"en dash diapit spasi", "tunai – hutang", "tunai, hutang"},
		{"em dash tanpa spasi", "stok berkurang—tanpa peringatan", "stok berkurang, tanpa peringatan"},
		{"spasi di depan saja", "dua opsi —jelas", "dua opsi, jelas"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			dapat := bersihkanGaya(k.masuk)
			if dapat != k.harap {
				t.Errorf("\n masuk: %q\n harap: %q\n dapat: %q", k.masuk, k.harap, dapat)
			}
		})
	}
}

// TestBersihkanGaya_TandaHubungKataTidakDisentuh: hyphen yang menempel pada kata
// adalah bagian istilah teknis, bukan pemisah klausa. Mengubahnya jadi koma
// akan merusak istilahnya.
func TestBersihkanGaya_TandaHubungKataTidakDisentuh(t *testing.T) {
	masuk := "mode offline-first, role-based access, clamp -1"
	dapat := bersihkanGaya(masuk)
	if dapat != masuk {
		t.Errorf("istilah teknis ikut berubah:\n masuk: %q\n dapat: %q", masuk, dapat)
	}
}

// TestBersihkanGaya_SpasiPinggirDibuang: balasan model sering diakhiri baris
// kosong; itu tidak boleh ikut tersimpan.
func TestBersihkanGaya_SpasiPinggirDibuang(t *testing.T) {
	dapat := bersihkanGaya("\n\n  halo  \n\n")
	if dapat != "halo" {
		t.Errorf("harap %q, dapat %q", "halo", dapat)
	}
}
