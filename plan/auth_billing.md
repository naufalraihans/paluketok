# Auth dan Billing - PaluKetok

Alur login, sinkron profil, top-up kredit lewat DOKU, dan pemakaian kredit.

## 1. Auth (Supabase)

### 1.1 Alur login

```
Browser -> Supabase Auth (email/password atau OAuth)
        <- access_token (JWT) + refresh_token
Browser -> Go API  Authorization: Bearer <access_token>
```

- Frontend memakai `supabase-js` dengan anon key untuk login dan refresh token.
- Frontend tidak pernah memegang service role key.
- Token dikirim ke Go lewat header. Go memverifikasi JWT sesuai `plan/security.md` bagian 2.

### 1.2 Sinkron `profiles`

Trigger Postgres pada `auth.users` insert:

```
AFTER INSERT ON auth.users
  -> INSERT INTO profiles (id, username, full_name, avatar_url)
```

- `profiles.id = auth.users.id`.
- `plan_tier = 'free'`, `is_admin = false`, `credits = 0` pada awal.
- Username dibuat dari metadata atau dari bagian email sebelum `@`, lalu dibuat unik dengan suffix angka.
- Trigger dijalankan sebagai fungsi `SECURITY DEFINER` dengan `search_path` terkunci.

### 1.3 Registrasi, login, lupa password

- Registrasi, login, dan reset password memakai Supabase Auth (email + password). Go tidak menyimpan password.
- Lupa password: `supabase-js` memanggil `resetPasswordForEmail`, link reset kembali ke `https://paluketok.web.id/reset`. Set password baru lewat Supabase.
- Email konfirmasi registrasi aktif. Profil dibuat saat insert `auth.users` (bagian 1.2). Tanpa konfirmasi, Supabase tidak mengeluarkan access token, jadi `/v1/me` tidak bisa dipanggil.
- Halaman ini di FE, ditunda sesuai scope BE dulu. Yang dibangun BE: trigger `profiles` dan `/v1/me`.

### 1.4 Endpoint `/v1/me`

- Hanya membaca profil milik `claims.sub`.
- Mengembalikan `id`, `username`, `full_name`, `plan_tier`, `is_admin`, `credits`.
- Tidak mengembalikan data user lain.

### 1.5 Role

Admin pertama dibuat lewat `cmd/seeder` dengan argumen email, yang men-set `profiles.is_admin = true`. Bukan lewat migrasi, supaya tidak ada email tertanam di SQL.

Matriks izin dan aturan role ada di `plan/roles.md`. Admin tidak bisa mengubah `is_admin` lewat API publik; hanya lewat SQL migrasi atau dashboard Supabase.

## 2. Billing (DOKU Checkout)

### 2.1 Model

- Top-up kredit. Bukan langganan.
- 1 kredit = 1 run. Tarif regenerate satu tahap = 1 kredit (lihat `plan/planner.md` bagian 7).
- Paket harga ditetapkan di kode konfigurasi, bukan dari klien.

### 2.2 Paket (awal, bisa diubah)

| Paket | Kredit | Harga (IDR) |
|---|---|---|
| Starter | 10 | 25.000 |

Harga dan jumlah kredit yang dikirim klien diabaikan. Server mengambil nilai dari tabel paket di server.

### 2.3 Alur order

```
1. POST /v1/orders  { package_id }
   Go: buat orders (status pending)
       invoice_number unik "PK-<random>" (dibuat server, maks 30 karakter)
       request_id unik (UUID, idempotency key DOKU)
   Go -> DOKU: POST https://api.doku.com/checkout/v1/payment
        header Client-Id, Request-Id, Request-Timestamp, Signature (HMAC-SHA256)
        body order.amount, order.invoice_number, payment.payment_due_date
   DOKU -> Go: response.payment.url, response.payment.expired_date
   Go -> klien { order_id, invoice_number, payment_url, expired_date, amount_idr, credits_granted }

2. Klien redirect ke payment_url (halaman Checkout DOKU)

3. DOKU -> POST /v1/webhooks/doku  (lihat plan/security.md bagian 6)
   Go verifikasi signature HMAC-SHA256 -> cabang status -> update orders
   transaction.status SUCCESS -> insert credits_ledger (delta +credits, reason topup,
                                 ref_type order, ref_id orders.id)
                              -> update profiles.credits

4. Klien poll GET /v1/orders/{id} sampai status settlement
```

Status order: `pending | settlement | expire`. Transisi hanya ke arah itu. Dari `settlement`
tidak boleh kembali.

Retry saat membuat order: pakai `request_id` yang SAMA. DOKU membalas `409 Conflict` dengan
body respons yang sama, dan itu dibaca sebagai "order sudah dibuat", bukan gagal. `409`
diteruskan sebagai `200` dengan order yang sudah ada.

### 2.4 Aturan

- Order `pending` tidak dibayar sampai expire: status jadi `expire`, tidak ada kredit.
- Expire tanpa sweeper. Dua jalan: webhook DOKU dengan `order.status = ORDER_EXPIRED`, atau `GET /v1/orders/{id}` yang cek status ke DOKU saat order masih `pending`.
- Sumber kedaluwarsa adalah `expired_date` dari respons DOKU (format `yyyyMMddHHmmss`, UTC+7). Docs DOKU menyuruh memakainya "to set the expiry order on merchant side". Nilai ini disimpan di `orders`, bukan dihitung ulang oleh timer sendiri.
- Webhook bisa datang berkali-kali. Proses idempoten lewat `request_id` unik; `request_id` yang sudah diproses dibalas `200` tanpa perubahan (lihat `security.md` bagian 6 langkah 8).
- Retry webhook DOKU sampai +12 jam. Karena itu umur `Request-Timestamp` TIDAK dipakai sebagai gerbang tolak.
- Refund kredit (kasus gagal sistem) lewat `credits_ledger` dengan `reason = refund`. Tidak ada refund uang di sistem ini. Refund uang ditangani manual lewat dashboard DOKU.

## 3. Pemakaian kredit

### 3.1 Saat run dibuat

Dalam satu transaksi Postgres:

1. `SELECT credits FROM profiles WHERE id = $user FOR UPDATE`
2. Jika `credits < 1` -> `402 insufficient_credits`, transaksi batal.
3. Insert `plan_runs` (status `queued`, `credits_charged = 1`).
4. Insert `credits_ledger` (delta `-1`, reason `run`, ref_type `plan_run`, ref_id run.id).
5. Update `profiles.credits = credits - 1`.
6. Commit. Balas `202`.

### 3.2 Saat run gagal

Dalam satu transaksi:

1. Update `plan_runs.status = failed`.
2. Insert `credits_ledger` (delta `+1`, reason `refund`, ref_type `plan_run`, ref_id run.id).
3. Update `profiles.credits`.

Refund hanya sekali per run. Dijamin unique index `uq_ledger_one_refund_per_run` (`erd.dbml`), ditambah cek status sebelum refund.

### 3.3 Saat run selesai

Tidak ada transaksi kredit tambahan. Kredit sudah terpotong di 3.1.

### 3.4 Saat regenerate (charge dan refund)

Charge, dalam satu transaksi (pola sama dengan 3.1):

1. `SELECT credits FROM profiles WHERE id = $user FOR UPDATE`
2. Jika `credits < 1` maka `402 insufficient_credits`, transaksi batal.
3. Insert `credits_ledger` (delta `-1`, reason `regenerate`, ref_type `artifact`, ref_id `artifacts.id`). Simpan `id` baris ini sebagai `charge_id`.
4. Update `profiles.credits = credits - 1`.
5. Commit, lalu jalankan regenerate.

Refund jika regenerate gagal, dalam satu transaksi:

1. Insert `credits_ledger` (delta `+credits_charged` dari baris charge, reason `refund`, ref_type `ledger`, ref_id `charge_id`).
2. Update `profiles.credits`.

Idempoten: `uq_ledger_one_refund_per_charge` (`erd.dbml`) menolak refund kedua untuk `charge_id` yang sama. Konflik berarti refund sudah ada, abaikan tanpa error.

## 4. Ledger

Aturan `credits_ledger` (reason, delta, ref) ada di `plan/erd.md`.

## 5. Checklist DOKU yang perlu disiapkan

Ini bagian untuk akun DOKU yang akan didaftarkan:

- [ ] Daftar akun DOKU, aktifkan mode Sandbox untuk pengembangan.
- [ ] Ambil `Client-Id` dan `Secret Key` Sandbox dari DOKU Back Office.
- [ ] Daftarkan `Notification URL` di Back Office ke `https://paluketok.web.id/v1/webhooks/doku`.
      Catatan: DOKU melarang domain ngrok, URL dengan authentication, localhost, dan VPN.
      Untuk lokal, pakai tunnel yang bukan ngrok (misalnya cloudflared atau localhost.run).
- [ ] Catat path notifikasi yang terdaftar. Path ini harus SAMA PERSIS dengan `DOKU_NOTIFICATION_PATH` di `server/.env`, karena dipakai sebagai `Request-Target` saat verifikasi signature.
- [ ] Aktifkan metode pembayaran yang diinginkan di Back Office.
- [ ] Catat durasi pembayaran (`payment.payment_due_date`, default 60 menit) dan simpan nilainya di `gaps.md` setelah dicek.
- [ ] Simpan `DOKU_CLIENT_ID` dan `DOKU_SECRET_KEY` di `server/.env`. Tidak ada key DOKU di sisi klien.
- [ ] Base URL: sandbox `https://api-sandbox.doku.com`, produksi `https://api.doku.com`.
- [ ] Untuk produksi: verifikasi bisnis, aktivasi akun, ganti kredensial ke produksi.
- [ ] Uji webhook dengan transaksi sandbox sampai `SUCCESS` dan buktikan kredit bertambah SEKALI saja walau webhook dikirim dua kali.
- [ ] Uji retry: kirim ulang webhook manual dari Back Office, pastikan tidak ada kredit tambahan.
- [ ] Uji `409`: buat order dua kali dengan `request_id` sama, pastikan tidak ada order dobel.
