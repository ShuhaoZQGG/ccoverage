#!/usr/bin/env python3
"""Generate CCoverage app icon as .icns file.

Uses the same dual concentric C-arc motif from the menubar icon,
rendered on a modern macOS rounded-rectangle background.
"""

import math
import os
import subprocess
import sys
from pathlib import Path

from PIL import Image, ImageDraw

# Icon color scheme
BG_COLOR = (30, 30, 30)          # Dark background
ARC_COLOR = (195, 115, 77)       # Claude Code orange


def draw_icon(size: int) -> Image.Image:
    """Draw the CCoverage icon at the given pixel size."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # --- Rounded-rectangle background (macOS squircle approximation) ---
    margin = size * 0.04
    corner = size * 0.22
    x0, y0 = margin, margin
    x1, y1 = size - margin, size - margin
    draw.rounded_rectangle([x0, y0, x1, y1], radius=corner, fill=BG_COLOR)

    # --- Dual C arcs ---
    cx = size * 0.52          # Center slightly right so opening faces right
    cy = size * 0.50

    # Arc angles: opening on the right side (50° to 310°)
    arc_start = 50
    arc_end = 310

    # Outer C
    outer_r = size * 0.32
    outer_w = max(size * 0.09, 2)
    _draw_arc(draw, cx, cy, outer_r, outer_w, arc_start, arc_end, ARC_COLOR)

    # Inner C
    inner_r = size * 0.16
    inner_w = max(size * 0.07, 1.5)
    _draw_arc(draw, cx, cy, inner_r, inner_w, arc_start, arc_end, ARC_COLOR)

    return img


def _draw_arc(draw: ImageDraw.ImageDraw, cx, cy, radius, width, start_deg, end_deg, color):
    """Draw a thick arc with round caps by filling between two ellipses."""
    half_w = width / 2

    # Pillow arc uses bounding box and clockwise angles from 3 o'clock
    # Our angles are math-convention (CCW from east), Pillow uses CW from east
    # In image coords (y flipped), math CCW = Pillow CW, so:
    # Pillow start = 360 - end_deg, Pillow end = 360 - start_deg
    pil_start = 360 - end_deg
    pil_end = 360 - start_deg

    # Draw the arc as a filled region using pieslice difference approach:
    # Create a mask with the thick arc shape
    from PIL import Image as _Img

    size = draw.im.size
    mask = _Img.new("L", size, 0)
    mask_draw = ImageDraw.Draw(mask)

    # Outer edge of the arc stroke
    outer_r = radius + half_w
    outer_bbox = [cx - outer_r, cy - outer_r, cx + outer_r, cy + outer_r]

    # Inner edge of the arc stroke
    inner_r = radius - half_w
    inner_bbox = [cx - inner_r, cy - inner_r, cx + inner_r, cy + inner_r]

    # Draw filled outer pieslice, then erase inner pieslice and center wedge
    mask_draw.pieslice(outer_bbox, pil_start, pil_end, fill=255)
    if inner_r > 0:
        mask_draw.pieslice(inner_bbox, pil_start, pil_end, fill=0)
    # Erase the wedge lines from pieslice by drawing only the annular region
    # Redraw: use chord approach - outer chord minus inner ellipse
    mask2 = _Img.new("L", size, 0)
    mask2_draw = ImageDraw.Draw(mask2)
    # Full outer disc
    mask2_draw.ellipse(outer_bbox, fill=255)
    # Remove inner disc
    if inner_r > 0:
        mask2_draw.ellipse(inner_bbox, fill=0)
    # AND with the pieslice mask to get just the arc band
    from PIL import ImageChops
    mask = ImageChops.multiply(mask, mask2)

    # But pieslice includes wedge lines we don't want - use arc sector approach instead
    # Simpler: draw the arc region using the sector mask
    # Actually let's use a cleaner approach: draw filled arc via polygon
    mask3 = _Img.new("L", size, 0)
    mask3_draw = ImageDraw.Draw(mask3)

    n_points = 200
    total_angle = (pil_end - pil_start) % 360
    if total_angle == 0:
        total_angle = 360

    # Outer arc points
    outer_points = []
    for i in range(n_points + 1):
        angle = math.radians(pil_start + total_angle * i / n_points)
        x = cx + outer_r * math.cos(angle)
        y = cy + outer_r * math.sin(angle)
        outer_points.append((x, y))

    # Inner arc points (reversed)
    inner_points = []
    for i in range(n_points + 1):
        angle = math.radians(pil_start + total_angle * i / n_points)
        x = cx + inner_r * math.cos(angle)
        y = cy + inner_r * math.sin(angle)
        inner_points.append((x, y))

    polygon = outer_points + list(reversed(inner_points))
    mask3_draw.polygon(polygon, fill=255)

    # Round caps at endpoints
    for angle_deg in (pil_start, pil_end):
        angle_rad = math.radians(angle_deg)
        cap_cx = cx + radius * math.cos(angle_rad)
        cap_cy = cy + radius * math.sin(angle_rad)
        mask3_draw.ellipse(
            [cap_cx - half_w, cap_cy - half_w, cap_cx + half_w, cap_cy + half_w],
            fill=255,
        )

    # Apply color through mask
    overlay = _Img.new("RGBA", size, color + (255,))
    base = draw._image
    base.paste(overlay, mask=mask3)


def main():
    script_dir = Path(__file__).resolve().parent.parent
    iconset_dir = script_dir / "AppIcon.iconset"
    iconset_dir.mkdir(exist_ok=True)

    # macOS .iconset required sizes
    sizes = [16, 32, 64, 128, 256, 512, 1024]
    filenames = {
        16:   ["icon_16x16.png"],
        32:   ["icon_16x16@2x.png", "icon_32x32.png"],
        64:   ["icon_32x32@2x.png"],
        128:  ["icon_128x128.png"],
        256:  ["icon_128x128@2x.png", "icon_256x256.png"],
        512:  ["icon_256x256@2x.png", "icon_512x512.png"],
        1024: ["icon_512x512@2x.png"],
    }

    for sz in sizes:
        icon = draw_icon(sz)
        for fname in filenames[sz]:
            icon.save(iconset_dir / fname)
            print(f"  {fname} ({sz}x{sz})")

    # Convert to .icns using macOS iconutil
    icns_path = script_dir / "AppIcon.icns"
    subprocess.run(
        ["iconutil", "-c", "icns", str(iconset_dir), "-o", str(icns_path)],
        check=True,
    )
    print(f"\nCreated {icns_path}")

    # Clean up iconset directory
    import shutil
    shutil.rmtree(iconset_dir)


if __name__ == "__main__":
    main()
