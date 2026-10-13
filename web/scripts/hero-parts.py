"""Ukur tinggi sub-elemen hero: mockup vs build. Melokalisasi selisih +28px."""
import pathlib
from playwright.sync_api import sync_playwright

MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()
BUILD = "http://127.0.0.1:5180/"

Q = """() => {
  const hero = document.querySelector('section');
  const out = [];
  const push = (name, el) => {
    if (!el) return out.push([name, null, null]);
    const r = el.getBoundingClientRect();
    out.push([name, Math.round(r.height), Math.round(r.top - hero.getBoundingClientRect().top)]);
  };
  push('h1', hero.querySelector('h1'));
  push('p.sub', hero.querySelector('p'));
  const cta = hero.querySelector('.mb-16');
  push('div.cta(mb-16)', cta);
  push('a#1', cta && cta.querySelector('a'));
  push('a>span#1', cta && cta.querySelector('a > span'));
  push('window', hero.querySelector('[class*="shadow-app"]'));
  push('winHeader', hero.querySelector('[class*="rounded-full bg-red-400"]')?.parentElement);
  push('sidebar', hero.querySelector('nav'));
  const btns = hero.querySelectorAll('nav > a, nav > button');
  push('sidebarItem1', btns[0]);
  push('sidebarItem2', btns[1]);
  return {rows: out, heroH: Math.round(hero.getBoundingClientRect().height),
          navH: Math.round(hero.querySelector('nav')?.getBoundingClientRect().height ?? -1)};
}"""


def grab(pg, url, label):
    pg.goto(url, wait_until="networkidle", timeout=60000)
    pg.wait_for_timeout(2500)
    d = pg.evaluate(Q)
    return d


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000})
    m = grab(pg, MOCK, "mockup")
    t = grab(pg, BUILD, "build")
    b.close()

print(f"{'elemen':<18} {'mockup h':>9} {'build h':>9} {'delta':>7}   {'m.top':>7} {'b.top':>7}")
for (n, mh, mt), (_, bh, bt) in zip(m["rows"], t["rows"]):
    if mh is None or bh is None:
        print(f"{n:<18} {'-':>9} {'-':>9} {'(tidak ada)':>7}")
        continue
    print(f"{n:<18} {mh:>9} {bh:>9} {bh-mh:>+7}   {mt:>7} {bt:>7}")
print(f"\nhero total: mockup {m['heroH']}  build {t['heroH']}  delta {t['heroH']-m['heroH']:+}")