"""Flow bukti Fase 10: register -> konfirmasi (service role) -> login ->
project -> topup DOKU (redirect payment_url + webhook sandbox bertanda tangan)
-> kredit naik -> run sampai done (MOCK LLM) -> regenerate.

Screenshot: fase10-flow-1..6. Password/key tidak pernah dicetak.
"""
import json
import pathlib
import secrets
import subprocess
import sys

from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parents[2]
DEV_MOCK = ROOT / "tools" / "dev-mock"
sys.path.insert(0, str(DEV_MOCK))
from flow_lib import send_webhook, latest_order, doku_env  # noqa: E402

DEV = "http://127.0.0.1:5181"
OUT = ROOT / "docs" / "screenshots"
CREDS = DEV_MOCK / "creds.local.json"
EMAIL = f"fase10.flow.{secrets.token_hex(3)}@paluketok.test"   # user flow
EMAIL_UI = f"fase10.flow.{secrets.token_hex(3)}@mailinator.com"  # percobaan form register
PWD = "Flow10!" + secrets.token_hex(6)


def shot(pg, name: str) -> None:
    pg.screenshot(path=str(OUT / name), full_page=True)
    print("saved", name)


def seed(*args: str) -> str:
    r = subprocess.run([sys.executable, str(DEV_MOCK / "seed_users.py"), *args],
                       capture_output=True, text=True)
    if r.returncode != 0:
        print("seed gagal:", r.stdout[-200:], r.stderr[-200:])
    return r.stdout


def login(pg, email: str, pwd: str) -> None:
    pg.goto(f"{DEV}/login", wait_until="networkidle")
    pg.fill('input[type="email"]', email)
    pg.fill('input[type="password"]', pwd)
    pg.click('button[type="submit"]')
    pg.wait_for_url(f"{DEV}/projects", timeout=30000)


def credits(pg) -> int:
    """Baca saldo kredit dari teks tombol beli bila tampil, else 0."""
    pg.goto(f"{DEV}/billing", wait_until="networkidle")
    pg.wait_for_timeout(1500)
    txt = pg.inner_text("main")
    for line in txt.splitlines():
        if "kredit" in line and "Rp" not in line:
            digits = "".join(c for c in line if c.isdigit())
            if digits:
                return int(digits)
    return -1


def main() -> None:
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": 1440, "height": 1000})
        errs: list[str] = []
        pg.on("pageerror", lambda e: errs.append(str(e)[:150]))

        # --- 1. register ---
        # Form register diisi pakai email @mailinator.com (domain yang
        # diterima validator Supabase). Supabase kini membalas 429
        # over_email_send_rate_limit karena kuota email sandbox habis; user
        # flow lalu dibuat lewat service role (izin Master) dan itu dicatat.
        pg.goto(f"{DEV}/register", wait_until="networkidle")
        pg.fill('input[type="email"]', EMAIL_UI)
        pg.fill('input[type="password"]', PWD)
        pg.click('button[type="submit"]')
        pg.wait_for_timeout(2500)
        shot(pg, "fase10-flow-1-register.png")
        t = pg.inner_text("main").lower()
        ok_reg = "cek emailmu" in t
        print("1 register sukses:", ok_reg, "| rate-limit:", "rate limit" in t)

        # user flow: dibuat lewat service role (proyek paluketok saja)
        cr = json.loads(CREDS.read_text()) if CREDS.exists() else {}
        cr[EMAIL] = {"password": PWD}
        CREDS.write_text(json.dumps(cr, indent=2))
        if not ok_reg:
            print("buat via service role:", seed("create", EMAIL, "--pwd", PWD).strip()[:80])
        print("confirm:", seed("confirm", EMAIL).strip()[:80])

        # --- 2. login ---
        login(pg, EMAIL, PWD)
        shot(pg, "fase10-flow-2-login.png")
        print("2 login ->", pg.url.rsplit("/", 1)[-1])

        # --- 3. buat project ---
        pg.goto(f"{DEV}/projects", wait_until="networkidle")
        pg.wait_for_timeout(1200)
        pg.click('button:has-text("Proyek baru")')
        pg.wait_for_timeout(600)
        pg.fill('input[placeholder="Nama proyek"]', "Fase 10 Flow")
        pg.fill('textarea[placeholder="Brief kebutuhan software"]',
                "Aplikasi kasir kecil untuk warung, satu kasir, cetak struk.")
        pg.click('button:has-text("Simpan proyek")')
        pg.wait_for_timeout(2500)
        shot(pg, "fase10-flow-3-project.png")
        ids = pg.eval_on_selector_all("[data-purpose='project-card'] button",
                                      "els => els.map(e => e.textContent.trim())")
        print("3 kartu project:", ids[:3])
        pg.click('button:has-text("Buka proyek")')
        pg.wait_for_url("**/projects/*", timeout=30000)
        pg.wait_for_timeout(1500)
        print("3 halaman proyek:", pg.url.split("/projects/")[-1][:12], "...")
        proj_url = pg.url
        shot(pg, "fase10-flow-3-project.png")

        # --- 4. topup DOKU: klik beli, ambil payment_url dari order ---
        pg.goto(f"{DEV}/billing", wait_until="networkidle")
        pg.wait_for_timeout(1500)
        before = pg.inner_text("main")
        print("kredit awal dari UI:", [l for l in before.splitlines() if "kredit" in l][:2])
        shot(pg, "fase10-flow-4-topup.png")
        auth_uid = pg.evaluate("() => JSON.parse(localStorage.getItem(Object.keys(localStorage).find(k => k.endsWith('-auth-token')))).user.id")

        # klik "Beli paket Starter" -> BE buat order, FE redirect ke
        # payment_url DOKU. Redirect dibiarkan jalan (itu buktinya); order
        # diambil dari DB via service role
        # (helper latest_order di flow_lib).
        pg.click('button:has-text("Beli paket Starter")')
        pg.wait_for_timeout(5000)
        shot(pg, "fase10-flow-4-topup.png")
        print("setelah klik beli ->", pg.url[:66])
        o = latest_order(doku_env(), auth_uid)
        if not o:
            b.close()
            raise SystemExit("order tak ada di DB: flow berhenti")
        print("order:", o["invoice_number"], o["status"],
              "amount", o["amount_idr"], "kredit", o["credits_granted"])

        # --- 5. settle lewat webhook sandbox bertanda tangan HMAC ---
        res = send_webhook(o["invoice_number"], o["amount_idr"], "SUCCESS")
        print("webhook:", res)
        pg.goto(f"{DEV}/billing", wait_until="networkidle")
        pg.wait_for_timeout(2500)
        after = pg.inner_text("main")
        print("5 kredit setelah settle:", [l for l in after.splitlines() if "kredit" in l][:2])
        shot(pg, "fase10-flow-5-kredit-naik.png")

        # --- 6. run sampai done (mock LLM) lalu regenerate satu artefak ---
        pg.goto(proj_url, wait_until="networkidle")
        pg.wait_for_timeout(1200)
        pg.click('button:has-text("Mulai generate")')
        print("6 run dimulai, tunggu status done...")
        for _ in range(40):
            pg.wait_for_timeout(3000)
            body = pg.inner_text("main")
            if "Tahap selesai" in body or "6/6" in body or "done" in body.lower():
                break
            if "gagal" in body.lower() or "Gagal" in body:
                print("RUN GAGAL:", [l for l in body.splitlines() if "gagal" in l.lower()][:2])
                break
        pg.wait_for_timeout(2000)
        body = pg.inner_text("main")
        print("6 status akhir:", [l for l in body.splitlines() if "tahap" in l.lower()][:2])
        shot(pg, "fase10-flow-6-regenerate.png")
        regen = pg.query_selector('button:has-text("Regenerate")')
        if regen:
            regen.click()
            pg.wait_for_timeout(4000)
            shot(pg, "fase10-flow-6-regenerate.png")
            print("6 regenerate diklik")
        else:
            print("6 tombol Regenerate belum tampil (run belum done)")

        b.close()


if __name__ == "__main__":
    sys.exit(main())