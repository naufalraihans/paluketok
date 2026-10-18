# Perubahan

Semua perubahan yang menggeser rencana, kontrak, atau arah produk dicatat di sini.
Bukan catatan commit: yang dibaca orang berikutnya adalah **keputusan dan alasannya**,
lengkap dengan berkas mana yang kena.

Changelog lama (riwayat Fase 5-8, format Keep a Changelog) tetap di
`docs/plan/changelog.md`. Berkas itu bagian dari planning beku dan tidak
ditambahi lagi. Semua perubahan setelah folder planning diarsipkan masuk ke sini.

## Cara mengisi

Satu berkas per perubahan besar di `docs/update/changelog/NNNN-judul-singkat.md`,
nomor urut empat digit. Setiap berkas wajib memuat:

| Bagian | Isi |
| --- | --- |
| Status | `rencana` / `sedang dikerjakan` / `selesai` / `dibatalkan` |
| Tanggal | tanggal keputusan, bukan tanggal commit |
| Kenapa | masalah nyata yang memicu, bukan keinginan abstrak |
| Yang berubah | daftar berkas + apa yang berubah di masing-masing |
| Yang TIDAK berubah | batas scope, supaya tidak melebar |
| Bukti | perintah dan hasil nyata, bukan klaim |
| Sisa | yang sengaja ditunda, dengan alasan |

Untuk perubahan kecil (satu berkas, satu baris) cukup satu baris di
`docs/update/changelog/ringkas.md` dengan tanggal + ringkasan.

## Mengapa dipisah dari `docs/plan/`

`docs/plan/` adalah **rencana yang sudah dibekukan**: menggambarkan produk
sebagaimana dirancang, bukan sebagaimana ia sekarang. Menambahi berkas itu
setiap kali kode berubah akan merusak nilainya sebagai titik acuan.

`docs/update/` adalah lapisan di atasnya: apa yang bergeser dari rencana, dan
kenapa. Kalau dua berkas berbeda pendapat, `docs/update/` yang lebih baru
menang, dan perbedaannya harus disebut di sana.
