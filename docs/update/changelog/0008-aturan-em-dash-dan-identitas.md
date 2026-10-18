# 0008 - Identitas "Ketok Magic" dan larangan em dash

- **Status**: selesai
- **Tanggal**: 2026-10-18
- **Commit**: `c4a1b56`

## Kenapa

Dua keluhan dari pengguna saat menguji:

1. Model menjawab sebagai model AI, bukan sebagai produk. Pemakai melihat nama
   model/vendor muncul di percakapan.
2. Keluaran memakai em dash (`—`) dan en dash (`–`). Gaya tulis yang diinginkan
   tidak memakai tanda pisah panjang.

## Yang berubah

| Berkas | Perubahan |
| --- | --- |
| `server/internal/usecase/message.go` | `SystemPrompt` memuat identitas "Ketok Magic"; aturan larangan em dash; fungsi baru `bersihkanGaya` |
| `server/internal/usecase/message_test.go` | Berkas baru: 5 test (identitas, larangan, em dash jadi koma, hyphen kata tidak disentuh, spasi pinggir) |

Identitas di prompt:

```
Kamu "Ketok Magic", asisten perancangan untuk proyek bernama "<nama>".
Kalau ditanya siapa kamu, jawab: Ketok Magic, asisten perancangan di sini.
Jangan pernah mengaku sebagai model AI atau produk lain, dan jangan sebut
nama model, vendor, atau versi apa pun.
```

Em dash dijaga **dua lapis**, karena instruksi di prompt tidak menjamin
kepatuhan:

1. Aturan tertulis di `SystemPrompt`.
2. `bersihkanGaya` di server mengganti em dash/en dash yang lolos jadi koma,
   dipanggil di jalur simpan pesan (`Content: bersihkanGaya(resp.Content)`).

`offline-first` dan `role-based` tidak disentuh; hanya em dash/en dash yang
memisahkan klausa.

## Yang TIDAK berubah

- Hyphen biasa (`-`) tidak diapa-apakan, jadi istilah teknis seperti
  `offline-first` tetap utuh.
- Bentuk pesan, endpoint, dan penyimpanan percakapan.

## Bukti

```
$ cd server && go test -count=1 ./internal/usecase/ -run 'TestSystemPrompt|TestBersihkanGaya' -v
--- PASS: TestSystemPrompt_IdentitasKetokMagic
--- PASS: TestSystemPrompt_LarangEmDash
--- PASS: TestBersihkanGaya_EmDashJadiKoma   (4 sub-kasus)
--- PASS: TestBersihkanGaya_TandaHubungKataTidakDisentuh
--- PASS: TestBersihkanGaya_SpasiPinggirDibuang

$ go test -count=1 ./...  -> 10/10 paket ok
```

Di produksi, `strings /opt/paluketok/bin/paluketok` memuat `Ketok Magic`,
`em dash`, dan `bersihkanGaya`.

## Sisa

Jawaban lama di database yang tersimpan sebelum perubahan ini masih memakai em
dash. `bersihkanGaya` hanya berlaku untuk keluaran baru. Membersihkan data lama
butuh migrasi tersendiri dan belum dilakukan.
