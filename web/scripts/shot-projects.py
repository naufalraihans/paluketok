"""Screenshot /projects dari dev server ke docs/screenshots/."""
import pathlib
from playwright.sync_api import sync_playwright

OUT = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\docs\screenshots\fase9-projects.png")
OUT.parent.mkdir(parents=True, exist_ok=True)

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900}, device_scale_factor=1)
    pg.goto("http://127.0.0.1:5181/projects", wait_until="networkidle")
    pg.wait_for_timeout(800)
    fonts = pg.evaluate(
        """() => [...document.fonts].map(f => f.family + ' [' + f.weight + '] ' + f.status)"""
    )
    print("font:", fonts)
    print("tinggi dokumen:", pg.evaluate("() => document.documentElement.scrollHeight"))
    pg.screenshot(path=str(OUT), full_page=True)
    b.close()
print("screenshot:", OUT, OUT.stat().st_size, "byte")