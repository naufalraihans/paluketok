"""Screenshot ketiga halaman Fase 9, full-page, dari dev server 5181."""
import pathlib
from playwright.sync_api import sync_playwright

OUT = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\docs\screenshots")
PAGES = [
    ("fase9-landing.png", "http://127.0.0.1:5181/"),
    ("fase9-projects.png", "http://127.0.0.1:5181/projects"),
    ("fase9-summary.png", "http://127.0.0.1:5181/projects/paluketok-demo"),
]

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    for name, url in PAGES:
        pg.goto(url)
        pg.wait_for_load_state("networkidle")
        pg.wait_for_timeout(400)
        h = pg.evaluate("() => document.documentElement.scrollHeight")
        pg.screenshot(path=str(OUT / name), full_page=True)
        print(f"{name}: {h}px")
    b.close()
