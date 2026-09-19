#!/usr/bin/env python3
"""Сборщик анонсов с TimePad (страницы организатора, без токена).

Первый источник — Европейский университет в СПб (eusp.timepad.ru).

Почему по HTML, а не через API: api.timepad.ru требует токен из личного
кабинета организатора, а у нас его нет. Страница организатора отдаёт все
события server-side и без ключа.

Запуск с машины, где открывается TimePad:

    python3 ingest.py                 # собрать кандидатов в out/
    python3 ingest.py --days 90       # горизонт шире
    python3 ingest.py --all           # включая уже импортированные

Скрипт ничего не публикует: готовит черновики (status: draft) на проверку
редактору. Описание он НЕ пишет — авторский текст анонса идёт только в отчёт,
редактор пересказывает своими словами (та же политика, что в tools/tg-ingest).
"""
import argparse, json, html, os, re, sys, time, urllib.request
from datetime import datetime, timedelta, timezone

HERE = os.path.dirname(os.path.abspath(__file__))
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130 Safari/537.36"
MSK = timezone(timedelta(hours=3))
MONTHS = {m: i + 1 for i, m in enumerate(
    "январ феврал март апрел ма июн июл август сентябр октябр ноябр декабр".split())}
MONTH_RE = "|".join(MONTHS)


def fetch(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept-Language": "ru,en"})
    with urllib.request.urlopen(req, timeout=25) as r:
        return r.read().decode("utf-8", "ignore")


def text_of(chunk: str) -> str:
    return " ".join(html.unescape(re.sub(r"<[^>]+>", " ", chunk)).split())


def cards(list_html: str):
    """Карточки со страницы организатора.

    Режем HTML по границам карточек (`t-card_event`), а не окном вокруг ссылки:
    окно захватывало соседнюю карточку и сдвигало даты на одно событие —
    поймано на «Язык, или Главная загадка человечества» (30.09 вместо 02.10).

    Из карточки берём относительную пометку TimePad («через 13 дней», «69 дней
    назад», «идет сейчас»). Года в датах нет, вычислять его самому опасно:
    ошибка превращает прошедшее событие в будущее. TimePad считает пометку сам.
    """
    out, seen = [], set()
    for chunk in re.split(r'(?=<div class="[^"]*t-card_event)', list_html):
        m = re.search(r"/event/(\d+)", chunk)
        if not m:
            continue
        eid = m.group(1)
        if eid in seen:
            continue
        seen.add(eid)
        frag = text_of(chunk)
        rel = None
        if re.search(r"идет сейчас|идёт сейчас", frag):
            rel = "ongoing"
        elif re.search(r"через\s+\d+\s+(?:день|дня|дней|час|часа|часов|минут)", frag) or "сегодня" in frag:
            rel = "future"
        elif re.search(r"\d+\s+(?:день|дня|дней|месяц\w*|год\w*)\s+назад", frag):
            rel = "past"
        out.append({"id": eid, "fragment": frag, "when": rel})
    return out


def parse_dates(fragment: str, year_hint: int):
    """Из карточки: одно- или многодневное событие.

    Многодневное («с 10:00 18 сентября до 12:00 20 сентября») — признак
    конференции: по правилу владельца она заводится ОДНИМ событием на весь
    диапазон, сессии в отчёт.
    """
    year = year_hint
    ym = re.search(r"(\d{1,2})[–—-](\d{1,2})\s+(" + MONTH_RE + r")\w*\s+(\d{4})", fragment)
    if ym:
        year = int(ym.group(4))

    multi = re.search(
        r"с\s+(\d{1,2}):(\d{2})\s+(\d{1,2})\s+(" + MONTH_RE + r")\w*\s+до\s+(\d{1,2}):(\d{2})\s+(\d{1,2})\s+(" + MONTH_RE + r")\w*",
        fragment)
    if multi:
        h1, m1, d1, mon1, h2, m2, d2, mon2 = multi.groups()
        start = datetime(year, MONTHS[mon1.lower()], int(d1), int(h1), int(m1), tzinfo=MSK)
        end = datetime(year, MONTHS[mon2.lower()], int(d2), int(h2), int(m2), tzinfo=MSK)
        return start, end, True

    one = re.search(
        r"(\d{1,2})\s+(" + MONTH_RE + r")\w*\s+c\s+(\d{1,2}):(\d{2})(?:\s+до\s+(\d{1,2}):(\d{2}))?",
        fragment)
    if one:
        d, mon, h1, m1, h2, m2 = one.groups()
        start = datetime(year, MONTHS[mon.lower()], int(d), int(h1), int(m1), tzinfo=MSK)
        end = None
        if h2:
            end = start.replace(hour=int(h2), minute=int(m2))
            if end <= start:                 # программа за полночь
                end += timedelta(days=1)
        return start, end, False
    return None, None, False


def title_of(fragment: str, page: str) -> str:
    m = re.search(r'<meta property="og:title" content="([^"]*)"', page)
    if m:
        return html.unescape(m.group(1)).replace(" / События на TimePad.ru", "").strip()
    return fragment[:80]


def price_of(page: str):
    """У TimePad цена лежит числом в разметке билетов: "price":0 — бесплатно."""
    prices = sorted({int(float(p)) for p in re.findall(r'"price"\s*:\s*"?(\d+(?:\.\d+)?)', page)})
    if not prices:
        return {"price_type": "free"}, "цена на странице не найдена — проверить"
    if prices == [0]:
        return {"price_type": "free"}, None
    paid = [p for p in prices if p > 0]
    low = min(paid)
    if len(set(paid)) > 1 or 0 in prices:
        return {"price_type": "from", "price_min": low}, None
    return {"price_type": "fixed", "price_min": low, "price_max": low}, None


def classify(title: str, topics: dict, conference_words: list, multi_day: bool):
    low = title.lower()
    if any(w in low for w in conference_words) or multi_day:
        return "conference", "многодневная программа" if multi_day else "конференция по названию"
    if any(w in low for w in topics["skip"]):
        return "skip", "формат вне темы площадки (набор, день открытых дверей, профильный семинар)"
    if any(w in low for w in topics["fits"]):
        return "fit", None
    return "unsure", "по названию тема не очевидна — решает редактор"


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--days", type=int, default=90, help="горизонт вперёд")
    ap.add_argument("--all", action="store_true", help="не пропускать уже импортированные")
    ap.add_argument("--pause", type=float, default=1.5)
    args = ap.parse_args()

    cfg = json.load(open(f"{HERE}/config.json", encoding="utf-8"))
    state = json.load(open(f"{HERE}/state.json", encoding="utf-8"))
    now = datetime.now(MSK)

    candidates, unsure, rejected = [], [], []
    for src in cfg["sources"]:
        print(f"→ {src['slug']}: читаю {src['list_url']}")
        page = fetch(src["list_url"])
        found = cards(page)
        print(f"  событий на странице: {len(found)}")
        for card in found:
            eid = card["id"]
            key = f"{src['slug']}/{eid}"
            if not args.all and key in state["imported"]:
                continue
            if card["when"] == "past":
                rejected.append({"key": key, "reason": "событие прошло", "title": card["fragment"][:60]})
                continue

            start, end, multi = parse_dates(card["fragment"], now.year)
            if not start:
                rejected.append({"key": key, "reason": "не разобрана дата", "title": card["fragment"][:60]})
                continue
            if card["when"] != "ongoing" and start < now:
                rejected.append({"key": key, "reason": f"дата в прошлом ({start:%d.%m})", "title": card["fragment"][:60]})
                continue
            if start > now + timedelta(days=args.days):
                rejected.append({"key": key, "reason": f"дальше горизонта ({start:%d.%m})", "title": card["fragment"][:60]})
                continue

            url = f"https://{src['slug']}.timepad.ru/event/{eid}/"
            detail = fetch(url)
            time.sleep(args.pause)
            title = title_of(card["fragment"], detail)
            bucket, note = classify(title, cfg["topics"], cfg["conference_words"], multi)
            price, price_note = price_of(detail)
            desc = re.search(r'<meta property="og:description" content="([^"]*)"', detail)
            cover = re.search(r'<meta property="og:image" content="([^"]*)"', detail)

            notes = [n for n in (note, price_note) if n]
            if multi:
                notes.append("многодневное: заводится ОДНИМ событием на весь диапазон, сессии — в отчёт")
            if card["when"] == "ongoing":
                notes.append("идёт прямо сейчас")

            body = {
                "title": title,
                "description": "",           # пишет редактор своими словами
                "city": src["city"],
                "venue_id": src["venue"]["id"],
                "category_ids": [],
                "status": "draft",
                "format": "offline",
                "starts_at": start.isoformat(),
                "signup_mode": "external",
                "external_registration_url": url,
                "source_url": url,
                "source_label": cfg["source_label"],
                "organizer_id": cfg["organizer_id"],
            }
            if end:
                body["ends_at"] = end.isoformat()
            body.update(price)

            item = {
                "key": key, "url": url, "bucket": bucket,
                "category_slug": src["default_category"],
                "cover_url": cover.group(1) if cover else None,
                "announcement": html.unescape(desc.group(1)) if desc else "",
                "notes": notes, "event_input": body,
            }
            if bucket == "skip":
                rejected.append({"key": key, "reason": note, "title": title})
            elif bucket == "unsure":
                unsure.append(item)
            else:
                candidates.append(item)

    candidates.sort(key=lambda c: c["event_input"]["starts_at"])
    unsure.sort(key=lambda c: c["event_input"]["starts_at"])
    stamp = now.strftime("%Y%m%d-%H%M")
    json.dump(candidates, open(f"{HERE}/out/candidates-{stamp}.json", "w", encoding="utf-8"),
              ensure_ascii=False, indent=1)
    json.dump(unsure, open(f"{HERE}/out/unsure-{stamp}.json", "w", encoding="utf-8"),
              ensure_ascii=False, indent=1)

    def block(item):
        ei = item["event_input"]
        price = {"free": "бесплатно", "fixed": f"{ei.get('price_min')} ₽",
                 "from": f"от {ei.get('price_min')} ₽"}[ei["price_type"]]
        when = datetime.fromisoformat(ei["starts_at"]).strftime("%d.%m %H:%M")
        if ei.get("ends_at"):
            when += " — " + datetime.fromisoformat(ei["ends_at"]).strftime("%d.%m %H:%M")
        return ["### " + ei["title"], f"- когда: **{when}**", f"- цена: {price}",
                f"- страница: {item['url']}",
                f"- обложка: {'есть (права у площадки — не перезаливаем автоматически)' if item['cover_url'] else 'нет'}",
                ("- на что смотреть: " + "; ".join(item["notes"])) if item["notes"] else "",
                "", "> " + (item["announcement"][:400] or "—"), ""]

    lines = [f"# TimePad — кандидаты на {now:%d.%m.%Y %H:%M}", "",
             f"Подходящих: {len(candidates)}, спорных: {len(unsure)}, отброшено: {len(rejected)}.", "",
             "Описание пишет редактор своими словами: ниже цитата анонса только для чтения.", "",
             "## Подходят", ""]
    for c in candidates:
        lines += block(c)
    lines += ["## Спорные — решает редактор", ""]
    for c in unsure:
        lines += block(c)
    lines += ["## Отброшено", ""]
    for r in rejected:
        lines.append(f"- {r['key']} — {r['reason']} — {r['title'][:70]}")
    open(f"{HERE}/out/review-{stamp}.md", "w", encoding="utf-8").write("\n".join(lines))

    print(f"\nподходят: {len(candidates)}, спорные: {len(unsure)}, отброшено: {len(rejected)}")
    print(f"  {HERE}/out/candidates-{stamp}.json\n  {HERE}/out/review-{stamp}.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
