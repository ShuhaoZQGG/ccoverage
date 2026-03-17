#!/usr/bin/env python3
"""Generate DMGIcon.icns by adding a download arrow to the existing AppIcon.

Run once to regenerate the committed asset:
    python3 scripts/generate-dmg-icon.py
"""

import os
import shutil
import subprocess
import sys
import tempfile

from PIL import Image, ImageDraw

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
APP_ICON = os.path.join(REPO_ROOT, "menubar", "Release", "AppIcon.icns")
OUT_ICNS = os.path.join(REPO_ROOT, "menubar", "Release", "DMGIcon.icns")

# Copper color matching the C shapes in AppIcon
ARROW_COLOR = (192, 112, 64, 255)

# Arrow geometry on a 1024×1024 canvas
# Center of the inner-C negative space (slightly right of center)
CX, CY = 530, 510
SHAFT_W = 56       # shaft width
SHAFT_H = 130      # shaft height (above arrowhead)
HEAD_W = 160       # arrowhead total width
HEAD_H = 80        # arrowhead height
BASE_W = 200       # horizontal base line width
BASE_H = 22        # base line thickness
GAP = 6            # gap between shaft and arrowhead top


def draw_arrow(img: Image.Image) -> Image.Image:
    """Draw a centered download arrow onto a copy of img."""
    out = img.copy().convert("RGBA")
    draw = ImageDraw.Draw(out)

    # Shaft (rectangle, above the arrowhead)
    shaft_x0 = CX - SHAFT_W // 2
    shaft_y0 = CY - SHAFT_H - HEAD_H // 2
    shaft_x1 = CX + SHAFT_W // 2
    shaft_y1 = CY - HEAD_H // 2 - GAP
    draw.rectangle([shaft_x0, shaft_y0, shaft_x1, shaft_y1], fill=ARROW_COLOR)

    # Arrowhead (downward-pointing triangle)
    tip_x = CX
    tip_y = CY + HEAD_H // 2
    head_top_y = CY - HEAD_H // 2
    triangle = [
        (CX - HEAD_W // 2, head_top_y),
        (CX + HEAD_W // 2, head_top_y),
        (tip_x, tip_y),
    ]
    draw.polygon(triangle, fill=ARROW_COLOR)

    # Base line (horizontal bar below the arrowhead)
    base_y0 = tip_y + GAP * 2
    base_y1 = base_y0 + BASE_H
    draw.rectangle(
        [CX - BASE_W // 2, base_y0, CX + BASE_W // 2, base_y1],
        fill=ARROW_COLOR,
    )

    return out


# Iconset sizes: (logical_size, scale) → filename
SIZES = [
    (16, 1),
    (16, 2),
    (32, 1),
    (32, 2),
    (128, 1),
    (128, 2),
    (256, 1),
    (256, 2),
    (512, 1),
    (512, 2),
]


def main() -> None:
    if not os.path.exists(APP_ICON):
        sys.exit(f"AppIcon.icns not found at {APP_ICON}")

    # Extract the 1024×1024 PNG from the icns using sips
    tmp_dir = tempfile.mkdtemp()
    try:
        base_png = os.path.join(tmp_dir, "AppIcon_1024.png")
        subprocess.run(
            ["sips", "-s", "format", "png", APP_ICON, "--out", base_png],
            check=True,
            capture_output=True,
        )

        base_img = Image.open(base_png).convert("RGBA")
        if base_img.size != (1024, 1024):
            # sips may produce a multi-image output; grab the largest
            base_img = base_img.resize((1024, 1024), Image.LANCZOS)

        arrow_img = draw_arrow(base_img)

        # Build iconset directory
        iconset_dir = os.path.join(tmp_dir, "DMGIcon.iconset")
        os.makedirs(iconset_dir)

        for logical, scale in SIZES:
            px = logical * scale
            resized = arrow_img.resize((px, px), Image.LANCZOS)
            if scale == 1:
                fname = f"icon_{logical}x{logical}.png"
            else:
                fname = f"icon_{logical}x{logical}@2x.png"
            resized.save(os.path.join(iconset_dir, fname))
            print(f"  wrote {fname} ({px}×{px})")

        # Convert iconset → icns
        subprocess.run(
            ["iconutil", "-c", "icns", iconset_dir, "-o", OUT_ICNS],
            check=True,
            capture_output=True,
        )
        print(f"\nDone: {OUT_ICNS}")

    finally:
        shutil.rmtree(tmp_dir, ignore_errors=True)


if __name__ == "__main__":
    main()
