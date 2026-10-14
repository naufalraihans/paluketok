"""Login password grant Supabase + probe endpoint /v1 (tanpa mencetak kredensial).

Dipakai untuk memverifikasi bentuk respons backend sungguhan sebelum dan
sesudah membangun UI, dan untuk menggerakkan flow uji.
"""
import json
import pathlib
import sys
import urllib.error
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[2]
ENV = ROOT / "server" / ".env"
CREDS = pathlib.Path(__file__).parent / "creds.local.json"
API = "http://127.0.0.1:8080"


def env() -> dict:
    out = {}
    for line in ENV.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            out[k.strip()] = v.strip().strip('"')
    return out


def token(email: str) -> str:
    e = env()
    creds = json.loads(CREDS.read_text())
    pwd = creds[email]["password"]
    url = e["SUPABASE_URL"].rstrip("/") + "/auth/v1/token?grant_type=password"
    req = urllib.request.Request(
        url,
        data=json.dumps({"email": email, "password": pwd}).encode(),
        headers={"apikey": e["SUPABASE_ANON_KEY"], "Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read())["access_token"]


def call(tok: str, method: str, path: str, body=None):
    req = urllib.request.Request(
        API + path,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Authorization": f"Bearer {tok}", "Content-Type": "application/json"},
        method=method,
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read()
            return r.status, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(detail)
        except Exception:
            return e.code, detail[:300]


if __name__ == "__main__":
    email = sys.argv[1] if len(sys.argv) > 1 else "flow-admin@paluketok.test"
    path = sys.argv[2] if len(sys.argv) > 2 else "/v1/me"
    method = sys.argv[3] if len(sys.argv) > 3 else "GET"
    st, d = call(token(email), method, path)
    keys = list(d.keys()) if isinstance(d, dict) else type(d).__name__
    print("status:", st)
    print("keys:", keys)
    items = None
    if isinstance(d, dict):
        for v in d.values():
            if isinstance(v, list) and v and isinstance(v[0], dict):
                items = list(v[0].keys())
    if items:
        print("item keys:", items)
    print(json.dumps(d, indent=2)[:1200])