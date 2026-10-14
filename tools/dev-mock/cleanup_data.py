"""Cleanup data uji fase 10: hapus baris anak per user uji, lalu user auth.

Urutan mengikuti FK: artifact_versions -> artifacts -> plan_runs ->
decisions/credits_ledger/orders -> projects -> profiles -> auth user.
Hanya menyentuh user berdomain @paluketok.test di project yang diizinkan.
"""
import json
import pathlib
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[2]
ALLOWED_REF = "uzlpexcixmffbjxaomwv"
DOMAIN = "paluketok.test"


def load_env() -> dict:
    env = {}
    for line in (ROOT / "server" / ".env").read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if "=" in line and not line.startswith("#"):
            k, v = line.split("=", 1)
            env[k.strip()] = v.strip().strip('"')
    ref = env["SUPABASE_URL"].split("//")[-1].split(".")[0]
    if ref != ALLOWED_REF:
        raise SystemExit(f"tolak: project {ref} bukan {ALLOWED_REF}")
    return env


def rest(env, table, param, method="DELETE"):
    base = env["SUPABASE_URL"].rstrip("/") + "/rest/v1/" + table
    url = base + "?" + urllib.parse.urlencode(param)
    key = env["SUPABASE_SERVICE_ROLE_KEY"]
    req = urllib.request.Request(url, method=method, headers={
        "apikey": key, "Authorization": f"Bearer {key}", "Prefer": "return=representation"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status
    except urllib.error.HTTPError as e:
        return f"{e.code}: {e.read().decode()[:120]}"


def main() -> None:
    env = load_env()
    base = env["SUPABASE_URL"].rstrip("/")
    key = env["SUPABASE_SERVICE_ROLE_KEY"]
    q = urllib.parse.urlencode({"page": 1, "per_page": 200})
    req = urllib.request.Request(base + "/auth/v1/admin/users?" + q, headers={
        "apikey": key, "Authorization": f"Bearer {key}"})
    users = json.load(urllib.request.urlopen(req, timeout=60))["users"]
    mine = [u for u in users if u.get("email", "").endswith("@" + DOMAIN)]
    ids = [u["id"] for u in mine]
    idlist = ",".join(ids)
    print("user uji:", len(ids))
    if not ids:
        return
    inu = {"user_id": f"in.({idlist})"}
    print("artifact_versions:", rest(env, "artifact_versions", inu))
    print("artifacts:", rest(env, "artifacts", inu))
    # plan_runs/orders/credits_ledger/decisions: cek nama kolom user
    print("plan_runs:", rest(env, "plan_runs", inu))
    print("decisions:", rest(env, "decisions", inu))
    print("credits_ledger:", rest(env, "credits_ledger", inu))
    print("orders:", rest(env, "orders", inu))
    print("projects:", rest(env, "projects", {"owner_id": f"in.({idlist})"}))
    # hapus profil lalu user auth
    print("profiles:", rest(env, "profiles", {"id": f"in.({idlist})"}))
    for uid in ids:
        req = urllib.request.Request(
            base + "/auth/v1/admin/users/" + uid, method="DELETE",
            headers={"apikey": key, "Authorization": f"Bearer {key}"})
        try:
            urllib.request.urlopen(req, timeout=60)
        except urllib.error.HTTPError as e:
            print("user", uid[:8], "->", e.code, e.read().decode()[:80])
    print("selesai")


if __name__ == "__main__":
    main()
