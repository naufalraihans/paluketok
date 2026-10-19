package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// Rekaman request LLM: tes harus bisa memeriksa BENTUK pesan yang keluar,
// bukan cuma jumlahnya. Inilah yang membuktikan bug riwayat benar-benar mati.
type recordingLLM struct {
	got      []port.LLMRequest
	balasan  string
	balasanN int
}

func (r *recordingLLM) Complete(_ context.Context, req port.LLMRequest) (*port.LLMResponse, error) {
	r.got = append(r.got, req)
	r.balasanN++
	if r.balasan != "" {
		return &port.LLMResponse{Content: r.balasan + "TAHAP BERIKUTNYA: -", Model: "uji"}, nil
	}
	return &port.LLMResponse{Content: "balasan uji TAHAP BERIKUTNYA: -", Model: "uji"}, nil
}

// fakeMessages menyimpan pesan dan menyaring per sesi, seperti SQL.
type fakeMessages struct{ all []entity.ProjectMessage }

func (f *fakeMessages) Create(_ context.Context, m *entity.ProjectMessage) error {
	f.all = append(f.all, *m)
	return nil
}

func (f *fakeMessages) ListByProject(_ context.Context, projectID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	var out []entity.ProjectMessage
	for _, m := range f.all {
		if m.ProjectID == projectID {
			out = append(out, m)
		}
	}
	return tail(out, limit), nil
}

func (f *fakeMessages) ListBySession(_ context.Context, sessionID entity.ID, limit int) ([]entity.ProjectMessage, error) {
	var out []entity.ProjectMessage
	for _, m := range f.all {
		if m.SessionID == sessionID {
			out = append(out, m)
		}
	}
	return tail(out, limit), nil
}

func tail(in []entity.ProjectMessage, limit int) []entity.ProjectMessage {
	if limit > 0 && len(in) > limit {
		return in[len(in)-limit:]
	}
	return in
}

type fakeSessions struct{ all []entity.ChatSession }

func (f *fakeSessions) Create(_ context.Context, s *entity.ChatSession) error {
	f.all = append(f.all, *s)
	return nil
}

func (f *fakeSessions) GetByID(_ context.Context, id entity.ID) (*entity.ChatSession, error) {
	for i := range f.all {
		if f.all[i].ID == id {
			cp := f.all[i]
			return &cp, nil
		}
	}
	return nil, port.ErrNotFound
}

func (f *fakeSessions) ListByProject(_ context.Context, projectID entity.ID) ([]entity.ChatSession, error) {
	var out []entity.ChatSession
	for i := len(f.all) - 1; i >= 0; i-- { // terbaru dulu, seperti SQL
		if f.all[i].ProjectID == projectID {
			out = append(out, f.all[i])
		}
	}
	return out, nil
}

func (f *fakeSessions) Delete(_ context.Context, id entity.ID) error {
	kept := f.all[:0]
	for _, s := range f.all {
		if s.ID != id {
			kept = append(kept, s)
		}
	}
	f.all = kept
	return nil
}

func (f *fakeSessions) Touch(_ context.Context, id entity.ID, _ time.Time) error { return nil }

var (
	_ port.MessageRepo     = (*fakeMessages)(nil)
	_ port.ChatSessionRepo = (*fakeSessions)(nil)
)

// resolverTetap mengembalikan proyek yang sudah disiapkan tes, sehingga
// ChatUsecase bisa dites tanpa lapisan proyek.
type resolverTetap struct{ p *entity.Project }

func (r resolverTetap) Resolve(_ context.Context, _ entity.Actor, _ string) (*entity.Project, error) {
	if r.p == nil {
		return nil, ErrNotFound
	}
	return r.p, nil
}

// chatKonteks menyusun Deps minimal untuk ChatUsecase.
func chatKonteks(t *testing.T) (*ChatUsecase, *recordingLLM, *fakeMessages, *fakeSessions, entity.Actor, *entity.Project) {
	t.Helper()
	msgs := &fakeMessages{}
	sess := &fakeSessions{}
	rekam := &recordingLLM{}
	actor := entity.Actor{UserID: "u1"}
	p := &entity.Project{ID: "p1", OwnerID: "u1", Name: "POS Warung", Brief: "kasir warung"}
	u := &ChatUsecase{
		deps: Deps{
			Messages: msgs, Sessions: sess, LLM: rekam,
			Clock: &fakeClock{t: time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)},
			IDs:   &seqIDs{},
		},
		resolve: resolverTetap{p: p},
	}
	return u, rekam, msgs, sess, actor, p
}

// sendInSession mengirim pesan lewat usecase dan mengembalikan request LLM yang
// benar-benar keluar, supaya tes memeriksa bentuk pesannya, bukan hanya hasil.
func sendInSession(t *testing.T, u *ChatUsecase, actor entity.Actor, p *entity.Project, sessionRef, text string, rekam *recordingLLM) port.LLMRequest {
	t.Helper()
	before := len(rekam.got)
	if _, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), sessionRef, text); err != nil {
		t.Fatalf("SendMessageInSession: %v", err)
	}
	if len(rekam.got) != before+1 {
		t.Fatalf("LLM dipanggil %d kali, harusnya sekali", len(rekam.got)-before)
	}
	return rekam.got[len(rekam.got)-1]
}

func TestSesi_RiwayatTidakBercampurAntarSesi(t *testing.T) {
	u, rekam, _, sess, actor, p := chatKonteks(t)

	s1 := &entity.ChatSession{ID: "s1", ProjectID: "p1", UserID: "u1", Judul: "kasir"}
	s2 := &entity.ChatSession{ID: "s2", ProjectID: "p1", UserID: "u1", Judul: "laporan"}
	_ = sess.Create(context.Background(), s1)
	_ = sess.Create(context.Background(), s2)

	// Sesi 2 punya riwayat tentang laporan. Sesi 1 tidak boleh melihatnya.
	msgs := u.deps.Messages
	_ = msgs.Create(context.Background(), &entity.ProjectMessage{
		ID: "m1", ProjectID: "p1", UserID: "u1", SessionID: "s2",
		Role: "user", Content: "bikin laporan laba",
	})
	_ = msgs.Create(context.Background(), &entity.ProjectMessage{
		ID: "m2", ProjectID: "p1", UserID: "u1", SessionID: "s2",
		Role: "assistant", Content: "Laporan: harian, periode, laba",
	})

	// Kirim ke sesi 1 pesan pendek yang menggantung maknanya.
	req := port.LLMRequest{}
	got := sendInSession(t, u, actor, p, "s1", "kamu yakin?", rekam)
	req = got

	// 1. Pesan pertama user tidak boleh berada di daftar pesan LLM.
	for _, m := range req.Messages {
		if strings.Contains(m.Content, "bikin laporan laba") {
			t.Fatalf("riwayat sesi lain bocor ke sesi 1: %q", m.Content)
		}
	}
	// 2. Dua pesan pertama = system + konteks proyek, terakhir = pertanyaan baru.
	if len(req.Messages) < 3 {
		t.Fatalf("pesan LLM cuma %d, harusnya system + konteks + pertanyaan", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Fatalf("pesan pertama harus system, dapat %q", req.Messages[0].Role)
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "kamu yakin?" {
		t.Fatalf("pesan terakhir harus pertanyaan user, dapat %q=%q", last.Role, last.Content)
	}
}

// Riwayat harus dikirim sebagai PESAN ASLI bergantian, bukan transkrip di
// dalam satu pesan. Dengan transkrip, "kamu yakin?" tidak punya rujukan dan
// model mengulang jawaban lama.
func TestSesi_RiwayatDikirimSebagaiPesanAsli(t *testing.T) {
	u, rekam, msgs, sess, actor, p := chatKonteks(t)
	_ = sess.Create(context.Background(), &entity.ChatSession{
		ID: "s1", ProjectID: "p1", UserID: "u1", Judul: "kasir",
	})
	_ = msgs.Create(context.Background(), &entity.ProjectMessage{
		ID: "m1", ProjectID: "p1", UserID: "u1", SessionID: "s1",
		Role: "user", Content: "bikin matriks akses",
	})
	_ = msgs.Create(context.Background(), &entity.ProjectMessage{
		ID: "m2", ProjectID: "p1", UserID: "u1", SessionID: "s1",
		Role: "assistant", Content: "Roles: POS Warung",
	})

	req := sendInSession(t, u, actor, p, "s1", "hmmm, kamu yakin?", rekam)

	// Cari pasangan role user/assistant riwayat, terpisah satu pesan satu role.
	var pola []string
	for _, m := range req.Messages {
		pola = append(pola, m.Role)
	}
	want := []string{"system", "user", "user", "assistant", "user"}
	if strings.Join(pola, ",") != strings.Join(want, ",") {
		t.Fatalf("urutan peran %v, harusnya %v", pola, want)
	}
	// Riwayat TIDAK boleh ditempel sebagai teks tag di konteks.
	for _, m := range req.Messages {
		if strings.Contains(m.Content, "<riwayat>") {
			t.Fatal("riwayat masih ditempel sebagai transkrip di dalam pesan")
		}
	}
	// Pesan assistant riwayat harus utuh sebagai isinya sendiri.
	if req.Messages[3].Content != "Roles: POS Warung" {
		t.Fatalf("pesan assistant riwayat salah: %q", req.Messages[3].Content)
	}
}

// Tanpa session_id, sesi terakhir proyek dipakai. Proyek tanpa sesi dibuatkan
// satu, supaya pemanggil lama tetap bekerja.
func TestSesi_KosongPakaiSesiTerakhirDanBuatBilaBelumAda(t *testing.T) {
	u, rekam, _, sess, actor, p := chatKonteks(t)

	sendInSession(t, u, actor, p, "", "pertanyaan pertama", rekam)
	if len(sess.all) != 1 {
		t.Fatalf("sesi dibuat %d, harusnya 1", len(sess.all))
	}
	if sess.all[0].Judul != "pertanyaan pertama" {
		t.Fatalf("judul sesi %q, harusnya dari pesan pertama", sess.all[0].Judul)
	}

	// Panggilan kedua tanpa session_id harus masuk sesi yang sama.
	sendInSession(t, u, actor, p, "", "pertanyaan kedua", rekam)
	if len(sess.all) != 1 {
		t.Fatalf("sesi jadi %d, tidak boleh bertambah", len(sess.all))
	}
	if got := len(rekam.got[1].Messages); got != 5 {
		t.Fatalf("pesan LLM %d, harusnya 5 (system+konteks+2 riwayat+tanya)", got)
	}
}

// Sesi milik proyek lain tidak boleh dibaca walaupun idnya diketahui.
func TestSesi_TidakBisaLintasProyek(t *testing.T) {
	u, rekam, _, sess, actor, p := chatKonteks(t)
	_ = sess.Create(context.Background(), &entity.ChatSession{
		ID: "sAsing", ProjectID: "pLain", UserID: "u9", Judul: "punya orang",
	})
	_, err := u.SendMessageInSession(context.Background(), actor, string(p.ID), "sAsing", "halo")
	if err == nil {
		t.Fatal("harus error: sesi milik proyek lain")
	}
	if len(rekam.got) != 0 {
		t.Fatal("LLM tidak boleh dipanggil untuk sesi proyek lain")
	}
}

// Sesi terakhir tidak bisa dihapus: proyek tanpa sesi meninggalkan layar kosong.
func TestSesi_TerakhirTidakBisaDihapus(t *testing.T) {
	u, _, _, sess, actor, p := chatKonteks(t)
	_ = sess.Create(context.Background(), &entity.ChatSession{
		ID: "s1", ProjectID: "p1", UserID: "u1", Judul: "satu-satunya",
	})
	if err := u.DeleteSession(context.Background(), actor, string(p.ID), "s1"); err == nil {
		t.Fatal("menghapus sesi terakhir harus ditolak")
	}
}

// Judul panjang dipotong di batas kata dan tidak melewati batas kolom.
func TestJudulSesi(t *testing.T) {
	kasus := []struct{ explicit, dariPesan, want string }{
		{"", "", "Sesi baru"},
		{"  Kasir warung  ", "abaikan", "Kasir warung"},
		{"", "baris satu\nbaris dua", "baris satu"},
		{"", strings.Repeat("kata ", 60), strings.Repeat("kata ", 23) + "kata"},
	}
	for _, k := range kasus {
		got := judulSesi(k.explicit, k.dariPesan)
		if got != k.want {
			t.Errorf("judulSesi(%q,%q) = %q, harusnya %q", k.explicit, k.dariPesan, got, k.want)
		}
		if n := len([]rune(got)); n > SessionJudulMax {
			t.Errorf("judul %d rune, melewati batas %d", n, SessionJudulMax)
		}
	}
}
