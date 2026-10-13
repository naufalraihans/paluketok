"""Bandingkan tinggi section: mockup beku (render langsung) vs build SvelteKit.

Mockup memakai Tailwind Play CDN, jadi butuh jaringan. Kalau CDN gagal dimuat,
skrip melaporkan itu dan tidak menyimpulkan apa pun.
"""
import pathlib, json
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()
BUILD = "http://127.0.0.1:5180/"

MEASURE = """() => {
  const out = [];
  document.querySelectorAll('header, main > section, body > section, footer').forEach(el => {
    const r = el.getBoundingClientRect();
    out.push({tag: el.tagName.toLowerCase(), h: Math.round(r.height)});
  });
  return {rows: out, total: document.documentElement.scrollHeight,
          tw: typeof window.tailwind !== 'undefined'};
}"""


def measure(pg, url, label):
    pg.goto(url, wait_until="networkidle", timeout=60000)
    pg.wait_for_timeout(2500)
    d = pg.evaluate(MEASURE)
    print(f"--- {label} ---")
    print(f"tailwind aktif: {d['tw']}   total: {d['total']}px   section: {len(d['rows'])}")
    for i, r in enumerate(d["rows"], 1):
        print(f"  {i:>2} {r['tag']:<8} {r['h']:>6}px")
    return d


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000})
    m = measure(pg, MOCK, "MOCKUP (design/web/landingPage.html)")
    t = measure(pg, BUILD, "BUILD (web/ dev server)")
    b.close()

print("\n--- selisih tinggi section (build - mockup) ---")
for i, (a, c) in enumerate(zip(m["rows"], t["rows"]), 1):
    d = c["h"] - a["h"]
    pct = (d / a["h"] * 100) if a["h"] else 0
    print(f"  {i:>2} {a['tag']:<8} mockup {a['h']:>6}  build {c['h']:>6}  delta {d:>+5} ({pct:+.1f}%)")
print(f"  TOTAL    mockup {m['total']:>6}  build {t['total']:>6}  delta {t['total']-m['total']:>+5}")