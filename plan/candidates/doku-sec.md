# Security (DEFENSIVE) - Integrasi DOKU

Sumber: docs DOKU lokal (168 halaman) + `candidates/doku-arch.md`. Temuan yang tidak
terbukti dari docs sudah dibuang (lihat catatan di akhir).

## CRITICAL

### SEC-D1. Notifikasi EXPIRED tidak lewat `transaction.status`
`transaction.status` hanya `SUCCESS` atau `FAILED` (notification/best-practice:62).
Status EXPIRED ada di `order.status` = `ORDER_GENERATED | ORDER_EXPIRED | ORDER_RECOVERED`
(order-status-for-checkout-page). Rancangan awal menganggap EXPIRED datang sebagai
`transaction.status`, sehingga order bisa tertinggal `pending` selamanya.
Perbaikan: handler cabang pada `order.status`, bukan `transaction.status`.

### SEC-D2. Retry webhook DOKU (sampai +12 jam) ditolak oleh jendela freshness signature
Retry DOKU: 3 percobaan pada 30 menit, 5,5 jam, 12 jam (notification/retry-notification).
Kalau verifikasi menuntut `Request-Timestamp` segar (mis. 5 menit), retry ke-1 dibuat DOKU
pada menit ke-30 GAGAL verifikasi, padahal itu notifikasi sah.
Perbaikan: DELETE jendela freshness sebagai gerbang tolak. Anti-replay pakai
idempotensi (`invoice_number` + `request_id` unik), bukan umur timestamp.
Umur timestamp dilog saja pada tahap pertama.

## HIGH

### SEC-D3. Signature adalah SATU-SATUNYA gerbang
Docs melarang URL notifikasi pakai authentication, VPN, localhost, dan ngrok
(notification/setup-notification-url). Tidak ada IP allowlist untuk notifikasi.
Konsekuensi: kalau verifikasi signature lemah atau dilewati, siapa pun di internet bisa
mengirim notifikasi palsu dan menaikkan kredit sendiri.
Perbaikan: verifikasi WAJIB, tanpa jalur bypass, tanpa flag dev yang bisa lolos ke prod.
Digest dihitung dari body MENTAH (bukan hasil parse ulang), bandingkan `hmac.Equal`.

### SEC-D4. Tinggi/lebar jendela replay tanpa cek `Request-Id` berulang
Docs tidak menyatakan apakah `Request-Id` webhook berubah antar retry.
Perbaikan: simpan `request_id` webhook yang sudah diproses (unique), tolak yang berulang.
Ini menutup replay tanpa bergantung pada timestamp, yang memang tidak boleh dipakai (SEC-D2).

## MEDIUM

### SEC-D5. `expired_date` dari DOKU diabaikan
`response.payment.expired_date` (yyyyMMddHHmmss, UTC+7) wajib ada, dan docs menyuruh
memakainya "to set the expiry order on merchant side". Rancangan awal pakai timer sendiri.
Dampak: order bisa dianggap pending padahal sudah kedaluwarsa, atau sebaliknya.
Perbaikan: simpan `expired_date`, pakai sebagai satu-satunya kebenaran kedaluwarsa.

### SEC-D6. 409 idempotency harus diperlakukan sebagai sukses, bukan error
Docs idempotency: request ulang dengan Request-Id sama dan body sama -> HTTP 409 Conflict,
body respons sama. Kalau 409 diperlakukan sebagai gagal, klien mengulang terus lalu
order dobel dibuat dengan Request-Id baru.
Perbaikan: 409 dibaca sebagai "sudah dibuat", ambil order yang ada, balas 200.

## LOW

### SEC-D7. `invoice_number` tidak boleh berasal dari klien
Harus dibuat server, alfanumerik + `-`, dan dicek unik. Docs: maks 64 (30 bila kartu kredit).
Menerima `invoice_number` dari klien membuka order palsu dan tumbukan nomor.

### SEC-D8. `payment.url` tidak boleh dipakai ulang setelah tampil
URL itu membuka halaman bayar siapa pun yang memegangnya. Simpan untuk redirect, jangan
kembalikan lagi di `GET /v1/orders/{id}`.

## Pertanyaan terbuka (docs tidak menyebut, JANGAN diasumsikan aman)
1. Toleransi umur `Request-Timestamp` untuk notifikasi. Angka 10 detik yang sempat muncul
   TIDAK terbukti di docs (grep kosong) dan sudah dibuang. Karena retry sampai +12 jam,
   rancangan ini tidak memakai umur timestamp sebagai gerbang.
2. Apakah `Request-Id` webhook berubah antar retry.
3. Apakah ada IP allowlist notifikasi di sisi DOKU (docs tidak menyebut).
4. Panduan rotasi secret DOKU (docs tidak menyebut).

## Temuan yang DIBUANG setelah verifikasi
- "Toleransi +-10 detik" (klaim dari response-code) - grep kosong, tidak terbukti.
