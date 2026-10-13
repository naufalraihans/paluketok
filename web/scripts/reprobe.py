"""Ukur ulang span alur / label mikro setelah perbaikan token."""
from playwright.sync_api import sync_playwright

JS = """() => {
  const out = [];
  for (const t of ['Ruang Ramu /', 'uuid', 'sensitif', 'E-01']) {
    const el = [...document.querySelectorAll('span, div')].find(e => e.textContent.trim() === t || e.textContent.trim().startsWith(t));
    if (!el) { out.push([t, 'NONE']); continue; }
    const cs = getComputedStyle(el);
    out.push([t, cs.fontSize, cs.lineHeight, Math.round(el.getBoundingClientRect().height)]);
  }
  return out;
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1280, "height": 900})
    pg.goto("http://127.0.0.1:5180/", wait_until="networkidle")
    pg.wait_for_timeout(1200)
    for row in pg.evaluate(JS):
        print(row)
    b.close()