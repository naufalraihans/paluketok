# Changelog perubahan

Urut baru di atas. Satu baris per perubahan; yang besar punya berkasnya sendiri
di folder ini.

## 2026-10-18

- **[0010](./0010-plan-dipindah-ke-docs.md)** - Folder `plan/` dipindah ke `docs/plan/`. 108 rujukan di kode, README, dan dokumen ikut diperbaiki. Alasan: semua dokumen dalam satu tempat.
- **[0009](./0009-slot-judge-tidak-dikunci.md)** - Slot "Judge JEV" jadi "AI Judge"; model tidak lagi dikunci `jev-latest` dan dibaca dari database. Alasan: `judge` adalah peran, bukan produk.
- **[0008](./0008-aturan-em-dash-dan-identitas.md)** - Identitas "Ketok Magic" + larangan em dash, ditegakkan di prompt dan di server. Alasan: keluaran model bocor sebagai nama model dan tanda pisah panjang.
- **[0007](./0007-render-markdown-chat.md)** - Balasan chat dirender sebagai markdown (`marked` + daftar putih tag). Alasan: `**tebal**` tampil apa adanya.
- **[0006](./0006-indikator-mengetik.md)** - Pesan pengguna tampil seketika + titik berdenyut selama menunggu. Alasan: kotak chat tampak kosong sampai balasan datang.

## 2026-10-13 dan sebelumnya

Riwayat Fase 5-8 ada di `docs/plan/changelog.md` (format Keep a Changelog).
Berkas itu dibekukan bersama folder planning dan tidak ditambahi lagi.
