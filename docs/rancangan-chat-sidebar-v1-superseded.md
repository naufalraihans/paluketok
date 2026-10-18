# Rancangan: Chat + Sidebar Bertahap

Dokumen rancangan, **bukan kode**. Belum ada satu baris pun yang ditulis.
Untuk ditinjau Naufal sebelum diputuskan dikerjakan atau tidak.

Disusun 16 Oktober 2026, dari pembacaan kode langsung (`server/`, `web/src/`, `docs/plan/`).

---

## Ringkasan: yang kamu minta itu DUA hal, bukan satu

Kamu bilang: *"kita ngeprompt, AI kasih tau apa yang dibutuhkan, sidebar muncul bertahap."*

Itu terdengar seperti satu fitur. Setelah membaca kode, ternyata **dua hal terpisah** dengan
jalur teknis berbeda:

| # | Yang kamu mau | Status sekarang | Ukuran kerja |
|---|---------------|-----------------|--------------|
| A | Ada tempat ngeprompt + AI membalas | Kontrak **sudah ada** (`POST /v1/ai/chat`), kode **nol** | Sedang |
| B | Sidebar muncul bertahap saat AI memutuskan | Data **sudah ada** (`artifacts`), tampilan **statis** | Kecil |

Selain itu ada **satu penghalang** yang harus dibereskan lebih dulu:

| # | Penghalang | Akibat sekarang |
|---|------------|-----------------|
| C | Slot LLM `planner` belum diisi | Tombol `Mulai generate` **pasti gagal** |

---

## C. Penghalang: LLM belum diisi (kerjakan ini dulu)

**Fakta dari kode:**

```
.env VPS (13 variabel, sudah dicek):
  JEV_API_KEY ✓   MIDTRANS_* ✓   SUPABASE_* ✓
  PLANNER_LLM_BASE_URL  ✗ tidak ada
  PLANNER_LLM_MODEL     ✗ tidak ada
```

`server/cmd/web/llm_resolve.go:35` jatuh ke env itu kalau slot `ai_settings` kosong. Dua-duanya
kosong, jadi:

```
slot "planner" belum diisi admin (settings/ai)   ← pesan error aslinya
```

**Akibat:** kamu tekan `Mulai generate (1 kredit)` → run gagal → kredit refund otomatis
(mekanismenya sudah benar, `FailRun` + `RefundRun`). Jadi uangmu tidak hilang, tapi
tidak ada dokumen yang dihasilkan.

**Yang perlu dilakukan:** isi slot `planner` lewat panel admin — `https://admin.paluketok.web.id/admin`
→ tab `AI`. Butuh: base URL, nama model, API key (OpenAI-compatible).

**Ini bukan bug.** Ini konfigurasi yang belum diisi. Dan **chat pun mustahil** sebelum ini
beres, karena chat memakai slot yang sama.

---

## A. Chat: kontraknya sudah beku, kodenya belum ada

### Fakta

`docs/plan/openapi.md:44` sudah menetapkan **`POST /v1/ai/chat`**, dan `docs/plan/gaps.md` mencatat:

```
| Chatbot | gratis, tidak memotong kredit (JEV: free 0.94) | openapi.md |
| POST /v1/ai/chat nol implementasi | Master | Kontrak ada di openapi.md:23,44,
  kode tak dibangun di F0-F7. Fitur baru = butuh fase baru pasca-beku |
```

Kontraknya sudah diputuskan lewat JEV (confidence 0.94). **Tidak perlu dirancang ulang.**

### Kontrak yang sudah ada (jangan diubah)

```
POST /v1/ai/chat
Request:  { "messages": [ { "role": "user", "content": "..." } ] }
Response: { "reply": "...", "model": "...", "usage": { "input_tokens": n, "output_tokens": m } }

Aturan:
  messages  maks 20 item
  content   maks 4.000 karakter per item
  total     maks 16.000 karakter
  role      hanya "user" atau "assistant" — "system" dari klien DITOLAK 422
  tanpa streaming
Error: 422 validation_error · 429 rate_limited · 503 unavailable
```

### Penting: kontrak ini TIDAK terikat proyek

Payload-nya cuma `messages[]` — **tidak ada `project_id`**. Artinya chatbot ini dirancang
sebagai **tanya-jawab umum**, bukan asisten yang merancang proyekmu.

Kalau kamu mau chat yang **tahu konteks proyek** ("buatkan PRD untuk aplikasi kasir"),
itu **kontrak berbeda** dan butuh keputusan baru. Lihat bagian "Keputusan yang kamu ambil".

### Yang perlu dibangun

| Bagian | Perubahan | Berkas |
|--------|-----------|--------|
| Backend | Handler `POST /v1/ai/chat` + usecase | `httpapi/`, `usecase/` baru |
| Backend | Tabel riwayat pesan | migrasi `0005_*.sql` |
| Backend | Rate limit (kontrak: `429`) | sudah ada polanya di `middleware.go` |
| Frontend | Kotak chat: input + daftar balasan | komponen baru |
| Frontend | Panggil `/v1/ai/chat`, tampilkan `reply` | `lib/api.ts` + komponen |

**Slot LLM yang dipakai:** `planner` (satu-satunya yang cocok). Karena chat gratis,
biaya token ditanggung key admin — itu sebabnya kontraknya membatasi `messages` dan
sudah punya rate limit.

---

## B. Sidebar bertahap: datanya SUDAH ADA, tinggal dipakai

Ini bagian yang paling murah dan hasilnya paling terasa.

### Fakta: datanya sudah tersedia

`web/src/routes/projects/[id]/+page.svelte` **sudah** menghitung mana artefak yang ada:

```js
const STAGES = ['prd','erd','openapi','roles','security','structure'];

tiles = STAGES.map((s) => {
    const a = artifacts.find((x) => x.kind === s);
    return { label: ..., ok: Boolean(a), note: a ? `v${a.version} · siap` : 'Belum ada artefak', ... };
});
```

Jadi frontend **sudah tahu** `prd` ada atau belum. Tapi sidebar tidak memakainya:

```svelte
<!-- SummarySidebar.svelte — sekarang -->
{#each summaryNav as item}          ← daftar dari mock/summary.ts, statis 7 item
    <a href="#tahap">               ← SEMUA item href-nya sama
```

**Semua 7 item menunjuk `#tahap`.** Itu sebabnya dipencet terasa tidak terjadi apa-apa.
Bukan rusak — memang tidak punya tujuan.

### Perubahan yang diusulkan

Sidebar menerima daftar `tiles` yang sudah dihitung, lalu:

| Kondisi | Sidebar menampilkan |
|---------|---------------------|
| Belum ada generate | Hanya `Ringkasan`. Menu tahap **belum muncul** |
| Run berjalan | Tahap yang sedang jalan muncul, ditandai "berjalan…" |
| Ada artefak `prd` | Menu `Dokumen · PRD` **muncul** dan bisa diklik |
| Ada artefak `erd` | Menu `Model data · ERD` muncul |
| Seterusnya | Muncul satu per satu mengikuti artefak yang jadi |

Ini persis yang kamu gambarkan: *"saat AI sudah memberikan saran PRD, ERD, baru menu itu muncul."*

**Pemetaan tahap → menu sidebar:**

| Tahap | Label menu | Sumber data |
|-------|-----------|-------------|
| `prd` | Dokumen · PRD | `artifacts.kind = 'prd'` |
| `erd` | Model data · ERD | `artifacts.kind = 'erd'` |
| `openapi` | Kontrak OpenAPI | `artifacts.kind = 'openapi'` |
| `roles` | Role access | `artifacts.kind = 'roles'` |
| `security` | Proteksi | `artifacts.kind = 'security'` |
| `structure` | Struktur & prototipe | `artifacts.kind = 'structure'` |

**Menu yang tak punya pasangan** (`Aktivitas proyek`, `Referensi & aset`) tidak punya data
pendukung di backend — keduanya usulan untuk ditandai "segera" atau dibuang.

### Ukuran kerja

Kecil. Tidak ada endpoint baru, tidak ada migrasi, tidak ada perubahan backend.
Mengubah `SummarySidebar.svelte` supaya menerima prop `tiles`, plus `+page.svelte`
mengopernya. Datanya sudah ada di variabel yang sudah dihitung.

**Risiko: rendah.** Kalau salah, yang terlihat cuma menu muncul/enggak muncul — bukan
uang, bukan data.

---

## Urutan pengerjaan yang kusarankan

| # | Langkah | Kenapa urutannya di sini |
|---|---------|--------------------------|
| 1 | **Isi slot `planner`** (kamu, lewat panel admin) | Tanpa ini generate & chat dua-duanya mati |
| 2 | **Sidebar bertahap** (B) | Murah, risikonya kecil, hasilnya langsung terlihat |
| 3 | **Uji generate beneran** satu proyek | Membuktikan 6 tahap jalan + kredit terpotong benar |
| 4 | **Chat umum** (A) | Kontraknya sudah ada; kerjakan setelah jalur LLM terbukti |
| 5 | **Putuskan: chat konteks proyek?** | Ini keputusan baru, lihat bawah |

Alasan (4) tidak di depan: kalau LLM belum terbukti jalan, menulis UI chat cuma menambah
lapisan di atas sesuatu yang belum teruji. Jalur LLM dulu, UI belakangan.

---

## Keputusan yang kamu ambil

Tiga hal yang **tidak bisa aku putuskan sendiri**, semua butuh pilihanmu:

**1. Chat umum, atau chat yang tahu konteks proyek?**

| Pilihan | Artinya | Biaya |
|---------|---------|-------|
| Kontrak beku apa adanya | Chatbot tanya-jawab umum. Tanya "apa itu ERD" boleh; "buatkan PRD untuk aplikasi kasir" tidak nyambung ke proyekmu | Kontrak sudah ada, kerja lebih kecil |
| Chat terikat proyek | Chat tahu brief + artefak proyek, bisa jadi tempat ngeprompt perancangan seperti yang kamu gambarkan | Ubah kontrak beku = keputusan baru + JEV baru |

**Yang kamu gambarkan tadi lebih dekat ke pilihan kedua.** Tapi pilihan kedua membuang
keputusan yang sudah dibekukan, jadi harus kamu sadari dan setujui.

**2. Menu tanpa data (`Aktivitas proyek`, `Referensi & aset`) diapakan?**

- Dijadikan "segera" dan tak bisa dipencet, atau
- Dibuang sama sekali

**3. Menu di `/projects` (8 item, semua menunjuk `/projects`) diapakan?**

Sama pilihannya: tandai "segera", atau buang.

---

## Yang rancangan ini TIDAK lakukan

Supaya jelas batasnya:

- **Tidak menambah pipeline tahap baru.** Enam tahap tetap `prd → erd → openapi → roles → security → structure`. Kamu menyebut "AI memutuskan sesuatu" — pada rancangan ini, yang "diputuskan" adalah artefak yang sudah jadi, bukan tahap baru.
- **Tidak menyentuh uang.** Kredit tetap 1 per run, refund otomatis saat gagal, idempoten.
- **Tidak mengubah kontrak yang sudah beku** di `docs/plan/openapi.md`, kecuali kamu pilih opsi chat-terikat-proyek di keputusan 1.
- **Tidak mengerjakan apa pun** sebelum kamu baca dan setujui dokumen ini.

---

## Catatan JEV

Rancangan ini lewat gate JEV dua kali:

| Panggilan | Gate | Conf | Keputusan |
|-----------|------|------|-----------|
| 1 | `GATHER_EVIDENCE` | 0.80 | Di bawah ambang 0.90 — disuruh cari fakta dulu |
| 2 | `GATHER_EVIDENCE` | 0.76 | Masih di bawah — disuruh cari fakta lagi |
| 3 | `EXECUTE` | **0.96** | Lolos. Ikuti kontrak beku apa adanya |

Dua penolakan pertama itu berguna: gate memaksa aku mencari sampai menemukan bahwa
**kontrak chat sudah ada** dan **slot LLM belum diisi** — dua fakta yang mengubah
rancangan sepenuhnya. Tanpa itu, aku akan menulis spesifikasi endpoint baru dari nol
dan melewatkan bahwa `planner` kosong.

Fakta yang dipakai gate: `docs/plan/openapi.md`, `docs/plan/gaps.md`, `docs/plan/planner.md`,
`server/internal/worker/prompt.go`, `server/cmd/web/llm_resolve.go`,
`0001_init.sql`, `SummarySidebar.svelte`, `.env` VPS (nama variabel saja).
