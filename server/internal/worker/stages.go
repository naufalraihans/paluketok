// Penyusun skema tahap dinamis (docs/plan/planner.md, tugas "tahap planning
// dinamis").
//
// Sebelum ini jumlah dan nama tahap dipaku di enam lapis kode. Sekarang AI judge
// diminta menyusunnya dari brief + stack proyek, lalu hasilnya diperiksa ketat di
// sini sebelum dipakai.
//
// Kenapa jalur judge, bukan LLM bebas: port.Judge memang bentuknya memilih dari
// daftar opsi yang DISEDIAKAN worker (gate.go). Worker menyediakan beberapa
// kandidat skema, judge memilih satu, worker memeriksa pilihannya. Tidak ada
// keluaran bebas yang perlu ditebak bentuknya.
//
// Aturan yang tidak bisa dilanggar (tugas bagian Fase B poin 3):
//   - jumlah tahap 2..12
//   - key cocok plan_stages.key, unik
//   - output_format WAJIB markdown/json/dbml
//   - depends_on hanya menunjuk key yang ada
//
// Apa pun yang gagal salah satu aturan di atas DITOLAK UTUH dan run memakai
// susunan awal enam tahap. Run tidak boleh gagal gara-gara judge menjawab aneh.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Pilihan gate penyusun skema.
const (
	stagePlanOption = "stage_plan"
	stageKeepOption = "keep_default"
)

// stageJudger kebutuhan penyusun skema atas gate JEV.
type stageJudger interface {
	Decide(ctx context.Context, req port.JudgeRequest) (*port.JudgeResponse, error)
}

// rawStage bentuk satu tahap seperti yang dijawab judge, di dalam satu pilihan
// gate. Dipisah dari entity.PlanStage karena isinya BELUM dipercaya.
type rawStage struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Intent       string   `json:"intent"`
	OutputFormat string   `json:"output_format"`
	CheckSpec    any      `json:"check_spec"`
	DependsOn    []string `json:"depends_on"`
}

// proposalStage adalah pilihan gate yang membawa usulan skema.
type proposalStage struct {
	Reasoning string     `json:"reasoning"`
	Stages    []rawStage `json:"stages"`
}

// buildStageCandidates menyusun daftar opsi untuk gate penyusun skema.
//
// Judge memilih di antara opsi yang DISEDIAKAN worker, jadi kandidat inilah yang
// menentukan mutu keluarannya. Tiga bentuk yang masuk akal:
//   - keep_default: enam tahap bawaan, aman untuk proyek yang tidak jelas
//   - ringkas: empat tahap, untuk proyek kecil
//   - rinci: sembilan tahap, untuk proyek yang butuh dokumen lapis operasional
//
// Masing-masing membawa daftar tahap LENGKAP, bukan cuma jumlahnya, supaya
// jawaban judge bisa langsung dipakai tanpa satu pun keluaran bebas.
func buildStageCandidates() []any {
	keep := DefaultRawStages()
	return []any{
		map[string]any{
			"id":     stageKeepOption,
			"stages": keep,
			"alasan": "enam tahap bawaan: cukup untuk proyek biasa dan sudah punya validator kuat",
		},
		map[string]any{
			"id":     "ringkas",
			"alasan": "empat tahap: proyek kecil, satu alur inti, tanpa dokumen operasional",
			"stages": rawStages(
				r("prd", "PRD", "Rumuskan masalah, pengguna, fitur inti, dan batas proyek.", entity.FormatMarkdown, []string{"Tujuan", "Scope", "Acceptance criteria", "Di luar scope"}, nil),
				r("erd", "ERD", "Susun skema data: tabel, kolom, primary key, dan relasi.", entity.FormatDBML, nil, []string{"prd"}),
				r("openapi", "OpenAPI", "Tetapkan kontrak HTTP: path, method, dan respons tiap operasi.", entity.FormatJSON, nil, []string{"prd", "erd"}),
				r("structure", "Struktur", "Tetapkan struktur folder dan aturan layer sesuai stack.", entity.FormatMarkdown, nil, []string{"prd", "erd", "openapi"}),
			),
		},
		map[string]any{
			"id":     "rinci",
			"alasan": "sembilan tahap: proyek besar, butuh peran akses, proteksi, rencana uji, dan operasional terpisah",
			"stages": rawStages(
				r("prd", "PRD", "Rumuskan masalah, pengguna, fitur inti, dan batas proyek.", entity.FormatMarkdown, []string{"Tujuan", "Scope", "Acceptance criteria", "Di luar scope"}, nil),
				r("erd", "ERD", "Susun skema data: tabel, kolom, primary key, dan relasi.", entity.FormatDBML, nil, []string{"prd"}),
				r("openapi", "OpenAPI", "Tetapkan kontrak HTTP: path, method, dan respons tiap operasi.", entity.FormatJSON, nil, []string{"prd", "erd"}),
				r("roles", "Role access", "Petakan peran dan hak akses untuk setiap path kontrak.", entity.FormatMarkdown, nil, []string{"prd", "openapi"}),
				r("security", "Proteksi", "Tetapkan trust boundary, validasi input, dan checklist proteksi.", entity.FormatMarkdown, nil, []string{"openapi", "erd", "roles"}),
				r("structure", "Struktur", "Tetapkan struktur folder dan aturan layer sesuai stack.", entity.FormatMarkdown, nil, []string{"prd", "erd", "openapi"}),
				r("contract-tests", "Skenario uji", "Turunkan skenario uji dari kriteria sukses dan kontrak HTTP.", entity.FormatMarkdown, nil, []string{"prd", "openapi"}),
				r("operations", "Operasional", "Tetapkan cara menjalankan, memantau, dan memulihkan layanan.", entity.FormatMarkdown, nil, []string{"structure", "security"}),
				r("rollout", "Peluncuran", "Tetapkan urutan peluncuran, kriteria siap, dan rencana mundur.", entity.FormatMarkdown, nil, []string{"operations", "contract-tests"}),
			),
		},
	}
}

// r membuat satu rawStage ringkas; deps nil artinya tanpa ketergantungan.
func r(key, label, intent string, f entity.OutputFormat, sections []string, deps []string) rawStage {
	st := rawStage{Key: key, Label: label, Intent: intent, OutputFormat: string(f), DependsOn: deps}
	if len(sections) > 0 {
		st.CheckSpec = map[string]any{"wajib_seksi": sections}
	}
	return st
}

func rawStages(st ...rawStage) []rawStage { return st }

// DefaultRawStages susunan awal enam tahap dalam bentuk raw, dipakai sebagai
// opsi "keep_default" dan sebagai cadangan saat judge tidak bisa dipakai.
// Sumbernya entity.DefaultPlanStages supaya label dan format tidak ditulis dua
// kali di dua tempat yang bisa berbeda.
func DefaultRawStages() []rawStage {
	base := entity.DefaultPlanStages("")
	out := make([]rawStage, 0, len(base))
	for _, s := range base {
		st := rawStage{
			Key:          string(s.Key),
			Label:        s.Label,
			Intent:       s.Intent,
			OutputFormat: string(s.OutputFormat),
		}
		if len(s.CheckSpec) > 0 {
			st.CheckSpec = json.RawMessage(s.CheckSpec)
		}
		for _, d := range s.DependsOn {
			st.DependsOn = append(st.DependsOn, string(d))
		}
		out = append(out, st)
	}
	return out
}

// defaultFormat format output tahap bawaan, diturunkan dari entity supaya hanya
// ada satu daftar. Dipakai validatePlanStage untuk memilih validator lama.
var defaultFormat = func() map[entity.Stage]entity.OutputFormat {
	m := make(map[entity.Stage]entity.OutputFormat, len(entity.Stages))
	for _, s := range entity.DefaultPlanStages("") {
		m[s.Key] = s.OutputFormat
	}
	return m
}()

// ProposePlanStages meminta judge memilih satu kandidat skema tahap.
//
// Mengembalikan skema yang SUDAH diperiksa ketat. Bila judge tidak bisa dipakai
// (error, key kosong, jawaban tak dikenal) atau usulannya melanggar aturan,
// hasilnya susunan awal enam tahap dan alasan penolakan dikembalikan sebagai
// catatan, bukan sebagai error: run tetap jalan.
func ProposePlanStages(ctx context.Context, judge stageJudger, brief string, stack map[string]string) ([]entity.PlanStage, string) {
	fallback := func(why string) ([]entity.PlanStage, string) {
		return entity.DefaultPlanStages(""), why
	}
	if judge == nil {
		return fallback("judge tidak tersedia, pakai susunan awal")
	}

	candidates := buildStageCandidates()
	choices := make([]port.JudgeChoice, 0, len(candidates))
	for _, c := range candidates {
		m, _ := c.(map[string]any)
		id, _ := m["id"].(string)
		alasan, _ := m["alasan"].(string)
		choices = append(choices, port.JudgeChoice{Value: id, Description: alasan})
	}
	req := port.JudgeRequest{
		Input: map[string]any{
			"tugas": "Pilih susunan tahap perancangan yang paling pas untuk proyek ini.",
			"brief": brief,
			"stack": stack,
			"aturan": []string{
				"jumlah tahap 2 sampai 12",
				"key huruf kecil, boleh angka, garis bawah, atau tanda hubung",
				"output_format wajib salah satu: markdown, json, dbml",
				"jangan usulkan pdf, docx, gambar, atau format lain",
				"depends_on hanya boleh menunjuk key yang ada di susunan itu",
			},
			"options": candidates,
		},
		Questions: []port.JudgeQuestion{{
			ID:   "q1",
			Type: "choice",
			Instructions: "Pilih satu susunan tahap dari options. Balas value pilihannya (" +
				stageKeepOption + ", ringkas, atau rinci).",
			Choices: choices,
		}},
	}

	resp, err := judge.Decide(ctx, req)
	if err != nil {
		return fallback("judge gagal dipanggil, pakai susunan awal")
	}
	ans, ok := resp.Answers["q1"]
	if !ok {
		return fallback("judge tidak menjawab, pakai susunan awal")
	}

	raw, ok := pickCandidate(candidates, ans.Choice)
	if !ok {
		return fallback(fmt.Sprintf("pilihan judge %q tidak dikenal, pakai susunan awal", ans.Choice))
	}
	stages, err := normalizeStages(raw)
	if err != nil {
		return fallback("usulan judge ditolak: " + err.Error())
	}
	return stages, ""
}

// pickCandidate mencari kandidat dengan id tertentu.
func pickCandidate(candidates []any, id string) ([]rawStage, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, c := range candidates {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := m["id"].(string); s != id {
			continue
		}
		rs, ok := m["stages"].([]rawStage)
		return rs, ok
	}
	return nil, false
}

// normalizeStages memeriksa usulan mentah dan mengubahnya jadi []entity.PlanStage.
//
// Mengembalikan error begitu satu saja aturan dilanggar: skema yang sebagian
// sah lebih berbahaya daripada skema bawaan, karena UI akan menampilkan tahap
// yang validatornya tidak jelas.
func normalizeStages(raw []rawStage) ([]entity.PlanStage, error) {
	if len(raw) < entity.MinStages || len(raw) > entity.MaxStages {
		return nil, fmt.Errorf("jumlah tahap %d di luar batas %d..%d",
			len(raw), entity.MinStages, entity.MaxStages)
	}

	seen := make(map[entity.Stage]bool, len(raw))
	out := make([]entity.PlanStage, 0, len(raw))
	for i, rs := range raw {
		key := entity.Stage(strings.TrimSpace(strings.ToLower(rs.Key)))
		if !entity.ValidStageKey(key) {
			return nil, fmt.Errorf("key tahap %q tidak sah", rs.Key)
		}
		if seen[key] {
			return nil, fmt.Errorf("key tahap %q dipakai dua kali", key)
		}
		seen[key] = true

		f := entity.OutputFormat(strings.TrimSpace(strings.ToLower(rs.OutputFormat)))
		if !entity.ValidOutputFormat(f) {
			return nil, fmt.Errorf("format %q tahap %s tidak dikenal", rs.OutputFormat, key)
		}

		label := strings.TrimSpace(rs.Label)
		if label == "" {
			label = string(key)
		}
		intent := strings.TrimSpace(rs.Intent)
		if intent == "" {
			intent = "Susun artefak " + label + " untuk proyek ini."
		}

		spec, err := encodeCheckSpec(rs.CheckSpec)
		if err != nil {
			return nil, fmt.Errorf("check_spec tahap %s tidak sah: %w", key, err)
		}

		out = append(out, entity.PlanStage{
			Key:          key,
			Label:        label,
			Position:     i,
			Intent:       intent,
			OutputFormat: f,
			CheckSpec:    spec,
			DependsOn:    normalizeDeps(rs.DependsOn, seen),
		})
	}

	// depends_on diperiksa setelah semua key terkumpul: rujukan ke tahap yang
	// datang belakangan tetap sah, rujukan ke key yang tidak ada tidak.
	keys := make(map[entity.Stage]bool, len(out))
	for _, s := range out {
		keys[s.Key] = true
	}
	for i := range out {
		var kept []entity.Stage
		for _, d := range out[i].DependsOn {
			if !keys[d] {
				return nil, fmt.Errorf("tahap %s bergantung pada %q yang tidak ada", out[i].Key, d)
			}
			kept = append(kept, d)
		}
		out[i].DependsOn = kept
	}
	return out, nil
}

// normalizeDeps membuang ketergantungan kosong, duplikat, dan rujukan ke diri
// sendiri. Bentuk akhirnya diperiksa lagi setelah semua key diketahui.
func normalizeDeps(deps []string, seen map[entity.Stage]bool) []entity.Stage {
	var out []entity.Stage
	for _, d := range deps {
		k := entity.Stage(strings.TrimSpace(strings.ToLower(d)))
		if k == "" || !entity.ValidStageKey(k) {
			continue
		}
		if seen[k] && !containsStage(out, k) {
			out = append(out, k)
		}
	}
	return out
}

func containsStage(list []entity.Stage, k entity.Stage) bool {
	for _, s := range list {
		if s == k {
			return true
		}
	}
	return false
}

// encodeCheckSpec menyimpan check_spec sebagai jsonb. Nil -> NULL.
func encodeCheckSpec(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" || string(b) == "{}" {
		return nil, nil
	}
	return b, nil
}
