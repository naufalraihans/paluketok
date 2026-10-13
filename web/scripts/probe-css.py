"""Cek cepat: apakah CSS terpasang di /projects?"""
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 900})
    pg.goto("http://127.0.0.1:5181/projects", wait_until="load")
    pg.wait_for_timeout(1500)
    print(pg.evaluate("""() => ({
      sheets: document.styleSheets.length,
      rules: [...document.styleSheets].reduce((n, s) => n + (s.cssRules ? s.cssRules.length : 0), 0),
      topbarH: getComputedStyle(document.querySelector('[data-purpose="top-navigation"]')).height,
      bodyBg: getComputedStyle(document.querySelector('div')).backgroundColor,
      sidebarDisplay: getComputedStyle(document.querySelector('[data-purpose="sidebar-navigation"]')).display
    })"""))
    b.close()