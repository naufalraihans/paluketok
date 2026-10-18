# 0006 - Pesan pengguna tampil seketika + indikator menunggu

- **Status**: selesai
- **Tanggal**: 2026-10-18
- **Commit**: `c4a1b56`

## Kenapa

Dilaporkan pengguna: setelah menekan kirim, pesannya **hilang dulu**, baru muncul
bersamaan dengan balasan model. Selama menunggu, kotak chat tampak kosong tanpa
tanda apa pun bahwa permintaan sedang diproses.

Penyebabnya urutan kerja di `ProjectChat.svelte`: pesan dikirim ke server lebih
dulu, baru riwayat dimuat ulang. Selama permintaan berjalan, tidak ada yang
ditampilkan.

## Yang berubah

| Berkas | Perubahan |
| --- | --- |
| `web/src/lib/components/ProjectChat.svelte` | Daftar `optimistis` untuk pesan pengguna yang belum dikonfirmasi server; blok indikator "sedang menyusun balasan" |
| `web/src/app.css` | Kelas `.chat-typing` + animasi 3 titik; menghormati `prefers-reduced-motion` |

Alurnya sekarang:

1. Pesan pengguna masuk daftar `optimistis` dan langsung tampil (biru pudar,
   menandakan belum dikonfirmasi server).
2. Tiga titik berdenyut selama menunggu.
3. Balasan datang, riwayat server dimuat, daftar `optimistis` dikosongkan.

Kalau pengiriman gagal, teks dikembalikan ke kotak ketik supaya tidak hilang.

## Yang TIDAK berubah

- Server tidak berubah. Satu permintaan, satu balasan penuh (belum streaming).
- Indikator ini **bukan** streaming token: ia hanya menandakan permintaan sedang
  diproses. Streaming sungguhan butuh perubahan protokol (SSE atau WebSocket) dan
  belum dikerjakan.

## Bukti

```
$ npm run check  -> 0 errors, 0 warnings
$ npm run build  -> sukses

Di produksi, node chunk halaman proyek memuat:
  "chat-typing"         ADA
  "chat-pending-user"   ADA
```

## Sisa

Indikator mengetik palsu. Streaming token asli adalah pekerjaan backend
tersendiri: perlu endpoint SSE, penanganan koneksi putus, dan penyusunan ulang
potongan markdown yang datang bertahap. Belum dijadwalkan.
