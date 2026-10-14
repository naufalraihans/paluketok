# Changelog

Format: Keep a Changelog (https://keepachangelog.com).

## 2026-10-13 - Fase 8 (pembersihan + konsolidasi, sebagian)

- Koreksi nama trigger di `docs/release-checklist.md` butir 3: `guard_profiles_update` -> `trg_guard_profiles_update` (function `guard_profiles_update`), dikonfirmasi dari `database/migrations/0001_init.sql:306-308`.
- Restore `design/favicon.svg`, `design/landing.html`, `design/prompt-figma-ai.md` (`git checkout -- design/`). Status `D` di working tree, bukan commit: `git log --diff-filter=D --name-only` kosong dan reflog penuh `checkout: moving from main to main`. `favicon.svg` aset logo wajib, `landing.html` + `prompt-figma-ai.md` acuan UI (`docs/agents/uiux.md:7`, output ke `/design/`). Tidak ada entri changelog penghapusan karena tidak pernah ada commit penghapusan.
- Audit mentah `server/`: `gofmt -l` kosong; `go build ./...` exit 0; `go vet ./...` exit 0; `go test -count=1 ./...` 9 paket `ok`, 0 FAIL (84 `func Test` dideklarasi, 74 PASS, 10 SKIP butuh DB/live); `go list -deps ./internal/usecase/... ./internal/entity/... | grep -E "pgx|gorm|net/http|delivery"` kosong. `grep -rn "TODO\|FIXME\|XXX" --include="*.go" server/` = 0. `staticcheck ./...` exit 1, 18 temuan (14 produksi + 4 test) - daftar di `docs/cleanup-candidates.md`, belum ada yang dihapus.
- Bocor plan-vs-kode yang belum diputuskan (dilaporkan, tidak diubah): `POST /v1/ai/chat` ada di `openapi.md:23,44` tapi nol implementasi; rate limit `security.md` bagian 8 punya `middleware.RateLimit` tapi tidak di-wire di `router.go`.
- Verifikasi working tree pasca-Fase 8: `git status design/ docs/` hanya `?? design/web/` dan `?? docs/release-checklist.md`.

### Fase 8 lanjutan - eksekusi keputusan Master (2026-10-13)

- `POST /v1/ai/chat`: **DITUNDA resmi.** Kontrak tetap di `openapi.md:23,44`, kode tidak dibangun di F0-F7. Fitur baru = butuh fase pasca-beku. Dicatat di `gaps.md` blok "Sisa sebelum deploy".
- 9 item dead code dihapus (`docs/cleanup-candidates.md` bagian "Sudah dihapus"): `doku.go` field `encryption`/`language`/`displayName` + func `createdOrNow`, `jwt.go` type `contextKey`, `repo.go` func `expectUnique`, `usecase.go` func `nowOrClock`, `e2e_test.go` type `noRewriteOrders`, `regenerate_test.go` field `running` di `fakeRegenRepo`. Tiap simbol diverifikasi ulang dengan grep sebelum hapus, tidak hanya percaya `staticcheck`.
- DITAHAN: `entity.ValidStage` (titik validasi, colok nanti) dan `config.IsProduction` (gate env). Dipindah ke tabel "JANGAN DIHAPUS" bagian C1.
- Tabel B (perbaikan lint) selesai seluruhnya: `SA1019` `strings.Title` di `order.go:101` dibereskan dengan field `Package.Label` (percobaan `golang.org/x/text/cases` dibatalkan karena melanggar aturan dependency `architecture.md` bagian 2 - `usecase` wajib bebas `x/text`); `ST1005` `validate.go:88`; `S1008` `gate.go:129`; 3x `S1016` di file test; `SA4006` `admin_test.go:252` (ditambah assertion `runs` 1 item, cakupan naik) dan `integration_test.go:95`.
- `gaps.md`: baris "Tidak ada. Planning beku dan siap dieksekusi." diganti karena tidak akurat; baris sanitizer prompt injection ditandai **DITUTUP F6** (`worker/prompt.go:50` menegakkan delimiter peran).
- Hasil: `gofmt -l` kosong, `go build`/`go vet` exit 0, `go test -count=1 ./...` 9 paket `ok` (74 PASS / 10 SKIP / 0 FAIL), `staticcheck ./...` exit 0 (dari 18 temuan ke 0), guard dependency `usecase`+`entity` bersih. `go.mod` kembali hanya `pgx` sebagai dependensi langsung.

## [Unreleased]

### Added
- Endpoint `POST /v1/ai/chat` (gratis, JEV: free 0.94).
- JEV judge internal, tidak diekspos (JEV: server_only 1.00).
- Tabel `ai_settings` (slot `planner`), endpoint admin `/v1/admin/settings/ai`.
- `plan/erd.md`, `plan/openapi.md`, `plan/gaps.md`, `plan/security.md` ditulis ulang.
- Struktur plan: main.md, prd.md, jev_ai.md, changelog.md.
- Stack final: Svelte+TS, Go, Supabase (PostgreSQL + Auth), DOKU, OpenAPI, DBDiagram.

### Changed
- Planning ditandai FINAL. Klaim expiry Snap 24 jam dihapus, jadi checklist setup.
- Charter `docs/agents/` dipakai sebagai label peran, bukan sumber path.
- Timeout eksplisit untuk semua klien HTTP keluar (LLM, JEV, DOKU).
- Auth BE: JWT/JWKS Supabase sejak awal. Mode admin-key dihapus dari `architecture.md`.
- Sinkronisasi lintas file: aturan dependency hanya di `architecture.md`, rahasia hanya di `security.md` bagian 9, status artefak punya jalur `final`.
- `auth_billing.md`: registrasi/login/reset password (bagian 1.3), bootstrap admin lewat `cmd/seeder` (bagian 1.5), expire order tanpa sweeper.
- Deploy: seluruh BE di satu VPS. Vercel dan serverless dibatalkan.
- Worker: keputusan `hybrid` (HTTP Vercel, worker VPS). Dev tanpa Docker di laptop.
- `architecture.md`: buang tree folder (pindah ke `structure.md`), buang pipeline/gate yang sudah ada di `planner.md`, samakan nama (`cmd/web`, `net/http`, `delivery/`). Sekarang hanya keputusan + aturan dependency.
- `prd.md`: seksi stack diganti pointer, tidak mengulang `main.md`.
- `planner.md`, `security.md`: limit dan timeout JEV jadi pointer ke `jev_ai.md` / `planner.md`.
- `deploy.md` ditambah: target deploy, Go 1.26, worker.
- Hapus `tech_stack.md` (sudah di `prd.md` dan `main.md`).
- Expire order: webhook + cek saat GET, tanpa sweeper.
- JWT: JWKS Supabase, bukan HS256.
- Duplikasi di `auth_billing.md` (role, ledger) diganti pointer ke `roles.md` dan `erd.md`.
- PRD diperjelas: PaluKetok adalah contoh, bukan target produk.
- Payment: Stripe diganti Midtrans, lalu diganti DOKU (2026-10-12).
- Env key JEV: EXPERIENTIAL_API_KEY menjadi JEV_API_KEY.
- Base URL JEV: backtick nyasar dihapus.

### Changed (2026-10-13)
- `profiles.token_version int not null default 0` ditambah ke `erd.dbml` (keputusan user 2026-10-13): kolom sudah dijanjikan `security.md` bagian 3 (lapis 1b) dan `changelog.md` 2026-10-12, tapi tidak pernah masuk skema. Typo nyasar `'al'` di note `artifact_versions.created_by_run_id` dibuang.
- `security.md` bagian 7: baris `token_version` dipindah ke dalam blok daftar (sebelumnya baris nyasar tanpa `-`).

### Added (2026-10-13)
- `server/database/migrations/0001_init.sql` (skema + RLS + grant cabut + policy + trigger `profiles`), `0002_grants_and_guard.sql`, `0003_profiles_fk.sql`.

### Status fase (2026-10-13)
- F4 doku live create-order 200 sukses token baru. reset key sandbox.
- **F0 SELESAI dengan satu pengecualian tercatat.**
  - F0.1 VPS: tuntas. Key-only login (`PasswordAuthentication no`, `PermitRootLogin without-password`), `ufw` aktif allow 22/20257/80/443. Deviasi: port SSH 20257, bukan 22 (`workflow.md:9`).
  - F0.2 secret: tuntas. `server/.env` ada dan diabaikan git (`.gitignore:5`).
  - F0.3 DOKU sandbox: **3 dari 4 tuntas.** `DOKU_CLIENT_ID`, `DOKU_SECRET_KEY`, `DOKU_BASE_URL=https://api-sandbox.doku.com`, `DOKU_NOTIFICATION_PATH=/v1/webhooks/doku` terisi. **PENGECUALIAN: Notification URL belum terdaftar di Back Office DOKU.** Sebab: DOKU melarang localhost/ngrok (`auth_billing.md:163`), sedangkan `paluketok.web.id` masih menunjuk `216.198.79.1`, bukan VPS `103.119.237.29`. Ditutup di Fase 7 saat cutover DNS + HTTPS. Aman ditunda karena webhook DOKU baru dipakai Fase 5 langkah 21.
  - F0.4 Supabase: tuntas. Signup lewat `{SUPABASE_URL}/auth/v1/signup` (anon key) balas `200`, `auth.users` terisi, `email_confirmed_at` NULL membuktikan konfirmasi email aktif (`auth_billing.md:1.3`), trigger `handle_new_user` membuat baris `profiles` otomatis.

### Deviasi dari `workflow.md` (ditemukan saat eksekusi Fase 1-2)
- Deviasi 1: `workflow.md:21` minta `config` pakai **viper**. Dipakai parser `.env` stdlib (~30 baris) karena hanya baca `KEY=VALUE`. Ganti ke viper bila butuh quoting/escape/var-expansion.
- Deviasi 2: `0001_init.sql` ditulis idempoten (`create table if not exists`, `drop policy if exists`) supaya bisa dijalankan ulang. `workflow.md:8` tidak melarang.
- Deviasi 3 (celah plan): `erd.dbml:172` punya `Ref: profiles.id - auth.users.id [delete: cascade]`, tapi tidak ada migrasi yang memasangnya. Ditemukan dari profil yatim saat cleanup uji RLS. Dipasang di `0003_profiles_fk.sql`.
- Deviasi 4 (celah plan): `security.md:51` minta `REVOKE ALL`, tapi tidak menyebut kewajiban `GRANT` ulang untuk tabel yang punya policy. Tanpa grant, `authenticated` kena `permission denied` sebelum policy dievaluasi. Dipasang di `0002_grants_and_guard.sql`: `profiles` (select + update kolom `username`/`full_name`/`avatar_url`), `projects`/`plan_runs`/`artifacts`/`decisions`/`artifact_versions` (select+insert+update+delete), `orders` (select saja). `credits_ledger` dan `ai_settings` nol grant, sesuai `security.md:54`.
- Deviasi 5: guard kolom sensitif `profiles` (credits, is_admin, plan_tier, token_version) hanya berlaku untuk klien `anon`/`authenticated`. Server (`postgres`/service_role) tetap boleh mengubahnya. Tanpa ini, jalur grant kredit server ikut terblokir.

## 2026-10-12 - Audit lintas file (startup-team) + perbaikan

Audit dijalankan lewat 4 subagent (3 architect, 3 security; 4 di antaranya mengulang karena kena HTTP 429). Temuan diterapkan hanya setelah diverifikasi ke isi file; 2 temuan yang tidak cocok dengan file dibuang (klaim ambiguitas `security.md` bagian 3, klaim `workflow.md` tidak terdaftar di `main.md`).

BLOCKER yang ditutup:
- RLS sebelumnya hanya flag `RLS ON` tanpa satu pun `POLICY`. Sekarang `security.md` bagian 3 memuat `REVOKE ALL` dari `anon`/`authenticated`, policy per tabel, dan larangan `UPDATE` `profiles` lewat PostgREST.
- Tiga index penjaga uang bisa bocor lewat `ref_id NULL` (Postgres menganggap NULL distinct). Ditambah predikat `and ref_id is not null`. Terbukti lewat uji: dua refund `ref_id NULL` lolos berdua.
- Grant admin tidak bisa diaudit dan bisa dobel. Ditambah `credits_ledger.actor_id` (FK `profiles`) dan `idempotency_key` dengan `uq_ledger_one_grant_per_key`.
- `credits_ledger.attempt_run_id` (dibuat di iterasi sebelumnya) dibuang: kolom yatim, tidak ada baris `plan_runs` untuk regenerate. Refund regenerate dikunci ke baris charge (`ref_type='ledger'`, `uq_ledger_one_refund_per_charge`).

PENTING yang ditutup:
- `profiles.token_version` ditambah untuk pencabutan sesi hidup.
- `GET /v1/decisions` tidak lagi mengekspos `state` mentah dan `criteria` internal.
- Batas chatbot diselaraskan: total gabungan maks 16.000 karakter.
- `ai_settings.base_url` wajib allowlist host; client menolak redirect lintas host (`CheckRedirect`).
- `CHECK credits >= 0` dan `delta <> 0` ditegakkan; `brief` 1-8.000 karakter.
- `plan_runs.current_stage` ditulis di AWAL tahap, supaya run `failed` menunjuk tahap yang benar.
- Regenerate ditolak `422` jika ada run `running` di project yang sama.
- Timeout adapter punya pemilik tunggal di `structure.md` (LLM 180, JEV 30, DOKU 20 detik).
- Transisi `orders` lewat `UPDATE ... WHERE status='pending'`; kebijakan late settlement ditulis eksplisit.
- `security.md` bagian 5b baru: mitigasi prompt injection berantai (delimiter peran, sanitasi keluaran, validasi makna, batas ukuran).

Uji Postgres 18 lokal (5 uji, semua sesuai harapan): double topup ditolak, saldo negatif ditolak, grant idempoten ditolak, `delta = 0` ditolak, NULL distinct terkonfirmasi.

Terbuka, belum ditutup: `artifact_versions.created_by_run_id` hanya berarti run asal; regenerate tidak dicatat sebagai baris `plan_runs`. Sanitizer dan validasi makna belum bisa diuji karena `server/` belum ada.

## 2026-10-12 - Payment gateway Midtrans -> DOKU (dokumen DOKU dibaca penuh)

- 168 halaman docs DOKU diunduh lokal lewat HTTP biasa (tanpa browser), 0 gagal.
- Kontrak baru: order POST `checkout/v1/payment`, signature HMAC-SHA256 (Client-Id, Request-Id, Request-Timestamp, Request-Target, Digest), webhook `POST /v1/webhooks/doku`, cek status `GET /orders/v1/status/{invoice_number}`.
- `erd.dbml`: `midtrans_order_id` -> `invoice_number`; `snap_token` dibuang; tambah `request_id` (unique, not null, idempotency key), `payment_url`, `expired_date`. `deny` dibuang dari status order.
- `security.md` bagian 6 ditulis ulang untuk DOKU. Umur `Request-Timestamp` TIDAK dipakai sebagai gerbang tolak karena retry DOKU sampai +12 jam; anti-replay lewat `request_id` unik.
- `auth_billing.md` bagian 2 dan 5 ditulis ulang. Sumber kedaluwarsa = `expired_date` dari respons DOKU, bukan timer sendiri. `409` DOKU dibaca sebagai order sudah ada.
- Webhook: `transaction.status FAILED` DIABAIKAN (Checkout mengizinkan ganti metode bayar); EXPIRED ada di `order.status`, bukan `transaction.status`.
- Audit security DEFENSIVE dijalankan untuk integrasi DOKU (`candidates/doku-sec.md`): 8 temuan.
- Uji Postgres 18: invoice_number dobel ditolak, request_id dobel ditolak, status `deny` ditolak CHECK, `expired_date` tersimpan benar (+07). 4/4.
- Klaim "toleransi timestamp 10 detik" dari audit DIBUANG karena tidak terbukti di docs.

## 2026-10-14 - Fase 10 Tahap B: /admin sebagai path, bukan subdomain (deviasi planner.md:136)

- `planner.md:136` menyebut panel admin di `admin.paluketok.web.id` (subdomain). Deviasi: UI dibangun sebagai route `/admin` di aplikasi web yang sama, satu halaman dengan lima tab (Projects, Runs, Users, Keputusan, AI Settings).
- Alasan: satu deploy, satu basis kode, guard `is_admin` dari `GET /v1/me` sudah cukup; subdomain butuh DNS + sertifikat terpisah yang belum ada di fase ini.
- Kontrak BE tidak berubah: pemanggilan tetap `/v1/admin/*` seperti `openapi.md:34-40`. Perubahan hanya di sisi frontend.
