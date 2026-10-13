"""Bandingkan sub-elemen satu section, mockup vs build. Pakai: python probe.py <idx_class>"""
import pathlib, sys
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()
BUILD = "http://127.0.0.1:5180/"

JS = """(i) => {
  const main = document.querySelector('main') || document.body;
  const secs = [...main.querySelectorAll('section, header')];
  const s = secs[i];
  if (!s) return ['NONE ' + secs.length];
  const out = [];
  for (const el of s.querySelectorAll('*')) {
    const t = (el.textContent || '').trim().slice(0, 20);
    if (!t) continue;
    const r = el.getBoundingClientRect();
    const cs = getComputedStyle(el);
    out.push([el.tagName, t, Math.round(r.height), cs.fontSize + '/' + cs.lineHeight]);
  }
  return out.slice(0, 40);
}"""


def probe(uri, i):
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": 1280, "height": 900})
        pg.goto(uri, wait_until="networkidle" if uri.startswith("http") else "load")
        pg.wait_for_timeout(1500)
        out = pg.evaluate(JS, i)
        b.close()
        return out


i = int(sys.argv[1])
m, d = probe(MOCK, i), probe(BUILD, i)
print(f"section #{i}: mockup {len(m)} elemen, build {len(d)} elemen")
for k in range(max(len(m), len(d))):
    a = m[k] if k < len(m) else None
    c = d[k] if k < len(d) else None
    same = a and c and a[0] == c[0] and a[2:] == c[2:]
    print(f"{'  ' if same else '>>'}M {a}")
    if not same:
        print(f"  B {c}")