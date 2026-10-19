package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Perintah garis miring di chat.
//
// Kenapa di sini dan bukan fitur terpisah: chat sudah punya riwayat, sesi, dan
// tempat menampilkan balasan. Perintah hanya mengubah SIAPA yang menjawab,
// bukan alurnya. Menaruhnya di tempat lain memaksa pengguna berpindah layar
// hanya untuk bertanya "menurutmu mana yang lebih baik?".
//
//	Daftar - Nama - Tujuan
//	/judge [opsi, ...] pertanyaan  -> judge.Decide, bukan LLM planner
//	/nama-skill pertanyaan         -> planner dengan instruksi skill disuntik
//
// Tanpa slash = perilaku lama, planner biasa.

// JudgeMaxOpsi batas gateway Decisions: 64 opsi per pertanyaan. Dipotong jauh
// lebih awal supaya opsi yang tidak sengaja ikut terbaca tidak mengirim
// pertanyaan yang tidak mungkin dijawab.
const JudgeMaxOpsi = 12

// hasilPerintah hasil satu perintah garis miring.
//
// Dua bentuk: perintah yang MENJAWAB sendiri (judge, help, daftar skill) mengisi
// `content` dan selesai; perintah yang hanya MENYIAPKAN prompt (skill) mengisi
// `promptTambahan` dan `pertanyaan`, lalu alur lanjut ke planner biasa.
type hasilPerintah struct {
	content string
	model   string

	// promptTambahan disuntik ke system prompt planner. Diisi oleh /nama-skill.
	promptTambahan string
	// pertanyaan adalah sisa teks setelah perintah, dipakai sebagai pertanyaan
	// planner menggantikan teks mentah "/skill apa itu".
	pertanyaan string
}

// jalankanPerintah mengenali perintah di awal teks. Mengembalikan (nil, false)
// bila teks bukan perintah, sehingga pemanggil lanjut ke jalur planner biasa.
func (u *ChatUsecase) jalankanPerintah(ctx context.Context, actor entity.Actor, p *entity.Project, text string) (*hasilPerintah, bool, error) {
	if !strings.HasPrefix(text, "/") {
		return nil, false, nil
	}
	nama, sisa := pisahPerintah(text)
	switch strings.ToLower(nama) {
	case "":
		return nil, false, nil
	case "judge":
		h, err := u.perintahJudge(ctx, p, sisa)
		return h, true, err
	case "skill":
		h, err := u.daftarSkill(ctx)
		return h, true, err
	case "help", "bantuan":
		return &hasilPerintah{content: bantuanPerintah()}, true, nil
	default:
		// Perintah tak dikenal: mungkin nama skill. Skill global, jadi tidak ada
		// pemeriksaan kepemilikan di sini.
		return u.perintahSkill(ctx, nama, sisa)
	}
}

// pisahPerintah memecah "/nama sisa teks" menjadi nama dan sisanya.
// Nama perintah hanya huruf/angka/tanda hubung; apa pun setelah itu jadi sisa.
func pisahPerintah(text string) (string, string) {
	body := strings.TrimPrefix(text, "/")
	i := strings.IndexFunc(body, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-')
	})
	if i < 0 {
		return body, ""
	}
	return body[:i], strings.TrimSpace(body[i:])
}

// perintahJudge mengajukan pertanyaan ke judge Decisions.
//
// Opsi ditulis dalam kurung siku di depan: /judge [gas, tunda] apakah aman?
// Bila tidak ada opsi, dipakai dua pilihan bawaan "ya" dan "tidak", karena
// pertanyaan tanpa pilihan tidak bisa dijawab API bertipe `choice`.
func (u *ChatUsecase) perintahJudge(ctx context.Context, p *entity.Project, sisa string) (*hasilPerintah, error) {
	if u.deps.Judge == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	opsi, pertanyaan := pisahOpsi(sisa)
	if pertanyaan == "" {
		return nil, ErrValidation
	}
	choices := make([]port.JudgeChoice, 0, len(opsi))
	for _, o := range opsi {
		choices = append(choices, port.JudgeChoice{Value: o, Description: "pilihan " + o})
	}

	// Konteks proyek ikut dikirim: pertanyaan "apakah aman?" tentang proyek ini
	// tidak bisa dinilai tanpa tahu proyeknya.
	var b strings.Builder
	b.WriteString("<konteks_proyek>\n")
	b.WriteString(p.Name)
	b.WriteString("\n")
	b.WriteString(p.Brief)
	b.WriteString("\n</konteks_proyek>\n\n")
	b.WriteString(pertanyaan)

	resp, err := u.deps.Judge.Decide(ctx, port.JudgeRequest{
		Input: b.String(),
		Questions: []port.JudgeQuestion{{
			ID:           "pilihan",
			Type:         "choice",
			Instructions: "Pilih satu opsi yang paling tepat untuk pertanyaan pengguna.",
			Choices:      choices,
		}},
	})
	if err != nil {
		return nil, err
	}
	return &hasilPerintah{
		content: renderJawabanJudge(pertanyaan, resp),
		model:   resp.Model,
	}, nil
}

// pisahOpsi membaca "[a, b] sisa" menjadi daftar opsi dan sisa pertanyaan.
// Tanpa tanda kurung, opsi bawaan ya/tidak dipakai.
func pisahOpsi(sisa string) ([]string, string) {
	s := strings.TrimSpace(sisa)
	if !strings.HasPrefix(s, "[") {
		return []string{"ya", "tidak"}, s
	}
	end := strings.Index(s, "]")
	if end < 0 {
		return []string{"ya", "tidak"}, s
	}
	var opsi []string
	for _, o := range strings.Split(s[1:end], ",") {
		o = strings.TrimSpace(o)
		if o != "" && len(opsi) < JudgeMaxOpsi {
			opsi = append(opsi, o)
		}
	}
	if len(opsi) < 2 {
		return []string{"ya", "tidak"}, strings.TrimSpace(s[end+1:])
	}
	return opsi, strings.TrimSpace(s[end+1:])
}

// renderJawabanJudge membentuk balasan chat dari keputusan judge.
//
// Probabilitas ditampilkan apa adanya. Angka itu alasan sebenarnya kenapa judge
// memilih, dan menyembunyikannya membuat keputusan tipis terlihat sama yakinnya
// dengan keputusan bulat. Baris "yakin" hanya muncul saat confidence rendah,
// supaya tidak jadi hiasan yang selalu ada.
func renderJawabanJudge(pertanyaan string, resp *port.JudgeResponse) string {
	var b strings.Builder
	b.WriteString(pertanyaan)
	b.WriteString("\n\n")
	if resp == nil || len(resp.Answers) == 0 {
		b.WriteString("**Judge tidak mengembalikan jawaban.** Coba lagi atau periksa pesan error.")
		return b.String()
	}
	// Answers berkunci id pertanyaan. Untuk perintah ini hanya ada satu
	// pertanyaan, jadi urutannya tidak penting; ambil satu yang ada.
	var a port.JudgeAnswer
	for _, v := range resp.Answers {
		a = v
		break
	}
	b.WriteString("**Keputusan: ")
	if a.Choice != "" {
		b.WriteString(a.Choice)
	} else {
		b.WriteString("tidak jelas")
	}
	b.WriteString("**\n")
	if len(a.Probabilities) > 0 {
		b.WriteString("\n")
		// Urut dari peluang terbesar supaya pilihan yang dipilih terbaca lebih
		// dulu daripada opsi yang peluangnya hampir nol.
		type pr struct {
			value string
			p     float64
		}
		list := make([]pr, 0, len(a.Probabilities))
		for v, p := range a.Probabilities {
			list = append(list, pr{v, p})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].p > list[j].p })
		for _, it := range list {
			b.WriteString(fmt.Sprintf("- %s: %.2f\n", it.value, it.p))
		}
	}
	b.WriteString(fmt.Sprintf("\nKeyakinan: %.2f", a.Confidence))
	if a.Confidence < 0.5 {
		b.WriteString(" - tipis, layak ditinjau ulang")
	}
	if resp.Model != "" {
		b.WriteString("\n\n_model: ")
		b.WriteString(resp.Model)
		b.WriteString("_")
	}
	return b.String()
}

// perintahSkill menyuntik instruksi skill ke system prompt lalu memanggil
// planner seperti biasa. Skill global, jadi tidak ada pemeriksaan pemilik.
func (u *ChatUsecase) perintahSkill(ctx context.Context, nama, pertanyaan string) (*hasilPerintah, bool, error) {
	if u.deps.Skills == nil {
		return nil, true, wrap(port.ErrUnavailable)
	}
	s, err := u.deps.Skills.GetByName(ctx, strings.ToLower(nama))
	if err != nil {
		if err == port.ErrNotFound {
			return &hasilPerintah{content: fmt.Sprintf(
				"Perintah atau skill `/%s` tidak dikenal. Ketik `/help` untuk daftar.", nama)}, true, nil
		}
		return nil, true, wrap(err)
	}
	if strings.TrimSpace(pertanyaan) == "" {
		return &hasilPerintah{content: fmt.Sprintf(
			"Skill `/%s` siap (%s). Tulis pertanyaannya setelah perintah, mis. `/%s cek rancangan ini`.",
			s.Nama, s.Ringkasan, s.Nama)}, true, nil
	}
	// Skill tidak menjawab sendiri: instruksinya disuntik ke system prompt, dan
	// pertanyaannya lanjut ke planner. `ya=false` menandai "bukan perintah yang
	// selesai", supaya pemanggil meneruskan ke jalur planner.
	return &hasilPerintah{
		promptTambahan: "Instruksi skill `/" + s.Nama + "` yang harus diikuti:\n\n" + s.Isi,
		pertanyaan:     pertanyaan,
	}, false, nil
}

// daftarSkill mengembalikan daftar skill yang terpasang.
func (u *ChatUsecase) daftarSkill(ctx context.Context) (*hasilPerintah, error) {
	if u.deps.Skills == nil {
		return nil, wrap(port.ErrUnavailable)
	}
	list, err := u.deps.Skills.List(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	var b strings.Builder
	b.WriteString("**Skill tersedia**\n")
	if len(list) == 0 {
		b.WriteString("\nBelum ada skill. Admin bisa menambah di panel admin.")
	} else {
		for _, s := range list {
			b.WriteString("\n- `/" + s.Nama + "`")
			if s.Ringkasan != "" {
				b.WriteString(" - " + s.Ringkasan)
			}
		}
	}
	b.WriteString("\n\n**Perintah**\n")
	b.WriteString("\n- `/judge [opsi, ...] pertanyaan` - minta penilaian judge atas pertanyaan")
	b.WriteString("\n- `/help` - tampilkan bantuan ini")
	return &hasilPerintah{content: b.String()}, nil
}

func bantuanPerintah() string {
	return "**Perintah di chat**\n" +
		"\n- `/judge [opsi, ...] pertanyaan` - judge menilai dan memilih satu opsi" +
		"\n- `/skill` - daftar skill tersedia" +
		"\n- `/nama-skill pertanyaan` - jawab memakai instruksi skill itu" +
		"\n- `/help` - bantuan ini" +
		"\n\nTanpa garis miring, pesan dijawab planner seperti biasa."
}
