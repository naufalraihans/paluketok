"""Screenshot full-page halaman "/" dari dev server lokal ke docs/screenshots/."""
import pathlib
from playwright.sync_api import sync_playwright

OUT = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\docs\screenshots")
OUT.mkdir(parents=True, exist_ok=True)
dest = OUT / "fase9-landing.png"

with sync_playwright() as p:
    b = p.chromium.launch()
    page = b.new_page(viewport={"width": 1440, "height": 1000}, device_scale_factor=1)
    page.goto("http://127.0.0.1:5180/", wait_until="networkidle")
    page.wait_for_timeout(1200)  # beri waktu font woff2 selesai dimuat
    page.screenshot(path=str(dest), full_page=True)
    h = page.evaluate("document.documentElement.scrollHeight")
    fonts = page.evaluate("Array.from(document.fonts).map(f => f.family + ' ' + f.weight + ' ' + f.status)")
    print(f"scrollHeight={h}px")
    print("fonts:", fonts)
    b.close()

print(f"{dest} {dest.stat().st_size} bytes")