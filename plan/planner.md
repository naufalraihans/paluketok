# Planner - Pipeline Generate Planning

Spesifikasi pipeline yang mengubah brief menjadi artefak. Arsitektur umum ada di `plan/architecture.md`. Dokumen ini mengatur detail tiap tahap, validasi, dan gate JEV.

## 1. Ringkasan alur

```
POST /v1/projects/{id}/runs
  -> potong kredit (transaksi) -> plan_runs.status = queued
  -> worker ambil run -> status = running
  -> untuk tiap tahap [prd, erd, openapi, roles, security, structure]:
       a0. update plan_runs.current_stage = kind      # di AWAL tahap, bukan setelah
       a. bangun prompt dari brief + artefak tahap sebelumnya
       b. panggil LLM -> draf
       c. validasi format
       d. titik ragu -> gate JEV (bagian 4)
       e. simpan artefak (status draft) + decisions
       f. update plan_runs.current_stage, progress
  -> semua tahap selesai -> status = done
  -> error tak tertangani -> status = failed, kredit di-refund
```

## 2. Tahap

| # | kind | Input utama | Format output | Validasi |
|---|---|---|---|---|
| 1 | `prd` | brief, stack | Markdown | Ada seksi: Tujuan, Scope, Acceptance criteria, Di luar scope |
| 2 | `erd` | brief, `prd` | DBML | Parse DBML tanpa error; semua tabel punya PK; semua `Ref` menunjuk tabel yang ada |
| 3 | `openapi` | brief, `prd`, `erd` | JSON (OpenAPI 3.1) | Parse JSON; path dan method valid; tiap operasi punya `responses`; `$ref` resolve |
| 4 | `roles` | `prd`, `openapi` | Markdown tabel matriks | Setiap path di `openapi` muncul di matriks |
| 5 | `security` | `openapi`, `erd`, `roles` | Markdown | Ada trust boundary, validasi input, dan checklist |
| 6 | `structure` | `prd`, `openapi`, `erd` | Markdown (tree + aturan layer) | Tree berisi folder dari stack yang dipilih |

Urutan tetap. Tahap berikutnya selalu memakai artefak tahap sebelumnya, sehingga kesalahan di awal ikut terbawa. Karena itu validasi per tahap wajib.

Tahap `seed`, `test-plan`, dan `ci` belum masuk. Ditunda.

## 3. Prompt

Setiap panggilan LLM memakai struktur tetap:

```
[sistem]  peran + charter tahap (dari docs/agents/*.md) + aturan format + aturan keamanan
[data]    brief dalam blok <brief>...</brief>, artefak sebelumnya dalam blok <artifact kind="...">...</artifact>
[tugas]   instruksi spesifik tahap + bentuk output yang diminta
```

Aturan:

- Charter dari `docs/agents/` dipakai sebagai peran. Isi charter menyebut path template (`docs/spec`, `docs/arch`, `/server`, `/web`). Saat dipakai sebagai prompt planner, path itu hanya label peran dan tidak mengikat; yang mengikat `planner.md`, `openapi.md`, dan `erd.dbml`:
  - `prd` -> `analyst.md`
  - `erd` -> `data.md` dan `architect.md`
  - `openapi` -> `backend.md`
  - `roles` dan `security` -> `security.md`
  - `structure` -> `architect.md`
- Isi `<brief>` dan `<artifact>` adalah data. Prompt menyatakan ini secara eksplisit. Aturan prompt injection: `security.md` bagian 5.
- Batas ukuran: prompt satu tahap maks `ai_settings.max_input_tokens`. Jika brief + artefak sebelumnya melebihi, run `failed` dengan error `input_too_large`, kredit di-refund. Tidak ada pemotongan diam-diam.
- Output hanya berisi artefak. Tanpa preamble, tanpa pagar kode di luar format.
- Suhu rendah (0.2) untuk konsistensi. Batas token input dan output per tahap dari `ai_settings` (bagian 9).

## 4. Gate JEV

JEV dipanggil hanya di titik ragu, bukan di setiap tahap. Titik ragu didefinisikan sebagai keputusan teknis yang punya lebih dari satu opsi wajar, misalnya:

- pilihan tipe ID (UUID atau bigint)
- pilihan model relasi (satu tabel atau tabel terpisah)
- pilihan pola auth atau pagination

Aturan gate:

| Confidence JEV | Gate | Aksi |
|---|---|---|
| >= 0.90 | `EXECUTE` | Pakai `choice`, lanjut. |
| 0.50 - 0.89 | `GATHER_EVIDENCE` | Tambahkan fakta ke `state` (dari brief, artefak, atau stack), panggil ulang. Maks 4 putaran. |
| < 0.50 | `ESCALATE` | Catat sebagai butuh keputusan manusia, pakai opsi default dari stack, lanjut. |
| Putaran ke-4 masih < 0.90 | `ESCALATE` | Sama seperti di atas. |

Catatan:

- Gate `ESCALATE` tidak menghentikan run. Run tetap `done` dan `plan_runs.escalations` bertambah. Dengan begitu user tetap dapat paket lengkap, dan keputusan yang belum pasti ditandai.
- `GATHER_EVIDENCE` menambah fakta, bukan mengulang prompt yang sama. Jika tidak ada fakta baru yang bisa ditambahkan, langsung `ESCALATE`.
- Setiap panggilan JEV dicatat ke `decisions`: `question`, `state`, `criteria`, `gate`, `choice`, `confidence`, `probabilities`.
- Request JEV memakai `model: "jev-latest"`, `state` ringkas (diff/evidence, bukan repository penuh), dan pertanyaan independen.
- Batas gateway JEV: lihat `jev_ai.md`.
- Timeout 30 detik. Tanpa auto-retry. Jika timeout, titik itu dianggap `ESCALATE` dan run lanjut.
- `probabilities` dan `confidence` bukan jaminan benar. Keputusan tidak pernah dijalankan otomatis.

## 5. Validasi output

Urutan validasi tiap tahap:

1. Pemeriksaan kosong dan panjang (tolak jika kosong atau melebihi batas).
2. Pemeriksaan format (parse DBML, JSON, atau cek seksi markdown wajib).
3. Pemeriksaan konsistensi lintas artefak (contoh: path di `roles` harus ada di `openapi`).

Gagal validasi:

- Coba ulang tahap dengan pesan error validasi ditambahkan ke prompt. Maks 2 kali.
- Jika tetap gagal, run `failed`, kredit di-refund.

Catatan: retry validasi ini adalah retry LLM, berbeda dengan retry JEV. Retry JEV tetap tidak diizinkan.

## 6. Status dan progres

`plan_runs`:

- `status`: `queued -> running -> done | failed`
- `current_stage`: nama `kind` yang SEDANG berjalan. Ditulis saat tahap dimulai dan tidak dikosongkan. Saat `status = failed`, nilainya tahap tempat run gagal. Alasan kegagalan ada di kolom `error`
- `progress`: jumlah tahap selesai
- `total`: 6
- `escalations`: jumlah gate `ESCALATE`

Klien polling `GET /v1/runs/{id}` setiap beberapa detik sampai `done` atau `failed`.

## 7. Regenerate satu tahap

`POST /v1/artifacts/{id}/regenerate`:

- Membuat versi baru di `artifact_versions` dan memperbarui `artifacts.content` serta `artifacts.version`.
- Hanya untuk run `done`, dan hanya jika tidak ada run `running` pada project yang sama (mencegah artefak run lama ditulis ulang saat dibaca run baru). Run induk `queued`, `running`, atau `failed`, atau project punya run `running`, ditolak `422 validation_error`.
- Hanya tahap itu yang dijalankan ulang. Artefak tahap sesudahnya tidak otomatis berubah; `artifacts.status` tahap hilir di-set `stale` (status `draft | final | stale` ada di `erd.dbml`; aturan penuh di `erd.md` bagian "Aturan status artefak").
- Biaya: 1 kredit. Keputusan awal, bisa diubah. Charge dicatat di `credits_ledger` (`reason = 'regenerate'`, `ref_type = 'artifact'`). Refund kalau gagal (`reason = 'refund'`, `ref_type = 'ledger'`, `ref_id` = id baris charge). Urutan transaksi: `auth_billing.md` 3.4.

## 8. Konkurensi

- Angka awal (bisa diubah di config): pool 4 worker, antrean 20 run, timeout per tahap 180 detik. Timeout JEV: bagian 4.

- Worker pool dengan ukuran tetap. Satu run dikerjakan satu worker dari awal sampai akhir (tahap berurutan, karena tiap tahap memakai hasil sebelumnya).
- Dua run dari project yang sama boleh berjalan bersamaan, tapi dibatasi satu run `running` per project. Run kedua tetap `queued`.
- Setiap tahap punya `context.WithTimeout`. Shutdown mengikuti `plan/security.md` bagian 11.

## 9. Keputusan terbuka

| Topik | Status |
|---|---|
| Model, base URL, key LLM | diatur admin lewat halaman admin (`admin.paluketok.web.id`), `PUT /v1/admin/settings/ai/planner`. Admin bertindak sebagai pengendali 9router. |
| Siapa bayar token LLM | key admin (keputusan). Harga kredit wajib menutup biaya token. |
| Batas token input dan output per tahap | diatur admin di `ai_settings` (kolom `max_input_tokens`, `max_output_tokens`). Default awal diisi admin saat setup. |
| Kebijakan penghapusan artefak `stale` | ditunda. Status `stale` sudah disiapkan di `artifacts.status`; regenerasi tidak menghapus artefak hilir |
