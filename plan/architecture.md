# Arsitektur PaluKetok

## 1. Apa ini

Platform yang mengubah 1 brief developer menjadi paket planning lengkap:
PRD, ERD, kontrak OpenAPI, role access, aturan proteksi, struktur folder.

Bukan toko online. Bukan generator kode. Output = dokumen planning.

## 2. Alur utama

```
User isi brief (nama project, deskripsi, stack, fitur)
  -> POST /v1/projects/{id}/runs
  -> worker pool (goroutine) jalankan pipeline bertahap:
       1. persist brief
       2. tiap tahap = 1 panggilan LLM -> draf
       3. titik ragu -> panggil JEV (gate)
          EXECUTE (>= 0.90)      -> pakai choice, lanjut
          GATHER_EVIDENCE (<0.90) -> cari fakta baru, ulang (maks 4x)
          ESCALATE (<0.50)       -> tandai butuh keputusan manusia
       4. simpan artefak + log keputusan
  -> status run = done
```

Pipeline tahap (urutan, tiap tahap tulis artefak sendiri):

1. `prd` - ringkas masalah, scope, acceptance criteria
2. `erd` - tabel, kolom, relasi, index
3. `openapi` - kontrak endpoint + schema
4. `roles` - role + matriks izin per endpoint
5. `security` - trust boundary, RLS, validasi input
6. `structure` - struktur folder BE/FE + aturan layer

Tahap besok, belum sekarang: `seed`, `test-plan`, `ci`.

## 3. Keputusan arsitektur

- **Monolith Go, satu proses.** Gate JEV: `monolith_go`, confidence 1.00. Satu deploy, satu CI, tanpa queue eksternal.
- **Async in-process.** Generation panjang -> worker pool goroutine, status di DB. Klien poll `GET /v1/runs/{id}`. Tanpa Redis, tanpa message broker.
- **Supabase = Postgres + Auth saja.** Go jadi sumber logika. FE bicara ke Go, bukan langsung ke DB.
- **RLS ON sebagai lapisan kedua.** Semua tabel user-scoped, policy `owner_id = auth.uid()`.
- **JEV = gate, bukan model chat.** Dipanggil server-side, key di env.
- **Midtrans = top-up kredit.** 1 run bayar sejumlah kredit. Webhook masuk ke Go.

## 4. Batas modul (dalam satu proses)

```
server/
  cmd/api/            entrypoint
  internal/http/      router, middleware, handler
  internal/planner/   pipeline tahap + prompt
  internal/judge/     klien JEV + gate logic
  internal/store/     query Postgres (sqlc)
  internal/pay/       klien Midtrans + webhook
  internal/worker/    goroutine pool
  migrations/         SQL migrasi
web/                  SvelteKit
```

## 5. Non-fungsi

- Timeout JEV: 30s per panggilan. Tanpa auto-retry (outcome tak diketahui = bisa sudah ditagih).
- 1 run maks 4 putaran GATHER_EVIDENCE per keputusan; lewat itu = ESCALATE.
- Idempoten webhook Midtrans via `midtrans_order_id` unik.
- Gateway JEV: maks 32 question/request, 64 opsi/choice, 2-10 level/score.

## 6. Di luar scope sekarang

- Fan-out multi-model (bandingkan output 2 LLM).
- Kolaborasi tim per project.
- Export PDF/DOCX.
- Stripe.
