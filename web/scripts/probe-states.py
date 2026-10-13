"""Cari sumber selisih blok states /projects: mockup vs build."""
import pathlib
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\dasrhboard.html")

JS = """() => {
  const root = document.querySelector('[data-purpose="page-states-preview"]');
  return [...root.querySelectorAll('[data-purpose^="state-"]')].map(c => {
    const inner = c.firstElementChild, foot = c.lastElementChild;
    return [c.getAttribute('data-purpose'), Math.round(c.getBoundingClientRect().height),
            Math.round(inner.getBoundingClientRect().height),
            Math.round(foot.getBoundingClientRect().height)];
  });
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    for tgt, url in (("mockup", MOCK.as_uri()), ("build", "http://127.0.0.1:5181/projects")):
        pg = b.new_page(viewport={"width": 1440, "height": 900})
        pg.goto(url, wait_until="networkidle")
        print(f"--- {tgt} ---")
        for r in pg.evaluate(JS):
            print(f"  {r[0]:16s} card={r[1]:4d} inner={r[2]:4d} foot={r[3]:4d}")
        pg.close()
    b.close()