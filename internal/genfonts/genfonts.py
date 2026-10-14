#!/usr/bin/env python3
"""Subsets Noto Sans CJK SC to GB2312 level-1 and writes render/fonts/.
Run once for Chinese labels in charts and PDF export:

    python3 internal/genfonts/genfonts.py /path/to/NotoSansCJK-Regular.ttc

The output is gitignored; the build works without it (Latin only).
"""
import os
import string
import sys

from fontTools.subset import Options, Subsetter
from fontTools.ttLib import TTCollection

def main():
    src = sys.argv[1] if len(sys.argv) > 1 else \
        "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
    chars = set(string.printable)
    chars.update("，。！？；：、（）《》“”‘’—…·「」『』【】～￥")
    for hi in range(0xB0, 0xF8):
        for lo in range(0xA1, 0xFF):
            cp = (hi << 8) | lo
            if 0xE000 <= cp <= 0xF8FF:
                continue
            try:
                ch = chr(cp)
                ch.encode("utf-32-be")
                chars.add(ch)
            except UnicodeEncodeError:
                pass
    text = "".join(chars)
    coll = TTCollection(src)
    font = None
    for f in coll.fonts:
        name = f["name"].getDebugName(4) or ""
        if "SC" in name and "Mono" not in name:
            font = f
            break
    if font is None:
        font = coll.fonts[0]
    opts = Options()
    opts.name_IDs = ["*"]
    opts.notdef_outline = True
    opts.drop_tables += ["GSUB", "GPOS", "GDEF", "vhea", "vmtx", "VORG"]
    ss = Subsetter(opts)
    ss.populate(text=text)
    ss.subset(font)
    outdir = os.path.join(os.path.dirname(__file__), "..", "..", "render", "fonts")
    os.makedirs(outdir, exist_ok=True)
    out = os.path.join(outdir, "NotoSansSC-Regular.ttf")
    font.save(out)
    print("wrote", out, os.path.getsize(out) // 1024, "KB")


if __name__ == "__main__":
    main()
