// Chat konteks proyek (Fase 4).
//
// Pengguna bertanya soal proyeknya sendiri; model menjawab dengan konteks
//
//	brief proyek + dokumen yang sudah jadi + riwayat percakapan.
//
// Cakupan SENGAJA hanya perancangan: model membantu menyusun keenam dokumen,
// bukan menulis kode aplikasi. Satu request, satu balasan — tidak ada
// streaming dan tidak ada WebSocket.
//
// Aturan pembatasan mengikuti kontrak lama `POST /v1/ai/chat`
// (plan/openapi.md:60) supaya batasnya satu suara untuk seluruh sistem:
// 20 pesan terakhir, 4.000 karakter per pesan, 16.000 karakter total.
package usecase

import (
	"context"
	"strings"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Batas chat (plan/openapi.md:60, dipakai ulang apa adanya).
const (
	// ChatMaxContent panjang maksimum satu pesan.
	ChatMaxContent = 4000
	// ChatMaxHistory jumlah pesan yang dikirim ke model.
	ChatMaxHistory = 20
	// ChatMaxTotal total karakter brief + dokumen + riwayat.
	ChatMaxTotal = 16000
	// ChatSystemReserve ruang yang disisakan untuk system prompt + instruksi
	// penutup. Yang dikirim ke provider adalah system prompt + konteks, dan
	// hanya konteks yang di-clip: tanpa cadangan ini totalnya melewati
	// ChatMaxTotal.
	ChatSystemReserve = 1500
)

// ChatUsecase melayani percakapan perancangan per proyek.
type ChatUsecase struct {
	deps Deps
	// resolve menegakkan policy owner + terjemah slug. Diisi composition root;
	// nil di usecase yang tidak butuh -> balas 503, bukan panik.
	resolve ProjectResolver
}

// NewChatUsecase merakit usecase chat.
func NewChatUsecase(d Deps) *ChatUsecase { return &ChatUsecase{deps: d} }

// SetResolver menyambungkan resolver proyek (ProjectUsecase).
func (u *ChatUsecase) SetResolver(r ProjectResolver) { u.resolve = r }

// SystemPrompt menyusun instruksi tetap chat. Ditulis di server, TIDAK PERNAH
// dari klien: kontrak lama menolak `role: system` dari luar, dan aturan itu
// dipertahankan di sini.
//
// Nama produk (PaluKetok) SENGAJA tidak disebut: model harus bicara tentang
// proyek PENGGUNA, bukan tentang aplikasi ini. Nama proyek disisipkan sebagai
// satu-satunya identitas yang relevan.
//
// Diekspor agar bisa diuji langsung: yang perlu dijaga adalah ISINYA (tidak
// menyebut produk, memakai nama proyek), dan itu tidak terlihat dari HTTP.
func SystemPrompt(projectName string) string {
	return `Kamu asisten perancangan untuk proyek bernama "` + projectName + `". ` +
		`Bahas HANYA proyek itu; jangan menyebut nama alat atau produk lain selain nama proyek di atas. ` +
		`Tugasmu HANYA planning: membantu menyusun enam dokumen (PRD, ERD, OpenAPI, Role access, Proteksi, Struktur). ` +
		`Kamu tidak menulis kode aplikasi. Balas ringkas dan konkret, dalam bahasa yang dipakai pengguna. ` +
		`Jangan menanyakan ulang hal yang sudah tertulis di <brief> atau di <artifact>; pakai yang sudah ada. ` +
		`Kalau brief masih terlalu tipis, ajukan satu asumsi konkret lalu minta konfirmasi singkat, ` +
		`bukan daftar pertanyaan. ` +
		`Kalau informasi sudah cukup untuk melanjutkan, sebutkan tahap berikutnya secara eksplisit ` +
		`(satu dari: prd, erd, openapi, roles, security, structure) di bagian paling akhir balasanmu, ` +
		`dengan format tepat: TAHAP BERIKUTNYA: <nama tahap>. Kalau belum cukup, tulis TAHAP BERIKUTNYA: -`
}

// SendMessage menyimpan pesan pengguna, memanggil model dengan konteks proyek,
// lalu menyimpan balasan. Chat GRATIS: tidak ada baris credits_ledger.
func (u *ChatUsecase) SendMessage(ctx context.Context, actor entity.Actor, ref, content string) (*entity.ProjectMessage, error) {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(content)
	if text == "" || len([]rune(text)) > ChatMaxContent {
		return nil, ErrValidation
	}

	// Riwayat SEBELUM pesan baru: percakapan sebelumnya dipakai sebagai konteks.
	history, err := u.deps.Messages.ListByProject(ctx, p.ID, ChatMaxHistory)
	if err != nil {
		return nil, wrap(err)
	}

	// Pesan pengguna disimpan lebih dulu. Kalau model gagal, pertanyaannya
	// tidak hilang — pengguna bisa mengirim ulang tanpa mengetik dari awal.
	userMsg := &entity.ProjectMessage{
		ID:        u.deps.IDs.New(),
		ProjectID: p.ID,
		UserID:    actor.UserID,
		Role:      "user",
		Content:   text,
		CreatedAt: u.deps.Clock.Now(),
	}
	if err := u.deps.Messages.Create(ctx, userMsg); err != nil {
		return nil, wrap(err)
	}

	req := port.LLMRequest{
		Slot:       "planner", // slot yang sama dengan penyusun dokumen
		MaxTokens:  1500,
		Temperature: 0.3,
		Messages: []port.LLMMessage{
			{Role: "system", Content: SystemPrompt(p.Name)},
			{Role: "user", Content: u.buildContext(ctx, p, history, text)},
		},
	}
	resp, err := u.deps.LLM.Complete(ctx, req)
	if err != nil {
		// Slot kosong / provider menolak = 503.
		return nil, wrap(port.ErrUnavailable)
	}

	reply := &entity.ProjectMessage{
		ID:             u.deps.IDs.New(),
		ProjectID:      p.ID,
		UserID:         actor.UserID,
		Role:           "assistant",
		Content:        strings.TrimSpace(resp.Content),
		SuggestedStage: parseSuggestedStage(resp.Content),
		Model:          resp.Model,
		TokensIn:       resp.Usage.InputTokens,
		TokensOut:      resp.Usage.OutputTokens,
		CreatedAt:      u.deps.Clock.Now(),
	}
	if reply.Content == "" {
		return nil, wrap(port.ErrUnavailable)
	}
	if err := u.deps.Messages.Create(ctx, reply); err != nil {
		return nil, wrap(err)
	}
	return reply, nil
}

// ListMessages mengembalikan percakapan satu proyek, urut waktu (lama -> baru).
//
// Page dipakai untuk MENELUSURI riwayat panjang; konteks yang dikirim ke model
// tetap dibatasi ChatMaxHistory pesan terakhir (lihat SendMessage), jadi dua
// batas itu tidak saling mengunci.
func (u *ChatUsecase) ListMessages(ctx context.Context, actor entity.Actor, ref string, page port.Page) ([]entity.ProjectMessage, error) {
	p, err := u.owned(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	page = pageOrDefault(page)
	// Repo memotong `limit` pesan TERAKHIR. Offset tidak bisa dipetakan lewat
	// interface itu, jadi jumlah yang diminta = limit + offset lalu offset
	// dibuang dari depan di sini. Menelusuri ke belakang tetap benar, dan
	// tidak perlu menambah method baru ke port.MessageRepo.
	need := page.Limit + page.Offset
	ms, err := u.deps.Messages.ListByProject(ctx, p.ID, need)
	if err != nil {
		return nil, wrap(err)
	}
	if page.Offset > 0 {
		if page.Offset >= len(ms) {
			return nil, nil
		}
		ms = ms[:len(ms)-page.Offset]
	}
	if len(ms) > page.Limit {
		ms = ms[len(ms)-page.Limit:]
	}
	return ms, nil
}

// owned menegakkan policy owner: hanya pemilik (atau admin) yang boleh
// menyentuh percakapan proyek. ref boleh UUID atau slug.
//
// SENGAJA memakai resolveProject milik ProjectUsecase, bukan aturan sendiri:
// satu tempat memutuskan 403-vs-404 dan satu tempat menerjemahkan slug. Aturan
// yang disalin ke sini cepat atau lambat akan menyimpang dari aslinya.
func (u *ChatUsecase) owned(ctx context.Context, actor entity.Actor, ref string) (*entity.Project, error) {
	if u.resolve == nil {
		return nil, ErrUnavailable
	}
	return u.resolve.Resolve(ctx, actor, ref)
}

// buildContext menyusun bahan untuk model: brief proyek, dokumen yang sudah
// ada, dan riwayat percakapan. Semuanya dibungkus tag dan diperlakukan sebagai
// DATA — instruksi di dalamnya diabaikan (aturan sama dengan worker/prompt.go).
func (u *ChatUsecase) buildContext(ctx context.Context, p *entity.Project, history []entity.ProjectMessage, question string) string {
	var b strings.Builder
	b.WriteString("<brief>\n")
	b.WriteString(p.Name)
	b.WriteString("\n")
	b.WriteString(p.Brief)
	b.WriteString("\n</brief>\n")

	if arts, err := u.deps.Artifacts.ListLatestByProject(ctx, p.ID); err == nil {
		for _, a := range arts {
			b.WriteString("<artifact kind=\"")
			b.WriteString(string(a.Kind))
			b.WriteString("\">\n")
			b.WriteString(clip(a.Content, ChatMaxTotal/len(arts)))
			b.WriteString("\n</artifact>\n")
		}
	}
	if len(history) > 0 {
		b.WriteString("<riwayat>\n")
		for _, m := range history {
			b.WriteString(m.Role)
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
		b.WriteString("</riwayat>\n")
	}
	b.WriteString("\nBalas pesan terakhir pengguna.\n")
	b.WriteString("<pesan>\n")
	b.WriteString(question)
	b.WriteString("\n</pesan>")
	// Sisakan ruang untuk system prompt: hard cap yang dihormati provider
	// adalah total SEMUA pesan, bukan cuma konteks ini.
	return clip(b.String(), ChatMaxTotal-ChatSystemReserve)
}

// clip memotong teks agar tidak melewati batas karakter (bukan byte: potongan
// di tengah rune bisa merusak UTF-8).
func clip(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// parseSuggestedStage membaca "TAHAP BERIKUTNYA: <tahap>" dari balasan model.
// Tidak ditemukan / "-" -> kosong, artinya belum ada usul.
func parseSuggestedStage(content string) string {
	const marker = "TAHAP BERIKUTNYA:"
	i := strings.LastIndex(strings.ToUpper(content), marker)
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(content[i+len(marker):])
	// Ambil kata pertama saja; model kadang menambah tanda baca atau kalimat.
	word := strings.ToLower(strings.Fields(rest)[0])
	word = strings.Trim(word, ".,;:*_`\"'")
	if entity.ValidStage(entity.Stage(word)) {
		return word
	}
	return ""
}
