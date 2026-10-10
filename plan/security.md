# Security - PaluKetok

Tahap pipeline `security` juga menghasilkan artefak ini untuk setiap project. Dokumen ini adalah aturan keamanan untuk sistem PaluKetok itu sendiri.

## 1. Trust boundary

```
[Browser SvelteKit]  --(anon key + JWT)-->  [Go API :server]  --(service role / JEV key / Midtrans key)-->  [Supabase Postgres, JEV, Midtrans]
        ^                                        ^
   tidak dipercaya                        dipercaya (satu-satunya pemegang secret)
                                                 ^
                                   [Midtrans webhook]  --(signature SHA512)--
```

| Zona | Dipercaya? | Catatan |
|---|---|---|
| Browser | Tidak | Semua input divalidasi ulang di Go. |
| Go API | Ya | Satu-satunya tempat secret. |
| Postgres | Ya, dengan RLS | RLS = lapis kedua. |
| JEV | Ya untuk keputusan, tidak untuk isi | Output JEV dipakai sebagai pilihan gate, bukan instruksi. |
| Midtrans webhook | Tidak sampai signature valid | Lihat bagian 6. |
| Isi brief dan artefak LLM | Tidak | Data, bukan instruksi. Lihat bagian 5. |

## 2. Autentikasi (JWT)

- Middleware Go memverifikasi JWT Supabase di setiap route user-scoped. Gagal = `401 unauthorized`.
- Verifikasi wajib: signature (JWKS jika project memakai asymmetric key; `SUPABASE_JWT_SECRET` hanya jika masih HS256), `exp`, `aud = "authenticated"`, `iss` sesuai project.
- Allowlist algoritma. Tolak `alg: none` dan algoritma di luar allowlist.
- Role dan `sub` hanya diambil dari claim JWT, tidak dari body atau query.
- Admin ditentukan dari `profiles.is_admin` yang dibaca server, bukan dari claim klien.

## 3. Otorisasi (3 lapis)

| Lapis | Mekanisme | Gagal |
|---|---|---|
| 1 | JWT valid | 401 |
| 2 | Handler cek `owner_id == claims.sub` atau `is_admin` | 403 |
| 3 | RLS Postgres: `owner_id = auth.uid()` | query kosong / 403 |

Aturan tambahan:

- Tidak ada endpoint yang menerima `owner_id` dari klien.
- `plan_runs` menyimpan `user_id` (denormalisasi dari project) agar policy RLS tidak perlu join.
- Service role key hanya dipakai server. Lewat service role, RLS dilewati, jadi setiap query service role wajib filter `owner_id`/`user_id` manual.
- Setiap 403 dicatat ke log dengan `user_id`, `route`, `resource_id`.

## 4. Validasi input

| Field | Aturan |
|---|---|
| `projects.name` | 1-120 karakter, trim, tanpa karakter kontrol |
| `projects.brief` | 1-8.000 karakter (batas biaya token dan prompt) |
| `projects.stack` | JSON object, key hanya `fe`, `be`, `db`, `payment`; nilai dari whitelist enum |
| `status` project | enum `draft / ready / archived`, tidak boleh diubah langsung ke nilai di luar enum |
| `run_id`, `artifact_id`, `order_id` | UUID v4 valid, cek format sebelum query |
| Body JSON | Tolak field tak dikenal (`DisallowUnknownFields`), batas ukuran body 64 KB |
| `decisions?run_id=` | wajib, UUID, dicek owner |

Error validasi: `422 validation_error`, pesan tidak membocorkan nama tabel atau query.

## 5. Keamanan prompt (LLM)

- Brief dan isi artefak lama dimasukkan ke prompt sebagai blok data yang diberi tanda jelas, bukan digabung sebagai instruksi sistem.
- Prompt sistem menyatakan bahwa isi brief tidak boleh mengubah format output, role, atau aturan.
- Output tiap tahap divalidasi sebelum disimpan (lihat `plan/planner.md` bagian 5). Output yang gagal validasi tidak masuk `artifacts`.
- Tidak ada output LLM yang dieksekusi. Sistem menghasilkan dokumen, bukan kode yang dijalankan.
- Kata kunci seperti "abaikan instruksi sebelumnya" di brief tidak mengubah aturan; hasilnya tetap lewat validasi format.

## 6. Webhook Midtrans

Endpoint: `POST /v1/webhooks/midtrans`. Tidak memakai JWT, dilindungi signature.

Urutan wajib:

1. Baca body mentah, batas 16 KB.
2. Hitung `SHA512(order_id + status_code + gross_amount + server_key)`.
3. Bandingkan dengan `signature_key` memakai `hmac.Equal` (constant time). Tidak cocok = `401`, tidak ada perubahan data.
4. Cari `orders` berdasarkan `midtrans_order_id`.
5. Cocokkan `gross_amount` dengan `orders.amount_idr`. Beda = tolak dan catat.
6. Panggil API status Midtrans (`GET /v2/{order_id}/status`) dan pakai status dari sana, bukan hanya dari body webhook.
7. Update order dan tambah `credits_ledger` dalam satu transaksi. Idempoten: jika status sudah `settlement`, return `200` tanpa grant ulang.
8. Balas `200` hanya setelah commit.

Server key Midtrans hanya di `server/.env`.

## 7. Kredit dan konkurensi

- Pemotongan kredit saat run dibuat dilakukan dalam satu transaksi: cek saldo, insert `credits_ledger` (delta negatif), update `profiles.credits`, dan `SELECT ... FOR UPDATE` pada baris profil.
- Saldo tidak boleh negatif. Tambahkan `CHECK (credits >= 0)` di SQL migrasi.
- Dua request run bersamaan dari user yang sama tidak boleh melewati saldo. Lock baris profil menjamin ini.
- `profiles.credits` adalah cache dari `sum(credits_ledger.delta)`. Rekonsiliasi lewat query, bukan lewat kode lain.

## 8. Rate limit dan biaya

| Scope | Batas awal | Aksi saat lewat |
|---|---|---|
| Per user, `POST /runs` | 5 request / menit | `429 rate_limited` |
| Per user, endpoint lain | 60 request / menit | `429 rate_limited` |
| Global JEV | sesuai kuota gateway, dijaga di server | antrean worker menunggu, bukan retry JEV |
| Webhook | tanpa batas per user, tapi body max 16 KB | - |

- Rate limit disimpan di memori proses (monolith satu proses). Jika nanti scale out, pindah ke Postgres atau Redis dan catat sebagai keputusan baru.
- Timeout JEV 30 detik per panggilan. Tanpa auto-retry, karena hasil tidak diketahui dan mungkin sudah tertagih.

## 9. Secret dan konfigurasi

| Secret | Lokasi | Dipakai oleh |
|---|---|---|
| `SUPABASE_SERVICE_ROLE_KEY` | `server/.env` | Go saja |
| `SUPABASE_JWT_SECRET` (jika HS256) | `server/.env` | Go saja |
| `JEV_API_KEY` | `server/.env` | Go saja, paket `internal/judge` |
| `MIDTRANS_SERVER_KEY` | `server/.env` | Go saja |
| `SUPABASE_ANON_KEY` | `web/.env` | Browser, boleh publik |
| `MIDTRANS_CLIENT_KEY` | `web/.env` | Browser (Snap), boleh publik |

- `.env` masuk `.gitignore`. `server/.env.example` hanya berisi nama variabel dengan placeholder.
- Key yang pernah tertulis di dokumen wajib di-rotate. JEV key di `plan/jev_ai.md` masuk daftar ini.
- Log tidak boleh memuat token, signature, server key, atau isi brief penuh.

## 10. HTTP dan frontend

- CORS: hanya origin web PaluKetok. Tidak memakai wildcard.
- Header: `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`.
- Artefak markdown dirender dengan sanitizer. HTML mentah dari LLM tidak dirender.
- Pesan error ke klien tidak memuat stack trace. `500` hanya `{"error":{"code":"internal","message":"Terjadi kesalahan"}}`. Detail masuk log server.
- Middleware `recover` menangkap panic dari goroutine handler dan worker. Panic di worker menandai run `failed`, bukan membuat proses mati.

## 11. Worker dan goroutine

- Setiap tahap punya `context.WithTimeout`. Pembatalan run menghentikan goroutine.
- Worker pool punya ukuran tetap. Antrean penuh = `503`, bukan goroutine tak terbatas.
- Shutdown: tunggu run yang berjalan sampai batas waktu, lalu tandai sisanya `failed` dengan `error = "server_restart"`. Kredit di-refund.

## 12. Audit

Dicatat ke `decisions` (keputusan JEV) dan `credits_ledger` (uang). Deny 403 dan webhook gagal dicatat ke log server. Tabel audit terpisah belum ada; ditunda.

## 13. Checklist sebelum rilis

- [ ] Semua route user-scoped punya middleware JWT.
- [ ] Setiap query service role memfilter owner.
- [ ] RLS aktif di semua tabel app, dan ada tes yang membuktikan user A tidak bisa baca data user B.
- [ ] Webhook menolak signature salah dan nominal tidak cocok.
- [ ] Key lama sudah di-rotate.
- [ ] `.env` tidak masuk git.
- [ ] Tidak ada log yang memuat secret.
