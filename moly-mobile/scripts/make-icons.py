#!/usr/bin/env python3
"""Draws the Android launcher icons and the start-up splash from the Moly flower (../moly-extension/images/Logo_base.png).

Needs Pillow. Writes into android/app/src/main/res. Run it again only when the logo or the colours change.
"""
import os
from PIL import Image, ImageChops, ImageDraw, ImageFilter

HERE = os.path.dirname(os.path.abspath(__file__))
RES = os.path.join(HERE, '..', 'android', 'app', 'src', 'main', 'res')
SRC = os.path.join(HERE, '..', '..', 'moly-extension', 'images', 'Logo_base.png')
INK = (0x20, 0x24, 0x26)    # the app's --ink
PAPER = (0xF5, 0xF4, 0xF0)  # the app's --bg
LIGHT = (0xF7, 0xF6, 0xF2)  # the app's --on-ink


def flower_mask():
    """The flower's lines as a square 8-bit mask (255 = line), a little thickened so it survives small sizes."""
    gray = Image.open(SRC).convert('L')
    lines = ImageChops.invert(gray).point(lambda v: min(255, int(v * 1.6)))
    box = lines.point(lambda v: 255 if v > 40 else 0).getbbox()
    lines = lines.crop(box)
    side = max(lines.size)
    square = Image.new('L', (side, side), 0)
    square.paste(lines, ((side - lines.width) // 2, (side - lines.height) // 2))
    return square.filter(ImageFilter.MaxFilter(3))


def tinted(mask, size, color):
    m = mask.resize((size, size), Image.LANCZOS)
    layer = Image.new('RGBA', (size, size), color + (0,))
    layer.putalpha(m)
    return layer


def save(img, *path):
    out = os.path.join(RES, *path)
    os.makedirs(os.path.dirname(out), exist_ok=True)
    img.save(out)


mask = flower_mask()
DENSITIES = {'mdpi': 1, 'hdpi': 1.5, 'xhdpi': 2, 'xxhdpi': 3, 'xxxhdpi': 4}

for name, k in DENSITIES.items():
    # adaptive icon: 108 dp layer, the flower inside the 66 dp safe circle
    fg = round(108 * k)
    layer = Image.new('RGBA', (fg, fg), (0, 0, 0, 0))
    f = tinted(mask, round(fg * 0.52), LIGHT)
    layer.paste(f, ((fg - f.width) // 2, (fg - f.height) // 2), f)
    save(layer, f'mipmap-{name}', 'ic_launcher_foreground.png')
    # older phones: a 48 dp icon, square with rounded corners, and a round one
    px = round(48 * k)
    for kind in ('ic_launcher', 'ic_launcher_round'):
        big = px * 4
        base = Image.new('RGBA', (big, big), (0, 0, 0, 0))
        d = ImageDraw.Draw(base)
        if kind == 'ic_launcher_round':
            d.ellipse((0, 0, big - 1, big - 1), fill=INK + (255,))
        else:
            d.rounded_rectangle((0, 0, big - 1, big - 1), radius=round(big * 0.18), fill=INK + (255,))
        f = tinted(mask, round(big * 0.62), LIGHT)
        base.paste(f, ((big - f.width) // 2, (big - f.height) // 2), f)
        save(base.resize((px, px), Image.LANCZOS), f'mipmap-{name}', kind + '.png')

# the adaptive icon's background colour
with open(os.path.join(RES, 'values', 'ic_launcher_background.xml'), 'w') as fh:
    fh.write('<?xml version="1.0" encoding="utf-8"?>\n<resources>\n    <color name="ic_launcher_background">#%02X%02X%02X</color>\n</resources>\n' % INK)

# the splash shown while the app starts: the flower on the app's paper colour
SPLASH = {
    'drawable': (480, 320), 'drawable-port-mdpi': (320, 480), 'drawable-port-hdpi': (480, 800), 'drawable-port-xhdpi': (720, 1280),
    'drawable-port-xxhdpi': (960, 1600), 'drawable-port-xxxhdpi': (1280, 1920), 'drawable-land-mdpi': (480, 320),
    'drawable-land-hdpi': (800, 480), 'drawable-land-xhdpi': (1280, 720), 'drawable-land-xxhdpi': (1600, 960), 'drawable-land-xxxhdpi': (1920, 1280),
}
for folder, (w, h) in SPLASH.items():
    img = Image.new('RGB', (w, h), PAPER)
    f = tinted(mask, round(min(w, h) * 0.26), INK)
    img.paste(f, ((w - f.width) // 2, (h - f.height) // 2), f)
    save(img, folder, 'splash.png')
print('icons and splash written')
