"""Derive build/windows/icon.ico from build/appicon.png for HVRIns.

Based on NVRINS_BUILD_GUIDE.md section 2.1 (.ico portion).

Requirements: Pillow (pip install Pillow)

Input:  e:/WEMAKE/HVRIns/build/appicon.png (1024x1024 PNG)
Output: e:/WEMAKE/HVRIns/build/windows/icon.ico (sizes 16,32,48,64,128,256)
"""
import os
import sys
from PIL import Image

# Resolve target paths relative to this script (scripts/ -> project root)
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC_PNG = os.path.join(PROJECT_ROOT, "build", "appicon.png")
WINDOWS_DIR = os.path.join(PROJECT_ROOT, "build", "windows")
OUT_ICO = os.path.join(WINDOWS_DIR, "icon.ico")

SIZES = [16, 32, 48, 64, 128, 256]


def main():
    os.makedirs(WINDOWS_DIR, exist_ok=True)

    tmp_ico = OUT_ICO + ".tmp"
    try:
        ico = Image.open(SRC_PNG).convert('RGBA')
        # Render frames largest-first. Pillow's ICO writer never upscales, so the
        # base frame must be the largest size for every requested size to be kept.
        sizes_desc = sorted(SIZES, reverse=True)
        frames = [ico.resize((s, s), Image.LANCZOS) for s in sizes_desc]
        # Write to a temp file first, then atomically replace the final output
        # so a pre-existing icon.ico is never left corrupt/partial on failure.
        frames[0].save(
            tmp_ico,
            format='ICO',
            sizes=[(s, s) for s in sizes_desc],
            append_images=frames[1:],
        )
        os.replace(tmp_ico, OUT_ICO)
    except Exception as exc:
        # Clean up any partial temp file so no corrupt artifact remains.
        try:
            if os.path.exists(tmp_ico):
                os.remove(tmp_ico)
        except OSError:
            pass
        print("ERROR: failed to generate build/windows/icon.ico: %s" % exc, file=sys.stderr)
        sys.exit(1)

    print("Saved:", OUT_ICO, sorted(SIZES))


if __name__ == "__main__":
    main()
