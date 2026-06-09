"""Generate build/appicon.png for HVRIns — Instagram gradient app icon.

Based on NVRINS_BUILD_GUIDE.md section 2.1, adapted for the app name "HVRIns".

Requirements: Pillow (pip install Pillow)

Output: e:/WEMAKE/HVRIns/build/appicon.png (1024x1024 PNG, rounded corners)
"""
import os
import sys
from PIL import Image, ImageDraw, ImageFont

SIZE = 1024
RADIUS = 160

# Resolve target paths relative to this script (scripts/ -> project root)
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BUILD_DIR = os.path.join(PROJECT_ROOT, "build")
OUT_PNG = os.path.join(BUILD_DIR, "appicon.png")

STOPS = [
    (0.0,  (64,  93,  230)),   # #405DE6
    (0.25, (131, 58,  180)),   # #833AB4
    (0.5,  (225, 48,  108)),   # #E1306C
    (0.75, (253, 29,  29)),    # #FD1D1D
    (1.0,  (252, 175, 69)),    # #FCAF45
]


def interp(t, stops):
    for i in range(len(stops) - 1):
        t0, c0 = stops[i]
        t1, c1 = stops[i + 1]
        if t0 <= t <= t1:
            f = (t - t0) / (t1 - t0)
            return tuple(int(c0[j] + f * (c1[j] - c0[j])) for j in range(3))
    return stops[-1][1]


def main():
    os.makedirs(BUILD_DIR, exist_ok=True)

    tmp_png = OUT_PNG + ".tmp"
    try:
        img = Image.new('RGBA', (SIZE, SIZE))
        px = img.load()
        for y in range(SIZE):
            for x in range(SIZE):
                r, g, b = interp((x + y) / (2 * (SIZE - 1)), STOPS)
                px[x, y] = (r, g, b, 255)

        mask = Image.new('L', (SIZE, SIZE), 0)
        ImageDraw.Draw(mask).rounded_rectangle([0, 0, SIZE - 1, SIZE - 1], radius=RADIUS, fill=255)
        img.putalpha(mask)

        draw = ImageDraw.Draw(img)
        fb = ImageFont.truetype('C:/Windows/Fonts/arialbd.ttf', 280)
        fs = ImageFont.truetype('C:/Windows/Fonts/arial.ttf', 66)

        main_text = "HVRIns"
        b = draw.textbbox((0, 0), main_text, font=fb)
        tx = (SIZE - (b[2] - b[0])) // 2 - b[0]
        draw.text((tx, 310), main_text, font=fb, fill=(255, 255, 255, 255))

        sub = "Ha Vu VIP PRO"
        b = draw.textbbox((0, 0), sub, font=fs)
        sx = (SIZE - (b[2] - b[0])) // 2 - b[0]
        draw.text((sx, 730), sub, font=fs, fill=(255, 255, 255, 185))

        # Write to a temp file first, then atomically replace the final output
        # so a pre-existing appicon.png is never left corrupt/partial on failure.
        img.save(tmp_png, 'PNG')
        os.replace(tmp_png, OUT_PNG)
    except Exception as exc:
        # Clean up any partial temp file so no corrupt artifact remains.
        try:
            if os.path.exists(tmp_png):
                os.remove(tmp_png)
        except OSError:
            pass
        print("ERROR: failed to generate build/appicon.png: %s" % exc, file=sys.stderr)
        sys.exit(1)

    print("Saved:", OUT_PNG, img.size)


if __name__ == "__main__":
    main()
