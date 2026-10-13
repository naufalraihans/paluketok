# Kandidat: Midtrans -> DOKU (PaluKetok)

Status: **USULAN, belum diterapkan.** Tidak ada file di `plan/` yang diubah. Semua nomor baris di bawah menunjuk kondisi file saat dokumen ini ditulis (2026-10-12). Sumber fakta DOKU: 168 halaman markdown di `C:/Users/naufal raihan s/AppData/Local/Temp/doku/`, nama file = URL dengan `/` -> `__`.

Prioritas keputusan: (1) kesederhanaan, (2) integritas uang, (3) konsistensi.

## 0. Ringkasan

- Port `port.Payment` tetap **satu**, dependensinya hanya `entity` + stdlib. Adapter `adapter/payment/midtrans` diganti `adapter/payment/doku`. Usecase tidak berubah bentuk, hanya nama field. Aturan dependency `architecture.md` bagian 2 tidak perlu diubah isinya, hanya daftar larangan (baris 26) dan daftar alasan (baris 44) yang menyebut nama vendor.
- Perubahan kontrak terbesar, berurutan menurut risiko: **verifikasi signature webhook** (SHA512 atas 4 nilai yang dikirim di body -> HMAC-SHA256 atas 5 komponen termasuk body), **redirect alih-alih popup** (`snap_token` -> `payment.url`), **`FAILED` wajib diabaikan**, **cek status tidak boleh dipanggil sebelum 60 detik**.
- Jaring pengaman uang yang sudah ada (`uq_ledger_one_topup_per_order`, lock `FOR UPDATE`, transisi `WHERE status='pending'`) **tetap dipakai apa adanya**. Ganti gateway tidak melemahkan satu pun.
- Yang paling mungkin jadi sumber bug: **`request_id` harus disimpan** dan **`Request-Target` harus persis path URL notifikasi** (dan path itu harus sama dengan yang didaftarkan di Back Office). Dua hal ini tidak ada padanannya di desain Midtrans.
- Kolom `snap_token` (setelah rename `payment_url`) menjadi **tak terpakai** di BFF: `payment.url` dipakai sekali untuk redirect. Usul: drop, bukan simpan. Satu kolom, satu variabel, satu keputusan.

## 1. Tabel perubahan per file

Kolom **Isi baru** = teks pengganti yang siap dipakai. Kolom **Catatan** hanya untuk yang perlu penjelasan.

### 1.1 `main.md`

| Baris | Isi lama (potongan) | Isi baru | Catatan |
|---|---|---|---|
| 6 | `Supabase (PostgreSQL + Auth), Midtrans, OpenAPI, DBDiagram` | `Supabase (PostgreSQL + Auth), DOKU Checkout, OpenAPI, DBDiagram` | |
| 22 | `\| auth_billing.md \| login, profil, Midtrans, pemakaian kredit \|` | `\| auth_billing.md \| login, profil, DOKU, pemakaian kredit \|` | |

### 1.2 `architecture.md`

| Baris | Isi lama | Isi baru | Catatan |
|---|---|---|---|
| 15 | `- **Midtrans = top-up kredit.** Webhook masuk ke Go.` | `- **DOKU Checkout = top-up kredit.** Webhook masuk ke Go. Redirect ke payment.url.` | |
| 26 | `SDK LLM/JEV/Midtrans \|` | `SDK LLM/JEV/DOKU \|` | Isi aturan dependency tidak berubah, hanya label vendor |
| 44 | `Ganti Midtrans, JEV, atau LLM cukup ganti adapter.` | `Ganti DOKU, JEV, atau LLM cukup ganti adapter.` | |

Tambahan usul (baris baru, di bagian 1 Keputusan): `- **Ganti gateway = tukar adapter `adapter/payment/<vendor>`.** Kontrak gateway tidak pernah masuk `port` atau `usecase`; yang boleh muncul di usecase hanya `entity.Order`.`

### 1.3 `auth_billing.md` — bagian yang berubah

| Baris | Isi lama | Isi baru |
|---|---|---|
| 3 | `top-up kredit lewat Midtrans` | `top-up kredit lewat DOKU Checkout` |
| 52 | `## 2. Billing (Midtrans Snap)` | `## 2. Billing (DOKU Checkout)` |
| 70-84 | blok ``` alur order ``` (Snap + webhook + poll) | blok di bawah (bagian 2 dokumen ini) |
| 86 | `Status order: pending \| settlement \| expire \| deny.` | tetap, ditambah: `settlement` diisi saat `transaction.status = SUCCESS` |
| 91 | `Expire tanpa sweeper. Dua jalan: webhook Midtrans dengan transaction_status = expire, atau GET /v1/orders/{id} yang cek status ke Midtrans` | `Expire tanpa sweeper. Dua jalan: notifikasi EXPIRED dari DOKU, atau GET /v1/orders/{id} yang memanggil Check Status DOKU saat order masih pending. Check Status tidak dipanggil lebih cepat dari 60 detik setelah pembayaran.` |
| 92 | `Durasi expire diatur di Snap Preference (dashboard Midtrans)` | `Durasi expire ditetapkan di request (payment.payment_due_date, menit, default 60) DAN notifikasi EXPIRED diaktifkan di Back Office (Settings > Checkout Appearance > Expired Settings). Nilai yang dipakai sistem = payment.payment_due_date.` |
| 93 | `idempoten lewat midtrans_order_id unik` | `idempoten lewat invoice_number unik dan lock `SELECT ... FOR UPDATE` pada baris orders` |
| 94 | `Refund uang ditangani manual lewat dashboard Midtrans.` | `Refund uang ditangani manual lewat DOKU Back Office. Cancel Order API TIDAK dipakai untuk refund; hanya membatalkan order belum dibayar dan tidak didukung untuk semua kanal.` |
| 144 | `## 5. Checklist Midtrans yang perlu disiapkan` | `## 5. Checklist DOKU yang perlu disiapkan` |
| 146-157 | isi checklist | ganti dengan daftar di bagian 3 dokumen ini |

Baris 152 (`Atur payment expire time di Snap Preference`) dihapus dari checklist, karena durasi expire sekarang ditentukan kode, bukan dashboard.

### 1.4 `auth_billing.md` — bagian 2.3 pengganti (siap tempel)

```
1. POST /v1/orders  { package_id }
   Go: buat orders (status pending, invoice_number unik "PK-<random>")
       invoice_number <= 30 karakter (batas aman: kartu kredit 30, umum 64)
       request_id = UUID baru, disimpan di orders.request_id
   Go -> DOKU Checkout API (POST /checkout/v1/payment, signature request)
      -> response.payment.url
   Go -> klien { order_id, invoice_number, payment_url, amount_idr, credits_granted }

2. Klien redirect ke payment_url (halaman Checkout DOKU; tanpa JS klien, tanpa client key)

3. DOKU -> POST /v1/webhooks/doku  (lihat plan/security.md bagian 6)
   Go verifikasi signature HMAC-SHA256 -> cocokkan amount -> update orders
   transaction.status = SUCCESS -> insert credits_ledger (delta +credits, reason topup,
                                    ref_type order, ref_id orders.id)
                               -> update profiles.credits
   transaction.status = FAILED  -> DIABAIKAN (pembeli bisa ganti metode di halaman Checkout)

4. Klien poll GET /v1/orders/{id} sampai status settlement atau expire
```

### 1.5 `auth_billing.md` — bagian 5 pengganti (checklist)

```
- [ ] Daftar merchant DOKU, pakai environment Sandbox.
- [ ] Ambil Client-Id dan Secret Key dari DOKU Back Office (Integration > API Keys).
      Secret = sandbox, dipakai untuk signature. Tidak ada client key di browser.
- [ ] Set Notification URL di Back Office ke https://<domain>/v1/webhooks/doku.
      Catat PATH-nya (/v1/webhooks/doku): dipakai sebagai Request-Target saat verifikasi.
      Untuk lokal: pakai tunnel, dan pastikan path override sama dengan path terdaftar.
- [ ] Aktifkan Expired Notification (Settings > Checkout Appearance > Expired Settings).
- [ ] Aktifkan kanal pembayaran yang diinginkan (minimal QRIS dan satu e-wallet).
- [ ] Pastikan tidak ada simbol pada invoice_number (dibatasi alfanumerik + '-'; syarat KKI).
- [ ] Simpan DOKU_CLIENT_ID, DOKU_SECRET_KEY, DOKU_BASE_URL, DOKU_NOTIFICATION_PATH di server/.env.
- [ ] Untuk produksi: verifikasi bisnis, aktifkan akun, ganti Client-Id + Secret ke produksi
      (api-sandbox.doku.com -> api.doku.com).
- [ ] Uji webhook transaksi sandbox sampai transaction.status = SUCCESS dan kredit bertambah sekali.
- [ ] Uji webhook dengan signature salah -> 401.
- [ ] Uji kirim notifikasi SUCCESS dua kali -> tetap satu grant.
- [ ] Uji notifikasi bertimestamp lebih dari 5 menit -> ditolak.
```

### 1.6 `security.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 10 | `... LLM key / Midtrans key)-->  [Supabase Postgres, JEV, LLM, Midtrans]` | `... LLM key / DOKU secret key)-->  [Supabase Postgres, JEV, LLM, DOKU]` |
| 14 | `[Midtrans webhook] --(signature SHA512)--` | `[DOKU webhook] --(signature HMAC-SHA256)--` |
| 24 | `\| Midtrans webhook \| Tidak sampai signature valid \| Lihat bagian 6. \|` | `\| Webhook DOKU \| Tidak sampai signature valid \| Lihat bagian 6. \|` |
| 69 | `\| run_id, artifact_id, order_id \| UUID valid, cek format sebelum query \|` | tambah baris: `\| invoice_number \| dari server (generate sendiri), alfanumerik + '-', maks 30; tidak pernah dari klien \|` |
| 94-109 | `## 6. Webhook Midtrans` + urutan 1-8 | diganti seluruhnya (bagian 4 dokumen ini) |
| 137 | `\| MIDTRANS_SERVER_KEY \| server/.env \| Go saja \|` | `\| DOKU_SECRET_KEY \| server/.env \| Go saja \|` |
| 139 | `\| SUPABASE_ANON_KEY \| web/.env \| Browser, boleh publik \|` | tetap |
| 140 | `\| MIDTRANS_CLIENT_KEY \| web/.env \| Browser (Snap), boleh publik \|` | `\| (dihapus) \| - \| DOKU Checkout tidak butuh key di browser \|` |
| 170 | `\| Webhook menolak signature salah dan nominal tidak cocok. \|` | `\| Webhook menolak signature salah, timestamp basi, dan nominal tidak cocok. \|` |
| 175 | `\| Tidak ada log yang memuat secret. \|` | tetap |

### 1.7 `security.md` — bagian 6 pengganti (siap tempel)

```
## 6. Webhook DOKU

Endpoint: `POST /v1/webhooks/doku`. Tidak memakai JWT, dilindungi signature.

Urutan wajib:

1. Baca body mentah, batas 16 KB. Body mentah ini dipakai apa adanya untuk Digest.
2. Ambil header `Client-Id`, `Request-Id`, `Request-Timestamp`, `Signature`.
3. Tolak lebih awal bila `Client-Id` != `DOKU_CLIENT_ID` -> `401`.
4. Tolak bila `Request-Timestamp` lebih dari 5 menit dari waktu server (Clock). Alasan: tanpa
   batas umur, webhook sah yang direkam bisa diputar ulang selamanya. 5 menit dipilih supaya
   retry DOKU (30 menit, 6 jam, 12 jam) tetap masuk lewat Check Status (langkah 8), bukan lewat
   body lama.
5. Bandingkan `Signature` dengan `HMACSHA256=` + base64(HMAC-SHA256(secret, komponen)), komponen
   tepat 5 baris, dipisah "\n", TANPA "\n" di akhir:
      Client-Id:<dari header>
      Request-Id:<dari header>
      Request-Timestamp:<dari header>
      Request-Target:<DOKU_NOTIFICATION_PATH, konstanta konfigurasi>
      Digest:<base64(SHA256(body mentah))>
   Bandingkan memakai `hmac.Equal` (constant time) setelah memotong prefix. Tidak cocok = `401`,
   tanpa perubahan data.
6. Parse body sebagai JSON longgar: `json.Decoder` tanpa `DisallowUnknownFields` (DOKU boleh
   menambah field kapan saja). Ini pengecualian sadar terhadap aturan body JSON di bagian 4,
   yang berlaku untuk input dari klien, bukan untuk webhook.
7. Cocokkan `order.amount` (angka JSON) dengan `orders.amount_idr`. Beda = tolak dan catat.
8. Cari `orders` berdasarkan `order.invoice_number`. Tidak ditemukan = `404`, tanpa perubahan data.
9. `SELECT ... FOR UPDATE` pada baris order dalam transaksi yang sama dengan grant.
   - `transaction.status = FAILED` -> DIABAIKAN (pembeli boleh ganti metode pembayaran di halaman
     Checkout; status kegagalan bukan status order). Balas `200`, tidak ada perubahan.
   - `transaction.status = SUCCESS` -> transisi `UPDATE orders SET status='settlement'
     WHERE id=$1 AND status='pending'`. Rows affected 0 = tidak ada grant.
   - Grant hanya setelah transisi berhasil: insert `credits_ledger` (`+credits_granted`,
     reason `topup`, ref_type `order`, ref_id `orders.id`), lalu update `profiles.credits`.
     Jaring terakhir tetap `uq_ledger_one_topup_per_order`.
   - Order yang sudah `settlement` -> `200` langsung, tanpa grant ulang.
10. Balas `200` hanya setelah commit. Balas 2xx sesegera mungkin: DOKU mengirim ulang 3 kali
    (30 menit, 6 jam, 12 jam) bila tidak menerima 2xx.

Notifikasi EXPIRED ditangani lewat urutan yang sama (status `expire`, tanpa kredit).
Notifikasi `FAILED` (status NO/final di mapping DOKU) tidak pernah menurunkan order `settlement`
dan tidak pernah memberi kredit.

Verifikasi silang ke Check Status DOKU (langkah tambahan, bukan pengganti langkah 9):
- Dipanggil HANYA bila order masih `pending` setelah webhook SUCCESS diproses, atau saat
  `GET /v1/orders/{id}`.
- TIDAK PERNAH lebih cepat dari 60 detik setelah pembayaran (batas keras dari DOKU).
- GET /orders/v1/status/{invoice_number}, signature tanpa Digest, timeout 20 detik, tanpa auto-retry.
- Hasil dipetakan: SUCCESS -> `settlement`, EXPIRED -> `expire`, PENDING/TIMEOUT/REDIRECT ->
  tidak ada perubahan, FAILED -> tidak ada perubahan (diabaikan, lihat langkah 9).
- Gagal panggil = `503`, biarkan DOKU mengirim ulang. Jangan memutuskan dari body saja.
```

### 1.8 `openapi.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 24 | `\| POST \| /v1/orders \| buat order top-up (Midtrans Snap) \| user \|` | `\| POST \| /v1/orders \| buat order top-up (DOKU Checkout) \| user \|` |
| 26 | `\| POST \| /v1/webhooks/midtrans \| callback Midtrans \| signature \|` | `\| POST \| /v1/webhooks/doku \| callback DOKU \| signature \|` |
| 88 | `"payment": "midtrans"` | `"payment": "doku"` |
| 109 | `{ "order_id": "uuid", "midtrans_order_id": "PK-...", "snap_token": "...", "amount_idr": 25000, "credits_granted": 10 }` | `{ "order_id": "uuid", "invoice_number": "PK-...", "payment_url": "https://sandbox.doku.com/checkout-link-v2/...", "amount_idr": 25000, "credits_granted": 10 }` |
| 130 | `- Webhook Midtrans: verifikasi signature_key (security.md bagian 6).` | `- Webhook DOKU: verifikasi header Signature HMAC-SHA256 (security.md bagian 6).` |

Catatan untuk baris 88: `projects.stack.payment` adalah pilihan stack **untuk project yang digenerate** (whitelist enum di `security.md` bagian 4), bukan gateway PaluKetok. Menambah nilai `doku` ke whitelist enum adalah perubahan terpisah; di sini hanya contoh di dokumen yang diganti.

### 1.9 `structure.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 26 | `│   │   └── payment/midtrans/    # Snap create, status check, verify signature` | `│   │   └── payment/doku/        # Checkout create (payment.url), check status, verify signature` |
| 70 | `LLM 180 detik, JEV 30 detik, Midtrans 20 detik` | `LLM 180 detik, JEV 30 detik, DOKU 20 detik` |

Tetap berlaku di baris 70: semua adapter keluar memakai `http.Client` dengan timeout eksplisit, tanpa `http.DefaultClient`. DOKU 20 detik sama dengan angka Midtrans, jadi tidak ada angka baru.

### 1.10 `workflow.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 3 | `checklist Midtrans (auth_billing.md bagian 5)` | `checklist DOKU (auth_billing.md bagian 5)` |
| 13 | `3. **Midtrans sandbox.** Ikuti checklist auth_billing.md bagian 5, kecuali baris expire (dicek setelah dashboard dibuka).` | `3. **DOKU sandbox.** Ikuti checklist auth_billing.md bagian 5.` (pengecualian baris expire tidak perlu lagi: expire diatur di request) |
| 14 | `Selesai: MIDTRANS_SERVER_KEY dan MIDTRANS_CLIENT_KEY ada, mode sandbox.` | `Selesai: DOKU_CLIENT_ID dan DOKU_SECRET_KEY ada, sandbox, Notification URL terdaftar dengan path /v1/webhooks/doku.` |
| 53 | `15. **adapter/midtrans**: Snap create, status check (timeout 20 detik, tanpa retry), verifikasi signature SHA512 dengan hmac.Equal.` | `15. **adapter/doku**: Checkout create (POST /checkout/v1/payment -> payment.url), Check Status (timeout 20 detik, tanpa retry, tidak lebih cepat dari 60 detik setelah bayar), verifikasi signature HMAC-SHA256 dengan hmac.Equal.` |
| 54 | `Selesai: test signature salah ditolak. Test pakai transaksi sandbox sampai settlement.` | tambah: `Test notifikasi SUCCESS dikirim dua kali -> satu grant.` |
| 65 | `21. **Webhook Midtrans** (POST /v1/webhooks/midtrans). Urutan langkah 1 sampai 8 di security.md 6.` | `21. **Webhook DOKU** (POST /v1/webhooks/doku). Urutan langkah 1 sampai 10 di security.md 6.` |

### 1.11 `erd.dbml` — tabel `orders`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 119 | `midtrans_order_id text [unique, not null, note: 'format PK-<random>']` | `invoice_number text [unique, not null, note: 'format PK-<random>, alfanumerik + dash, maks 30 karakter']` |
| 123 | `snap_token text` | `request_id text [unique, not null, note: 'UUID v4, idempotency key DOKU; dikirim sebagai header Request-Id']` |

Usul tambahan (1 kolom, bukan 2):

```
  payment_url text [note: 'response.payment.url; ditampilkan sekali untuk redirect']
```

Alasan `request_id` **not null** dan **unique**: DOKU memakai `Request-Id` sebagai idempotency key. Tanpa menyimpannya, retry create-order dari sisi kita tidak bisa mengirim request yang identik, dan order kedua bisa tercipta untuk topup yang sama. `unique` juga mencegah dua order berbagi satu Request-Id. `orders.id` (uuid) sendiri tidak bisa dipakai sebagai `Request-Id` karena nilai itu harus sudah diketahui **sebelum** request dikirim; UUID acak terpisah lebih sederhana daripada mencoba memakai kembali `orders.id`.

Alasan drop `snap_token`: nama kolom menyebut vendor, dan nilainya hanya dipakai sekali saat redirect. Menyimpan `payment_url` adalah bentuk termudah; kalau mau paling minimal, kolomnya dihapus dan URL dikembalikan langsung dari create-order tanpa menyimpan apa pun. Pilihan kedua sedikit lebih murah tetapi menghilangkan kemampuan melihat ulang URL saat debugging.

### 1.12 `erd.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 40 | `orders: pending ke settlement, expire, atau deny.` | tetap; tambah baris: `FAILED dari gateway bukan status order dan tidak disimpan (lihat security.md 6).` |
| 41 | `Lock webhook: SELECT order FOR UPDATE dalam transaksi yang sama dengan grant.` | tetap, tidak berubah |
| 42 | `Transisi order hanya lewat UPDATE orders SET status = $new WHERE id = $1 AND status = 'pending'.` | tetap, tidak berubah |
| 43 | `Late settlement: order expire yang kemudian dibayar boleh pindah ke settlement HANYA jika status API Midtrans mengonfirmasi settlement.` | `Late settlement: order expire yang kemudian dibayar boleh pindah ke settlement HANYA jika Check Status DOKU mengonfirmasi SUCCESS, dan dipanggil tidak lebih cepat dari 60 detik setelah pembayaran.` |
| 44 | `Webhook hanya diproses untuk midtrans_order_id yang ada di orders. Tidak dikenal = 404` | `Webhook hanya diproses untuk invoice_number yang ada di orders. Tidak dikenal = 404` |

### 1.13 `gaps.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 43 | `- Durasi expire order di Snap Preference Midtrans (auth_billing.md 2.4, 5).` | dihapus dari daftar "Dicek saat setup". Diganti di bagian "Ditunda": `- Batas umur Request-Timestamp webhook (usulan 5 menit, security.md 6) perlu dikonfirmasi bukan preferensi DOKU.` |

### 1.14 `changelog.md`

| Baris | Isi lama | Isi baru |
|---|---|---|
| 13 | `Supabase (PostgreSQL + Auth), Midtrans, OpenAPI, DBDiagram.` | `Supabase (PostgreSQL + Auth), DOKU, OpenAPI, DBDiagram.` |
| 16 | `Klaim expiry Snap 24 jam dihapus, jadi checklist setup.` | jangan disentuh (catatan sejarah) |
| 18 | `Timeout eksplisit untuk semua klien HTTP keluar (LLM, JEV, Midtrans).` | `Timeout eksplisit untuk semua klien HTTP keluar (LLM, JEV, DOKU).` |
| 33 | `- Payment: Stripe diganti Midtrans.` | jangan disentuh (catatan sejarah) |
| 55 | `Timeout adapter punya pemilik tunggal di structure.md (LLM 180, JEV 30, Midtrans 20 detik).` | `... (LLM 180, JEV 30, DOKU 20 detik).` |

Tambahan: satu entri `Changed` baru di `[Unreleased]`, bukan penulisan ulang sejarah: `- Payment: Midtrans diganti DOKU Checkout (keputusan user). Desain penuh: plan/candidates/doku-arch.md.`

### 1.15 `prd.md`, `roles.md`, `deploy.md`

| File:baris | Isi lama | Isi baru | Catatan |
|---|---|---|---|
| `prd.md:14` | `Stack contoh: Go, Svelte, PSQL + Supabase, Midtrans, Supabase Auth, UI minimalis.` | `... DOKU, Supabase Auth, ...` | |
| `prd.md:41` | `- Stripe (diganti Midtrans).` | biarkan apa adanya (riwayat) atau hapus; kalau dipertahankan tambahkan baris baru `- Midtrans (diganti DOKU).` | Milik user |
| `roles.md:26` | `\| webhook Midtrans \| - \| - \| - (tanpa JWT, dilindungi signature, security.md 6) \|` | `\| webhook DOKU \| - \| - \| - (tanpa JWT, dilindungi signature, security.md 6) \|` | |
| `deploy.md:34` | `- Outbound wajib bisa ke: Supabase pooler, Midtrans, endpoint LLM (base URL admin), dan gateway JEV.` | `- Outbound wajib bisa ke: Supabase pooler, api.doku.com, endpoint LLM (base URL admin), dan gateway JEV.` | Tambahkan juga: DOKU kirim notifikasi **masuk** ke /v1/webhooks/doku; pastikan tidak ada rule firewall/proxy yang memblokir POST dari luar |

Total: 42 baris di 14 file. Hanya `plan/candidates/doku-arch.md` yang dibuat; tidak ada file `plan/` lain yang disentuh.

## 2. Perbedaan kontrak: Midtrans -> DOKU

| Aspek | Midtrans (sekarang di plan) | DOKU (terverifikasi dari docs lokal) | Dampak |
|---|---|---|---|
| Endpoint buat order | Snap API, respons `token` | `POST https://api-sandbox.doku.com/checkout/v1/payment` (prod `https://api.doku.com/...`), respons `response.payment.url` | Adapter berubah; cara klien bayar berubah |
| Cara klien bayar | Popup Snap di browser dengan `snap_token` + client key | **Redirect** ke `payment.url`. Tidak ada JS/SDK di klien | Hapus `snap_token` + `MIDTRANS_CLIENT_KEY`; tambah `payment_url` |
| Signing request kita | `Authorization: Basic base64(serverkey:)`, tanpa signature | Header `Client-Id`, `Request-Id`, `Request-Timestamp` (UTC ISO8601 `Z`), `Signature: HMACSHA256=<base64>`. Komponen 5 baris: Client-Id, Request-Id, Request-Timestamp, Request-Target, Digest. `Digest` = base64(SHA256(body JSON)), **hanya POST**. Secret dari Back Office | Logika signature baru, dipakai di 3 tempat (create, check status, cancel) |
| Signing webhook masuk | `SHA512(order_id + status_code + gross_amount + server_key)`, dibandingkan dengan `signature_key` di body | HMAC-SHA256 atas 5 komponen (4 dari header + Digest body), header `Signature: HMACSHA256=` | Ditulis ulang; lihat 1.7 |
| ID order | `order_id` kita + `midtrans_order_id` (PK-<random>) | `order.invoice_number` (satu-satunya ID), maks 64, **30 bila kartu kredit aktif**, tanpa simbol bila ada KKI | Pakai satu kolom `invoice_number`, maks 30; hapus konsep dua-ID |
| Body webhook | Datar: `order_id`, `transaction_status`, `gross_amount`, `signature_key` | Bersarang: `order.invoice_number`, `order.amount`, `transaction.status`, `service.id`, `channel.id`, `acquirer.id`, `transaction.original_request_id`. Field baru bisa ditambah kapan saja -> parse longgar | Parser baru; **wajib** longgar |
| Nilai status | `capture`, `settlement`, `pending`, `deny`, `cancel`, `expire` | `SUCCESS`, `FAILED`, `PENDING`, `EXPIRED`, `REFUNDED`, `TIMEOUT`, `REDIRECT` (+ `order.status`: `ORDER_GENERATED`, `ORDER_EXPIRED`, `ORDER_RECOVERED`, `CANCELLED`) | Tabel mapping baru |
| Status gagal | `deny`/`cancel` bisa jadi status order | **`FAILED` wajib DIABAIKAN untuk Checkout**: pembeli boleh ganti metode pembayaran. `FAILED` juga bukan final (Final Status = NO) | Aturan baru, penting: jangan pernah tandai order `deny` dari webhook FAILED |
| Cek status | `GET /v2/{order_id}/status` | `GET https://api-sandbox.doku.com/orders/v1/status/{invoice_number}`, signature GET **tanpa Digest** | Adapter berubah |
| Kapan cek status boleh | kapan saja | **Tidak boleh sebelum 60 detik setelah pembayaran selesai** | Aturan waktu baru; kalau dilanggar bisa dapat `Not Found`/PENDING |
| Expire | Diatur di Snap Preference (dashboard); webhook `transaction_status=expire` | `payment.payment_due_date` (menit) **di request**, default 60; notifikasi EXPIRED otomatis, tapi **harus diaktifkan di Back Office** (Settings > Checkout Appearance > Expired Settings). Response mengembalikan `expired_date` (`yyyyMMddHHmmss`, UTC+7) | Expire jadi milik kode + konfigurasi; `expired_at` diisi dari `expired_date` |
| Retry webhook | Midtrans kirim ulang bila tidak 2xx | Harus balas `2xx`; bila tidak, **3 kali**: +30 menit, +6 jam, +12 jam. Retry manual tersedia di Back Office (Integration > HTTP Notification) | Sama-sama butuh idempotensi; jendela retry lebih panjang |
| Duplikat | Ada | Ada, eksplisit: "might occasionally receive the same event more than once" | Idempotensi wajib (sudah ada jaring DB) |
| Rate limit status | tidak disebut | 1 request / 1 detik / invoice_number | Tambah `ponytail` note di adapter |
| Idempotency request | tidak ada | `Request-Id` = idempotency key. Request identik diulang -> response sama dengan HTTP `409 Conflict` | **Simpan `request_id`**; perlakukan 409 sebagai "sudah dibuat", bukan error |
| Cancel order | Snap `cancel` | `POST /checkout/v3/cancellations` — **hanya VA dan QRIS**, harus diaktifkan di Back Office, tidak bisa untuk kartu/e-wallet/CS | Tidak dipakai (di luar scope) |
| Kredensial | server key + client key | Client-Id + Secret Key (Back Office). Tidak ada key di browser | `security.md` bagian 9 |
| Timeout adapter | 20 detik | 20 detik (tetap) | Tidak berubah |
| Refund | manual di dashboard | manual di Back Office. Refund API ada, di luar scope | Tidak berubah |

## 3. Keputusan arsitektur (ADR ringkas)

**ADR-1: Port tetap satu, adapter diganti.**
`port.Payment` tetap satu interface dengan dependensi `entity` + stdlib. `adapter/payment/midtrans` diganti `adapter/payment/doku`. Usecase tidak pernah mengimpor nama vendor (`architecture.md` bagian 2). Bukti aturan ini sudah ada di baris 26 dan 44; yang berubah hanya label.
Bentuk port yang cukup (tiga metode, tidak lebih):

```
CreateOrder(ctx, entity.Order) (entity.Order, error)   // mengisi invoice_number/payment_url
GetStatus(ctx, invoiceNumber string) (string, error)     // "SUCCESS" | "PENDING" | "EXPIRED" | ...
VerifyNotification(header http.Header, rawBody []byte) error
```

`VerifyNotification` menerima `http.Header`; ini membuat paket `port` menyentuh `net/http`. Dua pilihan: (a) terima `map[string]string` yang dibangun di `delivery` sebelum memanggil usecase — port tetap bebas `net/http`; (b) terima `http.Header` dan terima bahwa header HTTP adalah tipe data, bukan transport. Usul: **(a)**, supaya aturan "usecase tidak tahu `net/http`" tetap harfiah. Verifikasi signature sebaiknya dijalankan di usecase (yang punya akses `Clock`), bukan di handler: handler tetap tidak mengambil keputusan.

**ADR-2: Verifikasi signature di usecase, bukan di adapter.**
Signature webhook adalah keputusan keamanan uang, bukan detail transport. Adapter hanya menyediakan primitif (hitung HMAC dari komponen mentah). Store verifikasi di usecase memakai `Clock` untuk batas umur timestamp.

**ADR-3: Satu ID order.**
`invoice_number` menggantikan `midtrans_order_id`. `orders.id` (uuid) tetap PK internal dan tetap yang dipakai `credits_ledger.ref_id`. `invoice_number` adalah yang dikirim ke DOKU dan yang dicari saat webhook masuk. Dua lapis ini mencegah ID gateway (yang kita buat sendiri, format `PK-<random>`) menjadi PK, sehingga migrasi gateway berikutnya tidak menyentuh ledger.

**ADR-4: `request_id` unik + not null, disimpan di `orders`.**
Tanpa ini, idempotensi create-order hilang. 409 dari DOKU diperlakukan sebagai "sudah pernah dibuat" -> ambil `payment_url` yang tersimpan.

**ADR-5: Webhook 2xx cepat, keputusan uang tetap di server.**
Balas `2xx` setelah commit (sama seperti aturan Midtrans sekarang di `security.md` 107). Jangan menjadikan webhook pintu masuk satu-satunya: `GET /v1/orders/{id}` tetap bisa memanggil Check Status, dengan hormat pada batas 60 detik.

**ADR-6: `FAILED` bukan status order.**
`orders.status` tetap `pending | settlement | expire | deny`. `FAILED` tidak pernah menulis ke `orders`. `deny` tetap ada di enum tetapi hanya diisi manual (mis. sistem menolak order), bukan dari gateway. Ini menghindari order yang sudah `settlement` diturunkan oleh notifikasi `FAILED` dari percobaan bayar sebelumnya.

**ADR-7: Tidak memakai Cancel Order API, tidak memakai DOKU Jokul/Snap, tidak memakai kartu kredit di v1.**
Checkout non-Snap cukup. Cancel hanya VA/QRIS dan butuh aktivasi — di luar scope. Kalau kartu kredit nanti diaktifkan, batas `invoice_number` turun ke 30; karena kita langsung membatasi 30 sekarang, tidak ada perubahan skema saat itu terjadi.

## 4. Kolom DB (tepat, jangan diubah dulu)

`orders`:

| Aksi | Kolom | Tipe | Atribut | Alasan |
|---|---|---|---|---|
| RENAME | `midtrans_order_id` -> `invoice_number` | text | unique, not null | Nama vendor keluar dari skema; ID tunggal |
| DROP | `snap_token` | text | - | Tidak ada padanannya; Checkout redirect, bukan popup |
| ADD | `request_id` | text | unique, not null | Idempotency key DOKU (header `Request-Id`) |
| ADD (usul) | `payment_url` | text | - | Simpan `response.payment.url` untuk redirect dan debug |
| TETAP | `id`, `user_id`, `package_id`, `amount_idr`, `credits_granted`, `status`, `paid_at`, `expired_at`, `created_at`, `updated_at` | - | - | Tidak terpengaruh vendor |
| TETAP | index `idx_orders_user_created`, `idx_orders_status`, plus unique di `invoice_number` | - | - | Sama |

`credits_ledger`: **tidak berubah.** `uq_ledger_one_topup_per_order`, `actor_id`, `idempotency_key` tetap.

Catatan penting (jangan diabaikan):
- `invoice_number` **tidak bisa** memakai `orders.id`; karena unique-nya sudah ada di `invoice_number` dari Midtrans, migrasi hanya rename + tambah `request_id`. Order lama (kalau ada data sandbox) harus diberi `request_id` sebelum `not null` ditegakkan, atau migrasi diizinkan default.
- `expired_at` diisi dari `response.payment.expired_date` (format `yyyyMMddHHmmss`, UTC+7) hasil create-order, bukan dari `payment_due_date` yang dikirim.
- `status` enum **tidak berubah**: `pending | settlement | expire | deny`. `FAILED` bukan nilai kolom ini.

## 5. Pertanyaan terbuka (jangan ditebak)

1. **`request_id` vs invoice_number saat Check Status.** Docs bilang Check Status menerima `invoice_number` ATAU `Request-Id`. Konfirmasi mana yang berlaku untuk order Checkout v1 (bukan SNAP/VA), supaya tidak ada tebak-tebakan saat order belum punya transaksi.
2. **`order.status` (`ORDER_GENERATED`/`ORDER_EXPIRED`/`ORDER_RECOVERED`) hanya untuk merchant yang daftar setelah Desember 2024** (docs: "contact our support team to activate this for existing merchant"). Akun baru kita dapat ini atau tidak? Kalau tidak, `GET /v1/orders/{id}` bisa mengembalikan `Not Found` sebelum kanal dipilih.
3. **Request-Timestamp expiry.** Berapa toleransi yang DOKU pakai untuk menolak request basi? Tidak disebut di docs. Usulan 5 menit. Bisa salah di dua arah: terlalu ketat -> webhook sah ditolak; terlalu longgar -> replay window.
4. **`expired_date` ada di response Checkout non-SNAP?** Terlihat di Full Response (`yyyyMMddHHmmss`, UTC+7). Perlu dikonfirmasi muncul juga di Basic Response, karena Basic Response yang akan kita pakai.
5. **Perbedaan status Check Status vs notifikasi** untuk order yang dibatalkan/di-recover: nilai `CANCELLED` dan `ORDER_RECOVERED` muncul di report dashboard; apakah muncul di API? Kalau ya, mapping `expire` harus menangani keduanya.
6. **Bentuk body notifikasi EXPIRED untuk Checkout** (field mana yang menjamin ini notifikasi order, bukan notifikasi kanal) tidak ada contoh di docs lokal. Perlu contoh nyata dari sandbox.
7. **`payment_method_types` wajib diisi atau tidak** untuk Checkout non-SNAP. Full Request memuatnya; Basic Request tidak. Kalau boleh kosong, semua kanal yang aktif di Back Office tampil — itu yang diinginkan.
8. **Rate limit Check Status** 1 req/detik/invoice_number: apakah 429/4xx, dan apakah retry otomatis dilarang total? Adapter akan tanpa auto-retry, tetapi perlu tahu kode balasannya.
9. **Apakah `Signature` DOKU memakai `Request-Target` dengan path yang terdaftar di Back Office, atau path dari request yang kita kirim?** Untuk webhook docs jelas (path URL notifikasi). Untuk create-order, `Request-Target` = path endpoint DOKU yang kita hit (`/checkout/v1/payment`). Perlu dipastikan tidak ada stripping query string saat proxy.
10. **Notifikasi untuk kanal kartu kredit** membawa `transaction.status` selain `SUCCESS`, mis. `VOIDED`/`REFUNDED` (contoh ada di docs). Perlu diputuskan apakah `REFUNDED` diabaikan juga (usul: ya, diabaikan; refund uang manual di Back Office).

## 6. Catatan eksekusi

- Perubahan menyentuh `plan/` saja. Implementasi (`server/`) belum ada, jadi tidak ada kode aplikasi yang perlu diubah sekarang.
- Kalau DOKU dipakai juga untuk `projects.stack.payment` (pilihan stack project hasil generate), itu keputusan terpisah dan tidak diusulkan di sini.
- Batas 30 karakter `invoice_number` dipilih sekarang supaya aktivasi kartu kredit di masa depan tidak memicu migrasi skema.
