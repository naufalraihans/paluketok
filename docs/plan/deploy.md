# Deploy - PaluKetok

Target: satu VPS. Semua di sana (HTTP + worker). Tanpa Vercel, tanpa serverless.
Domain `paluketok.web.id` dan `admin.paluketok.web.id` mengarah ke VPS.

## Kenapa VPS penuh, bukan serverless

Pipeline planner 6 tahap berurutan (`planner.md` bagian 1) memanggil LLM tiap tahap. Di serverless, durasi total bisa melewati batas fungsi. JEV memberi `ESCALATE` 0.45 untuk pertanyaan itu (`hybrid` 0.59, `vercel_qstash` 0.34, `vps` 0.07). User memutuskan: seluruh BE di satu VPS. Keputusan ini menghapus batas durasi dan antrean eksternal.

## Spesifikasi VPS

| Sumber daya | Nilai | Catatan |
|---|---|---|
| vCPU | 1 | worker 99% menunggu jaringan |
| RAM | 1 GB | binary Go + pool pgx jauh di bawah 100 MB |
| Disk | 20 GB | binary, log, tanpa Docker image |
| OS | Ubuntu 24.04 LTS | `systemd` sudah ada |

Naikkan hanya jika nanti jalan Postgres lokal, build di VPS, atau alat tambahan.

## Setup

- SSH pakai key, login password dimatikan.
- `ufw`: buka 22, 80, 443 saja.
- Jalan sebagai `systemd` service: restart otomatis, log ke journal.
- Build di laptop: `GOOS=linux GOARCH=amd64 go build -o paluketok ./cmd/web` (sesuaikan arsitektur VPS), kirim binary via `scp`.
- Tanpa Docker di mana pun, dev maupun prod.
- Go: 1.26 (`go.mod`). Build pakai toolchain 1.26.x.

## Yang perlu dicek

- Arsitektur VPS saat beli (`amd64` atau `arm64`) supaya `GOARCH` benar.
- Port 80 dan 443 hanya dipakai kalau ada reverse proxy (nginx/caddy) untuk TLS. Worker sendiri tidak butuh port publik.
- Outbound wajib bisa ke: Supabase pooler, DOKU, endpoint LLM (base URL admin), dan gateway JEV.
