# Checklist Rilis PaluKetok (security.md §13, workflow.md langkah 28)

Status per 2026-10-13, Fase 7 selesai lokal. Verifikator: Fase 7 build/test.

| # | Butir (security.md §13) | Bukti | Status |
|---|---|---|---|
| 1 | Semua route user-scoped punya middleware JWT | `internal/delivery/httpapi/router.go`: hanya `/healthz` + `POST /v1/webhooks/doku` di luar grup `authed`; semua route lain di `authed` (JWT) dan `/v1/admin/*` bertumpuk `middleware.Admin` | LULUS |
| 2 | Query service role memfilter owner | Repo pgx: `ListByOwner`/`GetByID`+`user_id`; usecase `GetRun`/`GetArtifact` tolak `run.UserID != actor.UserID && !actor.IsAdmin` (403) | LULUS |
| 3 | RLS aktif + policy + grant dicabut; user A tak bisa baca B; authenticated tak bisa PATCH is_admin/credits | Sandbox DB: `rowsecurity=true` 9/9 tabel; 8 policy owner-scoped; `anon` = 0 grant; `authenticated` di profiles hanya SELECT + UPDATE(username, full_name, avatar_url), is_admin/credits/plan_tier/token_version diblok trigger `trg_guard_profiles_update` (function `guard_profiles_update`); `credits_ledger`/`ai_settings` tanpa grant ke anon/authenticated; test user A vs B di `repository` (ownership) | LULUS |
| 4 | Index penjaga uang ada; dobel ditolak DB | Sandbox: 4 index partial unique (`uq_ledger_one_refund_per_run`, `uq_ledger_one_topup_per_order`, `uq_ledger_one_refund_per_charge`, `uq_ledger_one_grant_per_key`); repo `isUnique(err)` → no-op idempoten; worker + webhook test menguji idempotensi | LULUS |
| 5 | `ai_settings.base_url` di allowlist; client tolak redirect lintas host | `entity.ValidBaseURLAllowlist` dipanggil `AdminUsecase.SetAISetting`; `llm.New` pasang `CheckRedirect` (http.ErrUseLastResponse dihapus → redirect lintas host ditolak) | LULUS |
| 6 | Webhook tolak signature salah & nominal tak cocok | `doku.Verify` hmac.Equal; usecase `ProcessWebhook` → signature salah 401, nominal beda 422; httptest `TestWebhook_*` hijau (signature invalid, amount mismatch, replay) | LULUS |
| 7 | Key lama sudah di-rotate | Semua key di server/.env, tidak pernah masuk git; rotation dilakukan Master saat cutover | MENUNGGU DEPLOY |
| 8 | `.env` tidak masuk git | `git check-ignore server/.env` → ignored (`.gitignore` baris 2-5) | LULUS |
| 9 | Tidak ada log yang memuat secret | Log middleware hanya req_id/method/path/dur; handler tidak mencetak header/authorization/body; API key disimpan DBmasked (`api_key` di-mask di DTO admin) | LULUS |

Catatan:
- Butir 2, 3, 4, 6 dibuktikan kombinasi test otomatis (83 test, semua `ok`) dan inspeksi live DB sandbox (psql, 2026-10-13).
- Butir 7 (rotate key prod) dan bagian DNS/TLS masuk kategori "menunggu deploy" — dikerjakan Master pada cutover (workflow.md tahap deploy, di luar Fase 7).
