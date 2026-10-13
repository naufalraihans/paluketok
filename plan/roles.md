# Role Access - PaluKetok

## Role

| Role | Deskripsi |
|---|---|
| `anon` | belum login. hanya lihat landing + login |
| `user` | login. punya project + kredit sendiri |
| `admin` | lihat semua project/run, atur tier + kredit, lihat log JEV |

Tidak ada role `editor`/`viewer`. Satu project = satu owner. Sharing besok.

## Matriks izin

| Resource | anon | user | admin |
|---|---|---|---|
| landing, login | read | read | read |
| `me` | - | read | read |
| projects | - | CRUD milik sendiri | read semua |
| runs | - | create/read milik sendiri | read semua |
| artifacts | - | read/regenerate milik sendiri | read semua |
| decisions | - | read milik sendiri | read semua |
| orders | - | create/read milik sendiri | read semua |
| kredit user lain | - | - | update (manual grant) |
| tier user lain | - | - | update |
| webhook DOKU | - | - | - (tanpa JWT, dilindungi signature HMAC-SHA256, `security.md` 6) |

## Enforcement (3 lapis)

1. **JWT** - middleware Go validasi token Supabase. Gagal = 401.
2. **App** - handler cek `owner_id == claims.sub`. Gagal = 403.
3. **RLS** - Postgres policy `owner_id = auth.uid()`. Lapisan jaring, bukan satu-satunya.

Admin ditandai di kolom `profiles.is_admin` (default `false`). `plan_tier` tidak menandai admin.

## Aturan

- Role dari JWT claim, bukan dari request body.
- Tidak ada endpoint yang menerima `owner_id` dari klien.
- Setiap deny 403 dicatat ke log server (`security.md` bagian 3).
