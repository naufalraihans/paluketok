# Kontrak API (OpenAPI) - PaluKetok v1

Base: `https://paluketok.web.id/v1`. Auth: header `Authorization: Bearer <access_token>` dari Supabase Auth, diverifikasi JWKS (`security.md` bagian 2).

Endpoint JEV judge tidak diekspos. Dipakai pipeline planner di server (`planner.md` bagian 4).

## Endpoint user

| Method | Path | Isi | Auth |
|---|---|---|---|
| GET | `/v1/me` | profil + saldo kredit | user |
| GET | `/v1/projects` | list project milik user | user |
| POST | `/v1/projects` | buat project (name, brief, stack) | user |
| GET | `/v1/projects/{id}` | detail project + artefak terakhir | owner |
| PATCH | `/v1/projects/{id}` | update brief/stack/status | owner |
| DELETE | `/v1/projects/{id}` | arsip project (status `archived`) | owner |
| POST | `/v1/projects/{id}/runs` | mulai generate planning (potong 1 kredit) | owner |
| GET | `/v1/runs/{id}` | status run + progres tahap | owner |
| GET | `/v1/runs/{id}/artifacts` | semua artefak run | owner |
| GET | `/v1/artifacts/{id}` | satu artefak + versi | owner |
| POST | `/v1/artifacts/{id}/regenerate` | generate ulang satu tahap (1 kredit). Hanya jika run induk `done` DAN tidak ada run `running` di project yang sama; selain itu `422` | owner |
| GET | `/v1/decisions?run_id=` | jejak keputusan JEV milik sendiri (tanpa `state` mentah dan `criteria` internal) | owner |
| POST | `/v1/ai/chat` | chatbot (gratis, lihat bagian AI) | user |
| POST | `/v1/orders` | buat order top-up (DOKU Checkout) | user |
| GET | `/v1/orders/{id}` | status order (aturan expire di `auth_billing.md` 2.4) | owner |
| POST | `/v1/webhooks/doku` | callback DOKU | signature HMAC-SHA256 |

## Endpoint admin

Semua butuh `profiles.is_admin = true` yang dibaca server. Tanpa itu `403`.

| Method | Path | Isi |
|---|---|---|
| GET | `/v1/admin/projects` | list semua project |
| GET | `/v1/admin/runs` | list semua run (filter `status`, `user_id`) |
| PATCH | `/v1/admin/users/{id}` | ubah `plan_tier` (tidak bisa ubah `is_admin`) |
| POST | `/v1/admin/users/{id}/credits` | grant manual, tulis `credits_ledger` reason `grant` |
| GET | `/v1/admin/decisions` | log JEV semua user |
| GET | `/v1/admin/settings/ai` | daftar slot AI, `api_key` masked |
| PUT | `/v1/admin/settings/ai/{slot}` | set provider, base_url, model, api_key (plaintext di DB), max_input_tokens, max_output_tokens |

## Endpoint AI

### `POST /v1/ai/chat` (chatbot)

Gratis, tidak memotong kredit (keputusan JEV: `free`, 0.94). Biaya token ditanggung key admin, jadi dibatasi rate limit (`security.md` bagian 8).

Request:

```json
{
  "messages": [
    { "role": "user", "content": "Apa beda Supabase Auth dan JWT?" }
  ]
}
```

Aturan:

- `messages` maks 20 item, `content` maks 4.000 karakter per item, dan total gabungan maks 16.000 karakter. Yang terlampaui ditolak `422`.
- `role` hanya `user` atau `assistant`. Pesan `system` dari klien ditolak `422`.
- Tidak ada streaming di v1.

Response `200`:

```json
{
  "reply": "Supabase Auth menerbitkan JWT...",
  "model": "nama model dari ai_settings",
  "usage": { "input_tokens": 120, "output_tokens": 240 }
}
```

Error: `422 validation_error`, `429 rate_limited`, `503 unavailable` (LLM gagal atau key belum diisi admin).

### JEV judge (internal, tidak ada di API publik)

Dipanggil `usecase` planner di server. Kontrak gateway ada di `jev_ai.md`. Hasil penuh dicatat ke `decisions`; ke klien hanya ringkasan (stage, gate, ringkas keputusan), tanpa `state` mentah dan `criteria` internal

## Contoh

`POST /v1/projects`

```json
{
  "name": "PaluKetok Demo",
  "brief": "Website toko online sederhana, fitur lengkap",
  "stack": { "fe": "svelte", "be": "go", "db": "supabase", "payment": "doku" }
}
```

`201`: `{ "id": "uuid", "status": "draft", "created_at": "..." }`

`POST /v1/projects/{id}/runs` -> `202`

```json
{ "run_id": "uuid", "status": "queued", "stages": ["prd","erd","openapi","roles","security","structure"], "credits_charged": 1 }
```

`GET /v1/runs/{id}`

```json
{ "id": "uuid", "status": "running", "progress": 3, "total": 6, "current_stage": "openapi", "escalations": 0 }
```

`POST /v1/orders` -> `201`

```json
{ "order_id": "uuid", "invoice_number": "PK-...", "payment_url": "https://...", "expired_date": "20261012104711", "amount_idr": 25000, "credits_granted": 10 }
```

`GET /v1/admin/settings/ai` -> `200`

```json
{ "slots": [ { "slot": "planner", "provider": "9router", "base_url": "https://...", "model": "...", "api_key": "sk-...abcd", "is_active": true } ] }
```

## Error

```json
{ "error": { "code": "insufficient_credits", "message": "Saldo kurang" } }
```

Kode: `unauthorized` 401, `forbidden` 403, `not_found` 404, `validation_error` 422, `insufficient_credits` 402, `rate_limited` 429, `unavailable` 503, `internal` 500.

## Aturan

- Semua path user-scoped, selain webhook, wajib JWT valid.
- `runs` async: `202` + poll. Tanpa long-poll.
- Webhook DOKU: verifikasi signature `HMACSHA256` (`security.md` bagian 6). Header `Client-Id`, `Request-Id`, `Request-Timestamp`, `Signature`. `409` dari DOKU saat buat order = order sudah ada, bukan error.
- Billing: 1 run = 1 kredit, `regenerate` = 1 kredit, chat gratis. Gagal run = refund lewat `credits_ledger`.
- Tidak ada endpoint yang menerima `owner_id` atau `is_admin` dari klien.
