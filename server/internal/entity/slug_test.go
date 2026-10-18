package entity

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Kasir Warung", "kasir-warung"},
		{"Testing Jembut", "testing-jembut"},
		{"  Spasi   Banyak  ", "spasi-banyak"},
		{"Aplikasi Kasir (v2)!", "aplikasi-kasir-v2"},
		{"sudah-slug", "sudah-slug"},
		{"HurufBesarKecil123", "hurufbesarkecil123"},
		{"a--b", "a-b"},
		{"***", ""},                   // hanya tanda baca -> kosong, usecase ganti "proyek"
		{"Café Santai", "caf-santai"}, // non-ASCII jadi pemisah, slug tetap ASCII
		{"", ""},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, mau %q", c.in, got, c.want)
		}
	}
}

func TestSlugify_PanjangDipotong(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	got := Slugify(long)
	if len(got) != SlugMaxLen {
		t.Fatalf("panjang slug %d, mau %d", len(got), SlugMaxLen)
	}
	if !ValidSlug(got) {
		t.Fatalf("slug hasil potong tidak valid: %q", got)
	}
}

func TestValidSlug(t *testing.T) {
	ok := []string{"a", "kasir-warung", "x1-y2", "testing-jembut-2"}
	for _, s := range ok {
		if !ValidSlug(s) {
			t.Errorf("ValidSlug(%q) harus true", s)
		}
	}
	// Catatan: UUID secara sintaks juga cocok pola slug (hex + tanda hubung).
	// resolveProject memeriksa UUID lebih dulu, jadi nama proyek yang berbentuk
	// UUID diberi akhiran saat dibuat (lihat ProjectUsecase.uniqueSlug).
	bad := []string{"", "-awal", "akhir-", "HurufBesar", "spasi ada", "a--b"}
	for _, s := range bad {
		if ValidSlug(s) {
			t.Errorf("ValidSlug(%q) harus false", s)
		}
	}
}
