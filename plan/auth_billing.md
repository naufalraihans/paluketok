# Auth dan Billing - PaluKetok

Alur login, sinkron profil, top-up kredit lewat Midtrans, dan pemakaian kredit.

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

### 1.3 Endpoint `/v1/me`

- Hanya membaca profil milik `claims.sub`.
- Mengembalikan `id`, `username`, `full_name`, `plan_tier`, `is_admin`, `credits`.
- Tidak mengembalikan data user lain.

### 1.4 Role

Sesuai `plan/roles.md`:

- `anon`: landing dan login.
- `user`: project dan run milik sendiri.
- `admin`: baca semua, atur tier dan kredit secara manual.

Admin tidak bisa mengubah `is_admin` user lain lewat API publik. Perubahan `is_admin` hanya lewat SQL migrasi atau langsung di Supabase dashboard.

## 2. Billing (Midtrans Snap)

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
   Go: buat orders (status pending, midtrans_order_id unik "PK-<random>")
   Go -> Midtrans Snap API (server key)  -> snap_token
   Go -> klien { order_id, midtrans_order_id, snap_token, amount_idr, credits_granted }

2. Klien buka Snap popup dengan snap_token (client key)

3. Midtrans -> POST /v1/webhooks/midtrans  (lihat plan/security.md bagian 6)
   Go verifikasi signature -> cek status ke Midtrans API -> update orders
   status settlement -> insert credits_ledger (delta +credits, reason topup, ref_type order, ref_id orders.id)
                     -> update profiles.credits

4. Klien poll GET /v1/orders/{id} sampai status settlement
```

Status order: `pending | settlement | expire | deny`. Transisi hanya ke arah itu. Dari `settlement` tidak boleh kembali.

### 2.4 Aturan

- Order yang tidak dibayar sampai expire: status jadi `expire`, tidak ada kredit.
- Webhook bisa datang berkali-kali. Proses idempoten lewat `midtrans_order_id` unik dan cek status sebelum grant.
- Refund kredit (kasus gagal sistem) lewat `credits_ledger` dengan `reason = refund`. Tidak ada refund uang di sistem ini. Refund uang ditangani manual lewat dashboard Midtrans.

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

Refund hanya sekali per run. Cegah dengan cek status sebelumnya dan constraint unik opsional pada `(ref_id, reason)` untuk `refund`.

### 3.3 Saat run selesai

Tidak ada transaksi kredit tambahan. Kredit sudah terpotong di 3.1.

## 4. Ringkasan `credits_ledger`

| reason | delta | ref_type | ref_id |
|---|---|---|---|
| `topup` | + | `order` | `orders.id` |
| `run` | - | `plan_run` | `plan_runs.id` |
| `refund` | + | `plan_run` | `plan_runs.id` |
| `regenerate` | - | `artifact` | `artifacts.id` |
| `grant` (manual admin) | + atau - | `none` | null |

## 5. Checklist Midtrans yang perlu disiapkan

Ini bagian untuk akun Midtrans yang akan didaftarkan:

- [ ] Daftar akun Midtrans, pilih mode Sandbox untuk pengembangan.
- [ ] Ambil `Client Key` dan `Server Key` Sandbox dari dashboard.
- [ ] Isi Payment Notification URL di dashboard ke `https://<domain>/v1/webhooks/midtrans`. Untuk lokal, pakai tunnel (misalnya ngrok atau cloudflared).
- [ ] Aktifkan metode pembayaran yang diinginkan (minimal QRIS dan e-wallet untuk sandbox).
- [ ] Simpan `MIDTRANS_SERVER_KEY` di `server/.env`, `MIDTRANS_CLIENT_KEY` di `web/.env`.
- [ ] Pilih environment: `MIDTRANS_IS_PRODUCTION=false` untuk sandbox.
- [ ] Untuk produksi: verifikasi bisnis, aktivasi akun, dan ganti key ke Production.
- [ ] Uji webhook dengan transaksi sandbox sampai `settlement` dan kredit bertambah sekali saja.
- [ ] Uji webhook dengan signature salah, harus ditolak.

## 6. Keputusan terbuka

| Topik | Status |
|---|---|
| Metode login selain email | belum, ditunda |
| Pajak dan invoice | belum, di luar scope awal |
| Kebijakan expire order | perlu dicek nilai default Midtrans (Snap) |
