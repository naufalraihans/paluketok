"""Bandingkan tinggi blok /projects/[id]: mockup vs build (via dev server).

Mockup pakai Tailwind Play CDN (butuh jaringan); kalau gagal, skrip berhenti.
"""
import pathlib
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\ringkasanProyek.html")
URL = "http://127.0.0.1:5181/projects/paluketok-demo"

SELECTORS = [
    ("sidebar", 'aside[data-purpose="sidebar"]'),
    ("topbar", 'header[data-purpose="top-navigation-bar"]'),
    ("page-title", 'section[data-purpose="page-title-header"]'),
    ("overview", 'section[data-purpose="project-metadata-card"]'),
    ("metrics", 'section[data-purpose="metrics-summary"]'),
    ("mid-split", 'section[data-purpose="traceability-and-protection-section"]'),
    ("bottom-split", 'section[data-purpose="actions-and-activity-logs"]'),
    ("micro-footer", 'footer[data-purpose="canvas-footer-metadata"]'),
]

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})

    pg.goto(MOCK.as_uri())
    pg.wait_for_load_state("networkidle")
    ok = pg.evaluate("() => !!document.querySelector('script[src*=\"cdn.tailwindcss\"]')?.dataset")
    h_m = {}
    for name, sel in SELECTORS:
        el = pg.query_selector(sel)
        h_m[name] = el.evaluate("e => e.getBoundingClientRect().height") if el else None

    pg.goto(URL)
    pg.wait_for_load_state("networkidle")
    h_b = {}
    for name, sel in SELECTORS:
        el = pg.query_selector(sel)
        h_b[name] = el.evaluate("e => e.getBoundingClientRect().height") if el else None

    doc_m = pg.evaluate("() => document.querySelector('main').scrollHeight")
    pg.goto(MOCK.as_uri())
    pg.wait_for_load_state("networkidle")
    main_m = pg.query_selector("main")
    doc_mm = main_m.evaluate("e => e.scrollHeight") if main_m else None

    b.close()

print(f"{'blok':14} {'mockup':>8} {'build':>8}   selisih")
for name, _ in SELECTORS:
    m, v = h_m[name], h_b[name]
    mark = "" if m == v else "  <-- BEDA"
    print(f"{name:14} {m!s:>8} {v!s:>8}   {'' if m == v else round((v or 0) - (m or 0), 1)}{mark}")
print(f"\nmain scrollHeight: mockup={doc_mm} build={doc_m}")
