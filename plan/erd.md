# ERD PaluKetok

Sumber kebenaran skema: `plan/erd.dbml`. Paste isinya ke https://dbdiagram.io (New Diagram > Import > DBML).
Dokumen ini menjelaskan relasi, aturan, dan alasan desain. Jika ada konflik, `erd.dbml` yang berlaku.

PostgreSQL via Supabase. `auth.users` dikelola Supabase. RLS ON di semua tabel app.

## 1. Daftar tabel

| Tabel | Fungsi | Pemilik data |
|---|---|---|
| `profiles` | profil, tier, admin, saldo kredit | `auth.users` 1:1 |
| `projects` | project milik user (brief mentah + stack) | user |
| `plan_runs` | satu kali generate planning | project |
| `artifacts` | satu dokumen per tahap per run | run |
| `artifact_versions` | riwayat versi artefak saat regenerate | artefak |
| `decisions` | jejak keputusan JEV per tahap | run |
| `orders` | order top-up Midtrans | user |
| `credits_ledger` | catatan mutasi kredit (sumber saldo) | user |

## 2. Relasi

```
auth.users 1 ── 1 profiles
profiles   1 ── * projects          (owner_id)
projects   1 ── * plan_runs         (project_id, cascade)
plan_runs  1 ── * artifacts         (run_id, cascade)
plan_runs  1 ── * decisions         (run_id, cascade)
artifacts  1 ── * artifact_versions (artifact_id, cascade)
profiles   1 ── * orders            (user_id, restrict)
profiles   1 ── * credits_ledger    (user_id, restrict)

credits_ledger.(ref_type, ref_id) ──> orders | plan_runs | artifacts   (polymorphic, tanpa FK)
```

`profiles`, `orders`, dan `credits_ledger` memakai `restrict` agar histori uang tidak ikut hilang saat user dihapus. Penghapusan user ditangani lewat proses terpisah yang sengaja.

## 3. Polymorphic: `credits_ledger`

Satu tabel mencatat semua mutasi kredit. Target tergantung `ref_type`:

| `reason` | `delta` | `ref_type` | `ref_id` menunjuk ke |
|---|---|---|---|
| `topup` | + | `order` | `orders.id` |
| `run` | - | `plan_run` | `plan_runs.id` |
| `refund` | + | `plan_run` | `plan_runs.id` |
| `regenerate` | - | `artifact` | `artifacts.id` |
| `grant` | + atau - | `none` | null |

Kenapa tanpa FK: Postgres tidak mendukung FK polymorphic. Integritas dijaga aplikasi. Jika nanti perlu lebih ketat, tambahkan trigger validasi per `ref_type`.

Aturan:

- `refund` hanya sekali per run. Dijaga lewat unique partial index `(ref_id, reason) WHERE reason = 'refund'`.
- `ref_id` wajib ada jika `ref_type <> 'none'`. Dijaga aplikasi (dan bisa ditambah `CHECK`).

## 4. Denormalisasi untuk RLS

`plan_runs`, `artifacts`, dan `decisions` menyimpan `user_id` walau bisa dicari lewat join. Alasannya:

- Policy RLS `user_id = auth.uid()` langsung tanpa join. Lebih murah dan lebih mudah diaudit.
- Query daftar per user (`/v1/runs`, `/v1/decisions`) tidak perlu join ke `projects`.

Konsistensi: `user_id` di `plan_runs` harus sama dengan `projects.owner_id` saat run dibuat. Nilainya tidak boleh diubah setelah insert.

## 5. Kredit

- Sumber kebenaran saldo: `credits_ledger`.
- `profiles.credits` adalah cache untuk baca cepat. Harus sama dengan `sum(credits_ledger.delta)` untuk user tersebut.
- `CHECK (credits >= 0)` pada `profiles`.
- Pemotongan kredit dilakukan dalam satu transaksi dengan `SELECT ... FOR UPDATE` pada baris `profiles`. Detail di `plan/auth_billing.md` bagian 3.

## 6. Status

| Tabel | Kolom | Nilai |
|---|---|---|
| `projects` | `status` | `draft`, `ready`, `archived` |
| `plan_runs` | `status` | `queued`, `running`, `done`, `failed` |
| `artifacts` | `status` | `draft`, `final`, `stale` |
| `orders` | `status` | `pending`, `settlement`, `expire`, `deny` |
| `decisions` | `gate` | `EXECUTE`, `GATHER_EVIDENCE`, `ESCALATE` |

Aturan `plan_runs`:

- Maksimal satu run `running` per project. Dijaga dengan unique partial index `(project_id) WHERE status = 'running'`.
- `current_stage` berisi kind tahap yang sedang berjalan.
- `progress` (0-6) dan `total` (6) dipakai klien untuk polling `GET /v1/runs/{id}`.

Aturan `artifacts`:

- Satu artefak per tahap per run: unique `(run_id, kind)`.
- `stale` dipasang pada artefak hilir saat tahap hulu di-regenerate.

Aturan `orders`:

- Transisi hanya `pending` ke `settlement`, `expire`, atau `deny`. Dari `settlement` tidak boleh kembali.
- `amount_idr` dan `credits_granted` diambil dari tabel paket di server, bukan dari klien.

## 7. Versi artefak

- `artifacts.version` dan `artifacts.content` selalu versi terbaru.
- Setiap regenerate menulis versi lama ke `artifact_versions` sebelum menimpa.
- `artifact_versions.version` unik per `artifact_id`.
- `created_by_run_id` mencatat run penyebab versi itu. Nullable agar aman jika run dihapus (`set null`).

## 8. Index

| Tabel | Index | Untuk |
|---|---|---|
| `profiles` | `is_admin` partial | daftar admin |
| `projects` | `(owner_id, created_at)` | `GET /v1/projects` |
| `plan_runs` | `(project_id, status)` | cek run aktif |
| `plan_runs` | `(user_id, created_at)` | daftar run user |
| `plan_runs` | unique `(project_id) WHERE status = 'running'` | satu run aktif per project |
| `artifacts` | unique `(run_id, kind)` | ambil artefak per tahap |
| `artifacts` | `(user_id, created_at)` | daftar artefak user |
| `artifact_versions` | unique `(artifact_id, version)` | riwayat versi |
| `decisions` | `(run_id, stage)` | `GET /v1/decisions?run_id=` |
| `decisions` | `(user_id, created_at)` | jejak keputusan user |
| `orders` | `midtrans_order_id` unique | idempotensi webhook |
| `orders` | `(user_id, created_at)`, `status` | daftar order, pembersihan expire |
| `credits_ledger` | `(user_id, created_at)` | riwayat kredit |
| `credits_ledger` | `(ref_type, ref_id)` | telusur mutasi per objek |
| `credits_ledger` | unique `(ref_id, reason) WHERE reason = 'refund'` | refund sekali |

## 9. Keputusan desain

| Keputusan | Alasan |
|---|---|
| UUID sebagai PK | Aman diekspos ke klien, tidak bisa ditebak urutannya |
| `jsonb` untuk `stack`, `state`, `criteria`, `probabilities` | Bentuknya berubah seiring fitur; tidak perlu tabel tambahan |
| `numeric` untuk `cost` dan `confidence` | Presisi desimal, tidak ada pembulatan float |
| Tidak ada tabel `stages` terpisah | Enam tahap tetap dan dikenal kode; cukup `kind` di `artifacts` |
| `decisions` terpisah dari `artifacts` | Satu tahap bisa punya banyak keputusan (beberapa putaran GATHER_EVIDENCE) |

## 10. Di luar skema sekarang

- Tabel audit terpisah (ditunda, lihat `plan/security.md` bagian 12).
- Sharing project antar user (ditunda, lihat `plan/roles.md`).
- Tabel seed (ditunda).
- Tabel paket harga. Sementara paket disimpan di konfigurasi server.
