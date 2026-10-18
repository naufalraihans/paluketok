# 0011 - Tahap planning dinamis + halaman tahap per URL

Tanggal: 2026-10-19
Branch: `feat/tahap-dinamis`

## Masalah

Jumlah dan nama tahap planning dipaku di enam lapis kode (entity, worker,
sidebar, halaman proyek, prompt, validator). Proyek kecil dan proyek besar
dipaksa lewat enam dokumen yang sama, dan tidak ada URL yang bisa dibuka
langsung untuk satu tahap.

## Keputusan pemilik yang dijalankan

1. Skema tahap disusun **AI judge** (bukan LLM langsung, bukan pilihan saya).
2. Skema disusun **sebelum run dibuat**, user melihat layar konfirmasi, lalu
   menekan Mulai. Keputusan ini diambil lewat gerbang JEV
   (`EXECUTE`, 0.92) karena `plan_stages.run_id` ber-FK ke `plan_runs`.
3. `output_format` dibatasi tiga: `markdown`, `json`, `dbml`.
4. Batas jumlah tahap 2..12.

## Yang berubah

### Fase A - halaman tahap punya URL sendiri

- Rute baru `/projects/[id]/[stage]` (`+page.ts`, `+page.svelte`). `id` boleh
  UUID atau slug, `stage` boleh key skema mana pun.
- `SummarySidebar` menerima prop `slug` + `stages`; tautan menu jadi
  `/projects/{slug}/{key}`.
- Label dan ikon tahap pindah ke satu berkas: `web/src/lib/stages.ts`.
  Tipe `PlanStage` hanya didefinisikan sekali (di `api.ts`), diekspor ulang.

### Fase B - tahap disusun dinamis, disimpan per run

- Migrasi `0007_plan_stages.sql`: tabel `public.plan_stages`. FK ke
  `plan_runs` ON DELETE CASCADE, unik `(run_id, key)`, CHECK bentuk key dan
  position tidak negatif. RLS aktif tanpa policy klien: hanya server.
- `entity.PlanStage`, `entity.OutputFormat`, `entity.MinStages/MaxStages`,
  `entity.ValidStageKey`, `entity.NormalizePlanStages`,
  `entity.DefaultPlanStages`. Enam konstanta `Stage` turun pangkat jadi
  **susunan awal**, bukan daftar tertutup.
- `worker/stages.go`: menyusun kandidat skema, memanggil judge dengan
  pertanyaan `choice`, memvalidasi hasilnya ketat. Satu pelanggaran saja
  menggagalkan seluruh usulan, dan run jatuh ke enam tahap awal.
- `worker/pipeline.go`: mengulang `plan_stages` milik run, bukan
  `entity.Stages`. `plan_runs.total` diisi panjang skema sebenarnya.
- `worker/prompt.go`: `[tugas]` disusun dari `intent` + `output_format` milik
  tahap, bukan dari peta tetap.
- `worker/validate.go`: menerima `check_spec` tahap; validator lama tetap
  dipakai untuk tahap bawaan sehingga perilakunya tidak berubah.
- `usecase/run_stages.go`: `PreviewStages` (tanpa potong kredit),
  `PlanStagesOfRun`, `proposeFor`, `saveStages`.
- Endpoint baru: `GET /v1/projects/{id}/stages` (usulan),
  `GET /v1/runs/{id}/stages` (skema tersimpan). `POST /v1/projects/{id}/runs`
  menerima `stages` opsional dan menolak skema tidak sah dengan 400
  **sebelum** kredit terpotong.

### Fase C - UI membaca skema, bukan menebaknya

- Sidebar, label tahap, dan petak progres diambil dari skema run.
- Chip usul tahap di percakapan hanya ditampilkan bila key-nya ada di skema.
- Layar konfirmasi: urutkan naik/turun, buang tahap, susun ulang, lalu Mulai.

## Kompatibilitas

Run lama tanpa baris `plan_stages` tetap dibuka dan tetap selesai: worker dan
kedua endpoint jatuh ke `entity.DefaultPlanStages`. `artifacts.kind` dan
`plan_runs.current_stage` sengaja **tetap TEXT tanpa FK** ke `plan_stages`,
supaya artefak hasil regenerate tetap sah walau skema run asalnya sudah hilang.

## Verifikasi (bukan klaim)

| Pemeriksaan | Hasil |
| --- | --- |
| `gofmt -l .` | kosong |
| `go vet ./...` | exit 0 |
| `go test -count=1 ./...` | 10 paket ok, 0 FAIL |
| Jumlah test func Go | 143 (135 lama tetap lulus + 8 baru) |
| `npm run check` | 0 error, 0 warning |
| `npm run build` | sukses, adapter-static menulis `build/` |
| Migrasi 0001..0007 di Postgres 18.3 | ketujuh berkas OK |
| Constraint `plan_stages` | key salah bentuk ditolak CHECK, key dobel ditolak unique, position negatif ditolak CHECK, `run_id` asing ditolak FK, cascade menghapus skema saat run dihapus |
| `PlanStageRepo` terhadap Postgres nyata | insert/baca/replace-ganti-total lulus |
| E2E HTTP | pratinjau tidak memotong kredit; skema 3 tahap tersimpan, terbaca, dan worker membuat 3 artefak termasuk key bebas `rancangan-ui`; 5 bentuk skema tidak sah ditolak 400 tanpa memotong kredit; run tanpa skema tetap selesai |

Catatan: `progress` tetap berupa indeks 0-based seperti sebelumnya (run 3 tahap
selesai dengan `progress = 2`, `total = 3`). Ini perilaku lama yang tidak diubah.

## Belum dikerjakan

- Belum ada cache untuk `PreviewStages`: setiap pratinjau memanggil judge sekali.
  Dicatat sebagai `ponytail:` di kode; tambah cache per (proyek, brief) bila
  biayanya jadi keluhan.
- Tidak ada endpoint daftar riwayat run. Halaman riwayat belum dibangun, dan UI
  memakai `latest_run` yang sudah ada.
- Migrasi belum dijalankan di Supabase prod. Perlu izin pemilik.
