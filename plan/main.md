# Project PaluKetok

Platform untuk developer membuat planning proyek secara sistematis.
Input: brief developer. Output: paket planning (PRD, ERD, OpenAPI, roles, security, struktur).

Stack: FE Svelte + TS, BE Go (Clean Architecture), Supabase (PostgreSQL + Auth), DOKU, OpenAPI, DBDiagram.
Judge keputusan teknis: JEV AI.
Domain: `paluketok.web.id`.

## Daftar file

| File | Isi |
|---|---|
| `prd.md` | tujuan, stack, aturan kerja, scope |
| `architecture.md` | keputusan + aturan dependency (tanpa tree) |
| `structure.md` | tree folder `server/` (satu-satunya tempat tree) |
| `deploy.md` | target VPS, spek, setup |
| `erd.dbml` | sumber kebenaran skema |
| `erd.md` | aturan di luar DBML (kredit, ledger, RLS) |
| `openapi.md` | kontrak endpoint user, admin, AI |
| `roles.md` | role dan matriks izin |
| `auth_billing.md` | login, profil, DOKU, pemakaian kredit |
| `planner.md` | pipeline generate planning, gate JEV, validasi |
| `security.md` | trust boundary, JWT, webhook, rate limit, secret |
| `jev_ai.md` | kontrak gateway JEV |
| `gaps.md` | status keputusan dan yang masih terbuka |
| `changelog.md` | riwayat perubahan planning |
| `workflow.md` | urutan kerja fase 0 sampai 7 (final) |
| `main.md` | indeks ini |

## Charter role

`docs/agents/` berisi 11 charter (analyst, architect, uiux, backend, frontend, reviewer, qa, security, devops, sre, data).

## Status

Status: **FINAL dan BEKU**. Ditetapkan 2026-10-12. Belum ada kode `server/`.

Aturan beku:
- `plan/` tidak diedit lagi. Perubahan hanya boleh untuk memperbaiki kontradiksi internal, dan wajib dicatat di `changelog.md`.
- Keputusan baru yang mengubah kontrak (skema, endpoint, kredit, auth) tidak lewat edit langsung: ditulis sebagai catatan revisi di `changelog.md` dulu, baru diterapkan setelah ditinjau.
- Yang ditunda (fase lanjut) tetap ditunda: lihat `gaps.md` bagian Ditunda.
