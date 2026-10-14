"""Screenshot 5 tab admin (login dulu sebagai flow-admin via UI nyata).

Password dibaca dari creds.local.json; tidak pernah dicetak.
"""
import json
import pathlib
import sys
from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[2]
DEV = "http://127.0.0.1:5181"
OUT = ROOT / "docs" / "screenshots"
CREDS = ROOT / "tools" / "dev-mock" / "creds.local.json"
ADMIN_EMAIL = "flow-admin@paluketok.test"

TABS = ["projects", "runs", "users", "decisions", "ai"]
# UUID user flow (punya run), dipakai mengisi filter tab Runs karena
# GET /v1/admin/runs tanpa user_id membalas 500 (bug BE dilaporkan).
UID = ""


def main() -> None:
    global UID
    creds = json.loads(CREDS.read_text())
    pwd = creds[ADMIN_EMAIL]["password"]
    sys.path.insert(0, str(ROOT / "tools" / "dev-mock"))
    flow = next(e for e in creds if e.startswith("fase10.flow")
                and e.endswith("@paluketok.test") and "9bcd25" in e)
    envm = {}
    for line in (ROOT / "server" / ".env").read_text(encoding="utf-8").splitlines():
        if "=" in line and not line.startswith("#"):
            k, v = line.split("=", 1)
            envm[k.strip()] = v.strip()
    import urllib.request
    req = urllib.request.Request(
        envm["SUPABASE_URL"].rstrip("/") + "/auth/v1/admin/users?page=1&per_page=200",
        headers={"apikey": envm["SUPABASE_SERVICE_ROLE_KEY"],
                 "Authorization": f"Bearer {envm['SUPABASE_SERVICE_ROLE_KEY']}"})
    users = json.load(urllib.request.urlopen(req, timeout=30))["users"]
    UID = next(u["id"] for u in users if u["email"] == flow)
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": 1440, "height": 1000})
        pg.goto(f"{DEV}/login", wait_until="networkidle")
        pg.fill('input[type="email"]', ADMIN_EMAIL)
        pg.fill('input[type="password"]', pwd)
        pg.click('button[type="submit"]')
        pg.wait_for_url(f"{DEV}/projects", timeout=30000)
        print("login admin ok ->", pg.url)

        for t in TABS:
            pg.goto(f"{DEV}/admin", wait_until="networkidle")
            pg.wait_for_timeout(1200)
            label = {"projects": "Projects", "runs": "Runs", "users": "Users",
                     "decisions": "Keputusan", "ai": "AI Settings"}[t]
            pg.click(f'nav[aria-label="Tab admin"] button:has-text("{label}")')
            pg.wait_for_timeout(1500)
            # Bug BE: GET /v1/admin/runs tanpa user_id -> 500. Isi UUID user
            # flow (yang punya run) lalu Terapkan supaya tabel terisi.
            if t == "runs":
                pg.fill('input[placeholder="UUID (kosong = semua)"]', UID)
                pg.click('button:has-text("Terapkan")')
                pg.wait_for_timeout(2000)
            # AI Settings butuh scroll ke form
            if t == "ai":
                pg.mouse.wheel(0, 600)
                pg.wait_for_timeout(500)
            path = OUT / f"fase10-admin-{t}.png"
            pg.screenshot(path=str(path), full_page=True)
            print(f"saved {path.name}")
        b.close()


if __name__ == "__main__":
    sys.exit(main())