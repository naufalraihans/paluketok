# PROMPT UNTUK AI EKSEKUTOR - PaluKetok: tahap dinamis + halaman dokumen

> Salin seluruh berkas ini sebagai pesan pertama ke AI eksekutor.
> Ganti hanya bagian `## Jawaban tiga keputusan` di bawah sebelum mengirim.

---

## Peranmu

Kamu senior full-stack engineer yang mengerjakan satu repo nyata milik orang
lain. Kamu bekerja sendiri, tanpa bisa bertanya di tengah jalan. Kalau ada yang
tidak jelas, tulis asumsimu di satu baris lalu lanjut, jangan berhenti.

Kamu **wajib** mengerjakan sampai tuntas dan terverifikasi: kode jalan, test
lulus, dipakai di produksi. Bukan sekadar "sudah ditulis".

## Repo

- Remote: `https://github.com/naufalraihans/paluketok.git`, branch `main`
- Produksi: `https://paluketok.web.id` (VPS, **bukan** Vercel)
- Panel admin: `https://admin.paluketok.web.id/admin`
- Backend: Go 1.26, `server/` (stdlib `net/http`, PostgreSQL via `pgx`)
- Frontend: SvelteKit 3 + Svelte 5 + TypeScript 6, `web/` (adapter-static)
- Database: Supabase (PostgreSQL). Migrasi di `server/database/migrations/`
- Deploy: otomatis lewat `.github/workflows/deploy.yml` tiap push ke `main`.
  Gerbang: `gofmt` + `go vet` + `go test`. Deploy di-rollback otomatis kalau
  `healthz` gagal.
- Dokumentasi: `docs/plan/` (rencana, **beku**), `docs/update/` (perubahan
  setelah beku), `docs/plan/changelog.md` (riwayat lama)

135 test Go ada di `server/`. Semuanya harus tetap lulus.

## Aturan kerja yang tidak boleh dilanggar

1. **Dilarang memakai em dash atau en dash di keluaran mana pun**, termasuk
   komentar kode, dokumen, dan pesan commit. Pakai koma, titik dua, tanda
   kurung, atau tulis ulang kalimatnya. Hyphen biasa (`-`) untuk kata majemuk
   seperti `offline-first` tetap boleh.
   (Menuliskan karakter itu di sini malah melanggarnya, jadi sengaja tidak
   ditulis. Yang dimaksud: tanda pisah panjang, bukan hyphen.)
2. **`docs/plan/` beku.** Jangan ubah isinya. Semua perubahan rencana dicatat di
   `docs/update/changelog/NNNN-judul.md` dengan format yang sudah ada di sana
   (lihat `0010-plan-dipindah-ke-docs.md` sebagai contoh lengkap).
3. **Jangan membuat akun atau data uji di produksi** tanpa izin pemilik.
4. **Jangan commit atau push** kecuali diminta. Bekerja di branch terpisah.
5. **Satu commit satu baris pesan pendek** (mis. `feat: skema tahap dinamis`).
6. **Bukti, bukan klaim.** Jangan pernah tulis "sudah jalan", "beres", atau
   "lulus" untuk sesuatu yang belum kamu jalankan sendiri. Tempelkan perintah
   dan keluarannya yang nyata.
7. **Jangan menulis ke DB produksi** tanpa `pg_dump` dulu.
8. Kalau ada kredensial, **jangan cetak nilainya** ke keluaran atau log.

## Keadaan sekarang (baca dulu, jangan percaya ringkasan ini saja)

Enam tahap planning **tertanam di enam lapis**, bukan sekadar daftar di UI:

| Lapis | Berkas | Isi |
| --- | --- | --- |
| Tipe + urutan | `server/internal/entity/entity.go` sekitar baris 90-108 | `Stage`, 6 konstanta, `Stages`, `TotalStages = 6` |
| Prompt per tahap | `server/internal/worker/prompt.go` sekitar baris 16-43 | peta `charter`, `outFormat`, `inputs` |
| Validator khusus | `server/internal/worker/validate.go` | `validatePRD`, `validateERD`, `validateOpenAPI`, `validateRoles`, `validateSecurity`, `validateStructure` |
| Loop pipeline | `server/internal/worker/pipeline.go` sekitar baris 105-131 | `for i, stage := range entity.Stages` |
| Kolom DB | `server/database/migrations/0001_init.sql` | `plan_runs.total`, `plan_runs.progress`, `artifacts.kind`, `decisions.stage` |
| Daftar UI | `web/src/routes/projects/[id]/+page.svelte` sekitar baris 50-58 | `STAGES`, `STAGE_LABEL` |

**Verifikasi sendiri keenam baris di atas sebelum mulai.** Nomor baris bisa
sudah bergeser.

Yang perlu kamu ketahui tentang validator: **tiap tahap bawaan punya aturan
periksa yang berbeda.** ERD harus DBML dengan PK di tiap tabel dan `Ref` hanya
menunjuk tabel yang ada. OpenAPI harus JSON valid dengan `responses` di tiap
operasi. Roles harus mencakup setiap path OpenAPI. Tahap karangan pengguna tidak
punya aturan seperti itu, dan itulah inti masalahnya.

AI judge adalah penyedia penilai keputusan. Slotnya di tabel `ai_settings`
dengan `slot = 'judge'`, dibaca per panggilan (jadi ganti penyedia dari panel
admin berlaku tanpa restart). Adapternya di `server/internal/adapter/judge/`.
Sekarang modelnya dibaca dari database, bukan konstanta.

## Tugas

Tiga fase, **kerjakan berurutan**. Jangan lompat ke B sebelum A selesai dan
terverifikasi.

### FASE A - Perbaiki bug sidebar + buat halaman dokumen

**Bug yang dilaporkan pemilik:** menekan "Dokumen · PRD" di sidebar melempar
kembali ke halaman chat, bukan ke halaman dokumen.

**Sebabnya sudah ditemukan, jangan cari lagi:**
`web/src/lib/components/SummarySidebar.svelte` memakai `href="#tahap-{item.key}"`.
Anchor `#` menunjuk halaman yang sama, jadi kliknya hanya menggeser scroll.
Halaman `/projects/{slug}/{tahap}` belum pernah ada.

Yang dikerjakan:

1. Buat rute `web/src/routes/projects/[id]/[stage]/` (`+page.ts` dan
   `+page.svelte`). Nama parameternya `stage`, bukan `slug`.
   - Muat proyek dan artefaknya, cari artefak dengan `kind === params.stage`
   - Ada: tampilkan isinya. Markdown lewat `mdToHtml` dari `web/src/lib/markdown.ts`.
     DBML atau JSON lewat `<pre>` + tombol salin.
   - Tidak ada: tampilkan "Dokumen ini belum disusun" + tautan balik
   - Karena rute proyek dinamis tidak diprerender, ikuti pola `prerender`/`ssr`
     di `web/src/routes/projects/[id]/+page.ts`
2. Ubah sidebar: tambah prop `slug`, ganti `href` jadi `/projects/{slug}/{key}`.
   Kirim `slug` dari `web/src/routes/projects/[id]/+page.svelte`.
3. Pindahkan `STAGE_LABEL` ke `web/src/lib/stages.ts` supaya tidak ada dua
   sumber kebenaran. Judul halaman dokumen memakainya.

**Selesai kalau:** `npm run check` bersih, `npm run build` sukses, dan membuka
`/projects/testing-jembut/prd` menampilkan isi PRD yang nyata (bukan kosong).

### FASE B - Skema tahap dinamis

Pemilik ingin jumlah dan nama tahap **ditentukan AI judge**, bukan dipaku enam.

1. **Migrasi baru** `server/database/migrations/0002_plan_stages.sql`. Tabel
   `plan_stages`: `id`, `run_id` (FK ke `plan_runs`, cascade), `key`, `label`,
   `position`, `intent`, `output_format`, `check_spec` (jsonb), `depends_on`
   (text[]), `created_at`. Unik pada `(run_id, key)`. **Aditif saja**, jangan
   ubah tabel lama.
2. **`entity.PlanStage`** sebagai struct. `entity.Stages` tetap ada tapi
   kedudukannya turun jadi **nilai awal**, bukan satu-satunya kemungkinan.
   `plan_runs.total` diisi dari jumlah baris `plan_stages`.
3. **Judge menentukan tahap.** Fungsi baru di `server/internal/worker/stages.go`
   yang memanggil AI judge dan meminta: daftar tahap, masing-masing dengan
   `key`, `label`, `intent`, `output_format`, `check_spec`, `depends_on`.
   Batas wajar 2 sampai 12 tahap. Di luar batas, atau `key` duplikat, atau
   gagal parse: **tolak dan pakai susunan awal enam tahap.** Jangan sampai run
   gagal hanya karena judge mengembalikan yang aneh.
4. **Validasi memakai `check_spec`.** `server/internal/worker/validate.go`
   menerima `check_spec`. **Tahap bawaan tetap memakai validator lamanya
   (perilaku tidak boleh berubah, 6 test lama harus lulus tanpa diubah).**
   `check_spec` dipakai untuk tahap karangan. Kalau `check_spec` kosong,
   minimal periksa: tidak kosong, di bawah batas karakter, format sesuai
   `output_format`.
5. **Pipeline membaca skema.** `pipeline.go` mengulang `plan_stages` milik run,
   bukan `entity.Stages`. `prompt.go` menyusun `[tugas]` dari `intent` +
   `output_format`, bukan peta `charter`/`outFormat`.
6. **UI membaca skema.** `STAGES`/`STAGE_LABEL` statis di
   `web/src/routes/projects/[id]/+page.svelte` diganti data dari API. Teks
   "Enam tahap planning" jadi jumlah yang sebenarnya. Kalau belum ada run,
   tampilkan susunan awal enam tahap sebagai pratinjau dengan catatan bahwa AI
   yang akan menentukannya.
7. **Endpoint.** Sertakan `stages` di DTO run yang sudah ada supaya UI tidak
   perlu panggilan tambahan.

**Selesai kalau:** satu run penuh tersimpan dengan `kind` sesuai `key` skema,
proyek dengan 4 tahap menampilkan 4 kartu, dan 135 test lama tetap lulus.

### FASE C - Halaman dokumen mengikuti skema dinamis

1. Halaman `[stage]` mengambil label dari skema run, bukan `STAGE_LABEL` statis.
2. `stageMenu` di `SummarySidebar.svelte` dari skema run, bukan peta tetap.
   **Menu yang artefaknya belum ada tetap ditampilkan, ditandai "belum", jangan
   disembunyikan.** Menyembunyikannya adalah sebab lain dari bug yang sama.
3. Chip usul tahap di `ProjectChat.svelte` hanya muncul kalau `key` itu ada di
   skema run.

## Jawaban tiga keputusan

Ini sudah diputuskan pemilik. Pakai apa adanya, jangan tanya balik.

1. **Kredit:**
   <!-- GANTI BARIS INI: "tetap 1 kredit per run" ATAU "kredit mengikuti jumlah tahap, rumus: ..." -->
2. **Kendali pengguna atas susunan usulan judge:**
   <!-- GANTI BARIS INI: "langsung dipakai" ATAU "pengguna boleh mengubah dulu di layar konfirmasi" -->
3. **Format tahap karangan:**
   <!-- GANTI BARIS INI: "batasi ke markdown/tabel/JSON" ATAU "bebas, check_spec yang menjaga" -->

## Cara membuktikan pekerjaanmu

Untuk tiap fase, jalankan dan tempelkan keluarannya:

```bash
cd server
gofmt -l .                                  # harus kosong
go vet ./...                                # exit 0
go test -count=1 ./...                      # semua paket ok, 135+ test

cd ../web
npm run check                               # 0 errors, 0 warnings
npm run build                               # sukses
```

Perangkap yang harus kamu hindari:

- **Jangan baca exit code dari ujung pipa.** `go vet ./... | head` memberi exit
  code `head` (selalu 0), sehingga kegagalan `vet` tersembunyi. Alihkan ke
  berkas lalu `echo $?` terpisah.
- **`gofmt -l` menandai berkas yang sudah tidak rapi sebelum kamu menyentuhnya.**
  Cek `git stash && gofmt -l <berkas>; git stash pop` untuk tahu baseline.
- **Bukti rute butuh panggilan HTTP nyata.** `go build` hanya membuktikan kode
  bisa dikompilasi, bukan bahwa rutenya berperilaku benar. Jalankan servernya,
  panggil rutenya, periksa kode status dan isi responsnya. Beda 404 lawan 403
  lawan 400 cuma ada saat berjalan.

## Cara melapor

Untuk tiap fase, laporkan dalam bentuk ini:

```
Fase: A
Status: selesai / sebagian / gagal
Yang diubah: <daftar berkas>
Bukti:
  <perintah>
  <keluaran nyata>
Yang TIDAK dikerjakan: <daftar, beserta alasannya>
Asumsi: <kalau ada>
```

Jangan tulis "semua lulus" tanpa keluaran. Jangan sembunyikan yang gagal. Kalau
ada yang tidak bisa kamu selesaikan, **katakan terus terang** dan jelaskan
penghalangnya. Laporan jujur yang menyebut kegagalan jauh lebih berguna daripada
laporan mulus yang ternyata bohong.

## Jangan dikerjakan

- Streaming token di chat. Bukan bagian tugas ini.
- Membersihkan em dash pada jawaban lama di database. Butuh migrasi tersendiri.
- Menyelaraskan `docs/plan/jev_ai.md`. Folder planning beku.
- Mengubah `docs/plan/openapi.md`. Kalau penambahan endpoint perlu dicatat,
  catat di `docs/update/changelog/`.
- Menyentuh layanan `sfm-server` dan `sfm-worker` di VPS.
