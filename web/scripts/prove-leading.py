"""Bukti mekanisme: apakah `leading-*` mati di mockup karena kalah dari `sm:text-*`?

Uji dua arah pada h1 yang sama:
  A. mockup apa adanya            -> line-height render?
  B. mockup + paksa .leading-[1.15] menang -> berubah?
  C. build saya apa adanya        -> line-height?
Lalu: berapa tinggi h1 di tiap kondisi. Kalau B == C, mekanisme terbukti.
"""
import pathlib
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()
BUILD = "http://127.0.0.1:5180/"
FORCE = "h1{line-height:1.15 !important}"

Q = """() => {
  const h1 = document.querySelector('section h1');
  const h2 = document.querySelectorAll('section')[1]?.querySelector('h2');
  const c1 = getComputedStyle(h1), c2 = h2 ? getComputedStyle(h2) : null;
  return {h1: [c1.fontSize, c1.lineHeight, Math.round(h1.getBoundingClientRect().height)],
          h2: c2 ? [c2.fontSize, c2.lineHeight, Math.round(h2.getBoundingClientRect().height)] : null,
          h2cls: h2 ? h2.className : null};
}"""


def run(pg, url, force=None, label=""):
    pg.goto(url, wait_until="networkidle", timeout=60000)
    pg.wait_for_timeout(2500)
    if force:
        pg.add_style_tag(content=force)
        pg.wait_for_timeout(400)
    d = pg.evaluate(Q)
    print(f"{label:<34} h1 fs/lh/h={d['h1']}   h2 fs/lh/h={d['h2']}")
    return d


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000})
    a = run(pg, MOCK, None, "A. mockup apa adanya")
    bb = run(pg, MOCK, FORCE, "B. mockup + leading dipaksa 1.15")
    c = run(pg, BUILD, None, "C. build saya")
    b.close()

print()
print(f"h1 tinggi: mockup={a['h1'][2]}  mockup+force={bb['h1'][2]}  build={c['h1'][2]}")
print(f"h2 tinggi: mockup={a['h2'][2]}  mockup+force={bb['h2'][2]}  build={c['h2'][2]}")
print(f"h2 class : {a['h2cls']}")
print()
print(f"B == C ? h1: {bb['h1'][2] == c['h1'][2]}   h2: {bb['h2'][2] == c['h2'][2]}")