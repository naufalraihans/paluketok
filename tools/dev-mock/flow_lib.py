"""Helper flow Fase 10: tanda tangan webhook DOKU + panggil API BE.

Rahasia tidak pernah dicetak.
"""
import base64
import hashlib
import hmac
import json
import pathlib
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[2]
WEBHOOK_PATH = "/v1/webhooks/doku"


def doku_env() -> dict:
    env = {}
    for line in (ROOT / "server" / ".env").read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if "=" in line and not line.startswith("#"):
            k, v = line.split("=", 1)
            env[k.strip()] = v.strip()
    return env


def sign_webhook(raw: bytes) -> dict:
    """Header bertanda tangan persis seperti doku.signRequest (verbatim)."""
    env = doku_env()
    cid = env["DOKU_CLIENT_ID"]
    secret = env["DOKU_SECRET_KEY"]
    rid = "flow-req-0001"
    ts = "2026-10-14T00:00:00Z"
    digest = base64.b64encode(hashlib.sha256(raw).digest()).decode()
    comp = (
        f"Client-Id:{cid}\nRequest-Id:{rid}\nRequest-Timestamp:{ts}\n"
        f"Request-Target:{WEBHOOK_PATH}\nDigest:{digest}"
    )
    sig = base64.b64encode(
        hmac.new(secret.encode(), comp.encode(), hashlib.sha256).digest()
    ).decode()
    return {
        "Client-Id": cid,
        "Request-Id": rid,
        "Request-Timestamp": ts,
        "Signature": "HMACSHA256=" + sig,
        "Content-Type": "application/json",
    }


def latest_order(env: dict, uid: str) -> dict:
    """Order terbaru milik user dari Supabase REST (service role)."""
    url = env["SUPABASE_URL"].rstrip("/") + (
        f"/rest/v1/orders?user_id=eq.{uid}&order=created_at.desc&limit=1")
    req = urllib.request.Request(url, headers={
        "apikey": env["SUPABASE_SERVICE_ROLE_KEY"],
        "Authorization": f"Bearer {env['SUPABASE_SERVICE_ROLE_KEY']}",
    })
    with urllib.request.urlopen(req, timeout=30) as r:
        rows = json.load(r)
    return rows[0] if rows else {}


def send_webhook(invoice: str, amount: int, status: str = "SUCCESS") -> dict:
    raw = json.dumps({
        "order": {
            "invoice_number": invoice,
            "amount": amount,
            "status": "ORDER_PAID",
        },
        "transaction": {"status": status},
    }, separators=(",", ":")).encode()
    import urllib.request
    req = urllib.request.Request(
        "http://127.0.0.1:8080" + WEBHOOK_PATH, data=raw, method="POST",
        headers=sign_webhook(raw),
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return {"http": r.status, **json.loads(r.read())}
    except urllib.error.HTTPError as e:
        return {"http": e.code, "body": e.read().decode()[:200]}