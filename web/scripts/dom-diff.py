"""Cocokkan elemen berteks berurutan (indeks ke-N berteks) antara mockup dan build.

Elemen ke-N yang berteksnya sama = pasangan. Lapor beda computed style.
"""
import pathlib
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()
BUILD = "http://127.0.0.1:5180/"

Q = """() => {
  const out = [];
  document.querySelectorAll('p,h1,h2,h3,h4,span,a,button,li,div')
    .forEach(el => {
      const t = (el.textContent || '').replace(/\\s+/g, ' ').trim();
      if (!t || t.length < 3) return;
      if (el.querySelector('p,h1,h2,h3,h4,span,a,button')) return; // daun saja
      const c = getComputedStyle(el);
      const r = el.getBoundingClientRect();
      out.push({txt: t.slice(0, 60), fs: c.fontSize, lh: c.lineHeight,
                h: Math.round(r.height), w: Math.round(r.width),
                mb: c.marginBottom, cls: (el.className || '').toString().slice(0, 70)});
    });
  return out;
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000})
    pg.goto(MOCK, wait_until="networkidle", timeout=60000); pg.wait_for_timeout(2500)
    m = pg.evaluate(Q)
    pg.goto(BUILD, wait_until="networkidle", timeout=60000); pg.wait_for_timeout(1500)
    t = pg.evaluate(Q)
    b.close()

print(f"daun-berteks: mockup={len(m)} build={len(t)}\n")
n = min(len(m), len(t))
bad = []
for i in range(n):
    a, c = m[i], t[i]
    same_txt = a["txt"] == c["txt"]
    d = (a["fs"] != c["fs"], a["lh"] != c["lh"], abs(a["h"] - c["h"]) > 1)
    if any(d):
        bad.append((i, same_txt, a, c))

print(f"pasangan elemen ke-N yang beda style: {len(bad)} (dari {n})\n")
for i, same, a, c in bad[:30]:
    mark = "OK " if same else "?? "
    f = f"{a['fs']}->{c['fs']}" if a["fs"] != c["fs"] else "-"
    l = f"{a['lh']}->{c['lh']}" if a["lh"] != c["lh"] else "-"
    h = f"{a['h']}->{c['h']}" if abs(a["h"] - c["h"]) > 1 else "-"
    print(f"{mark}{i:>4} {f:<19} {l:<21} {h:<11} | {a['txt'][:40]}")
    if not same:
        print(f"     mockup cls: {a['cls']}")
        print(f"     build  cls: {c['cls']}")