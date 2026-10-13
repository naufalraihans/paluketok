"""Analisis peta selisih: di mana beda terkumpul, dan apakah beda itu murni pergeseran vertikal.

Kalau profil pergeseran naik mulus (bukan lompatan), selisih berasal dari
perbedaan tinggi kecil yang menumpuk -- bukan blok yang hilang/bergeser.
"""
import pathlib
from PIL import Image, ImageChops

TMP = pathlib.Path(r"E:\rekapProject\projecct_build_with_AI\web\.diff")
a = Image.open(TMP / "mockup.png").convert("RGB")
c = Image.open(TMP / "build.png").convert("RGB")
h = min(a.height, c.height)
diff = ImageChops.difference(a.crop((0, 0, 1440, h)), c.crop((0, 0, 1440, h))).convert("L")

print("--- selisih per band 500px ---")
print(f"{'band':>14} {'beda>23/255':>12} {'rata2':>8}")
for y0 in range(0, h, 500):
    y1 = min(y0 + 500, h)
    band = diff.crop((0, y0, 1440, y1))
    hist = band.histogram()
    tot = sum(hist)
    big = sum(hist[24:]) / tot * 100
    avg = sum(i * v for i, v in enumerate(hist)) / tot
    bar = "#" * int(big / 2)
    print(f"{y0:>6}-{y1:<7} {big:>10.1f}% {avg:>8.2f}  {bar}")

print("\n--- profil pergeseran vertikal (baris build -> baris mockup paling mirip) ---")
print(f"{'baris build':>12} {'shift':>7} {'beda saat sejajar':>18}")
for y in range(0, h - 80, 400):
    row = c.crop((0, y, 1440, y + 40)).convert("L")
    best, besty = None, None
    for dy in range(-80, 81, 2):
        yy = y + dy
        if yy < 0 or yy + 40 > a.height:
            continue
        d = ImageChops.difference(row, a.crop((0, yy, 1440, yy + 40)).convert("L"))
        m = sum(i * v for i, v in enumerate(d.histogram())) / (1440 * 40)
        if best is None or m < best:
            best, besty = m, dy
    print(f"{y:>12} {besty:>+7} {best:>18.2f}")