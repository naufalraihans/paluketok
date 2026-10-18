# 0009 - Slot "Judge JEV" jadi "AI Judge", model tidak lagi dikunci

- **Status**: selesai
- **Tanggal**: 2026-10-18
- **Commit**: (`feat: judge generik...` menyusul)

## Kenapa

Slot penilai dipanggil "Judge JEV" dan modelnya dikunci `jev-latest` di kode.
Dua masalah:

1. **Nama vendor dipakai sebagai nama fitur.** `judge` adalah PERAN (penilai
   keputusan), bukan produk. Penyedia penilai bisa diganti.
2. **Model terkunci di konstanta Go.** Admin bisa mengganti base URL dan API
   key, tapi server tetap mengirim `model: "jev-latest"`. Artinya mengganti
   penyedia tidak mungkin tanpa mengubah kode dan deploy ulang.

Ditemukan saat menyiapkan penyedia penilai lain: panel menampilkan kolom Model
`(dikunci)` dengan latar abu dan tidak bisa diketik.

## Yang berubah

| Berkas | Perubahan |
| --- | --- |
| `server/internal/entity/entity.go` | Komentar `Stage` menunjuk `docs/plan/planner.md` |
| `server/internal/adapter/judge/judge.go` | `const Model = "jev-latest"` jadi `DefaultModel`; `Client` dapat field `Model`; `Decide` pakai `c.Model`, kosong berarti `DefaultModel` |
| `server/cmd/web/llm_resolve.go` | `resolvedJudge.Decide` membaca `s.Model` dari slot database, bukan konstanta |
| `web/src/lib/components/admin/AdminAI.svelte` | Label `Judge JEV` jadi `AI Judge`; kolom Model tidak lagi `disabled`; `JEV_MODEL`/`JEV_DEFAULT_URL` jadi `JUDGE_DEFAULT_*`; `modelLocked` dihapus dari jalur simpan |
| `server/internal/adapter/judge/judge_test.go` | 2 test baru: model dari slot dipakai, model kosong jatuh ke default |
| `docs/plan/jev_ai.md` | TIDAK diubah (planning beku) |

Alur simpan sekarang: apa pun yang diketik admin di kolom Model dikirim apa
adanya (`model: model.trim()`), bukan ditimpa konstanta.

## Yang TIDAK berubah

- Nilai awal JEV tetap terisi di panel untuk slot yang belum pernah diisi, supaya
  data lama tetap jalan. Slot tanpa model jatuh ke `DefaultModel` = `jev-latest`.
- Endpoint penilai, bentuk request, timeout 30 detik, tanpa retry.
- Validasi backend: `s.Model == ""` tetap ditolak `ErrValidation`. Model apa pun
  diterima selama tidak kosong.
- `docs/plan/jev_ai.md` masih menyebut JEV sebagai penyedia tunggal. Tidak
  diselaraskan karena folder planning beku.

## Bukti

```
$ cd server && go test -count=1 ./internal/adapter/judge/ -v -run TestDecide
--- PASS: TestDecide_OK
--- PASS: TestDecide_429Unavailable
--- PASS: TestDecide_ModelDariSlot
--- PASS: TestDecide_ModelKosongJatuhKeDefault
ok  paluketok/server/internal/adapter/judge

$ go test -count=1 ./...   -> 10/10 paket ok
$ gofmt -l .               -> kosong
$ go vet ./...             -> exit 0

$ cd web && npm run check  -> 0 errors, 0 warnings
$ npm run build            -> sukses
```

## Sisa

`docs/plan/jev_ai.md` masih menyebut JEV sebagai satu-satunya penyedia penilai.
Kalau dokumen itu mau diselaraskan, harus lewat keputusan pemilik produk karena
folder planning beku.
