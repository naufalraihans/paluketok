# PRD: Sistem Planning AI (PaluKetok)

## 1. Tujuan

Membangun sistem yang membuat planning proyek software lebih cepat, teliti, dan akurat.
Output sistem: dokumen planning lengkap sebelum coding dimulai.

Sistem ini BUKAN aplikasi toko online. PaluKetok hanya nama project contoh untuk menguji sistem.

## 2. Contoh Input (hanya contoh)

User minta: website toko online sederhana dengan fitur lengkap.

Stack contoh: Go, Svelte, PSQL + Supabase, DOKU, Supabase Auth, UI minimalis.

Dari input itu, sistem harus menghasilkan aspek planning developer:

- ERD / DBMS design (DBDiagram)
- Kontrak API (OpenAPI)
- Konkurensi (goroutine)
- PRD
- Aturan API
- Role access
- Proteksi
- Sistem auth
- Struktur file/folder
- Tech stack

## 3. Stack

Ringkas di `main.md`. Keputusan teknis di `architecture.md`. Deploy di `deploy.md`.

## 4. Aturan Kerja

- AI wajib pakai JEV saat ragu atau potensi halusinasi.
- Keputusan teknis di planning harus lewat gate JEV (EXECUTE / GATHER_EVIDENCE / ESCALATE).
- Planning ditulis di folder `plan/`, dimulai dari `main.md` sebagai index.

## 5. Di Luar Scope (sekarang)

- Stripe (diganti Midtrans, lalu diganti DOKU).
- Eksekusi kode proyek contoh.
- Changelog otomatis.
