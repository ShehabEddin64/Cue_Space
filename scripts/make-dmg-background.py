"""Draws the Cue Space disk image background (600x400 points, 1x and 2x).

    python3 scripts/make-dmg-background.py

Writes build/darwin/dmg-background.png and dmg-background@2x.png; the
release script combines them into a Retina-ready TIFF. The layout matches
the icon positions in scripts/release.sh: the app at (160, 190) and
Applications at (440, 190), icons 112 points wide.
"""
import os
from PIL import Image, ImageDraw, ImageFont

W, H = 600, 400
CANVAS = (246, 245, 243)  # Canvas #F6F5F3
INK = (24, 24, 26)  # Ink #18181A
GRAPHITE = (110, 110, 115)
RED = (217, 48, 37)  # Cue Red #D93025
FONT = "/System/Library/Fonts/SFNS.ttf"
ROOT = os.path.join(os.path.dirname(__file__), "..", "build", "darwin")


def font(size, weight):
    f = ImageFont.truetype(FONT, size)
    try:
        # SF's axes: width, optical size, grade, weight.
        f.set_variation_by_axes([100, max(17, min(96, size)), 400, weight])
    except Exception:
        pass
    return f


def draw(scale):
    s = lambda v: int(round(v * scale))
    img = Image.new("RGB", (s(W), s(H)), CANVAS)
    d = ImageDraw.Draw(img)

    # The ring-and-dot mark, small, top left.
    cx, cy, r = s(36), s(34), s(9)
    d.arc([cx - r, cy - r, cx + r, cy + r], start=40, end=320, fill=RED, width=max(2, s(3)))
    dr = s(2.3)
    dx = cx + s(11.4)
    d.ellipse([dx - dr, cy - dr, dx + dr, cy + dr], fill=INK)
    d.text((s(54), s(34)), "Cue Space", font=font(s(15), 650), fill=INK, anchor="lm")

    # A dashed arrow from the app to Applications.
    y = s(182)
    x0, x1 = s(240), s(352)
    dash, gap = s(9), s(7)
    x = x0
    while x < x1:
        d.line([(x, y), (min(x + dash, x1), y)], fill=RED, width=max(2, s(3)))
        x += dash + gap
    head = s(12)
    d.polygon([(x1 + head, y), (x1 - s(2), y - head * 0.75), (x1 - s(2), y + head * 0.75)], fill=RED)

    # The instruction, in English and French.
    d.text((s(W / 2), s(318)), "Drag Cue Space to Applications", font=font(s(15), 600), fill=INK, anchor="mm")
    d.text((s(W / 2), s(342)), "Glissez Cue Space dans Applications", font=font(s(13), 450), fill=GRAPHITE, anchor="mm")
    return img


draw(1).save(os.path.join(ROOT, "dmg-background.png"))
draw(2).save(os.path.join(ROOT, "dmg-background@2x.png"))
print("wrote build/darwin/dmg-background.png and dmg-background@2x.png")
