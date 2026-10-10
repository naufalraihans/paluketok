# Kontrak API (OpenAPI) - PaluKetok v1

Base: `/v1`. Auth: Supabase JWT di header `Authorization: Bearer <jwt>`.

## Endpoints

| Method | Path | Isi | Auth |
|---|---|---|---|
| GET | `/v1/me` | profil + saldo kredit | user |
| GET | `/v1/projects` | list project milik user | user |
| POST | `/v1/projects` | buat project (name, brief, stack) | user |
| GET | `/v1/projects/{id}` | detail project + artefak terakhir | owner |
| PATCH | `/v1/projects/{id}` | update brief/stack/status | owner |
| DELETE | `/v1/projects/{id}` | arsip project | owner |
| POST | `/v1/projects/{id}/runs` | mulai generate planning (potong kredit) | owner |
| GET | `/v1/runs/{id}` | status run + progres tahap | owner |
| GET | `/v1/runs/{id}/artifacts` | semua artefak run | owner |
| GET | `/v1/artifacts/{id}` | satu artefak + versi | owner |
| POST | `/v1/artifacts/{id}/regenerate` | generate ulang satu tahap | owner |
| GET | `/v1/decisions?run_id=` | jejak keputusan JEV | owner |
| POST | `/v1/orders` | buat order top-up (Midtrans Snap) | user |
| GET | `/v1/orders/{id}` | status order | owner |
| POST | `/v1/webhooks/midtrans` | callback Midtrans | signature |

## Contoh

`POST /v1/projects`

```json
{
  "name": "PaluKetok Demo",
  "brief": "Website toko online sederhana, fitur lengkap",
  "stack": { "fe": "svelte", "be": "go", "db": "supabase", "payment": "midtrans" }
}
```

`201`

```json
{ "id": "uuid", "status": "draft", "created_at": "..." }
```

`POST /v1/projects/{id}/runs` -> `202`

```json
{
  "run_id": "uuid",
  "status": "queued",
  "stages": ["prd","erd","openapi","roles","security","structure"],
  "credits_charged": 1
}
```

`GET /v1/runs/{id}`

```json
{
  "id": "uuid",
  "status": "running",
  "progress": 3,
  "total": 6,
  "current_stage": "openapi",
  "escalations": 0
}
```

`POST /v1/orders` -> `201`

```json
{ "order_id": "uuid", "midtrans_order_id": "PK-...", "snap_token": "...", "amount_idr": 25000, "credits_granted": 10 }
```

## Error

```json
{ "error": { "code": "insufficient_credits", "message": "Saldo kurang" } }
```

Kode: `unauthorized` 401, `forbidden` 403, `not_found` 404, `validation_error` 422, `insufficient_credits` 402, `rate_limited` 429, `internal` 500.

## Aturan

- Semua path user-scoped, selain webhook, wajib JWT valid.
- `runs` sengaja async: `202` + poll. Tidak ada long-poll.
- Webhook Midtrans diverifikasi `signature_key` (SHA512 order_id+status_code+gross_amount+server_key).
- Billing: 1 run = 1 kredit. Gagal = tidak dipotong. Refund manual lewat `credits_ledger`.
- Rate limit JEV di sisi server, bukan per user request.
