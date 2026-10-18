# 0010 - Folder `plan/` dipindah ke `docs/plan/`

- **Status**: selesai
- **Tanggal**: 2026-10-18
- **Commit**: (menyusul)

## Kenapa

Dokumen tersebar di dua tempat: `plan/` di akar repo dan `docs/` di sebelahnya.
Sulit ditebak harus mencari di mana. Digabung supaya semua dokumen ada di satu
tempat.

## Yang berubah

| Berkas | Perubahan |
| --- | --- |
| `plan/` (17 berkas) | Dipindah utuh ke `docs/plan/` lewat `git mv`, riwayat git terjaga |
| 33 berkas kode (`.go`, `.svelte`, `.ts`) | Komentar `plan/x.md` jadi `docs/plan/x.md` |
| `README.md` | 4 rujukan diperbaiki |
| `docs/cleanup-candidates.md` | Rujukan diperbaiki |
| `docs/rencana-chat-planning.md` | Rujukan diperbaiki |
| `docs/rancangan-chat-sidebar-v1-superseded.md` | Rujukan diperbaiki |
| `server/deploy/DEPLOY.md` | Rujukan diperbaiki |
| `docs/update/` | Dibuat: README + changelog |

Total 108 rujukan `docs/plan/...` setelah perbaikan.

## Yang TIDAK berubah

- Isi setiap dokumen planning. Tidak ada satu baris pun di dalam `docs/plan/`
  yang diubah; ini pemindahan murni.
- Status beku folder planning tetap berlaku.

## Bukti

```
$ grep -rn '[^s]plan/' --include='*.go' --include='*.md' . \
    | grep -v node_modules | grep -v 'docs/plan/' | wc -l
0

$ grep -rn 'docs/plan/' --include='*.go' --include='*.md' . \
    | grep -v node_modules | wc -l
108

$ cd server && gofmt -l . && go vet ./... && go test -count=1 ./...
(bersih)
ok  paluketok/server/cmd/web
ok  paluketok/server/internal/adapter/judge
... 10/10 paket ok
```

## Sisa

Rujukan di dalam `docs/plan/` sendiri (antar-berkas planning) dibiarkan memakai
path relatif seperti semula. Itu cara berkas-berkas itu saling menunjuk sejak
awal, dan mengubahnya termasuk mengubah isi planning yang beku.
