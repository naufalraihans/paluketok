# PaluKetok

Platform yang mengubah brief developer menjadi paket planning lengkap: PRD, ERD, OpenAPI, role access, proteksi, dan struktur folder.

Dokumen planning ada di `plan/`, mulai dari `plan/main.md`.

## Status

- Planning `plan/`: FINAL dan BEKU.
- Backend Go `server/`: ada. Fase 0-7 tuntas.
- Uji: 9 paket hijau, 84 `func Test` dideklarasi (74 PASS, 10 SKIP butuh DB/live). `gofmt`/`go build`/`go vet` bersih.
- FE Svelte: belum dibangun (keputusan ditunda, fokus BE).
- Belum siap deploy: `POST /v1/ai/chat` belum diimplementasikan, rate limit belum di-wire, `PLANNER_LLM_*` belum diisi. Lihat `plan/gaps.md`.

## Stack

- FE: Svelte + TypeScript
- BE: Go
- DB + Auth: PostgreSQL via Supabase
- Payment: DOKU Checkout
- Judge keputusan: JEV AI

