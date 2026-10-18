#!/usr/bin/env python3
"""Bangun pitch deck PaluKetok jadi satu HTML offline (font di-inline base64)."""
import base64
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parent
FONT_DIR = ROOT / "_fonts"
PARTS = ROOT / "_parts"
OUT = ROOT / "PaluKetok-PitchDeck.html"

FONT_DEFS = [
    ("'Plus Jakarta Sans'", "normal", "pjs.b64.txt"),
    ("'JetBrains Mono'", "normal", "jbm.b64.txt"),
]


def font_css() -> str:
    out = []
    for family, style, fname in FONT_DEFS:
        b64 = (FONT_DIR / fname).read_text().strip()
        src = "url(data:font/woff2;base64," + b64 + ") format('woff2')"
        out.append(
            "@font-face{font-family:%s;font-style:%s;font-weight:400 800;"
            "font-display:block;src:%s}" % (family, style, src)
        )
    return "".join(out)


def main() -> None:
    shell = (PARTS / "shell.html").read_text(encoding="utf-8")
    slides = (PARTS / "slides.html").read_text(encoding="utf-8")
    appjs = (PARTS / "app.js").read_text(encoding="utf-8")

    html = shell.replace("/*__FONTS__*/", font_css())
    html = html.replace("<!--__SLIDES__-->", slides)
    html = html.replace("/*__APP__*/", appjs)

    # guard: tak boleh ada placeholder tersisa
    for token in ("__FONTS__", "__SLIDES__", "__APP__"):
        if token in html:
            raise SystemExit("placeholder tertinggal: " + token)
    # guard: tak boleh ada rujukan jaringan
    for bad in ("http://", "https://", "fetch(", "type=\"module\"", "<link"):
        hits = [m.start() for m in re.finditer(re.escape(bad), html)]
        if hits:
            raise SystemExit("rujukan luar ditemukan: %s (%d)" % (bad, len(hits)))
    # guard: tak boleh ada gradasi
    if "gradient" in html:
        raise SystemExit("gradasi ditemukan di deck")

    OUT.write_text(html, encoding="utf-8")
    n = html.count('class="slide')
    print("tulis:", OUT)
    print("ukuran: %.1f KB" % (len(html) / 1024), "| slide:", n)


if __name__ == "__main__":
    main()
