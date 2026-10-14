"""Pelunasan Tahap A: /projects + /billing login dengan data API nyata."""
import json
import pathlib
from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[2]
DEV = "http://127.0.0.1:5181"
CREDS = ROOT / "tools" / "dev-mock" / "creds.local.json"
OUT = ROOT / "docs" / "screenshots"
EMAIL = "fase10.flow.9bcd25@paluketok.test"
import json as _j
_creds = _j.loads(CREDS.read_text())
EMAIL = next(e for e in _creds if "9bcd25" in e)

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000})
    creds = json.loads(CREDS.read_text())
    pg.goto(f"{DEV}/login", wait_until="networkidle")
    pg.fill('input[type="email"]', EMAIL)
    pg.fill('input[type="password"]', creds[EMAIL]["password"])
    pg.click('button[type="submit"]')
    pg.wait_for_url(f"{DEV}/projects", timeout=30000)
    pg.wait_for_timeout(1500)
    pg.screenshot(path=str(OUT / "fase10-projects.png"), full_page=True)
    print("projects saved")

    pg.goto(f"{DEV}/billing", wait_until="networkidle")
    pg.wait_for_timeout(1800)
    body = pg.inner_text("main")
    print("billing memuat ropa:", [l for l in body.splitlines() if "Rp" in l][:3])
    pg.screenshot(path=str(OUT / "fase10-billing.png"), full_page=True)
    print("billing saved")
    b.close()