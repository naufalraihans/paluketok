# Struktur folder - PaluKetok

Acuan pola: `E:\rekapProject\project_cpns\backend` (cmd dua entrypoint, pkg kecil, internal per lapisan). Aturan dependency dan alasan ada di `plan/architecture.md`.

## Tree

```
server/
├── cmd/
│   ├── web/main.go              # composition root: config -> logger -> db -> route -> worker -> serve
│   └── seeder/main.go           # seed data dev + bootstrap admin pertama (argumen email)
├── pkg/                         # tanpa logika bisnis, boleh dipakai siapa saja
│   ├── config/                  # viper + env, satu struct config
│   ├── database/                # pgxpool, Connect()
│   ├── logger/                  # logger tunggal
│   ├── response/                # format JSON sukses + error (kode di openapi.md)
│   └── jwt/                     # verifikasi JWT via JWKS (cache + refresh kid)
├── internal/
│   ├── entity/                  # struct domain murni (Profile, Project, PlanRun, Order, ...)
│   ├── port/                    # interface: repository, LLM, judge, payment
│   ├── usecase/                 # logika bisnis + owner check; input = struct usecase
│   ├── repository/              # implementasi port dengan Postgres (pgx), transaksi FOR UPDATE
│   ├── adapter/
│   │   ├── llm/                 # client OpenAI-compatible (slot planner, chat)
│   │   ├── judge/               # client JEV gateway (internal)
│   │   └── payment/doku/        # create order, status check, verify signature HMAC-SHA256
│   ├── worker/                  # pool tetap, jalankan tahap planner
│   └── delivery/
│       ├── dto/                 # struct request/response HTTP (json tag, validasi)
│       ├── handler/             # parse dto -> usecase -> response
│       ├── middleware/          # jwt, admin, ratelimit, recover, bodylimit, cors
│       └── route/               # daftar route net/http
├── database/
│   ├── migrations/              # 0001_init.sql dari erd.dbml + RLS + trigger profiles
│   └── seeder/                  # data dev untuk cmd/seeder
├── go.mod
└── .env.example                 # nama variabel saja, tanpa nilai
```

## Yang diadopsi dari `project_cpns`

| Pola | Dipakai di sini |
|---|---|
| `cmd/web` + `cmd/seeder` | ya |
| `pkg/config`, `pkg/response`, `pkg/logger`, `pkg/database` | ya |
| `internal/entity`, `internal/repository`, `internal/usecase`, `internal/worker` | ya |
| `internal/dto` terpisah dari entity | ya, dipindah ke `delivery/dto` |
| `internal/delivery/{handler,middleware,route}` | ya |
| `database/migrations`, `database/seeder` | ya |

## Yang tidak disalin, dan kenapa

| Di `project_cpns` | Di sini | Alasan |
|---|---|---|
| `usecase` import `gorm` dan `dto` | usecase hanya import `entity` dan `port` | Clean Architecture. Usecase tidak tahu DB dan HTTP |
| `delivery/policy/*` | owner check di usecase | `security.md` 3 lapis 2. Satu tempat, tidak tersebar |
| `internal/helper` (json, time, param) | dihapus | `pkg/response` dan stdlib cukup |
| `docs/docs.go` (swag) | dihapus | kontrak ada di `openapi.md` |
| `nguwawor/`, binary di root (`app`, `e2e_web.exe`) | tidak ada | bukan kode, tidak masuk repo |
| `route/injector.go` | `cmd/web/main.go` | composition root di satu tempat |
| `gorm` | `pgx` | sesuai `architecture.md` |

## Aturan

- Aturan dependency: `architecture.md` bagian 2.
- `handler` tidak memanggil `repository` atau `adapter` langsung. Lewat `usecase`.
- `worker` memanggil `usecase` per tahap, tidak memanggil `adapter` langsung.
- Owner check di `usecase`. Actor dari JWT claim, bukan dari body atau query.
- Rahasia dan lokasinya: lihat `security.md` bagian 9.
- Sumber tunggal angka timeout adapter: LLM 180 detik, JEV 30 detik, DOKU 20 detik. Semua adapter keluar memakai `http.Client` dengan timeout eksplisit, tidak ada `http.DefaultClient`. File lain merujuk ke sini, bukan menyalin angkanya.
- Router: `net/http` stdlib sesuai constraint awal. `project_cpns` pakai Gin; ini satu-satunya perbedaan framework yang disengaja.
