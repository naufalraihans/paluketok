# Deploy PaluKetok ke VPS (docs/plan/workflow.md langkah 27, plan/deploy.md)

Build di laptop, kirim binary via `scp`, jalan sebagai service `systemd`.
Jangan Docker (docs/plan/deploy.md). Domain `paluketok.web.id` + TLS Caddy menunggu
langkah DNS (terpisah, butuh keputusan Master).

## 1. Build

```bash
cd server
GOOS=linux GOARCH=amd64 go build -o paluketok ./cmd/web
```

Cek arsitektur VPS dulu (`uname -m` di VPS: `x86_64` = amd64, `aarch64` = arm64).

## 2. Kirim

```bash
scp paluketok deploy/paluketok.service root@<VPS>:/tmp/
```

## 3. Pasang di VPS

```bash
# user service khusus, bukan root (docs/plan/security.md)
sudo useradd --system --home /opt/paluketok --shell /usr/sbin/nologin paluketok || true
sudo mkdir -p /opt/paluketok /etc/paluketok
sudo mv /tmp/paluketok /opt/paluketok/paluketok
sudo chmod 0755 /opt/paluketok/paluketok
sudo chown -R paluketok:paluketok /opt/paluketok

# env: isi manual, JANGAN dari repo (secret hanya server/.env Master)
sudo nano /etc/paluketok/.env          # SUPABASE_*, MIDTRANS_*, DATABASE_URL, APP_PORT, dst.
sudo chown root:paluketok /etc/paluketok/.env
sudo chmod 0640 /etc/paluketok/.env

# unit
sudo mv /tmp/paluketok.service /etc/systemd/system/paluketok.service
sudo systemctl daemon-reload
sudo systemctl enable --now paluketok
```

## 4. Cek

```bash
sudo systemctl status paluketok        # active (running)
curl -s http://127.0.0.1:8080/healthz  # {"status":"ok"}
sudo journalctl -u paluketok -f        # log
```

`APP_PORT` di `.env` mengatur port; healthz tanpa auth untuk probe.

## 5. Update versi berikutnya

```bash
# laptop
GOOS=linux GOARCH=amd64 go build -o paluketok ./cmd/web && scp paluketok root@<VPS>:/tmp/
# vps
sudo systemctl stop paluketok && sudo mv /tmp/paluketok /opt/paluketok/paluketok \
  && sudo chown paluketok:paluketok /opt/paluketok/paluketok \
  && sudo systemctl start paluketok
```

Restart service menutup run `running` sebagai failed + refund otomatis
(server_restart, Fase 6 langkah 25) dan mengulang run `queued`.

## Bootstrap admin

Setelah user mendaftar, jadikan admin:

```bash
cd server && go run ./cmd/seeder admin@paluketok.web.id
```

(id read dari `auth.users` Supabase; butuh `DATABASE_URL` di environment lokal.)
