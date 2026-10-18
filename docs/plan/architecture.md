# Arsitektur PaluKetok

Domain: `paluketok.web.id`. Dokumen ini hanya berisi keputusan dan aturan dependency.
Tree folder: `structure.md`. Pipeline dan gate: `planner.md`. Limit JEV: `jev_ai.md`. Keamanan: `security.md`. Skema: `erd.dbml`. Kontrak: `openapi.md`.

## 1. Keputusan

- **Clean Architecture di Go, monolith satu proses.** Satu deploy, satu codebase.
- **Stack BE: `net/http` stdlib, `pgx`, bukan `gorm`, bukan Gin, bukan chi.**
- **Supabase = Postgres + Auth saja.** Go jadi sumber logika. FE bicara ke Go, bukan langsung ke DB.
- **RLS ON sebagai lapis kedua.** Lapis utama owner check di usecase.
- **Auth BE: JWT via JWKS Supabase sejak awal.** Middleware memverifikasi token di setiap route user-scoped, sesuai `security.md` bagian 2. Owner check di usecase tetap lapis kedua.
- **Key LLM dikelola admin.** Satu key OpenAI-compatible, plaintext di `ai_settings` slot `planner`, diisi admin lewat panel. Bukan mode auth BE.
- **Chatbot gratis, JEV judge server-internal.** Keputusan JEV, lihat `openapi.md` bagian AI.
- **DOKU = top-up kredit.** Webhook masuk ke Go.
- **Semua di satu VPS.** HTTP dan worker goroutine pool satu proses (lihat `deploy.md`). Tanpa serverless, tanpa Docker.

## 2. Aturan dependency

Dependency mengarah ke dalam. Lapisan luar tahu lapisan dalam, tidak sebaliknya.

| Lapisan | Boleh import | Tidak boleh import |
|---|---|---|
| `entity` | stdlib | semua lainnya |
| `port` | `entity`, stdlib | adapter, SDK |
| `usecase` | `entity`, `port`, stdlib | `gorm`, `pgx`, `net/http`, `delivery/*`, SDK LLM/JEV/DOKU |
| `repository`, `adapter/*` | `port`, `entity`, SDK masing-masing | `usecase` tidak boleh di-import balik |
| `delivery/*` | `usecase`, `entity`, `pkg/response` | `repository`, `adapter/*` langsung |
| `worker` | `usecase`, `entity`, stdlib | `repository`, `adapter/*`, `delivery/*` |
| `cmd/web` | semua | — (composition root) |

Cara cek: `go list -deps ./internal/usecase/...` tidak boleh memuat `gorm`, `pgx`, `net/http`, atau `delivery`.

## 3. Aturan perilaku

- Owner check di usecase: `owner_id == actor.ID`. Handler tidak memutuskan akses data.
- Actor (`user_id`, `is_admin`) dari JWT claim lalu `profiles` di server. Tidak pernah dari body atau query.
- Handler tidak query DB. Usecase tidak tahu `*http.Request`, hanya `context.Context`.
- Waktu lewat port `Clock`, supaya usecase bisa dites deterministik.
- Transaksi kredit ada di `repository` (`FOR UPDATE`), dipanggil dari usecase lewat port `LedgerRepo`.

## 4. Alasan

Usecase bisa dites tanpa DB dan tanpa HTTP. Ganti DOKU, JEV, atau LLM cukup ganti adapter. Ini yang diminta user.

## 5. Yang belum diputuskan

- Worker pool dan antrean run: lihat `deploy.md`.
- Pipeline tahap: lihat `planner.md`.
