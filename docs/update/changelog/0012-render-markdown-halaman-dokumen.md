# 0012 - Halaman dokumen menampilkan markdown sebagai teks polos

- **Status**: selesai
- **Tanggal**: 2026-10-19
- **Commit**: `7b3ad40`

## Kenapa

Dilaporkan pemilik saat membuka `paluketok.web.id/projects/pos-warung/prd`:
isi dokumen tampil seperti teks polos. Judul tidak lebih besar, tidak ada
jarak antar bagian, daftar tidak bernomor.

## Sebabnya

Bukan parser. `marked` sudah benar mengubah `#` jadi `<h1>`, `##` jadi `<h2>`,
dan `1.` jadi `<ol>`; ini diverifikasi dengan menjalankan `marked` langsung di
atas isi artefak nyata.

Sebabnya satu kelas CSS: `prose-sm`. Itu kelas plugin
`@tailwindcss/typography`, dan **plugin itu tidak terpasang** di proyek ini
(Tailwind 4.3.3, tanpa plugin typography). Kelasnya jadi hantu: HTML benar,
tetapi tidak ada CSS yang membuatnya tampil sebagai judul, daftar, atau tabel.
Hasilnya terlihat seperti teks polos.

Kelas ini hanya muncul di satu tempat, halaman dokumen baru
(`/projects/[id]/[stage]`). Balasan chat tidak terpengaruh karena sudah memakai
kelas `.md`.

## Yang berubah

`web/src/routes/projects/[id]/[stage]/+page.svelte`, satu baris:

```
prose-sm   ->   .md
```

`.md` sudah ada di `src/app.css` (35 aturan) dan sudah dipakai balasan chat.
Isinya mencakup `p`, `h1`-`h6`, `ul`, `ol`, `table`, `blockquote`, `hr`, `code`.
Jadi tidak ada CSS baru dan tidak ada dependensi baru.

## Verifikasi (bukan klaim)

| Pemeriksaan | Hasil |
| --- | --- |
| `marked` atas isi artefak nyata | `<h1>`, `<h2>`, `<ol>` benar |
| `npm run check` | 0 error, 0 warning |
| `npm run build` | sukses |
| Bundle lokal | `class="md rounded-xl border ... p-6 max-w-none"` |
| Bundle prod | `md rounded-xl` ada di `nodes/9.CtHiwp9j.js` |
| Bundle prod | `prose-sm` sisa 0 berkas |
| Prod healthz | 200 |

CI `39201377116` sukses 54 detik.

## Catatan

Pola ini gampang terulang: kelas plugin Tailwind yang tidak terpasang tidak
menghasilkan error apa pun, hanya diam-diam tidak berfungsi. Sebelum memakai
kelas dari plugin, pastikan plugin-nya ada di `package.json`. Untuk proyek ini
plugin typography tidak dipakai, jadi pakai `.md`.
