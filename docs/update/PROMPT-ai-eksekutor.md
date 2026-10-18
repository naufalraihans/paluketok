# Tugas: PaluKetok - tahap planning dinamis + perbaikan sidebar

Repo: `E:\rekapProject\projecct_build_with_AI`
Branch kerja: `feat/tahap-dinamis` (buat dari `main`)

Baca dulu sebelum mulai:

- `docs/plan/main.md` dan `docs/plan/planner.md` - alur planning
- `docs/plan/workflow.md` - aturan dependency dan urutan kerja
- `docs/plan/changelog.md` - riwayat keputusan
- `docs/update/README.md` - cara mencatat perubahan

135 test Go ada. Semuanya harus tetap lulus.

## Aturan

1. Dilarang memakai tanda pisah panjang (em dash atau en dash) di keluaran mana
   pun: kode, komentar, dokumen, pesan commit. Pakai koma, titik dua, tanda
   kurung, atau tulis ulang kalimatnya. Hyphen biasa untuk kata majemuk seperti
   `offline-first` tetap boleh.
2. `docs/plan/` BEKU. Jangan ubah isinya. Semua perubahan dicatat di
   `docs/update/changelog/NNNN-judul.md` mengikuti contoh
   `docs/update/changelog/0010-plan-dipindah-ke-docs.md`.
3. Jangan bikin akun atau data uji di produksi. Jangan sentuh DB produksi tanpa
   `pg_dump` dulu.
4. Jangan commit atau push kecuali pemilik menyuruh.
5. Pesan commit satu baris pendek.
6. Bukti, bukan klaim. Jangan tulis "sudah jalan" atau "lulus" untuk hal yang
   belum kamu jalankan sendiri. Tempelkan perintah dan keluaran nyatanya.
7. Jangan cetak nilai kredensial ke mana pun.

## Kenapa tugas ini ada

Pemilik mau jumlah dan nama tahap planning **ditentukan AI judge**, bukan dipaku
enam. Sekarang enam tahap tertanam di enam lapis kode, bukan cuma daftar di UI:

| Lapis | Berkas | Isi |
| --- | --- | --- |
| Tipe dan urutan | `server/internal/entity/entity.go` | `Stage`, 6 konstanta, `Stages`, `TotalStages = 6` |
| Prompt per tahap | `server/internal/worker/prompt.go` | peta `charter`, `outFormat`, `inputs` |
| Validator khusus | `server/internal/worker/validate.go` | `validatePRD`, `validateERD`, `validateOpenAPI`, `validateRoles`, `validateSecurity`, `validateStructure` |
| Loop pipeline | `server/internal/worker/pipeline.go` | `for i, stage := range entity.Stages` |
| Kolom DB | `server/database/migrations/0001_init.sql` | `plan_runs.total`, `plan_runs.progress`, `artifacts.kind`, `decisions.stage` |
| Daftar UI | `web/src/routes/projects/[id]/+page.svelte` | `STAGES`, `STAGE_LABEL` |

Verifikasi sendiri keenam baris itu sebelum mulai. Nomor baris bisa bergeser.

Yang penting dipahami: **tiap tahap bawaan punya aturan periksa yang berbeda.**
ERD harus DBML dengan PK di tiap tabel dan `Ref` hanya menunjuk tabel yang ada.
OpenAPI harus JSON valid dengan `responses` di tiap operasi. Roles harus
mencakup setiap path OpenAPI. Tahap karangan pengguna tidak punya aturan seperti
itu, dan itulah inti masalahnya. Judge harus mengembalikan cara memeriksanya
sekalian, bukan cuma namanya.

Slot judge ada di tabel `ai_settings` (`slot = 'judge'`), dibaca per panggilan,
adapternya `server/internal/adapter/judge/`.

## Fase A - perbaiki sidebar + halaman dokumen

Bug: tombol "Dokumen - PRD" di sidebar melempar kembali ke halaman chat.

Sebabnya sudah ditemukan: `web/src/lib/components/SummarySidebar.svelte` memakai
`href="#tahap-{item.key}"`. Anchor `#` menunjuk halaman yang sama, jadi kliknya
cuma menggeser scroll. Halaman `/projects/{slug}/{tahap}` belum pernah ada.

1. Buat rute `web/src/routes/projects/[id]/[stage]/` (`+page.ts` +
   `+page.svelte`). Nama parameternya `stage`, bukan `slug`.
   - Muat proyek dan artefaknya, cari yang `kind === params.stage`
   - Ada: tampilkan isinya. Markdown lewat `mdToHtml` dari `web/src/lib/markdown.ts`.
     DBML atau JSON lewat `<pre>` + tombol salin.
   - Tidak ada: "Dokumen ini belum disusun" + tautan balik
   - Ikuti pola `prerender`/`ssr` di `web/src/routes/projects/[id]/+page.ts`
2. Sidebar: tambah prop `slug`, `href` jadi `/projects/{slug}/{key}`. Kirim
   `slug` dari `web/src/routes/projects/[id]/+page.svelte`.
3. Pindahkan `STAGE_LABEL` ke `web/src/lib/stages.ts` supaya satu sumber.

Selesai kalau `npm run check` bersih, `npm run build` sukses, dan
`/projects/testing-jembut/prd` menampilkan isi PRD nyata.

## Fase B - skema tahap dinamis

1. Migrasi baru `server/database/migrations/0002_plan_stages.sql`. Tabel
   `plan_stages`: `id`, `run_id` (FK `plan_runs` cascade), `key` (slug), `label`,
   `position`, `intent`, `output_format`, `check_spec` (jsonb), `depends_on`
   (text[]), `created_at`. Unik `(run_id, key)`. Aditif saja, tabel lama jangan
   diubah.
2. `entity.PlanStage` struct. `entity.Stages` turun pangkat jadi nilai awal,
   bukan satu-satunya kemungkinan. `plan_runs.total` diisi jumlah baris
   `plan_stages`.
3. `server/internal/worker/stages.go`: panggil AI judge, minta daftar tahap
   beserta `key`, `label`, `intent`, `output_format`, `check_spec`, `depends_on`.
   Batas 2 sampai 12 tahap. Di luar batas, `key` duplikat, atau gagal parse:
   tolak dan pakai susunan awal enam tahap. Run tidak boleh gagal gara-gara judge
   mengembalikan yang aneh.
4. `server/internal/worker/validate.go` menerima `check_spec`. Tahap bawaan TETAP
   pakai validator lamanya, perilakunya tidak boleh berubah, 6 test lama harus
   lulus tanpa diubah. `check_spec` untuk tahap karangan. Kalau kosong, minimal
   periksa: tidak kosong, di bawah batas karakter, format sesuai `output_format`.
5. `pipeline.go` mengulang `plan_stages` milik run, bukan `entity.Stages`.
   `prompt.go` menyusun `[tugas]` dari `intent` + `output_format`, bukan peta
   `charter`/`outFormat`.
6. UI: `STAGES`/`STAGE_LABEL` statis diganti data dari API. Teks "Enam tahap
   planning" jadi jumlah sebenarnya. Belum ada run: tampilkan susunan awal enam
   tahap sebagai pratinjau, dengan catatan AI yang akan menentukannya.
7. Sertakan `stages` di DTO run yang sudah ada, jangan bikin UI memanggil dua kali.

Selesai kalau satu run penuh tersimpan dengan `kind` sesuai `key` skema, proyek
4 tahap menampilkan 4 kartu, dan 135 test lama tetap lulus.

## Fase C - halaman dokumen ikut skema

1. Halaman `[stage]` ambil label dari skema run, bukan `STAGE_LABEL` statis.
2. `stageMenu` di `SummarySidebar.svelte` dari skema run. Menu yang artefaknya
   belum ada TETAP ditampilkan, ditandai "belum", jangan disembunyikan.
   Menyembunyikannya adalah sebab lain dari bug yang sama.
3. Chip usul tahap di `ProjectChat.svelte` hanya muncul kalau `key`-nya ada di
   skema run.

## Tiga keputusan pemilik

1. Kredit: **1 kredit untuk semuanya, berapa pun jumlah tahapnya.** Tidak ada
   hitungan per tahap. Kalau perlu dibatasi, batasi jumlah tahapnya (maksimal
   12), bukan kreditnya.
2. Susunan usulan judge: **dibuat layar konfirmasi, pengguna boleh mengubah
   sebelum mulai.** Alasan: judge hanya dipanggil saat LLM penyusun tidak yakin
   dan butuh keputusan berbasis statistika. LLM gampang berhalusinasi, jadi
   usulan judge tetap harus bisa dikoreksi manusia sebelum dipakai.
3. Format tahap karangan: **dibatasi ke markdown, JSON, dan DBML.** Alasannya:
   `check_spec` hanya bisa memastikan "ada", tidak bisa memastikan "benar".
   Membatasi format menjaga jurang mutu itu tetap kecil.

## Bukti yang harus kamu tempelkan

```bash
cd server
gofmt -l .                    # harus kosong
go vet ./...                  # exit 0
go test -count=1 ./...        # semua paket ok

cd ../web
npm run check                 # 0 errors
npm run build                 # sukses
```

Dua perangkap:

- Jangan baca exit code dari ujung pipa. `go vet ./... | head` memberi exit code
  `head` (selalu 0), kegagalan `vet` tersembunyi. Alihkan ke berkas, lalu `echo $?`
  terpisah.
- `gofmt -l` menandai berkas yang sudah tidak rapi sebelum kamu sentuh. Cek
  baseline dengan `git stash && gofmt -l <berkas>; git stash pop`.

Bukti rute butuh panggilan HTTP nyata. `go build` cuma membuktikan kode bisa
dikompilasi. Jalankan servernya, panggil rutenya, periksa kode status dan isi
responsnya. Beda 404 lawan 403 lawan 400 cuma ada saat berjalan.

## Cara melapor

```
Fase: A
Status: selesai / sebagian / gagal
Diubah: <daftar berkas>
Bukti: <perintah + keluaran nyata>
Tidak dikerjakan: <daftar + alasan>
Asumsi: <kalau ada>
```

Kalau ada yang gagal atau tidak bisa diselesaikan, katakan terus terang dan
jelaskan penghalangnya. Laporan jujur yang menyebut kegagalan jauh lebih berguna
daripada laporan mulus yang ternyata karangan.

## Jangan dikerjakan

- Streaming token di chat
- Membersihkan tanda pisah panjang pada jawaban lama di database
- Menyelaraskan `docs/plan/jev_ai.md` (folder beku)
- Mengubah `docs/plan/openapi.md`. Catat di `docs/update/changelog/` kalau perlu
- Menyentuh layanan `sfm-server` dan `sfm-worker` di VPS
