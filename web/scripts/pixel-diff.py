"""Pixel-diff full-page: mockup beku vs build. Ukur seberapa berbeda secara visual."""
import pathlib
from playwright.sync_api import sync_playwright
from PIL import Image, ImageChops

TMP = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\web\.diff")
TMP.mkdir(exist_ok=True)
MOCK = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\design\web\landingPage.html").as_uri()


def shot(pg, url, dest):
    pg.goto(url, wait_until="networkidle", timeout=60000)
    pg.wait_for_timeout(2500)
    pg.screenshot(path=str(dest), full_page=True)
    return pg.evaluate("document.documentElement.scrollHeight")


with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page(viewport={"width": 1440, "height": 1000}, device_scale_factor=1)
    hm = shot(pg, MOCK, TMP / "mockup.png")
    hb = shot(pg, "http://127.0.0.1:5180/", TMP / "build.png")
    b.close()

a, c = Image.open(TMP / "mockup.png").convert("RGB"), Image.open(TMP / "build.png").convert("RGB")
print(f"mockup {hm}px {a.size}   build {hb}px {c.size}")

h = min(a.height, c.height)
a, c = a.crop((0, 0, 1440, h)), c.crop((0, 0, 1440, h))
diff = ImageChops.difference(a, c).convert("L")

# Baris yang identik persis vs baris yang berbeda.
rows_equal = sum(1 for y in range(0, h, 4) if not diff.crop((0, y, 1440, y + 1)).getbbox())
rows_total = len(range(0, h, 4))
print(f"baris sampel identik persis: {rows_equal}/{rows_total} = {rows_equal/rows_total*100:.1f}%")

hist = diff.histogram()
tot = sum(hist)
big = sum(hist[24:])  # piksel dengan selisih > 23/255
print(f"piksel selisih >23/255: {big}/{tot} = {big/tot*100:.2f}%")
print(f"selisih rata-rata seluruh piksel: {sum(i*v for i,v in enumerate(hist))/tot:.2f}/255")

diff.point(lambda v: 255 if v > 23 else 0).save(TMP / "diffmap.png")
print(f"peta selisih: {TMP / 'diffmap.png'}")