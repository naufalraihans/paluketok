Lanjutkan desain PaluKetok di file ini. Kamu sudah membuat halaman-halaman ini di Page 1:

- PaluKetok · Landing page
- PaluKetok · Ringkasan proyek
- PaluKetok · Struktur dan prototipe
- PaluKetok · Model data ERD
- PaluKetok · Dokumen PRD
- PaluKetok · User Settings

Pakai gaya, komponen, warna, dan tipografi yang sama persis dengan halaman-halaman itu. Jangan bikin design system baru. Jangan ubah halaman yang sudah ada.

Sekarang buat halaman-halaman berikut, masing-masing frame 1440px, auto-layout, dan komponen yang konsisten dengan yang sudah ada:

1. Login / Register: email + password, tombol OAuth (placeholder), pesan error.
2. Dashboard / Daftar project: kartu project dengan status (draft/ready/archived), tombol buat project baru, saldo kredit di topbar.
3. Buat project baru: form nama, brief (textarea), stack (fe/be/db/payment).
4. Run berjalan: progres 6 tahap (prd, erd, openapi, roles, security, structure), current_stage, indikator loading, tombol batal.
5. Kontrak OpenAPI: daftar endpoint (method + path), panel detail request/response.
6. Role Access: matriks izin (resource x role: anon/user/admin), penanda enforcement 3 lapis.
7. Rencana Proteksi: trust boundary, validasi input, checklist sebelum rilis.
8. Keputusan JEV: daftar keputusan per tahap, badge gate (EXECUTE / GATHER_EVIDENCE / ESCALATE), confidence.
9. Billing / Top-up: saldo, paket Starter (10 kredit / Rp 25.000), riwayat kredit (ledger), status order.
10. Admin dashboard: semua project dan run lintas user, ringkasan penggunaan.
11. Admin log JEV: semua keputusan lintas user, filter per tahap dan gate.

Untuk setiap halaman app, sertakan state empty, loading, dan error.

Isi konten tiap halaman ambil dari dokumen plan PaluKetok (PRD, ERD, OpenAPI, roles, security, structure, auth_billing). Jangan mengarang fitur di luar itu.
