"""Isi slot planner lewat PANEL ADMIN (bukan env, bukan SQL).

base_url = tunnel HTTPS ke mock LLM. Key diisi nilai dummy (mock tak
memeriksa). Key tidak dicetak ke stdout.
"""
import json
import pathlib
import sys
from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[2]
DEV = "http://127.0.0.1:5181"
CREDS = ROOT / "tools" / "dev-mock" / "creds.local.json"
BASE_URL = sys.argv[1] if len(sys.argv) > 1 else ""
DUMMY_KEY = "mock-dev-key-tidak-nyata-0000"


def main() -> None:
    if not BASE_URL.startswith("https://"):
        sys.exit("butuh base_url https")
    creds = json.loads(CREDS.read_text())
    pwd = creds["flow-admin@paluketok.test"]["password"]
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": 1440, "height": 1000})
        pg.goto(f"{DEV}/login", wait_until="networkidle")
        pg.fill('input[type="email"]', "flow-admin@paluketok.test")
        pg.fill('input[type="password"]', pwd)
        pg.click('button[type="submit"]')
        pg.wait_for_timeout(4000)

        pg.goto(f"{DEV}/admin", wait_until="networkidle")
        pg.click('nav[aria-label="Tab admin"] button:has-text("AI Settings")')
        pg.wait_for_timeout(1000)
        pg.fill('input[placeholder="openai-compatible"]', "mock")
        pg.fill('input[placeholder="gpt-4o-mini"]', "mock-model")
        pg.fill('input[placeholder="https://api.example.com/v1"]', BASE_URL)
        pg.fill('input[type="number"] >> nth=0', "8000")
        pg.fill('input[type="number"] >> nth=1', "4000")
        pg.fill('input[type="password"]', DUMMY_KEY)
        pg.click('button:has-text("Simpan slot")')
        pg.wait_for_timeout(2500)
        body = pg.inner_text("main")
        ok = "tersimpan" in body
        print("simpan slot:", "OK" if ok else "GAGAL")
        print("pesan:", [l for l in body.splitlines() if "tersimpan" in l or "allowlist" in l][:2])
        pg.screenshot(path=str(ROOT / "docs" / "screenshots" / "fase10-admin-ai.png"), full_page=True)
        b.close()


if __name__ == "__main__":
    sys.exit(main())