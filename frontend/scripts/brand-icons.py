"""Generate the сб monogram icons (handoff README → Assets: Archivo 900,
tracking -.08em, white on #111 — Golos Text 900 substitutes for Archivo).

Run once from frontend/:  python3 scripts/brand-icons.py
Needs: pip install fonttools pillow  (use a throwaway venv)."""
import io, urllib.request
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from PIL import Image, ImageDraw, ImageFont

SRC = "https://github.com/google/fonts/raw/main/ofl/golostext/GolosText%5Bwght%5D.ttf"
TEXT, INK, PAPER, TRACK = "сб", "#111111", "#FFFFFF", -0.08

raw = urllib.request.urlopen(SRC).read()
font = instantiateVariableFont(TTFont(io.BytesIO(raw)), {"wght": 900})
buf = io.BytesIO(); font.save(buf); ttf = buf.getvalue()
upm = font["head"].unitsPerEm
cmap, gs, hmtx = font.getBestCmap(), font.getGlyphSet(), font["hmtx"]

# --- SVG: glyph outlines as paths, no font dependency at runtime ---
size = 64; em = 40  # glyph em in px inside a 64px square
scale = em / upm
names = [cmap[ord(c)] for c in TEXT]
advance = sum(hmtx[n][0] for n in names) + TRACK * upm * (len(names) - 1)
x0 = (size - advance * scale) / 2
cap = font["OS/2"].sxHeight or upm * 0.5
baseline = size / 2 + cap * scale / 2
paths, x = [], 0.0
for n in names:
    pen = SVGPathPen(gs)
    gs[n].draw(TransformPen(pen, (scale, 0, 0, -scale, x0 + x * scale, baseline)))
    paths.append(pen.getCommands())
    x += hmtx[n][0] + TRACK * upm
svg = (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {size} {size}">'
       f'<rect width="{size}" height="{size}" fill="{INK}"/>'
       f'<path fill="{PAPER}" d="{" ".join(paths)}"/></svg>\n')
open("app/icon.svg", "w").write(svg)

# --- PNG/ICO via Pillow from the same static instance ---
def raster(px: int) -> Image.Image:
    # RGBA, not RGB: Next.js's metadata image decoder (image-rs) rejects
    # ICO frames whose embedded PNGs aren't RGBA ("The PNG is not in RGBA
    # format!"), which 500s every page since favicon.ico feeds the <head>.
    img = Image.new("RGBA", (px, px), INK)
    d = ImageDraw.Draw(img)
    f = ImageFont.truetype(io.BytesIO(ttf), int(px * em / size))
    widths = [d.textlength(c, font=f) for c in TEXT]
    track = TRACK * f.size
    total = sum(widths) + track * (len(TEXT) - 1)
    x = (px - total) / 2
    for c, w in zip(TEXT, widths):
        d.text((x, px / 2), c, font=f, fill=PAPER, anchor="lm")
        x += w + track
    return img

raster(180).save("app/apple-icon.png")
raster(48).save("app/favicon.ico", sizes=[(16, 16), (32, 32), (48, 48)])
print("wrote app/icon.svg app/apple-icon.png app/favicon.ico")
