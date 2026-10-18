#!/usr/bin/env bash
# Verifikasi deck lewat Chrome headless: overflow geometris + kontras WCAG.
set -u
cd "$(dirname "$0")"
T="$LOCALAPPDATA/Temp"
CHROME="/c/Program Files (x86)/Google/Chrome/Application/chrome.exe"

python build_deck.py || exit 1
python - <<'PY'
import pathlib
src = pathlib.Path('PaluKetok-PitchDeck.html').read_text(encoding='utf-8')
src = src.replace('</body>', '<script src="verify_deck.js"></script></body>')
pathlib.Path('_verify.html').write_text(src, encoding='utf-8')
PY

rm -rf "$T/pk-verify"
"$CHROME" --headless=new --disable-gpu --no-first-run --hide-scrollbars \
  --window-size=1920,1080 --virtual-time-budget=6000 \
  --user-data-dir="$T/pk-verify" \
  --dump-dom "file:///$(pwd -W 2>/dev/null || pwd)/_verify.html" 2>/dev/null > "$T/dom.html"

python - <<'PY'
import re, json, pathlib, os, html as H, sys
T = pathlib.Path(os.environ['LOCALAPPDATA']) / 'Temp'
s = (T/'dom.html').read_text(encoding='utf-8', errors='replace')
m = re.search(r'<pre id="out">(.*?)</pre>', s, re.S)
if not m:
    print('GAGAL: skrip verifikasi tidak jalan'); sys.exit(1)
d = json.loads(H.unescape(m.group(1)))
bad = 0
print('protokol %s | font inline %s | gradasi %s' % (d['file_protocol'], d['font_inline'], d['gradient']))
for x in d['slides']:
    flag = 'OK ' if not x['overflow'] and not x['low_contrast'] else 'BAD'
    if flag == 'BAD':
        bad += 1
    print('%s slide %2d  overflow=%d  kontras_min=%s  kontras_rendah=%d'
          % (flag, x['n'], len(x['overflow']), x['min_contrast'], len(x['low_contrast'])))
    for o in x['overflow'][:5]:
        print('    overflow:', o)
    for c in x['low_contrast'][:5]:
        print('    kontras:', c)
print('HASIL:', 'BERSIH' if bad == 0 else '%d slide bermasalah' % bad)
PY
