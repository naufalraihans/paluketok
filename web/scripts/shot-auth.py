"""Screenshot 4 halaman baru Tahap A (login, register, reset, billing)."""
import pathlib
from playwright.sync_api import sync_playwright

OUT = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\docs\screenshots")
OUT.mkdir(parents=True, exist_ok=True)

PAGES = [
    ("/login", "fase10-login"),
    ("/register", "fase10-register"),
    ("/reset", "fase10-reset"),
    ("/billing", "fase10-billing-prelogin"),
]

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    for path, name in PAGES:
        pg.goto(f"http://127.0.0.1:5181{path}", wait_until="networkidle")
        pg.screenshot(path=str(OUT / f"{name}.png"), full_page=True)
        print(f"shot {name}")
    b.close()
print("done")
