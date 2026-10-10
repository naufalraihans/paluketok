# Struktur Folder - PaluKetok

```
PaluKetok/
├── server/                      # Go monolith
│   ├── cmd/api/main.go          # entrypoint
│   ├── internal/
│   │   ├── http/                # router, middleware, handler
│   │   │   ├── router.go
│   │   │   ├── middleware.go    # jwt, ratelimit, recover
│   │   │   └── handlers/
│   │   ├── planner/             # pipeline tahap + prompt
│   │   │   ├── pipeline.go
│   │   │   └── stages/
│   │   ├── judge/               # klien JEV + gate
│   │   │   ├── client.go
│   │   │   └── gate.go
│   │   ├── store/               # query Postgres (sqlc)
│   │   │   ├── queries/
│   │   │   └── db.go
│   │   ├── pay/                 # Midtrans
│   │   │   ├── client.go
│   │   │   └── webhook.go
│   │   └── worker/              # goroutine pool
│   │       └── pool.go
│   ├── migrations/              # SQL migrasi
│   ├── go.mod
│   └── .env.example
├── web/                         # SvelteKit + TS
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte           # landing
│   │   │   ├── (app)/
│   │   │   │   ├── dashboard/
│   │   │   │   ├── projects/[id]/
│   │   │   │   └── billing/
│   │   ├── lib/
│   │   │   ├── api.ts           # klien fetch ke Go
│   │   │   ├── auth.ts          # Supabase auth
│   │   │   └── components/
│   │   └── app.html
│   ├── package.json
│   └── svelte.config.js
├── docs/
│   └── agents/                  # charter 11 role
├── plan/                        # planning (folder ini)
└── README.md
```

## Aturan layer

- Handler tidak query DB langsung; lewat `store`.
- `planner` tidak tahu HTTP; hanya terima brief + kembalikan artefak.
- `judge` satu-satunya yang pegang base URL + key JEV.
- `worker` yang memanggil `planner`; handler hanya enqueue + poll.
- FE tidak pernah pegang service role key Supabase. Hanya anon key + JWT.
- Rahasia (service role, JEV key, Midtrans server key) hanya di `server/.env`.
