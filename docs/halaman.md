# PaluKetok — Daftar Halaman & Fungsinya

Dokumen ini inventaris **seluruh halaman** yang ada di aplikasi PaluKetok, apa fungsinya,
apa yang bisa diklik, dan mana yang masih placeholder.

Sumber: dibaca langsung dari kode di `web/src/routes/`, bukan dari mockup.
Terakhir diperiksa: 16 Oktober 2026.

---

## Ringkasan cepat

| # | URL | Status | Fungsi |
|---|-----|--------|--------|
| 1 | `/` | Nyata | Landing page / halaman pemasaran |
| 2 | `/login` | Nyata | Masuk akun |
| 3 | `/register` | Nyata | Daftar akun baru |
| 4 | `/reset` | Nyata | Lupa password / set password baru |
| 5 | `/auth/callback` | Nyata | Penukar kode tautan email jadi sesi |
| 6 | `/projects` | Nyata | Dashboard: daftar proyek milik pengguna |
| 7 | `/projects/{slug}` | Nyata | Ringkasan satu proyek, URL enak dibaca. UUID lama tetap jalan |
| 8 | `/billing` | Nyata | Saldo kredit, beli paket, riwayat order |
| 9 | `/admin` | Nyata | Konsol admin (5 tab) |

**Total: 9 halaman. Semuanya sudah nyata, tidak ada yang masih dummy di level halaman.**

Yang masih placeholder bukan halamannya, tapi **menu di sidebar dashboard** (lihat bagian
[Placeholder yang belum ada halamannya](#placeholder-yang-belum-ada-halamannya)).

---

## 1. `/` — Landing Page

**Fungsi:** halaman pemasaran. Menjelaskan produk ke pengunjung yang belum punya akun.

**Isi blok, urut dari atas:**

| Urutan | Komponen | Isi |
|--------|----------|-----|
| 1 | `SiteNav` | Navbar: logo, Produk, Cara kerja, Untuk siapa, Masuk/Daftar |
| 2 | `HeroSection` | Judul besar "Rancang dulu. Bangun dengan benang merah." + 2 tombol CTA |
| 3 | `ProblemSection` | id `#cara-kerja` — masalah yang dipecahkan |
| 4 | `FeatureSection` | id `#produk` — fitur produk |
| 5 | `TraceabilitySection` | jejak dari kebutuhan ke keputusan teknis |
| 6 | `ProcessSection` | alur kerja |
| 7 | `PersonaSection` | id `#untuk-siapa` — siapa yang cocok memakai |
| 8 | `SecuritySection` | soal proteksi/keamanan |
| 9 | `FinalCtaSection` | ajakan terakhir |
| 10 | `SiteFooter` | footer 4 kolom: brand, Produk, Sumber, Kontak |

**Bisa diklik:**

- `Produk` → anchor `#produk` (gulir ke FeatureSection)
- `Cara kerja` → anchor `#cara-kerja` (gulir ke ProblemSection)
- `Untuk siapa` → anchor `#untuk-siapa` (gulir ke PersonaSection)
- `Mulai rancanganmu →` → `/register`
- `Jelajahi contoh proyek ↗` → tautan luar
- `Masuk` / `Daftar` di navbar (kalau belum login)

**Kalau sudah login**, navbar berubah menampilkan nama akun + jumlah kredit, tombol
`Keluar`, dan tombol `Dashboard`. Itu perilaku normal navbar situs, bukan bug.

---

## 2. `/login` — Masuk

**Fungsi:** masuk ke akun yang sudah ada.

**Dua cara masuk:**

1. **Email + password** — isi form, tekan `Masuk`
2. **Tautan masuk (magic link)** — tombol teks "atau kirim tautan masuk via email",
   kirim tautan ke email, tautan membawa ke `/auth/callback`

**Bisa diklik:**

- `Daftar` → `/register`
- `Lupa password` → `/reset`

**Kalau sudah login** dan membuka `/login`, otomatis dialihkan ke `/projects`.

---

## 3. `/register` — Daftar

**Fungsi:** membuat akun baru.

**Alur:** isi form → Supabase kirim email konfirmasi → pengguna klik tautan →
`/auth/callback` → sesi aktif → masuk `/projects`.

Pengiriman email konfirmasi lewat **Brevo** (SMTP kustom Supabase), pengirim
`noreply@paluketok.web.id`.

**Bisa diklik:** `Masuk` → `/login`

**Kalau sudah login** dan membuka `/register`, otomatis dialihkan ke `/projects`.

---

## 4. `/reset` — Lupa Password

**Fungsi:** memulihkan akses akun. Dua mode dalam satu halaman:

| Mode | Kapan | Yang dilakukan |
|------|-------|----------------|
| 1 | Belum punya sesi reset | Masukkan email → kirim tautan reset |
| 2 | Sesi reset aktif (URL bawa `?code=`) | Set password baru |

Setelah password baru tersimpan, dialihkan ke `/login`.

---

## 5. `/auth/callback` — Penukar Kode

**Fungsi:** halaman teknis, bukan untuk dibuka manual.

Supabase mengirim tautan konfirmasi/reset yang berisi `?code=...`. Halaman ini menukar
kode itu jadi sesi login (`exchangeCodeForSession`), lalu mengalihkan ke `/projects`.

**Yang perlu diketahui:** kalau halaman ini muncul sebagai error `localhost:3000`, berarti
**Site URL di Supabase belum diganti** ke `https://paluketok.web.id`. Itu masalah konfigurasi
Supabase, bukan bug aplikasi.

---

## 6. `/projects` — Dashboard Daftar Proyek

**Fungsi:** halaman kerja utama. Menampilkan semua proyek milik pengguna yang login.

**Isi:**

- **Sidebar kiri** (260px): brand, pemilih proyek, daftar proyek (kalau >1), menu 10 item, badge profil di bawah
- **Topbar**: breadcrumb, chip status, jumlah kredit, tombol `Top-up` → `/billing`
- **Kartu metrik**: jumlah proyek, siap/ready, draf, arsip — semuanya data live dari `/v1/projects`
- **Daftar proyek**: kartu per proyek, masing-masing punya tombol `Buka proyek`
- **Banner pipeline**: penjelasan alur planning + tombol buat proyek
- **Form buat proyek**: nama + brief, tersimpan ke server

**Bisa diklik:**

- `Buka proyek` → `/projects/{slug}`
- `Top-up` → `/billing`
- `Buat proyek baru` → buka form
- Item sidebar → lihat catatan di bawah

**Data:** semua dari API `/v1`. Kalau muncul "Server tidak terjangkau", cek dulu apakah
kamu benar-benar sudah login — pesan itu dipakai untuk beberapa jenis kegagalan.

---

## 7. `/projects/{slug}` — Ringkasan Proyek

**Fungsi:** halaman detail satu proyek. Tempat menjalankan generate enam dokumen planning.

**URL memakai slug, bukan UUID:** `/projects/testing-jembut`, bukan
`/projects/fae59609-…-44970f`. Slug diturunkan dari nama proyek saat dibuat dan
**beku sejak itu** — mengganti nama proyek tidak mengubah URL, jadi tautan yang
sudah dibagikan tidak pernah mati.

**Dua bentuk tetap diterima:**

| Dibuka lewat | Yang terjadi |
| --- | --- |
| `/projects/{slug}` | langsung tampil (bentuk resmi) |
| `/projects/{uuid}` | address bar ditukar ke slug otomatis, data tidak dimuat ulang |

Karena itu tautan lama yang terlanjur dibagikan tetap hidup. Unik **per pemilik**:
dua orang boleh sama-sama punya `warung-kopi`, masing-masing mendapat proyeknya sendiri.

**Enam tahap planning, berurutan:**

```
prd → erd → openapi → roles → security → structure
```

Setiap tahap menghasilkan satu artefak. Biaya: **1 run = 1 kredit**, regenerate satu tahap = 1 kredit.

**Isi:**

- **Sidebar**: nama proyek, progres (mis. `2/6`)
- **Menu tahap progresif**: item muncul hanya kalau artefaknya sudah ada
  (`siap`), yang belum ada ditandai `segera` dan tidak bisa dipencet. Baca
  `GET /v1/projects/{ref}/artifacts` (artefak terbaru per tahap), jadi tetap
  benar walau dibuka di perangkat/browser baru.
- **Topbar**: breadcrumb, waktu simpan, kolaborator, tombol bagikan
- **Header ringkasan**: monogram, nama proyek, status, deskripsi, persentase kesiapan
- **Enam tile tahap**: hijau = artefak siap, abu = belum ada
- **Tombol `Mulai generate (1 kredit)`** — hanya muncul kalau belum ada run
- **Tombol `Regenerate`** per tahap — muncul kalau run sudah `done`

**Bisa diklik:**

- `Mulai generate` → jalankan run baru
- `Regenerate` → ulang satu tahap
- `Coba lagi` → muncul kalau ada error

**Catatan penting:** halaman ini **tidak diprerender** (id proyek tak diketahui saat build).
Root layout memaksa `prerender = true` untuk semua halaman; rute ini menimpanya dengan
`export const prerender = false` di `+page.ts`, dan SvelteKit melayaninya lewat SPA
fallback `200.html`. Tanpa dua hal itu, nginx jatuh ke `/index.html` dan halaman ini salah
menampilkan landing page — bukan halaman proyek sama sekali.

Sudah diperbaiki 16 Okt 2026. Terverifikasi: `/projects/fae59609-…` menampilkan proyek
"Testing Jembut" beserta tombol generate, bukan landing page.

---

## 8. `/billing` — Saldo & Pembelian

**Fungsi:** lihat saldo kredit, beli paket, lihat riwayat order.

**Isi:**

- **Kartu saldo**: jumlah kredit + nama paket (`free`)
- **Kartu paket**: nama paket, harga, tombol beli
- **Riwayat order** (sesi ini): tabel Invoice / Nominal / Kredit / Status
- **Banner pesan**: hijau kalau lunas, merah kalau gagal

**Satu-satunya paket:** `Paket Starter` — Rp 25.000 · 10 kredit.

**Bisa diklik:**

- `Beli paket Starter` → buat order, lalu diarahkan ke halaman pembayaran
- `Bayar` → muncul di baris order kalau status masih `pending`
- `Cek status` → tanya ulang status order ke server

**Pembayaran:** lewat Midtrans Snap (kunci sandbox saat ini). Setelah pembayaran berhasil,
Midtrans mengirim webhook ke `/v1/webhooks/midtrans`, server menambah kredit, dan kredit
di navbar ikut berubah.

**Kalau belum login**, halaman ini mengalihkan ke `/login`.

---

## 9. `/admin` — Konsol Admin

**Fungsi:** konsol operasional. Hanya bisa dibuka kalau akunmu `is_admin`.

**Alamat:** `https://admin.paluketok.web.id/admin` (subdomain terpisah)

**Lima tab:**

| Tab | Fungsi |
|-----|--------|
| `Projects` | semua proyek dari semua pengguna |
| `Runs` | semua run generate + statusnya |
| `Users` | daftar pengguna, ubah paket, tambah kredit |
| `Decisions` | catatan keputusan JEV |
| `AI` | pengaturan provider AI per slot |

**Sidebar di halaman ini berfungsi** — tiap item memilih tab, bukan berpindah halaman.

**Kalau bukan admin:** server membalas `403`, halaman menolak akses.

---

# Placeholder yang belum ada halamannya

Ini **temuan penting**. Sidebar dashboard menampilkan **10 menu**, tapi hanya **2** yang
menuju halaman yang benar-benar ada.

| Menu sidebar | Href sekarang | Halaman tujuan | Status |
|--------------|---------------|----------------|--------|
| Daftar proyek | `/projects` | ✓ ada | **Berfungsi** |
| Billing / Top-up | `/billing` | ✓ ada | **Berfungsi** |
| Run berjalan | `/projects` | ✗ tidak ada | Placeholder |
| Dokumen · PRD | `/projects` | ✗ tidak ada | Placeholder |
| Model data · ERD | `/projects` | ✗ tidak ada | Placeholder |
| Kontrak OpenAPI | `/projects` | ✗ tidak ada | Placeholder |
| Role access | `/projects` | ✗ tidak ada | Placeholder |
| Proteksi | `/projects` | ✗ tidak ada | Placeholder |
| Struktur & prototipe | `/projects` | ✗ tidak ada | Placeholder |
| Keputusan JEV | `/projects` | ✗ tidak ada | Placeholder |

Delapan menu itu **kerangka dari mockup Fase 9** — dipasang supaya tampilan cocok dengan
desain, bukan karena halamannya sudah dibangun. Semuanya menunjuk ke `/projects` supaya
tidak jadi tautan mati. Jadi kalau dipencet, kamu "dilempar balik" ke halaman yang sama.

Selain itu, **dropdown "pemilih proyek"** di sidebar (yang ada ikon panah atas-bawah) juga
belum berfungsi — masih elemen tampilan saja.

---

# Rute backend yang melayani halaman

Dari `server/`:

| Endpoint | Fungsi |
|----------|--------|
| `GET /v1/me` | profil + kredit + paket pengguna |
| `GET POST /v1/projects` | daftar / buat proyek |
| `GET /v1/projects/{id}` | detail satu proyek |
| `POST /v1/projects/{id}/runs` | mulai generate (potong kredit) |
| `GET /v1/runs/{id}` | status + progres run |
| `GET /v1/runs/{id}/artifacts` | artefak hasil generate |
| `GET /v1/artifacts/{id}` | satu artefak |
| `POST /v1/artifacts/{id}/regenerate` | ulang satu tahap |
| `GET /v1/decisions` | catatan keputusan |
| `GET POST /v1/orders` | order pembelian |
| `GET /v1/orders/{id}` | status satu order |
| `POST /v1/webhooks/midtrans` | notifikasi pembayaran dari Midtrans |
| `GET /v1/admin/*` | lima endpoint konsol admin |

Semua endpoint kecuali `/healthz` dan webhook butuh header `Authorization: Bearer <token>`.
Tanpa itu server membalas `401`.

---

# Catatan teknis

**Struktur berkas frontend:**

```
web/src/routes/      9 halaman (satu folder = satu rute)
web/src/lib/components/   28 komponen tampilan
web/src/lib/mock/    data statis kerangka halaman (bukan data pengguna)
web/src/lib/api.ts   semua panggilan ke backend /v1
web/src/lib/auth.svelte.ts   sesi Supabase
```

**Soal `mock/`:** data di folder ini **bukan** data pengguna. Fungsinya mengisi teks
statis yang tidak berubah (judul, label, kerangka blok). Data pengguna selalu datang dari
`/v1`. Jadi kalau kamu melihat teks contoh di halaman, itu dari `mock/`.

**Model rute:** SvelteKit dengan `adapter-static`.

| Jenis rute | Perlakuan | Contoh |
|------------|-----------|--------|
| Statis | diprerender jadi HTML nyata saat build | `/`, `/login`, `/projects`, `/billing` |
| Dinamis | **tidak** diprerender, dilayani SPA fallback `200.html` | `/projects/{id}` |

Pengaturan dinamis ini dipegang tiga tempat sekaligus, dan ketiganya harus benar:

| Berkas | Perannya |
|--------|----------|
| `web/vite.config.ts` | `adapter({ fallback: '200.html' })` — bikin SPA shell |
| `web/src/routes/projects/[id]/+page.ts` | `export const prerender = false` — tolak prerender |
| `web/src/routes/+layout.ts` | `prerender = true` untuk sisanya + catatan kenapa bukan fungsi |
| nginx | `try_files $uri $uri.html $uri/ /200.html` |

**Jebakan yang sudah pernah kejadian:** kalau `fallback` kosong atau rute `[id]` ikut
diprerender, nginx tak menemukan berkas untuk id asli lalu jatuh ke `/index.html` — dan
halaman detail proyek menampilkan **landing page**. Gejalanya: URL benar, isi salah.

Catatan tambahan: `prerender` **tidak bisa** berupa fungsi di `+layout.ts`. SvelteKit
meng-clone konfigurasi ke worker build, dan fungsi gagal di-clone
(`DataCloneError: could not be cloned`). Penolakan rute dinamis harus lewat `+page.ts`
rute itu sendiri.

**Alamat produksi:**

| Layanan | Alamat |
|---------|--------|
| Aplikasi | `https://paluketok.web.id` |
| Admin | `https://admin.paluketok.web.id` |
| Email pengirim | `noreply@paluketok.web.id` |
| Registrar + DNS | IDCloudHost |
| VPS | `103.119.237.29` (SSH port 20257) |
