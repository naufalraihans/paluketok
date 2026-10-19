package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"paluketok/server/internal/entity"
)

// Tes repo skills terhadap DB nyata (migrasi 0009).
//
// Data uji dibuat sendiri dan dihapus sendiri: nama selalu berawalan "zz-uji-"
// dan dibersihkan di defer, jadi tidak ada sisa yang mengganggu skill asli.
// Jalur singkat: go test ./internal/repository/ -run Skill -count=1
func TestSkillRepo_SiklusHidup(t *testing.T) {
	_, r := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	nama := fmt.Sprintf("zz-uji-%d", now.UnixNano())

	defer func() {
		// Bersihkan walau tes gagal di tengah.
		_ = r.Skills.Delete(ctx, entity.ID(nama))
	}()

	s := &entity.Skill{
		ID:   entity.ID("a4f40000-0000-4000-8000-0000000009" + fmt.Sprintf("%02d", now.UnixNano()%100)),
		Nama: nama, Ringkasan: "skill uji", Isi: "# Aturan\n\n- selalu periksa izin",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := r.Skills.Create(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := r.Skills.GetByName(ctx, nama)
	if err != nil {
		t.Fatalf("getByName: %v", err)
	}
	if got.Isi != s.Isi || got.Ringkasan != s.Ringkasan {
		t.Fatalf("isi tidak sama: %+v", got)
	}

	// Ubah isi, pastikan tersimpan.
	got.Isi = "# Aturan revisi"
	got.UpdatedAt = now.Add(time.Second)
	if err := r.Skills.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	lagi, err := r.Skills.GetByName(ctx, nama)
	if err != nil {
		t.Fatalf("getByName setelah update: %v", err)
	}
	if lagi.Isi != "# Aturan revisi" {
		t.Fatalf("update tidak tersimpan: %q", lagi.Isi)
	}

	// Daftar harus memuatnya, dan urut nama.
	list, err := r.Skills.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Nama > list[i].Nama {
			t.Fatalf("daftar tidak urut nama: %q > %q", list[i-1].Nama, list[i].Nama)
		}
	}

	// Nama unik: create kedua dengan nama sama harus gagal.
	if err := r.Skills.Create(ctx, &entity.Skill{
		ID:   entity.ID("a4f40000-0000-4000-8000-000000000999"),
		Nama: nama, Isi: "lain", CreatedAt: now, UpdatedAt: now,
	}); err == nil {
		t.Fatal("nama duplikat harus ditolak")
	}

	// Nama tak ada = ErrNotFound, bukan error lain.
	if _, err := r.Skills.GetByName(ctx, "zz-uji-tidak-ada-sama-sekali"); err == nil {
		t.Fatal("nama tak ada harus ErrNotFound")
	}

	if err := r.Skills.Delete(ctx, s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Hapus dua kali: yang kedua ErrNotFound, membuktikan barisnya benar hilang.
	if err := r.Skills.Delete(ctx, s.ID); err == nil {
		t.Fatal("delete baris yang sudah hilang harus ErrNotFound")
	}
}
