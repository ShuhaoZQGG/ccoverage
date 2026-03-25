#!/usr/bin/env python3
"""Generate DMG background image with arrow and instruction text."""

from PIL import Image, ImageDraw, ImageFont
import sys

WIDTH, HEIGHT = 600, 400
BG_COLOR = (237, 237, 237)
TEXT_COLOR = (68, 68, 68)
ARROW_COLOR = (140, 140, 140)

img = Image.new("RGB", (WIDTH, HEIGHT), BG_COLOR)
draw = ImageDraw.Draw(img)

# "Drag to Applications" text
try:
    font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", 22)
    small_font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", 13)
except OSError:
    font = ImageFont.load_default()
    small_font = font

text = "Drag the app to your Applications folder"
bbox = draw.textbbox((0, 0), text, font=font)
tw = bbox[2] - bbox[0]
draw.text(((WIDTH - tw) / 2, 50), text, fill=TEXT_COLOR, font=font)

# Arrow in the center (pointing right)
cx, cy = WIDTH // 2, HEIGHT // 2 + 10
shaft_len, shaft_h = 60, 10
head_len, head_h = 30, 24

# Shaft
draw.rectangle(
    [cx - shaft_len, cy - shaft_h // 2, cx + shaft_len - head_len, cy + shaft_h // 2],
    fill=ARROW_COLOR,
)
# Head
draw.polygon(
    [
        (cx + shaft_len - head_len, cy - head_h),
        (cx + shaft_len, cy),
        (cx + shaft_len - head_len, cy + head_h),
    ],
    fill=ARROW_COLOR,
)

out = sys.argv[1] if len(sys.argv) > 1 else "dmg-background.png"
img.save(out)
print(f"Created {out} ({WIDTH}x{HEIGHT})")
