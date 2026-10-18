// Test penyusun skema tahap: pemilihan kandidat, penolakan usulan, cadangan.
package worker

import (
	"context"
	"errors"
	"testing"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// stageJudgeFunc judge yang menjawab dengan choice tetap (atau error).
type stageJudgeFunc struct {
	choice string
	err    error
	got    port.JudgeRequest
}

func (f *stageJudgeFunc) Decide(_ context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	f.got = req
	if f.err != nil {
		return nil, f.err
	}
	return &port.JudgeResponse{Answers: map[string]port.JudgeAnswer{
		"q1": {QuestionID: "q1", Choice: f.choice, Confidence: 0.9},
	}}, nil
}

// TestProposePlanStages_PilihKandidat: pilihan judge yang sah dipakai apa adanya,
// dan jumlah tahapnya mengikuti kandidat itu (bukan selalu enam).
func TestProposePlanStages_PilihKandidat(t *testing.T) {
	j := &stageJudgeFunc{choice: "ringkas"}
	got, note := ProposePlanStages(context.Background(), j, "kasir warung", map[string]string{"be": "go"})
	if note != "" {
		t.Fatalf("pilihan sah tidak boleh beri catatan: %q", note)
	}
	if len(got) != 4 {
		t.Fatalf("kandidat ringkas harus 4 tahap, dapat %d", len(got))
	}
	// Kriteria yang dikirim ke judge harus membawa brief + stack: judge yang
	// memutuskan, jadi ia butuh konteksnya.
	raw, ok := j.got.State.(map[string]any)
	if !ok {
		t.Fatalf("state bukan map: %T", j.got.State)
	}
	if raw["brief"] != "kasir warung" {
		t.Fatalf("brief tidak diteruskan ke judge: %v", raw["brief"])
	}
	if raw["stack"] == nil {
		t.Fatalf("stack tidak diteruskan ke judge")
	}
	// depends_on kandidat harus ikut tersimpan, bukan dibuang.
	if len(got[1].DependsOn) == 0 || got[1].DependsOn[0] != entity.StagePRD {
		t.Fatalf("depends_on tahap kedua salah: %+v", got[1].DependsOn)
	}
}

// TestProposePlanStages_CadanganEnamTahap: semua jalur gagal harus berakhir pada
// susunan awal yang SAH — bukan error, bukan skema separuh.
func TestProposePlanStages_CadanganEnamTahap(t *testing.T) {
	cases := []struct {
		nama  string
		judge stageJudger
	}{
		{"judge nil", nil},
		{"judge error", &stageJudgeFunc{err: errors.New("jev mati")}},
		{"choice tak dikenal", &stageJudgeFunc{choice: "entah"}},
		{"choice kosong", &stageJudgeFunc{choice: ""}},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			got, note := ProposePlanStages(context.Background(), c.judge, "b", nil)
			if len(got) != len(entity.Stages) {
				t.Fatalf("%s: harus jatuh ke %d tahap awal, dapat %d", c.nama, len(entity.Stages), len(got))
			}
			if note == "" {
				t.Fatalf("%s: cadangan harus menjelaskan alasannya", c.nama)
			}
			// Cadangan pun harus lolos aturan yang sama dengan usulan judge.
			if _, err := entity.NormalizePlanStages(got); err != nil {
				t.Fatalf("%s: cadangan tidak sah: %v", c.nama, err)
			}
		})
	}
}

// TestNormalizeStages_TolakYangTidakSah: satu pelanggaran saja menggagalkan
// seluruh usulan — skema separuh sah lebih berbahaya daripada skema bawaan.
func TestNormalizeStages_TolakYangTidakSah(t *testing.T) {
	markdown := string(entity.FormatMarkdown)
	dbml := string(entity.FormatDBML)
	ok2 := []rawStage{
		{Key: "prd", Label: "PRD", OutputFormat: markdown},
		{Key: "erd", Label: "ERD", OutputFormat: dbml, DependsOn: []string{"prd"}},
	}
	cases := []struct {
		nama string
		in   []rawStage
	}{
		{"kurang dari dua", ok2[:1]},
		{"lebih dari dua belas", manyStages(13)},
		{"key karakter terlarang", []rawStage{
			{Key: "prd!", OutputFormat: markdown},
			{Key: "erd", OutputFormat: dbml},
		}},
		{"key mulai angka", []rawStage{
			{Key: "1prd", OutputFormat: markdown},
			{Key: "erd", OutputFormat: dbml},
		}},
		{"key dobel", []rawStage{
			{Key: "prd", OutputFormat: markdown},
			{Key: "prd", OutputFormat: markdown},
		}},
		{"format pdf", []rawStage{
			{Key: "prd", OutputFormat: "pdf"},
			{Key: "erd", OutputFormat: dbml},
		}},
		{"format kosong", []rawStage{
			{Key: "prd", OutputFormat: ""},
			{Key: "erd", OutputFormat: dbml},
		}},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			if _, err := normalizeStages(c.in); err == nil {
				t.Fatalf("%s: harus ditolak", c.nama)
			}
		})
	}

	// Huruf besar DITOLERANSI (model sering membalas "PRD") dan disimpan
	// lowercase; key di DB tetap harus cocok plan_stages.key.
	got, err := normalizeStages([]rawStage{
		{Key: "PRD", OutputFormat: markdown},
		{Key: "ERD", OutputFormat: dbml, DependsOn: []string{"PRD"}},
	})
	if err != nil {
		t.Fatalf("huruf besar harus ditoleransi: %v", err)
	}
	if got[0].Key != entity.StagePRD || got[1].Key != entity.StageERD {
		t.Fatalf("key harus dinormalkan ke huruf kecil: %q %q", got[0].Key, got[1].Key)
	}
	if len(got[1].DependsOn) != 1 || got[1].DependsOn[0] != entity.StagePRD {
		t.Fatalf("depends_on huruf besar harus ikut dinormalkan: %+v", got[1].DependsOn)
	}

	// Yang sah tetap diterima, dan depends_on menunjuk key yang tidak ada dibuang.
	got, err = normalizeStages(append(ok2, rawStage{
		Key: "ui", Label: "", OutputFormat: markdown, DependsOn: []string{"prd", "hantu"},
	}))
	if err != nil {
		t.Fatalf("susunan sah ditolak: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("harus 3 tahap, dapat %d", len(got))
	}
	// Label kosong diisi key; depends_on asing dibuang tanpa menggagalkan.
	if got[2].Label != "ui" {
		t.Fatalf("label kosong harus diisi key: %q", got[2].Label)
	}
	if len(got[2].DependsOn) != 1 || got[2].DependsOn[0] != entity.StagePRD {
		t.Fatalf("depends_on asing harus dibuang: %+v", got[2].DependsOn)
	}
}

func manyStages(n int) []rawStage {
	out := make([]rawStage, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, rawStage{
			Key:          string(rune('a'+i)) + "x",
			OutputFormat: string(entity.FormatMarkdown),
		})
	}
	return out
}

// TestDefaultRawStages_CocokDenganEntity: opsi "keep_default" tidak boleh
// menyimpang dari susunan awal entity, karena keduanya label yang sama.
func TestDefaultRawStages_CocokDenganEntity(t *testing.T) {
	raw := DefaultRawStages()
	base := entity.DefaultPlanStages("")
	if len(raw) != len(base) {
		t.Fatalf("jumlah berbeda: raw %d, entity %d", len(raw), len(base))
	}
	for i := range raw {
		if raw[i].Key != string(base[i].Key) {
			t.Fatalf("key %d berbeda: %q vs %q", i, raw[i].Key, base[i].Key)
		}
		if raw[i].Label != base[i].Label {
			t.Fatalf("label %d berbeda: %q vs %q", i, raw[i].Label, base[i].Label)
		}
		if raw[i].OutputFormat != string(base[i].OutputFormat) {
			t.Fatalf("format %d berbeda: %q vs %q", i, raw[i].OutputFormat, base[i].OutputFormat)
		}
	}
}
