# 0007 - Balasan chat dirender sebagai markdown

- **Status**: selesai
- **Tanggal**: 2026-10-18
- **Commit**: `68edbcd`

## Kenapa

Balasan model tampil sebagai teks mentah: `**Asumsi**` muncul apa adanya,
begitu juga daftar dan tabel. Penyebabnya `ProjectChat.svelte` menampilkan
`{m.content}` langsung dengan hanya `whitespace-pre-wrap`.

## Yang berubah

| Berkas | Perubahan |
| --- | --- |
| `web/package.json` | Tambah dependensi `marked@18.1.1` |
| `web/src/lib/markdown.ts` | Baru: `mdToHtml`, bungkus `marked` + daftar putih tag/atribut |
| `web/src/lib/markdown.test.ts` | Baru: 20 test, termasuk XSS |
| `web/src/lib/components/ProjectChat.svelte` | Balasan AI lewat `{@html mdToHtml(...)}`; gelembung biru hanya pesan pengguna |
| `web/src/app.css` | Gaya `.md` (judul, daftar, kutipan, tabel, kode) |

Pilihan pustaka: `marked` + daftar putih sendiri, bukan `markdown-it` + DOMPurify
(18 kB), bukan parser mini buatan sendiri, bukan `svelte-markdown`. Alasan: 2,7 kB
gzip, satu dependensi, parser teruji. XSS ditutup di lapisan daftar putih, semua
tag di luar daftar dibuang.

## Yang TIDAK berubah

- Bentuk data pesan dan endpoint. Ini murni lapisan tampilan.
- `markdown.test.ts` ikut di-commit sebagai jejak, dan dipastikan tidak masuk
  bundel (dicek `grep -rl 'SEMUA LULUS' .svelte-kit/output/client/`).
- Aturan em dash dari perubahan 0008 juga berlaku di dalam markdown.

## Bukti

```
$ node --experimental-strip-types src/lib/markdown.test.ts
20/20 lulus

$ npm run check   -> 0 errors, 0 warnings
$ npm run build   -> sukses

Di produksi, node chunk halaman proyek memuat penanda kode kita:
  "chat-markdown"   ADA
  marked: gfm ADA, breaks ADA, lexer ADA
```

## Sisa

Fitur markdown lanjutan (sorot sintaks, rumus matematika, gambar) sengaja tidak
diaktifkan: tag-nya di luar daftar putih, jadi dibuang. Balasan model tidak
membutuhkannya. Tambahkan ke daftar putih hanya kalau benar-benar perlu.
