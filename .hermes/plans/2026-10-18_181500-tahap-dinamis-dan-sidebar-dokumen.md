# Rencana: sidebar dokumen, tahap dinamis, dan halaman dokumen

**Tujuan:** Pengguna menentukan sendiri berapa dan tahap apa saja yang disusun.
Sidebar menautkan ke halaman dokumen nyata. Tahap karangan pengguna divalidasi
oleh AI judge, bukan hanya tahap bawaan.

**Arsitektur:** Tiga pekerjaan terpisah, dikerjakan berurutan. Nomor 3 dulu
(bug kecil, langsung terasa), lalu 1 (skema dinamis), terakhir 2 (halaman
dokumen mengikuti skema hasil nomor 1).

**Stack:** Go 1.26 (`server/`), SvelteKit 5 + TS (`web/`), PostgreSQL/Supabase,
worker HTTP, AI judge sebagai penentu tahap.

---

## Temuan awal (dari pembacaan kode, bukan dugaan)

Enam tahap **bukan daftar di UI**. Ia tertanam di enam lapis:

| Tempat | Berkas | Isi |
| --- | --- | --- |
| Tipe + urutan | `server/internal/entity/entity.go:90-108` | `Stage`, 6 konstanta, `Stages`, `TotalStages = 6` |
| Prompt per tahap | `server/internal/worker/prompt.go:16-43` | `charter`, `outFormat`, `inputs` |
| Validator khusus | `server/internal/worker/validate.go` | `validatePRD/ERD/OpenAPI/Roles/Security/Structure` |
| Loop pipeline | `server/internal/worker/pipeline.go:105-131` | `for i, stage := range entity.Stages` |
| Kolom DB | `server/database/migrations/0001_init.sql` | `plan_runs.total`, `progress`, `artifacts.kind`, `decisions.stage` |
| Daftar UI | `web/src/routes/projects/[id]/+page.svelte:50-58` | `STAGES`, `STAGE_LABEL` |

Setiap tahap bawaan punya **aturan validasi yang berbeda** (ERD harus DBML
ber-PK dan `Ref` menunjuk tabel yang ada; OpenAPI harus JSON dengan `responses`
per operasi; roles harus mencakup setiap path OpenAPI). Tahap karangan pengguna
tidak punya validator seperti itu.

**Keputusan kunci:** AI judge yang menentukan nama dan bentuk tahap, lalu
menentukan **cara memeriksa** hasilnya. Kalau tidak, "dinamis" berarti kehilangan
seluruh jaminan mutu yang jadi alasan alat ini ada.

## Bug nomor 3 - sebabnya sudah pasti

`web/src/lib/components/SummarySidebar.svelte:111` memakai
`href="#tahap-{item.key}"`. Anchor `#` menunjuk halaman yang sama, jadi klik
"Dokumen · PRD" hanya menggeser scroll. Halaman `/projects/{slug}/{tahap}`
belum pernah ada.

---

## Urutan pengerjaan

```
Fase A (kecil, 1 sesi)   ->  Bug sidebar + halaman dokumen statis
Fase B (besar, 2-3 sesi) ->  Skema tahap dinamis + judge penentu tahap
Fase C (sedang, 1-2 sesi)->  Halaman dokumen mengikuti skema dinamis
```

Fase A dulu supaya bug yang terasa sehari-hari hilang, dan karena Fase C
menggantikan sebagian Fase A (halaman dokumen akan dibuat ulang mengikuti
skema dinamis), jadi Fase A harus sengaja dibuat tipis.

---

# FASE A - Bug sidebar + halaman dokumen

**Sasaran:** Klik "Dokumen · PRD" membuka halaman dokumen PRD yang berisi
artefak asli. Rutenya siap menerima nama tahap bebas (untuk Fase C).

### Task A1: Rute halaman dokumen

**Berkas:**
- Buat: `web/src/routes/projects/[id]/[stage]/+page.ts`
- Buat: `web/src/routes/projects/[id]/[stage]/+page.svelte`

**Catatan penting:** nama folder `[stage]` (bukan `[slug]`) supaya jelas yang
diharapkan adalah nama tahap. Karena proyek dinamis tidak diprerender, rute
`[stage]` juga tidak boleh diprerender; ikuti pola di
`web/src/routes/projects/[id]/+page.ts`.

**Langkah 1:** Baca `web/src/routes/projects/[id]/+page.ts` untuk meniru cara
`prerender` dan `ssr` disetel.

**Langkah 2:** Buat `+page.ts` dengan pola yang sama.

**Langkah 3:** Buat `+page.svelte`:
- Muat proyek: `getProject(params.id)`
- Muat artefak: `listProjectArtifacts(params.id)`
- Cari artefak dengan `kind === params.stage`
- Ada: tampilkan isinya lewat `mdToHtml` bila markdown, atau `<pre>` bila DBML/JSON
- Tidak ada: tampilkan "Dokumen ini belum disusun" + tautan balik ke ringkasan

**Verifikasi:**
```bash
cd web && npm run check && npm run build
# lalu buka /projects/testing-jembut/prd -> harus tampil isi PRD
```

### Task A2: Sidebar menautkan ke halaman, bukan anchor

**Berkas:**
- Ubah: `web/src/lib/components/SummarySidebar.svelte:108-120`

**Langkah 1:** Tambah prop `slug: string` (slug proyek, dari halaman induk).

**Langkah 2:** Ganti `href="#tahap-{item.key}"` jadi
`href="/projects/{slug}/{item.key}"`.

**Langkah 3:** Kirim `slug` dari `web/src/routes/projects/[id]/+page.svelte`
(nilai `p.slug`, sudah tersedia).

**Verifikasi:** klik tiap menu tahap, pastikan pindah halaman dan isinya benar.

### Task A3: Tautan balik + judul

**Berkas:** `web/src/routes/projects/[id]/[stage]/+page.svelte`

**Langkah 1:** Judul halaman memakai `STAGE_LABEL` yang sama dengan
`+page.svelte`; pindahkan `STAGE_LABEL` ke `web/src/lib/stages.ts` supaya tidak
ada dua sumber kebenaran.

**Langkah 2:** Tautan "kembali ke ringkasan" ke `/projects/{slug}`.

**Verifikasi:** `npm run check` bersih, klik balik berfungsi.

---

# FASE B - Skema tahap dinamis

Ini bagian berat. **Jangan dikerjakan tanpa keputusan pemilik produk** pada
tiga pertanyaan di akhir dokumen ini.

### Task B1: Skema tahap sebagai data, bukan konstanta

**Berkas:**
- Buat: `server/database/migrations/0002_plan_stages.sql`
- Ubah: `server/internal/entity/entity.go`

**Langkah 1:** Tabel baru `plan_stages`:

```sql
create table plan_stages (
  id          uuid primary key default gen_random_uuid(),
  run_id      uuid not null references plan_runs(id) on delete cascade,
  key         text not null,              -- 'prd', 'erd', dst (slug)
  label       text not null,              -- 'PRD', 'Dokumen Hukum'
  position    int  not null,
  intent      text not null,              -- kenapa tahap ini ada
  output_format text not null,            -- 'markdown' | 'dbml' | 'json'
  check_spec  jsonb not null,             -- aturan periksa dari judge
  depends_on  text[] not null default '{}',
  created_at  timestamptz not null default now(),
  unique (run_id, key)
);
```

**Langkah 2:** `entity.PlanStage` sebagai struct; `Stages` tetap ada sebagai
**nilai awal** (bukan satu-satunya kemungkinan).

**Langkah 3:** `TotalStages` diganti: `plan_runs.total` diisi jumlah baris
`plan_stages`, bukan konstanta 6.

**Verifikasi:** migrasi jalan di DB scratch; `SELECT count(*)` sesuai harapan.

### Task B2: Judge menentukan tahap

**Berkas:**
- Buat: `server/internal/worker/stages.go`

**Langkah 1:** Fungsi `susunTahap(ctx, brief, stack) ([]PlanStage, error)` yang
memanggil AI judge dengan pertanyaan: tahap apa saja yang dibutuhkan proyek ini.

**Langkah 2:** Judge mengembalikan, untuk tiap tahap: `key`, `label`, `intent`,
`output_format`, `check_spec` (daftar hal yang wajib ada di keluaran), dan
`depends_on`.

**Langkah 3:** Batas wajar: minimal 2, maksimal 12 tahap. Di luar itu → tolak dan
pakai susunan awal 6 tahap.

**Verifikasi:** jalankan dengan 3 brief berbeda, periksa jumlah dan nama tahap
masuk akal. Uji juga brief kosong dan brief aneh.

### Task B3: Validasi memakai check_spec

**Berkas:**
- Ubah: `server/internal/worker/validate.go`

**Langkah 1:** `validate` menerima `check_spec`. Untuk tahap bawaan, check_spec
diisi setara validator lama, jadi perilakunya tidak berubah.

**Langkah 2:** Tahap karangan: periksa tiap butir `check_spec` (mis. wajib
memuat seksi bernama X, wajib JSON valid). Validator khusus lama tetap dipakai
untuk tahap yang `key`-nya `prd`/`erd`/`openapi`/`roles`/`security`/`structure`.

**Langkah 3:** Kalau check_spec kosong, minimal periksa: tidak kosong, di bawah
batas karakter, format sesuai `output_format`.

**Verifikasi:** 6 test lama harus tetap lulus tanpa diubah (regresi), plus test
baru untuk check_spec.

### Task B4: Pipeline membaca skema

**Berkas:**
- Ubah: `server/internal/worker/pipeline.go:105-131`
- Ubah: `server/internal/worker/prompt.go:47-73`

**Langkah 1:** `execute` mengulang `plan_stages` milik run, bukan
`entity.Stages`.

**Langkah 2:** `buildPrompt` memakai `intent` + `output_format` dari skema
untuk bagian `[tugas]`, bukan peta `charter`/`outFormat`.

**Verifikasi:** satu run penuh, semua tahap tersimpan sebagai `artifacts`
dengan `kind` sesuai `key`.

### Task B5: UI membaca skema

**Berkas:**
- Ubah: `web/src/routes/projects/[id]/+page.svelte:50-58`
- Ubah: `web/src/lib/api.ts`

**Langkah 1:** Ganti `STAGES`/`STAGE_LABEL` statis dengan data dari API
(`plan_stages` run terakhir).

**Langkah 2:** Teks "Enam tahap planning" jadi "Tahap planning" + jumlah asli.

**Langkah 3:** Kalau belum ada run, tampilkan susunan awal 6 tahap sebagai
pratinjau, dengan catatan bahwa AI akan menentukannya.

**Verifikasi:** proyek dengan 4 tahap menampilkan 4 kartu; dengan 9 menampilkan 9.

### Task B6: Endpoint skema

**Berkas:**
- Ubah: `server/internal/delivery/httpapi/handlers.go`
- Ubah: `docs/plan/openapi.md` **ATAU** catat di `docs/update/` kalau
  penambahan endpoint tidak boleh mengubah kontrak beku

**Langkah 1:** `GET /v1/runs/{id}/stages` mengembalikan skema run.

**Langkah 2:** Sertakan `stages` di DTO run yang sudah ada supaya UI tidak
perlu panggilan tambahan.

**Verifikasi:** uji endpoint dengan token asli, periksa bentuk respons.

---

# FASE C - Halaman dokumen mengikuti skema dinamis

### Task C1: Halaman `[stage]` membaca `key` bebas

**Berkas:** `web/src/routes/projects/[id]/[stage]/+page.svelte`

**Langkah 1:** Ambil label dari skema run, bukan `STAGE_LABEL` statis.

**Langkah 2:** Pilih cara render berdasar `output_format` (`markdown` →
`mdToHtml`; `dbml`/`json` → `<pre>` + tombol salin).

### Task C2: Sidebar dinamis

**Berkas:** `web/src/lib/components/SummarySidebar.svelte`

**Langkah 1:** `stageMenu` dari skema run, bukan peta tetap.

**Langkah 2:** Menu yang artefaknya belum ada tetap ditampilkan tapi ditandai
"belum", bukan disembunyikan. Alasannya: sekarang menu disembunyikan sebelum
artefak ada, dan itu yang membuat chip "Lanjut ke PRD" terasa mati.

### Task C3: Sinkronkan chip usul tahap

**Berkas:** `web/src/lib/components/ProjectChat.svelte`

**Langkah 1:** Chip `TAHAP BERIKUTNYA` memakai `key` dari skema, dan hanya
muncul kalau `key` itu benar-benar ada di skema run.

---

## Uji dan verifikasi

| Lapis | Perintah | Harapan |
| --- | --- | --- |
| Format Go | `gofmt -l .` | kosong |
| Statis Go | `go vet ./...` | exit 0 |
| Unit Go | `go test -count=1 ./...` | 10/10 paket ok, 6 test validator lama tetap lulus |
| Migrasi | jalankan di DB scratch | tabel dibuat, data lama utuh |
| E2E | server + panggilan HTTP nyata | rute baru balas 200 dengan isi benar |
| Svelte | `npm run check` | 0 errors |
| Bangun | `npm run build` | sukses |
| Produksi | buka `paluketok.web.id/projects/<slug>/<tahap>` | isi dokumen tampil |

Ikuti `backend-e2e-verification`: bukti harus dari eksekusi nyata, bukan klaim.
Untuk tiap laporan, sebutkan lapis mana yang benar-benar dijalankan.

## Risiko

| Risiko | Besar | Tanggapan |
| --- | --- | --- |
| Judge memberi tahap mengada-ada atau saling tumpang tindih | sedang | Batas 2-12 tahap; `key` wajib unik; tolak dan pakai susunan awal |
| check_spec terlalu longgar → mutu turun | tinggi | Tahap bawaan tetap pakai validator lama; check_spec hanya untuk tahap karangan |
| Data lama rusak | tinggi | Migrasi aditif; `plan_runs` lama tetap jalan lewat kolom `total` yang sudah ada |
| Biaya kredit berubah kalau tahap bisa 12 | sedang | `credits_charged` dihitung dari jumlah tahap, atau tetap 1 kredit per run (perlu keputusan) |
| `docs/plan/openapi.md` beku | rendah | Endpoint baru dicatat di `docs/update/`, kontrak beku tidak diubah tanpa izin |

## Tiga keputusan yang dibutuhkan sebelum Fase B

1. **Kredit.** Jumlah tahap bisa 2 sampai 12. Tetap 1 kredit per run, atau
   kredit mengikuti jumlah tahap? Kalau mengikuti, run 12 tahap jadi jauh lebih
   mahal dan bisa mengejutkan pengguna.
2. **Kendali pengguna.** Setelah judge mengusulkan susunan tahap, apakah
   pengguna boleh menambah/menghapus/mengganti nama sebelum menyusun? Atau
   susunan langsung dipakai tanpa ditanya?
3. **Tahap dengan format bebas.** Kalau pengguna minta tahap "Dokumen Hukum"
   yang isinya prosa, check_spec-nya cuma "wajib ada seksi A, B, C". Apakah itu
   cukup, atau tahap bebas sebaiknya dibatasi ke beberapa format saja
   (markdown, tabel, JSON)?

## Yang sengaja TIDAK dikerjakan

- Streaming token di chat. Bukan bagian permintaan ini.
- Membersihkan em dash di jawaban lama di database. Butuh migrasi sendiri.
- Menyelaraskan `docs/plan/jev_ai.md` (masih menyebut JEV penyedia tunggal).
  Folder planning beku.
