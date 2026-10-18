# ERD PaluKetok

Sumber kebenaran skema: `plan/erd.dbml`. Jika ada konflik, `erd.dbml` yang berlaku. Dokumen ini hanya mencatat aturan yang tidak bisa diekspresikan di DBML.

PostgreSQL via Supabase. `auth.users` dikelola Supabase. RLS ON di semua tabel app.

## Aturan kredit

- Sumber saldo: `credits_ledger`. `profiles.credits` = cache, harus sama dengan `sum(credits_ledger.delta)`.
- Pemotongan dalam satu transaksi dengan `SELECT ... FOR UPDATE` pada baris `profiles`. Detail: `auth_billing.md` bagian 3.
- Refund hanya sekali per run (`uq_ledger_one_refund_per_run`).
- Refund regenerate hanya sekali per baris charge regenerate, dikunci `uq_ledger_one_refund_per_charge` pada baris charge itu sendiri (`ref_type = 'ledger'`, `ref_id` = id baris charge).
- Grant topup hanya sekali per order, dikunci `uq_ledger_one_topup_per_order` (`reason = 'topup'`, `ref_type = 'order'`, `ref_id` = `orders.id`). Ini jaring terakhir di database, bukan pengganti lock di `security.md` bagian 6.
- Ketiga index penjaga uang di atas memuat predikat `ref_id is not null`. Tanpa itu, baris dengan `ref_id = NULL` dianggap distinct oleh Postgres dan bisa masuk berkali-kali — jaringnya bocor.
- Setiap baris ledger mencatat `actor_id` (pelaku: user atau admin). Grant admin wajib membawa `idempotency_key`, dikunci `uq_ledger_one_grant_per_key`, supaya double-submit tidak menaikkan saldo dua kali.
- `credits_ledger` bersifat append-only: tidak ada `UPDATE` atau `DELETE` pada barisnya. `profiles.credits` di-set dari `RETURNING` hasil `INSERT` ledger, bukan dihitung ulang di aplikasi.
- Refund memakai nilai `credits_charged` dari baris asal, bukan angka tetap `1`.

## Aturan `ledger` (polymorphic, tanpa FK)

| `reason` | `delta` | `ref_type` | `ref_id` menunjuk ke |
|---|---|---|---|
| `topup` | + | `order` | `orders.id` |
| `run` | - | `plan_run` | `plan_runs.id` |
| `refund` | + | `plan_run` | `plan_runs.id` |
| `regenerate` | - | `artifact` | `artifacts.id` |
| `refund` (regenerate gagal) | + | `ledger` | id baris charge `regenerate` |
| `grant` | + atau - | `none` | null (`actor_id` = admin pelaku, `idempotency_key` wajib) |

Integritas `ref_type` dijaga aplikasi. `ref_type = 'ledger'` hanya dipakai untuk refund regenerate dan hanya menunjuk baris `credits_ledger` dengan `reason = 'regenerate'`.

## Aturan status

- `projects.brief`: 1-8.000 karakter, ditegakkan `CHECK` di migrasi.
- `projects.status`: `draft | ready | archived` (`erd.dbml`). Project baru = `draft`. `archived` menolak `POST /runs` dengan `422`. `DELETE` = set `archived`, bukan hapus baris.

- `plan_runs`: maks satu `running` per project. Penegak utama: usecase di `POST /runs` menolak dengan `409` jika project punya run `running`. Jaring kedua: unique partial index `uq_plan_runs_one_running`. `progress` 0-6, `total` 6.
- `artifacts`: satu per tahap per run. `stale` dipasang pada tahap hilir saat tahap hulu di-regenerate (kebijakan penuh ditunda, status `stale` sudah disiapkan).
- `decisions`: pada regenerate, `round` melanjutkan penomoran untuk `(run_id, stage)` yang sama, bukan mulai dari 1 lagi. Nomor tertinggi sebelumnya `+1`.
- `orders`: `pending` ke `settlement` atau `expire`. Dari `settlement` tidak boleh kembali. Kedaluwarsa memakai `expired_date` dari respons DOKU, bukan timer sendiri. `amount_idr` dan `credits_granted` dari tabel paket server, bukan klien.
- Lock webhook: `SELECT` order `FOR UPDATE` dalam transaksi yang sama dengan grant. Order `settlement` langsung `200` tanpa grant.
- Transisi order hanya lewat `UPDATE orders SET status = $new WHERE id = $1 AND status = 'pending'`. Rows affected = 0 berarti tidak ada perubahan dan tidak ada grant.
- Late settlement: order `expire` yang kemudian dibayar boleh pindah ke `settlement` HANYA jika status API DOKU mengonfirmasi `SUCCESS`. Grant tetap dijaga `uq_ledger_one_topup_per_order`.
- Webhook hanya diproses untuk `invoice_number` yang ada di `orders`. Tidak dikenal = `404`, tanpa perubahan data.

## Aturan status artefak

- `draft`: dibuat saat tahap selesai dalam run yang masih `running`.
- `final`: otomatis saat run `done`.
- `stale`: tahap hulu di-regenerate (lihat `planner.md` bagian 7).

## Aturan versi artefak

- `artifacts.version` dan `artifacts.content` selalu versi terbaru.
- Regenerate menulis versi lama ke `artifact_versions` sebelum menimpa.

## Denormalisasi untuk RLS

`plan_runs`, `artifacts`, `decisions` menyimpan `user_id` agar policy `user_id = auth.uid()` tanpa join. Nilainya sama dengan `projects.owner_id` saat run dibuat dan tidak boleh diubah.

`artifact_versions` tidak punya `user_id`. Policy-nya lewat join: `artifact_id IN (SELECT id FROM artifacts WHERE user_id = auth.uid())`.

## Keputusan desain

| Keputusan | Alasan |
|---|---|
| UUID sebagai PK | Aman diekspos, tidak bisa ditebak |
| `jsonb` untuk `stack`, `state`, `criteria`, `probabilities` | Bentuk berubah seiring fitur |
| `numeric` untuk `cost` dan `confidence` | Presisi desimal |
| Tanpa tabel `stages` | Enam tahap tetap, cukup `kind` |
| `decisions` terpisah dari `artifacts` | Satu tahap bisa banyak putaran GATHER_EVIDENCE |
| `ai_settings.api_key` plaintext | Keputusan user. Akses hanya service role, GET API hanya masked |

## Ditunda

- Tabel audit terpisah (`security.md` bagian 12).
- Sharing project antar user (`roles.md`).
- Tabel seed dan tabel paket harga (paket sementara di konfigurasi server).
