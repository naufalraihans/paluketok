package usecase

import (
	"context"
	"strings"
	"testing"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// fakeJudge merekam request dan mengembalikan jawaban tetap.
type fakeJudge struct {
	got     []port.JudgeRequest
	answers map[string]port.JudgeAnswer
}

func (f *fakeJudge) Decide(_ context.Context, req port.JudgeRequest) (*port.JudgeResponse, error) {
	f.got = append(f.got, req)
	if f.answers != nil {
		return &port.JudgeResponse{Model: "gpt-6-luna-decisions", Answers: f.answers}, nil
	}
	return &port.JudgeResponse{
		Model: "gpt-6-luna-decisions",
		Answers: map[string]port.JudgeAnswer{"pilihan": {
			QuestionID: "pilihan", Choice: "gas", Confidence: 0.82,
			Probabilities: map[string]float64{"gas": 0.82, "tunda": 0.18},
		}},
	}, nil
}

// fakeSkills menyimpan skill di memori, urut nama seperti SQL.
type fakeSkills struct{ all []entity.Skill }

func (f *fakeSkills) Create(_ context.Context, s *entity.Skill) error {
	f.all = append(f.all, *s)
	return nil
}

func (f *fakeSkills) List(_ context.Context) ([]entity.Skill, error) { return f.all, nil }

func (f *fakeSkills) GetByName(_ context.Context, nama string) (*entity.Skill, error) {
	for i := range f.all {
		if f.all[i].Nama == nama {
			cp := f.all[i]
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (f *fakeSkills) Update(_ context.Context, s *entity.Skill) error { return nil }
func (f *fakeSkills) Delete(_ context.Context, _ entity.ID) error     { return nil }

var (
	_ port.Judge     = (*fakeJudge)(nil)
	_ port.SkillRepo = (*fakeSkills)(nil)
)

// chatDenganPerintah menyiapkan usecase lengkap dengan judge dan skill.
func chatDenganPerintah(t *testing.T) (*ChatUsecase, *fakeJudge, *fakeSkills, entity.Actor, *entity.Project) {
	t.Helper()
	j := &fakeJudge{}
	sk := &fakeSkills{}
	u, rekam, _, _, actor, p := chatKonteks(t)
	u.deps.Judge = j
	u.deps.Skills = sk
	_ = rekam
	return u, j, sk, actor, p
}

// Pertanyaan dan opsi harus sampai utuh ke judge, dan balasannya memuat
// keputusan serta probabilitas.
func TestPerintahJudge_PertanyaanDanOpsiSampai(t *testing.T) {
	u, j, _, actor, p := chatDenganPerintah(t)
	balasan, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "",
		"/judge [gas, tunda] apakah migrasi ini aman?")
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if len(j.got) != 1 {
		t.Fatalf("judge dipanggil %d kali, harusnya 1", len(j.got))
	}
	in, _ := j.got[0].Input.(string)
	if !strings.Contains(in, "apakah migrasi ini aman?") {
		t.Fatalf("pertanyaan tidak sampai ke judge: %q", in)
	}
	if !strings.Contains(in, "POS Warung") {
		t.Fatalf("konteks proyek tidak ikut: %q", in)
	}
	q := j.got[0].Questions[0]
	if q.Type != "choice" || len(q.Choices) != 2 {
		t.Fatalf("pertanyaan judge salah bentuk: %+v", q)
	}
	if q.Choices[0].Value != "gas" || q.Choices[1].Value != "tunda" {
		t.Fatalf("opsi tidak terbaca: %+v", q.Choices)
	}
	if !strings.Contains(balasan.Content, "gas") || !strings.Contains(balasan.Content, "0.82") {
		t.Fatalf("balasan tidak memuat keputusan/probabilitas: %q", balasan.Content)
	}
}

// Tanpa opsi eksplisit, pilihan bawaan ya/tidak dipakai: pertanyaan tipe choice
// tanpa choices akan ditolak provider.
func TestPerintahJudge_TanpaOpsiPakaiYaTidak(t *testing.T) {
	u, j, _, actor, p := chatDenganPerintah(t)
	if _, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "",
		"/judge apakah ini perlu?"); err != nil {
		t.Fatalf("kirim: %v", err)
	}
	q := j.got[0].Questions[0]
	if len(q.Choices) != 2 || q.Choices[0].Value != "ya" {
		t.Fatalf("opsi bawaan salah: %+v", q.Choices)
	}
}

// Confidence rendah harus terbaca sebagai peringatan, bukan disembunyikan.
func TestPerintahJudge_ConfidenceRendahDiperingatkan(t *testing.T) {
	u, j, _, actor, p := chatDenganPerintah(t)
	j.answers = map[string]port.JudgeAnswer{"pilihan": {
		QuestionID: "pilihan", Choice: "tunda", Confidence: 0.31,
		Probabilities: map[string]float64{"gas": 0.48, "tunda": 0.51},
	}}
	balasan, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "",
		"/judge [gas, tunda] lanjut?")
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if !strings.Contains(balasan.Content, "tipis") {
		t.Fatalf("peringatan confidence rendah tidak muncul: %q", balasan.Content)
	}
	// Probabilitas terbesar harus lebih dulu dari yang kecil.
	if strings.Index(balasan.Content, "tunda: 0.51") > strings.Index(balasan.Content, "gas: 0.48") {
		t.Fatalf("probabilitas tidak urut menurun: %q", balasan.Content)
	}
}

// Pertanyaan kosong = validasi gagal, judge TIDAK dipanggil (panggilan judge
// berbiaya, jadi tidak boleh terjadi karena input salah).
func TestPerintahJudge_TanpaPertanyaanTidakMemanggilJudge(t *testing.T) {
	u, j, _, actor, p := chatDenganPerintah(t)
	if _, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "", "/judge [a, b]"); err == nil {
		t.Fatal("pertanyaan kosong harus ditolak")
	}
	if len(j.got) != 0 {
		t.Fatalf("judge dipanggil %d kali padahal input tidak valid", len(j.got))
	}
}

// /nama-skill menyuntik instruksi ke SYSTEM prompt, dan pertanyaannya bersih
// tanpa teks perintah.
func TestPerintahSkill_InstruksiMasukSystemPrompt(t *testing.T) {
	u, _, sk, actor, p := chatDenganPerintah(t)
	_ = sk.Create(context.Background(), &entity.Skill{
		ID: "sk1", Nama: "cek-keamanan", Ringkasan: "periksa ancaman",
		Isi: "Selalu sebut risiko otorisasi lebih dulu.",
	})
	req := sendInSession(t, u, actor, p, "", "/cek-keamanan tinjau rancangan ini", u.deps.LLM.(*recordingLLM))

	if !strings.Contains(req.Messages[0].Content, "risiko otorisasi") {
		t.Fatalf("instruksi skill tidak masuk system prompt: %q", req.Messages[0].Content)
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Content != "tinjau rancangan ini" {
		t.Fatalf("pertanyaan masih memuat teks perintah: %q", last.Content)
	}
}

// Skill tak dikenal dijawab penjelasan, tidak diteruskan ke planner.
func TestPerintahSkill_NamaTidakDikenal(t *testing.T) {
	u, _, _, actor, p := chatDenganPerintah(t)
	balasan, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "",
		"/skill-yang-tidak-ada halo")
	if err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if !strings.Contains(balasan.Content, "tidak dikenal") {
		t.Fatalf("balasan tidak menjelaskan: %q", balasan.Content)
	}
}

// Teks tanpa garis miring tetap lewat planner, tidak tersentuh perintah.
func TestPerintah_BukanPerintahLewatPlanner(t *testing.T) {
	u, j, _, actor, p := chatDenganPerintah(t)
	rekam := u.deps.LLM.(*recordingLLM)
	before := len(rekam.got)
	if _, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "",
		"hmmm, kamu yakin?"); err != nil {
		t.Fatalf("kirim: %v", err)
	}
	if len(rekam.got) != before+1 {
		t.Fatal("planner tidak dipanggil untuk teks biasa")
	}
	if len(j.got) != 0 {
		t.Fatal("judge dipanggil untuk teks biasa")
	}
}

func TestPisahOpsi(t *testing.T) {
	kasus := []struct {
		in       string
		wantOpsi []string
		wantSisa string
	}{
		{"[a, b] tanya", []string{"a", "b"}, "tanya"},
		{"tanya polos", []string{"ya", "tidak"}, "tanya polos"},
		{"[a] tanya", []string{"ya", "tidak"}, "tanya"}, // 1 opsi tidak cukup
		{"[a, b, c, d] p", []string{"a", "b", "c", "d"}, "p"},
		// Kurung buka tanpa tutup: seluruh teks dianggap pertanyaan, opsi
		// kembali ke bawaan supaya tidak ada pertanyaan yang hilang.
		{"[a, b tanya tanpa tutup", []string{"ya", "tidak"}, "[a, b tanya tanpa tutup"},
	}
	for _, k := range kasus {
		opsi, sisa := pisahOpsi(k.in)
		if strings.Join(opsi, ",") != strings.Join(k.wantOpsi, ",") || sisa != k.wantSisa {
			t.Errorf("pisahOpsi(%q) = %v/%q, harusnya %v/%q", k.in, opsi, sisa, k.wantOpsi, k.wantSisa)
		}
	}
}

func TestPisahPerintah(t *testing.T) {
	kasus := []struct{ in, nama, sisa string }{
		{"/judge [a, b] tanya?", "judge", "[a, b] tanya?"},
		{"/cek-keamanan halo dunia", "cek-keamanan", "halo dunia"},
		{"/help", "help", ""},
		{"/", "", ""},
	}
	for _, k := range kasus {
		nama, sisa := pisahPerintah(k.in)
		if nama != k.nama || sisa != k.sisa {
			t.Errorf("pisahPerintah(%q) = %q/%q, harusnya %q/%q", k.in, nama, sisa, k.nama, k.sisa)
		}
	}
}
