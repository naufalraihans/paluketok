"""Bandingkan teks halaman /projects: mockup vs build. Deteksi teks hilang/berubah."""
import pathlib
import re
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\dasrhboard.html")
BUILD = "http://127.0.0.1:5181/projects"

JS = "() => document.querySelector('main').innerText.replace(/\\s+/g, ' ').trim()"


def norm(t):
    return [s.strip() for s in re.split(r"(?<=[a-z0-9)\u2026])(?=[A-Z])|  +| \u00b7 ", t) if s.strip()]


with sync_playwright() as p:
    b = p.chromium.launch()
    got = {}
    for tgt, url in (("mockup", MOCK.as_uri()), ("build", BUILD)):
        pg = b.new_page(viewport={"width": 1440, "height": 900})
        pg.goto(url, wait_until="networkidle")
        # mockup: main ada. build: main ada.
        got[tgt] = pg.evaluate(JS)
        pg.close()
    b.close()

m, bd = got["mockup"], got["build"]
print(f"chars mockup={len(m)} build={len(bd)}")
segs = norm(m)
missing = [s for s in segs if s not in bd]
print(f"segmen diuji={len(segs)}  tidak ditemukan di build={len(missing)}")
for s in missing[:10]:
    print("  MISSING:", s[:90])