"""Seed/normalisasi user Supabase untuk uji flow Fase 10 (sandbox paluketok).

Memakai SUPABASE_SERVICE_ROLE_KEY dari server/.env — nilainya tidak pernah
dicetak. Hanya diizinkan di project uzlpexcixmffbjxaomwv (dicek di awal).

Perintah:
  python seed_users.py create <email> [--admin]   -> cetak uuid user
  python seed_users.py confirm <email>            -> email_confirm=true
  python seed_users.py delete <uuid>|--all-created -> hapus user uji
  python seed_users.py list                       -> daftar user ber-domain uji

Semua user uji memakai domain TEST_DOMAIN agar cleanup aman dan tidak
menyentuh user lain.
"""
import json
import pathlib
import sys
import urllib.error
import urllib.parse
import urllib.request

ALLOWED_REF = "uzlpexcixmffbjxaomwv"
TEST_DOMAIN = "paluketok.test"
ENV = pathlib.Path(__file__).resolve().parents[2] / "server" / ".env"


def load_env() -> dict:
    out = {}
    for line in ENV.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            out[k.strip()] = v.strip().strip('"')
    ref = out["SUPABASE_URL"].split("//")[-1].split(".")[0]
    if ref != ALLOWED_REF:
        sys.exit(f"tolak: project {ref} bukan {ALLOWED_REF}")
    return out


def call(env: dict, method: str, url: str, body=None, extra=None):
    key = env["SUPABASE_SERVICE_ROLE_KEY"]
    headers = {"apikey": key, "Authorization": f"Bearer {key}",
               "Content-Type": "application/json"}
    headers.update(extra or {})
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            raw = r.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")
        # pesan Supabase bisa memuat email; aman ditampilkan (bukan rahasia kunci)
        sys.exit(f"HTTP {e.code}: {detail[:400]}")


def find(env: dict, email: str):
    """Cari user by email lewat admin list (paginasi sederhana)."""
    base = env["SUPABASE_URL"].rstrip("/")
    page = 1
    while page <= 20:
        q = urllib.parse.urlencode({"page": page, "per_page": 200})
        d = call(env, "GET", f"{base}/auth/v1/admin/users?{q}")
        users = d.get("users", [])
        for u in users:
            if u.get("email", "").lower() == email.lower():
                return u
        if len(users) < 200:
            return None
        page += 1
    return None


def cmd_create(env: dict, email: str, admin: bool, pwd: str = ""):
    if not email.endswith("@" + TEST_DOMAIN):
        sys.exit(f"tolak: email uji harus berdomain @{TEST_DOMAIN}")
    base = env["SUPABASE_URL"].rstrip("/")
    pwd = pwd or __import__("secrets").token_urlsafe(24)
    d = call(env, "POST", f"{base}/auth/v1/admin/users",
             {"email": email, "password": pwd, "email_confirm": True})
    uid = d["id"]
    print(f"create ok  id={uid}  email={email}  terkonfirmasi=True")
    if admin:
        call(env, "PATCH", f"{base}/rest/v1/profiles?id=eq.{uid}",
             {"is_admin": True}, extra={"Prefer": "return=minimal"})
        print(f"admin ok   id={uid}  profiles.is_admin=true")
    print(f"PASSWORD_UNTUK_LOGIN={pwd}")
    return uid


def cmd_confirm(env: dict, email: str):
    base = env["SUPABASE_URL"].rstrip("/")
    u = find(env, email)
    if not u:
        sys.exit("user tidak ada")
    call(env, "PUT", f"{base}/auth/v1/admin/users/{u['id']}",
         {"email_confirm": True})
    print(f"confirm ok id={u['id']}")


def cmd_setpass(env: dict, email: str, path: str):
    """Set password acak + tulis {email,password} ke file lokal tanpa cetak."""
    base = env["SUPABASE_URL"].rstrip("/")
    u = find(env, email)
    if not u:
        sys.exit("user tidak ada")
    pwd = __import__("secrets").token_urlsafe(24)
    call(env, "PUT", f"{base}/auth/v1/admin/users/{u['id']}",
         {"password": pwd, "email_confirm": True})
    p = pathlib.Path(path)
    data = json.loads(p.read_text()) if p.exists() else {}
    data[email] = {"id": u["id"], "password": pwd}
    p.write_text(json.dumps(data, indent=2))
    print(f"setpass ok id={u['id']} -> {path} (password tidak dicetak)")


def cmd_list(env: dict):
    base = env["SUPABASE_URL"].rstrip("/")
    q = urllib.parse.urlencode({"page": 1, "per_page": 200})
    d = call(env, "GET", f"{base}/auth/v1/admin/users?{q}")
    for u in d.get("users", []):
        em = u.get("email", "")
        if em.endswith("@" + TEST_DOMAIN):
            print(f"{u['id']}  {em}  confirmed={u.get('email_confirmed_at') is not None}")


def cmd_delete(env: dict, target: str):
    base = env["SUPABASE_URL"].rstrip("/")
    if target == "--all-created":
        q = urllib.parse.urlencode({"page": 1, "per_page": 200})
        d = call(env, "GET", f"{base}/auth/v1/admin/users?{q}")
        ids = [u["id"] for u in d.get("users", [])
               if u.get("email", "").endswith("@" + TEST_DOMAIN)]
    else:
        ids = [target]
    for uid in ids:
        call(env, "DELETE", f"{base}/auth/v1/admin/users/{uid}")
        print(f"hapus ok   id={uid}")
    if not ids:
        print("tidak ada user uji untuk dihapus")


def main():
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    env = load_env()
    cmd, *rest = sys.argv[1:]
    if cmd == "create":
        admin = "--admin" in rest
        args = [a for a in rest if a != "--admin"]
        pwd = ""
        if "--pwd" in args:
            i = args.index("--pwd")
            pwd = args[i + 1]
            args = args[:i] + args[i + 2:]
        cmd_create(env, args[0], admin, pwd)
    elif cmd == "confirm":
        cmd_confirm(env, rest[0])
    elif cmd == "setpass":
        cmd_setpass(env, rest[0], rest[1])
    elif cmd == "list":
        cmd_list(env)
    elif cmd == "delete":
        cmd_delete(env, rest[0])
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()