"""Типографская обложка площадки — для событий без своей картинки.

Обложки событий из постов и с TimePad мы не перезаливаем (права у авторов),
поэтому импортированные события получают изображение площадки: официальный
логотип, если он есть, иначе такой плейсхолдер — название и адрес на фоне в
палитре сайта. 1600×800, текст в безопасной зоне, чтобы пережить кроп карточки.

    python3 render.py out.jpg "Пространство F5" "проспект Римского-Корсакова, 39 · Санкт-Петербург" --dark
    python3 render.py out.jpg "Нефиктивное|образование" "Галерея «Сети» · улица Рылеева, 17–19" --mark mark.png

«|» в названии — перенос строки. Шрифт — Golos Text (OFL), как на сайте:
https://github.com/google/fonts/raw/main/ofl/golostext/GolosText%5Bwght%5D.ttf

Готовый файл загружается через POST /api/v1/uploads (поле file), а полученный
id прописывается как cover_file_id площадки в channels.json / config.json.
"""
import argparse

from PIL import Image, ImageDraw, ImageFont

W, H = 1600, 800
LIGHT, DARK = "#F0EEE9", "#151515"


def font(path, size, weight):
    f = ImageFont.truetype(path, size)
    f.set_variation_by_name(weight)
    return f


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("out")
    ap.add_argument("title")
    ap.add_argument("subtitle")
    ap.add_argument("--dark", action="store_true", help="светлый текст на тёмном фоне")
    ap.add_argument("--mark", help="PNG-знак площадки слева от названия")
    ap.add_argument("--font", default="GolosText[wght].ttf")
    a = ap.parse_args()

    bg, fg, muted = (DARK, LIGHT, "#9A968F") if a.dark else (LIGHT, DARK, "#6B6760")
    im = Image.new("RGB", (W, H), bg)
    d = ImageDraw.Draw(im)
    title_font, sub_font = font(a.font, 112, "Bold"), font(a.font, 40, "Regular")

    x = 180
    if a.mark:
        mark = Image.open(a.mark).convert("RGBA")
        im.paste(mark, (x, (H - mark.height) // 2), mark)
        x += mark.width + 90

    lines = a.title.split("|")
    y = (H - (124 * len(lines) + 80)) // 2
    for line in lines:
        d.text((x, y), line, font=title_font, fill=fg)
        y += 124
    y += 30
    d.line([(x, y), (x + 120, y)], fill=fg, width=4)
    d.text((x, y + 26), a.subtitle, font=sub_font, fill=muted)
    im.save(a.out, quality=82)


if __name__ == "__main__":
    main()
