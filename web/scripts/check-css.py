"""Baca aturan .text-3xs langsung dari document.styleSheets (halaman dev)."""
from playwright.sync_api import sync_playwright

JS = """() => {
  const hits = [];
  for (const ss of document.styleSheets) {
    let rules;
    try { rules = ss.cssRules; } catch (e) { continue; }
    for (const r of rules) {
      if (r.selectorText && r.selectorText.includes('.text-3xs')) {
        hits.push(r.selectorText + ' { ' + r.style.cssText + ' }');
      }
      if (r.cssRules) {
        for (const r2 of r.cssRules) {
          if (r2.selectorText && r2.selectorText.includes('.text-3xs')) {
            hits.push('@' + r.conditionText + ' ' + r2.selectorText + ' { ' + r2.style.cssText + ' }');
          }
        }
      }
    }
  }
  return hits;
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page()
    pg.goto("http://127.0.0.1:5180/", wait_until="networkidle")
    pg.wait_for_timeout(1200)
    for h in pg.evaluate(JS):
        print(h)
    b.close()