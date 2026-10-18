# Status planning

Dokumen ini hanya mencatat yang masih terbuka. Keputusan yang sudah jadi ada di file masing-masing.

## Sudah diputuskan

| Topik | Keputusan | Sumber |
|---|---|---|
| Arsitektur | Go Clean Architecture, monolith satu proses | `architecture.md` |
| Domain | `paluketok.web.id` | `architecture.md` |
| JWT | JWKS Supabase, tanpa HS256 | `security.md` 2 |
| Auth BE | JWT/JWKS Supabase sejak awal, RLS lapis kedua | `architecture.md`, `security.md` 2 |
| LLM planner | key dikelola admin, OpenAI-compatible, slot `planner` | `architecture.md` 3 |
| Key LLM | plaintext di DB, GET masked (JEV: masked 0.94) | `erd.md`, `openapi.md` |
| Chatbot | gratis, tidak memotong kredit (JEV: free 0.94) | `openapi.md` |
| JEV judge | server-internal, tidak diekspos (JEV: server_only 1.00) | `openapi.md`, `planner.md` 4 |
| Expire order | webhook + cek saat GET order, tanpa sweeper | `auth_billing.md` 2.4 |
| CORS | hanya `https://paluketok.web.id` + localhost saat dev | `security.md` 10 |
| Rate limit | per user, per admin, webhook | `security.md` 8 |
| FE | ditunda, fokus BE | - |
| Deploy target | satu VPS, semua di sana (HTTP + worker) | `deploy.md` |
| Go | 1.26.2 terpasang, `go 1.26` di `go.mod` | `deploy.md` |
| Worker | pool goroutine di VPS yang sama, tanpa serverless | `deploy.md` |
| Graph wiring | `graphify` terpasang sebagai skill Hermes | - |
| Struktur folder | adopsi pola `project_cpns`, usecase tetap bebas gorm/dto | `structure.md` |
| Router | `net/http` stdlib (beda dari Gin di `project_cpns`) | `structure.md` |

Sisa keterbatasan pasca-F0-F7 ada di tabel di bawah dan blok "Sisa sebelum deploy".

Sisa keterbatasan yang diketahui (tidak menghalangi eksekusi, ditutup saat `server/` ada):

| Hal | Kenapa dibiarkan | Ditutup kapan |
|---|---|---|
| `artifact_versions.created_by_run_id` hanya menunjuk run asal | regenerate tidak dicatat sebagai baris `plan_runs` (keputusan: hindari tabel/kolom baru) | saat panel admin butuh jejak per-percobaan regenerate |
| ~~Sanitizer prompt injection (`security.md` 5b) baru aturan~~ | **DITUTUP F6.** `worker/prompt.go:50` menegakkan delimiter peran + instruksi mengabaikan isi `<brief>`/`<artifact>`. Validasi makna menyusul | selesai |

Ditunda ke panel admin (bukan penghalang coding): nama model, base URL, dan key LLM diisi admin lewat `admin.paluketok.web.id`.

## Sisa sebelum deploy

Poin di bawah terbuka per 2026-10-13 (Fase 8). Sumber: audit F8 + `docs/cleanup-candidates.md`.

| Hal | Butuh | Alasan |
|---|---|---|
| FE Svelte belum ada | Eksekutor | Ditunda sejak awal, fokus BE (`architecture.md`). Deploy BE tidak wajib menunggu, tapi produk belum utuh |
| Repo belum commit sejak `723dcee` | Master | Seluruh `server/`, `docs/`, `plan/` masih untracked/berubah. Belum ada titik pulih selain commit awal |
| DNS + TLS | Master | `paluketok.web.id` masih menunjuk `216.198.79.1`, bukan VPS `103.119.237.29` (F0.3) |
| Rotate key DOKU | Master | Key lama pernah tertulis di dokumen, wajib di-rotate saat cutover (`security.md` bagian 9, checklist butir 7) |
| Notification URL DOKU | Master | Belum terdaftar di Back Office. DOKU melarang localhost/ngrok, jadi menunggu DNS dulu |
| `PLANNER_LLM_*` belum diisi | Master | `cmd/web/llm_resolve.go:35` jatuh ke `PLANNER_LLM_BASE_URL`/`PLANNER_LLM_MODEL` bila slot `planner` di `ai_settings` kosong, dan **run pertama gagal** sampai salah satu diisi. Env-nya tidak dibaca `pkg/config/config.go`, hanya lewat `os.Getenv` |
| ~~Rate limit belum di-wire~~ | **DITUTUP F11.** `router.go:76-93` memasang `RateLimit` (per user 60/mnt, POST /runs 5/mnt, admin 30/mnt; webhook tanpa limit). Diwajibkan `security.md` bagian 8 | selesai, `0c200da` |
| `POST /v1/ai/chat` nol implementasi | Master | Kontrak ada di `openapi.md:23,44`, kode tak dibangun di F0-F7. Fitur baru = butuh fase baru pasca-beku. Bukan penghalang deploy BE |
| Rujukan `tech_stack.md` dibersihkan | Selesai | Tak ada rujukan mati; sisa satu baris di riwayat `changelog.md` dan itu memang catatan historis |
| `GET /v1/admin/users` tidak ada di kontrak | Master | Tidak ada di `openapi.md:34-40`; UI admin menurunkan daftar user dari `GET /v1/admin/runs` (bawa `user_id`). Tambah endpoint bila produksi menuntut |
| List order per user tidak ada di kontrak | Master | Tak ada `GET /v1/orders` (hanya `POST` + `GET /v1/orders/{id}`); riwayat billing UI dari `sessionStorage` + status per order id. Tambah endpoint bila produksi menuntut |
| ~~`pageFrom` limit 0 saat tanpa query~~ | **DITUTUP F11.** `handlers.go:115-139` default 20/cap 100, offset aneh -> 0; `ListAllRuns` kirim NULL saat `user_id` kosong (bukan `uuid ""`). Bukti: `page_admin_test.go` | selesai, `1f5c4d5` |
| ~~JWKS default 404~~ | **DITUTUP F11.** `jwt.go:68-75` default `<SUPABASE_URL>/auth/v1/.well-known/jwks.json`; env `SUPABASE_JWKS_URL` eksplisit tak berubah. Bukti: `jwks_default_test.go` | selesai, `18f6275` |
| ~~CORS tanpa PUT~~ | **DITUTUP F11.** `middleware.go:83` Allow-Methods memuat `PUT`; Allow-Headers tak disentuh. Bukti: `middleware_test.go` | selesai, `df08227` |
| ~~Regenerate artefak tumbukan versi~~ | **DITUTUP F11.** `regenerate.go:85-98` tulis versi lama hanya bila belum ada di history (`HasVersionSet`, `order_artifact.go:125`); fake repo menegakkan unique. Bukti: `TestRegenerate_DuaKali_HistoryBerurutan` gagal-sebelum/lulus-sesudah | selesai, `1a6226b` |
| Kartu proyek navigasi pakai slug | Eksekutor (F10, diperbaiki di FE) | `DashProjectList.svelte` mengarahkan ke `/projects/<slug-nama>`, padahal route `[id]` mencari UUID -> halaman detail kosong. FE kini mengirim `card.id` (UUID) dan jatuh ke slug hanya untuk data contoh |

## Dicek saat setup (bukan keputusan planning)

- Durasi expire order: diambil dari `expired_date` yang dikembalikan DOKU (`auth_billing.md` 2.4, 5).

## Ditunda

Paket harga vs biaya token (user: tidak perlu estimasi sekarang), BYOK, stale policy regenerate, tahap `seed`/`test-plan`/`ci`, tabel audit terpisah, endpoint AI streaming, slot `ai_settings` selain `planner`, nama model dan base URL 9router (diisi admin di panel).
