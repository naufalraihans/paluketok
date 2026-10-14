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
| Rate limit belum di-wire | Eksekutor | `middleware.RateLimit` ada, `router.go` belum memasang. Diwajibkan `security.md` bagian 8 |
| `POST /v1/ai/chat` nol implementasi | Master | Kontrak ada di `openapi.md:23,44`, kode tak dibangun di F0-F7. Fitur baru = butuh fase baru pasca-beku. Bukan penghalang deploy BE |
| Rujukan `tech_stack.md` dibersihkan | Selesai | Tak ada rujukan mati; sisa satu baris di riwayat `changelog.md` dan itu memang catatan historis |
| `GET /v1/admin/users` tidak ada di kontrak | Master | Tidak ada di `openapi.md:34-40`; UI admin menurunkan daftar user dari `GET /v1/admin/runs` (bawa `user_id`). Tambah endpoint bila produksi menuntut |
| List order per user tidak ada di kontrak | Master | Tak ada `GET /v1/orders` (hanya `POST` + `GET /v1/orders/{id}`); riwayat billing UI dari `sessionStorage` + status per order id. Tambah endpoint bila produksi menuntut |
| `pageFrom` limit 0 saat tanpa query | Eksekutor (F10, dilaporkan) | `handlers.go:115-120`: `limit`/`offset` absen -> `Page{0,0}` -> `limit 0` -> hasil kosong (bukan default). `/v1/admin/runs` tanpa `user_id` juga kirim `uuid ""` ke SQL -> `500 SQLSTATE 22P02`. FE fase 10 selalu kirim `limit`, dan tab Runs selalu sertakan `user_id` |
| JWKS default 404 | Eksekutor (F10, dilaporkan) | `middleware.NewVerifier` default `<SUPABASE_URL>/.well-known/jwks.json` -> 404 di Supabase; jalur benar `/auth/v1/.well-known/jwks.json`. Kerja sekitar dev: env `SUPABASE_JWKS_URL` diisi eksplisit. Perbaikan disarankan di BE |
| CORS tanpa PUT | Eksekutor (F10, dilaporkan) | `middleware.go:83` `Access-Control-Allow-Methods` tak memuat `PUT`, jadi preflight browser memblokir `PUT /v1/admin/settings/ai/{slot}` dan tab AI Settings gagal simpan dari UI (curl lolos karena tanpa preflight). Kerja sekitar dev: proxy `tools/dev-mock/dev_cors_proxy.py` |
| Regenerate artefak tumbukan versi | Eksekutor (F10, dilaporkan, akar dikonfirmasi) | `POST /v1/artifacts/{id}/regenerate` gagal `duplicate key ... artifact_versions_artifact_id_version_key` (SQLSTATE 23505). AKAR (bukan "regenerate berturut-turut"): pipeline run sudah menulis history versi 1 lewat `ArtifactRepo.Upsert` (`repository/artifact_upsert.go:30-34`), lalu `RegenerateArtifact` memanggil `InsertVersion` dengan `Version: art.Version` (= 1, versi LAMA — `usecase/regenerate.go:82-83`) → tabrakan pasti untuk setiap artefak hasil run. Seharusnya yang masuk history adalah versi lama hanya bila belum ada, dan versi baru (v2) yang disimpan ke `artifacts`. Perbaikan ada di `server/` (tidak disentuh sesuai aturan). |
| Kartu proyek navigasi pakai slug | Eksekutor (F10, diperbaiki di FE) | `DashProjectList.svelte` mengarahkan ke `/projects/<slug-nama>`, padahal route `[id]` mencari UUID -> halaman detail kosong. FE kini mengirim `card.id` (UUID) dan jatuh ke slug hanya untuk data contoh |

## Dicek saat setup (bukan keputusan planning)

- Durasi expire order: diambil dari `expired_date` yang dikembalikan DOKU (`auth_billing.md` 2.4, 5).

## Ditunda

Paket harga vs biaya token (user: tidak perlu estimasi sekarang), BYOK, stale policy regenerate, tahap `seed`/`test-plan`/`ci`, tabel audit terpisah, endpoint AI streaming, slot `ai_settings` selain `planner`, nama model dan base URL 9router (diisi admin di panel).
