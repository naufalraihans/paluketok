"""Cek mana `d` mockup ringkasanProyek yang belum ada di Icon.svelte (single path saja)."""
import re
import pathlib

icon = pathlib.Path("src/lib/components/Icon.svelte").read_text(encoding="utf-8")
mine = set(re.findall(r"'([A-Z][^']+=+[^']*)'", icon))

mock = pathlib.Path(
    r"E:\rekapProject\projecct_build_with_AI\design\web\ringkasanProyek.html"
).read_text(encoding="utf-8")

seen, baru = set(), []
for m in re.finditer(r'd="([A-Z][^"]+)"', mock):
    d = m.group(1)
    if d in seen:
        continue
    seen.add(d)
    if d not in mine:
        baru.append(d)

print(f"mockup d unik: {len(seen)}, belum ada di Icon: {len(baru)}")
for d in baru:
    print("BARU:", d[:55])
