"""Bandingkan tinggi blok /projects: mockup dasrhboard.html vs build SvelteKit.

Mockup memakai Tailwind Play CDN (butuh jaringan). Kalau CDN gagal, skrip
berhenti tanpa menyimpulkan apa pun.
"""
import pathlib
from playwright.sync_api import sync_playwright

WEB = pathlib.Path(__file__).resolve().parents[1]
ROOT = WEB.parent
MOCK = ROOT / "design" / "web" / "dasrhboard.html"
URL = "http://127.0.0.1:5181/projects"

JS = """() => {
  const out = [];
  const push = (sel, label) => {
    const el = document.querySelector(sel);
    if (el) out.push([label, Math.round(el.getBoundingClientRect().height)]);
  };
  push('[data-purpose="sidebar-navigation"]', 'sidebar');
  push('[data-purpose="top-navigation"]', 'topbar');
  push('[data-purpose="dashboard-header"]', 'hero');
  push('[data-purpose="stat-metrics"]', 'metrics');
  push('[data-purpose="all-projects-list"]', 'projects');
  push('[data-purpose="secondary-action-banners"]', 'banners');
  push('[data-purpose="notice-banner"]', 'notice');
  push('[data-purpose="page-states-preview"]', 'states');
  out.push(['TOTAL-dok', Math.round(document.documentElement.scrollHeight)]);
  return out;
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    # Mockup: fixed sidebar + konten; viewport dasar seperti landing.
    for tgt, url in (("mockup", MOCK.as_uri()), ("build", URL)):
        pg = b.new_page(viewport={"width": 1440, "height": 900})
        pg.goto(url, wait_until="networkidle")
        tw = pg.evaluate(
            """() => {
              const t = document.querySelector('script[src*="cdn.tailwindcss"]');
              return t ? 'aktif' : 'tidak-ada';
            }"""
        )
        rows = pg.evaluate(JS)
        print(f"--- {tgt} (tailwind: {tw}) ---")
        for label, h in rows:
            print(f"  {label:10s} {h}")
        pg.close()
    b.close()