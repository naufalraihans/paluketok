# Kandidat Cleanup — Audit Fase 8

Sumber: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test -count=1 ./...`,
`go list -deps ./internal/usecase/... ./internal/entity/...`,
`grep -rn "TODO\|FIXME\|XXX" --include="*.go" server/`, `staticcheck ./...` (v0.6.1).
Tanggal: 2026-10-13. Basis commit: `723dcee` (kode `server/` belum di-commit).

**Tidak ada satu pun item di dokumen ini yang sudah dihapus.** Semua menunggu keputusan Master.

## Ringkasan

| Metrik | Angka |
|---|---|
| `TODO` / `FIXME` / `XXX` di `server/**/*.go` | 0 |
| `gofmt -l` | kosong |
| `go vet ./...` | exit 0 |
| `go test -count=1 ./...` | 9 paket `ok`, 0 FAIL |
| `func Test` dideklarasi / PASS / SKIP | 84 / 74 / 10 (10 SKIP butuh DB atau live API) |
| `staticcheck ./...` | exit 1, 18 temuan (14 produksi + 4 test) |

## Sudah dihapus (Fase 8, 2026-10-13)

Dihapus atas izin Master, 9 item. Verifikasi sebelum hapus: `grep -rn "\b<simbol>\b" --include="*.go" .` untuk tiap simbol (bukti di bawah), bukan hanya percaya `staticcheck`. Setelah hapus: `gofmt -l` kosong, `go build`/`go vet` exit 0, `go test -count=1 ./...` 9 paket `ok`, 74 PASS / 10 SKIP / 0 FAIL, `staticcheck ./...` exit 0 (nol temuan).

| Lokasi (sebelum hapus) | Jenis | Alasan | Bukti grep |
|---|---|---|---|
| `internal/adapter/payment/doku/doku.go:49` | field `encryption` | Sisa spekulatif, 0 referensi | 1 hit = deklarasinya sendiri |
| `internal/adapter/payment/doku/doku.go:50` | field `language` | idem | 1 hit = deklarasinya sendiri |
| `internal/adapter/payment/doku/doku.go:51` | field `displayName` | idem | 1 hit = deklarasinya sendiri |
| `internal/adapter/payment/doku/doku.go:314` | func `createdOrNow` | 0 referensi | 1 hit = deklarasinya sendiri |
| `internal/delivery/middleware/jwt.go:36` | type `contextKey` | Alias ke `typeKey` yang tak pernah dipakai | 1 hit = deklarasinya sendiri |
| `internal/repository/repo.go:69` | func `expectUnique` | 0 referensi; duplikat `uuid.go:19 isUnique` yang justru dipakai | 2 hit: deklarasi + komentar |
| `internal/usecase/usecase.go:102` | func `nowOrClock` | 0 referensi; duplikat `RunUsecase.clock()` | 2 hit: deklarasi + komentar |
| `internal/delivery/httpapi/e2e_test.go:224` | type `noRewriteOrders` | 0 referensi | 1 hit = deklarasinya sendiri |
| `internal/usecase/regenerate_test.go:25` | field `running` di `fakeRegenRepo` | 0 referensi pada struct ini | field `running` di `fakeRunRepo` (dipakai `r.running`) tidak disentuh |

Efek samping yang ikut dibereskan: import `"time"` di `internal/usecase/usecase.go` jadi yatim lalu dibuang; `gofmt -w` menyelaraskan perataan field `Client` di `doku.go` yang tadinya disejajarkan dengan field yang sudah dihapus.

### Perbaikan lint Fase 8 (sudah dikerjakan)

| Lokasi | Jenis | Perbaikan |
|---|---|---|
| `internal/usecase/order.go:101` | `SA1019` `strings.Title` deprecated | `strings.Title(pkg.ID)` dibuang; `Package` dapat field `Label` (`"Starter"`) untuk teks item DOKU. Tanpa dependensi baru: percobaan pertama pakai `golang.org/x/text/cases` **dibatalkan** karena menyusupkan `x/text` ke `internal/usecase`, melanggar aturan dependency `architecture.md` bagian 2 |
| `internal/worker/validate.go:88` | `ST1005` error string kapital | `"Ref menunjuk..."` → `"ref menunjuk..."` |
| `internal/worker/gate.go:129` | `S1008` | `if len(facts) == 0 { return false }; return true` → `return len(facts) != 0` |
| `internal/delivery/httpapi/e2e_harness_test.go:97` | `S1016` | `memProjects{m.s}` → `memProjects(m)` |
| `internal/delivery/httpapi/memdb_runs_test.go:93` | `S1016` | `memRuns{m.s}` → `memRuns(m)` |
| `internal/delivery/httpapi/memdb_runs_test.go:148` | `S1016` | idem |
| `internal/delivery/httpapi/admin_test.go:252` | `SA4006` nilai `out` ditimpa tanpa dibaca | Pisah variabel `respRuns`/`runsOut`, tambah assertion `runs` harus 1 item, jalur decisions pakai variabel sendiri. **Cakupan naik**, tidak turun |
| `internal/repository/integration_test.go:95` | `SA4006` nilai `id` ditimpa | Buang assignment mati, sisakan literal UUID (test SKIP tanpa DB) |

## A. Kandidat hapus — menunggu keputusan Master

Tidak ada. Semua kandidat dead code di dokumen ini sudah diputuskan: 9 dihapus (bagian "Sudah dihapus"), 2 ditahan (tabel C), sisanya false positive (tabel C).

## B. Kandidat perbaikan — bukan penghapusan fitur

Sudah dikerjakan seluruhnya di Fase 8, lihat "Perbaikan lint Fase 8" di atas. `staticcheck ./...` sekarang exit 0.

## C. JANGAN DIHAPUS

### C1. Ditahan atas keputusan Master (Fase 8)

| Lokasi | Fakta | Alasan ditahan |
|---|---|---|
| `internal/entity/validate.go:169` `ValidStage` | 0 pemanggil saat audit | Titik validasi `entity.Stage`. Master: colok ke jalur yang butuh nanti, jangan hapus |
| `pkg/config/config.go:71` `IsProduction` | 0 pemanggil saat audit | Gate env `APP_ENV`. Master: pertahankan sebagai jalur gate produksi |

### C2. False positive staticcheck

| Lokasi | Fakta | Kenapa dipertahankan |
|---|---|---|
| `internal/delivery/middleware/middleware.go:181` `RateLimit` + `:162 rateLimiter` | 0 pemanggilan di `router.go` | Bukan dead code. Diwajibkan `plan/security.md` bagian 8 (per user 5/menit `POST /runs`, 60/menit endpoint lain, 30/menit admin). Yang kurang adalah **wiring**, bukan fungsinya. Hapus = menghapus kewajiban keamanan |
| `internal/port/repository.go:158` `RateLimiter` | 0 referensi langsung | Kontrak port untuk `RateLimit` di atas |
| `internal/port/repository.go:75` `WorkerQueue` | 0 referensi langsung | Kontrak port; implementasi di `repository/worker_queue.go` |
| `pkg/response/response.go:71` `FailInternal` | 0 pemanggilan langsung (dipakai lewat jalur 500 di `handlers.go:36 writeErr`) | Menegakkan `plan/security.md` bagian 10: `500` hanya `{"error":{"code":"internal",...}}`, tanpa stack trace |
| `internal/entity/validate.go:97,126` `DecodeStack`/`EncodeStack` | 0 pemanggilan langsung | Dipakai lapisan repository saat encode/decode kolom stack |
| `internal/worker/pipeline.go:138 Stop` | 0 pemanggilan | shutdown pool; dijalankan saat drain, dicek lewat `pool.go:170 deadline2` |
| `internal/worker/pipeline.go:62 Process` | bukan nol | Dipakai API callback worker, bukan dari paket lain |
| `internal/adapter/{llm,judge}` `IsUnavailable` | 0 pemanggilan prod | Dipakai untuk klasifikasi error 503 gateway; rujukan ada di test |

Catatan: `staticcheck U1000` menandai hal yang tak terpakai **di dalam satu modul**. Item di tabel C muncul karena pemakaiannya lintas paket/melalui antarmuka, jadi jangan ikut dihapus.

## D. Batas audit ini

- Kode belum pernah di-commit (`723dcee` masih commit awal). Belum ada diff untuk dibandingkan, jadi audit ini memeriksa working tree apa adanya, bukan regresi.
- 10 test SKIP karena butuh Postgres atau API live; cakupan jalur uang pada kondisi itu belum tervalidasi di lingkungan ini.
- `staticcheck` tidak terpasang di lingkungan; dipasang manual dengan versi dipin (`honnef.co/go/tools/cmd/staticcheck@v0.6.1`) supaya hasil dapat direproduksi.
