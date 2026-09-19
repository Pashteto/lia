#!/usr/bin/env python3
"""Сборщик анонсов из публичных превью Telegram-каналов (t.me/s/<канал>).

Зачем именно так: с боевого сервера Telegram недоступен целиком (замер 2026-09-17),
а MTProto под личным аккаунтом грозит баном. Превью — обычные веб-страницы:
ни аккаунта, ни бота, банить нечего. Подробности: docs/research/2026-09-17-telegram-ingest/.

Запускать с машины, где Telegram открывается (Mac под VPN или VPS вне РФ):

    python3 ingest.py                  # собрать кандидатов в out/
    python3 ingest.py --days 30        # окно поиска шире
    python3 ingest.py --channel nefiktiv

Скрипт НИЧЕГО не публикует: он только готовит черновики. Тела событий из
out/candidates-*.json заводятся в PRESENCE отдельно (см. README.md), всегда как
`draft`, и публикуются руками редактора.
"""
import argparse, json, html, re, sys, time, urllib.request, os
from datetime import datetime, timedelta, timezone

HERE = os.path.dirname(os.path.abspath(__file__))
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130 Safari/537.36"
MSK = timezone(timedelta(hours=3))

RANGE_PREFIX = re.compile(r"(?:^|[\s(])(?:до|по|с|от|начиная с|продлена до|открыта до|работает до)\s*$", re.I)

MONTHS = {m: i + 1 for i, m in enumerate(
    "январ феврал март апрел ма июн июл август сентябр октябр ноябр декабр".split())}
WEEKDAYS = {"понедельник": 0, "вторник": 1, "сред": 2, "четверг": 3,
            "пятниц": 4, "суббот": 5, "воскресен": 6}

# «не событие»: реклама подписки, дайджесты, репортажи без конкретного сеанса
NOISE = re.compile(r"подписк|скидк|промокод|дайджест|итоги (?:недели|месяца)|"
                   r"напоминаем, что запись|смотрите запись|архив лекц", re.I)


def fetch(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept-Language": "ru,en"})
    with urllib.request.urlopen(req, timeout=25) as r:
        return r.read().decode("utf-8", "ignore")


def strip_tags(chunk: str) -> str:
    chunk = re.sub(r"<br\s*/?>", "\n", chunk)
    chunk = re.sub(r"<[^>]+>", "", chunk)
    return html.unescape(chunk).strip()


def posts_from_page(page_html: str, channel: str):
    """Разбирает страницу превью на посты: id, дата публикации, текст, ссылки, фото."""
    out = []
    for block in re.split(r'(?=<div class="tgme_widget_message[ "])', page_html):
        m = re.search(r'data-post="(%s/(\d+))"' % re.escape(channel), block)
        if not m:
            continue
        body = re.search(r'<div class="tgme_widget_message_text[^"]*"[^>]*>(.*?)</div>\s*'
                         r'(?:<div class="tgme_widget_message_footer|<div class="tgme_widget_message_bubble_end)',
                         block, re.S) or \
               re.search(r'<div class="tgme_widget_message_text[^"]*"[^>]*>(.*?)</div>', block, re.S)
        if not body:
            continue
        raw = body.group(1)
        t = re.search(r'<time[^>]+datetime="([^"]+)"', block)
        photos = [u for u in re.findall(r"background-image:url\('([^']+)'\)", block)
                  if "telesco.pe" in u or "cdn-telegram" in u]
        out.append({
            "post": m.group(1), "id": int(m.group(2)),
            "posted_at": t.group(1) if t else None,
            "text": strip_tags(raw),
            "links": [u for u in re.findall(r'href="([^"]+)"', raw) if not u.startswith("https://t.me/")],
            "photo": photos[0] if photos else None,
        })
    return out


def collect(channel: str, pages: int, pause: float):
    seen, before = {}, None
    for _ in range(pages):
        url = f"https://t.me/s/{channel}" + (f"?before={before}" if before else "")
        try:
            page = fetch(url)
        except Exception as exc:                      # сеть/ограничение частоты — не роняем весь прогон
            print(f"  ! {channel}: {exc}", file=sys.stderr)
            break
        batch = posts_from_page(page, channel)
        if not batch:
            break
        for p in batch:
            seen.setdefault(p["post"], p)
        before = min(p["id"] for p in batch)
        time.sleep(pause)
    return sorted(seen.values(), key=lambda p: p["id"])


def _date_in(fragment: str, posted: datetime, allow_range: bool = False):
    """Ищет дату во фрагменте. Границы периода («до 4 октября») пропускает:
    это конец выставки, а не начало события."""
    for m in re.finditer(r"\b(\d{1,2})[./](\d{1,2})(?:[./](\d{2,4}))?\b", fragment):
        if not allow_range and RANGE_PREFIX.search(fragment[:m.start()]):
            continue
        day, month = int(m.group(1)), int(m.group(2))
        if not (1 <= month <= 12 and 1 <= day <= 31):
            continue
        if m.group(3):
            year = int(m.group(3))
            year += 2000 if year < 100 else 0
            try:
                return datetime(year, month, day, tzinfo=MSK), "явная дата"
            except ValueError:
                pass
        return _fix_year(month, day, posted), "дата без года"
    for m in re.finditer(r"\b(\d{1,2})\s+(" + "|".join(MONTHS) + r")\w*", fragment, re.I):
        if not allow_range and RANGE_PREFIX.search(fragment[:m.start()]):
            continue
        return _fix_year(MONTHS[m.group(2).lower()], int(m.group(1)), posted), "дата словом"
    return None, None


def _fix_year(month: int, day: int, posted: datetime) -> datetime:
    for year in (posted.year, posted.year + 1):
        try:
            cand = datetime(year, month, day, tzinfo=MSK)
        except ValueError:
            continue
        if (cand.date() - posted.date()).days >= -1:
            return cand
    return datetime(posted.year, month, day, tzinfo=MSK)


def parse_datetime(text: str, posted: datetime):
    """Сначала ищем строку со временем и дату рядом с ней — так анонс «19.09 20:00»
    и «Суббота 19.09 / 19.00 — CITROMOoN» разбираются одинаково правильно."""
    lines = text.split("\n")
    for i, line in enumerate(lines):
        tm, doors = parse_time(line)
        if not tm:
            continue
        for frag in (line, "\n".join(lines[max(0, i - 2):i]), "\n".join(lines[i + 1:i + 3])):
            date, note = _date_in(frag, posted)
            if date:
                return date, tm, doors, note
    date, note = _date_in(text, posted)
    if date:
        tm, doors = parse_time(text)
        if tm:
            return date, tm, doors, note
    return None, None, None, None


def parse_date(text: str, posted: datetime):
    """Дату ищем в порядке надёжности. Год в постах почти никогда не указан —
    берём ближайший будущий относительно даты публикации."""
    def fix_year(month: int, day: int) -> datetime:
        for year in (posted.year, posted.year + 1):
            try:
                cand = datetime(year, month, day, tzinfo=MSK)
            except ValueError:
                continue
            if (cand.date() - posted.date()).days >= -1:
                return cand
        return datetime(posted.year, month, day, tzinfo=MSK)

    m = re.search(r"\b(\d{1,2})[./](\d{1,2})(?:[./](\d{2,4}))?\b", text)
    if m:
        day, month = int(m.group(1)), int(m.group(2))
        if 1 <= month <= 12 and 1 <= day <= 31:
            if m.group(3):
                year = int(m.group(3))
                year += 2000 if year < 100 else 0
                try:
                    return datetime(year, month, day, tzinfo=MSK), "явная дата"
                except ValueError:
                    pass
            return fix_year(month, day), "дата без года"
    m = re.search(r"\b(\d{1,2})\s+(" + "|".join(MONTHS) + r")\w*", text, re.I)
    if m:
        day = int(m.group(1))
        month = MONTHS[m.group(2).lower()]
        return fix_year(month, day), "дата словом"
    m = re.search(r"\b(?:в\s+)?(эту|этот|ближайш\w+|следующ\w+)?\s*(" + "|".join(WEEKDAYS) + r")\w*", text, re.I)
    if m:
        target = WEEKDAYS[m.group(2).lower()]
        delta = (target - posted.weekday()) % 7
        if m.group(1) and m.group(1).lower().startswith("следующ"):
            delta += 7
        return (posted + timedelta(days=delta or 7)).replace(hour=0, minute=0, second=0, microsecond=0), "день недели — ПРОВЕРИТЬ"
    return None, None


def parse_time(text: str):
    """Начало события.

    Две ловушки, обе встречались в реальных постах:
    • «18.09   20:00» — дата через точку выглядит как время. Различаем по второму
      числу: у времени это 00 или >12 (номера месяца 0 и 13+ не бывает), у даты — 01..12.
    • «19:30 - двери / 20:00 - начало» — подпись стоит ПОСЛЕ времени, поэтому
      ищем метку с обеих сторон, а двери отдаём отдельной заметкой.
    """
    cands = []  # (позиция, часы, минуты, помечено_как_начало)
    for m in re.finditer(r"\b([01]?\d|2[0-3])([:.])([0-5]\d)\b", text):
        hh, sep, mm = int(m.group(1)), m.group(2), int(m.group(3))
        if sep == "." and 1 <= mm <= 12:
            continue                      # это дата вида 18.09, а не время
        line_start = text.rfind("\n", 0, m.start()) + 1
        line_end = text.find("\n", m.end())
        line_end = len(text) if line_end == -1 else line_end
        before = text[max(line_start, m.start() - 14):m.start()]   # метку ищем в своей строке
        after = text[m.end():min(line_end, m.end() + 14)]
        is_start = bool(re.search(r"начал|музыка|старт|start|показ", before + after, re.I))
        if not is_start and re.search(r"двер|сбор гостей|open", before + after, re.I):
            continue                      # двери/сбор — не начало
        cands.append((m.start(), hh, mm, is_start))
    doors_note = None            # подпись «двери» бывает и до, и после времени
    for m in re.finditer(r"\b([01]?\d|2[0-3])([:.])([0-5]\d)\b", text):
        if m.group(2) == "." and 1 <= int(m.group(3)) <= 12:
            continue
        ls = text.rfind("\n", 0, m.start()) + 1
        le = text.find("\n", m.end())
        le = len(text) if le == -1 else le
        near = text[max(ls, m.start() - 14):m.start()] + text[m.end():min(le, m.end() + 14)]
        if re.search(r"двер|сбор гостей", near, re.I):
            doors_note = f"двери {m.group(1)}:{m.group(3)}"
            break
    if not cands:
        m = re.search(r"(?:начало|старт|start)\D{0,6}(\d{1,2})\s+(\d{2})\b", text, re.I)
        if m:                              # «start : 20 00»
            return (int(m.group(1)), int(m.group(2))), doors_note
        return None, doors_note
    labelled = [c for c in cands if c[3]]
    pos, hh, mm, _ = (labelled or cands)[0]
    return (hh, mm), doors_note


def parse_price(text: str):
    if re.search(r"бесплатн|вход свободн|free entry", text, re.I):
        return {"price_type": "free"}, None
    m = re.search(r"(?:донат|от)\s*(\d{3,5})\s*(?:₽|р\b|р\.|руб)", text, re.I)
    if m:
        return {"price_type": "from", "price_min": int(m.group(1))}, None
    m = re.search(r"(?:вход|билет|цена|стоимость)\D{0,15}(\d{3,5})\s*(?:₽|р\b|р\.|руб)?", text, re.I)
    if not m:
        m = re.search(r"\b(\d{3,5})\s*(?:₽|руб|р\.)", text)
    if m:
        v = int(m.group(1))
        return {"price_type": "fixed", "price_min": v, "price_max": v}, None
    return None, "цена не указана — уточнить"


def parse_venue(text: str, channel_cfg: dict, known: dict):
    for key, venue in known.items():
        if key in text.lower():
            return venue, f"площадка из текста: {venue['name']}"
    if re.search(r"\b(?:ул\.|улица|проспект|пр-т|наб\.|набережная|переулок)\b", text, re.I):
        # адрес есть, но он может быть и родным адресом канала — решает редактор
        pass
    return channel_cfg["default_venue"], None


def title_from(text: str) -> str:
    for line in [l.strip() for l in text.split("\n") if l.strip()]:
        clean = re.sub(r"^[^\wА-Яа-я«\"]+", "", line).strip()
        clean = re.sub(r"^\d{1,2}[./]\d{1,2}\s*[—-]?\s*", "", clean)
        clean = re.sub(r"^\d{1,2}[:.]\d{2}\s*[—-]?\s*", "", clean).strip()
        if len(clean) >= 12:
            return clean[:120]
    return text.strip().split("\n")[0][:120]


def analyse(post: dict, cfg: dict, channel_cfg: dict, horizon_days: int):
    text = post["text"]
    posted = datetime.fromisoformat(post["posted_at"].replace("Z", "+00:00")).astimezone(MSK)
    notes = []
    if NOISE.search(text):
        return None, "похоже на рекламу/дайджест"
    date, tm, doors, date_note = parse_datetime(text, posted)
    if not date:
        return None, "нет даты рядом со временем"
    if not tm:
        return None, "нет времени начала"
    if len(text) < 30:
        return None, "слишком короткий пост"
    starts = date.replace(hour=tm[0], minute=tm[1])
    now = datetime.now(MSK)
    if starts < now:
        return None, f"дата в прошлом ({starts:%d.%m %H:%M})"
    if starts > now + timedelta(days=horizon_days):
        return None, f"слишком далеко ({starts:%d.%m})"
    price, price_note = parse_price(text)
    if price_note:
        notes.append(price_note)
    if date_note and "ПРОВЕРИТЬ" in date_note:
        notes.append("дата вычислена по дню недели — проверить")
    elif date_note == "дата без года":
        notes.append("год в посте не указан — подставлен ближайший")
    if doors:
        notes.append(doors)
    if len(text) < 120:
        notes.append("короткий анонс — описание писать с нуля")
    if re.search(r"онлайн|online", text, re.I):
        notes.append("в посте упомянут онлайн — проверить формат")
    venue, venue_note = parse_venue(text, channel_cfg, cfg["known_venues"])
    if venue_note:
        notes.append(venue_note)
    if venue.get("id") is None:
        notes.append("площадку нужно завести вручную")
    body = {
        "title": title_from(text),
        "description": "",                      # пишет редактор своими словами
        "city": channel_cfg["city"],
        "venue_id": venue.get("id"),
        "category_ids": [],
        "status": "draft",
        "format": ("online" if re.search(r"полностью онлайн|только онлайн|\bZOOM\b", text, re.I)
                   else "offline"),
        "starts_at": starts.isoformat(),
        "signup_mode": "external",
        "external_registration_url": (post["links"][0] if post["links"] else f"https://t.me/{post['post']}"),
        # Атрибуция: площадки разрешили републикацию при условии ссылки на канал.
        # Это НЕ ссылка регистрации: t.me не в whitelist платформ, и подстановка
        # канала в external_registration_url отправила бы событие на модерацию.
        "source_url": f"https://t.me/{post['post']}",
        "source_label": f"Телеграм-канал «{channel_cfg['title']}»",
        "organizer_id": cfg["organizer_id"],
    }
    body.update(price or {"price_type": "free"})
    if not price:
        notes.append("цена не найдена — поставлено «бесплатно», проверить")
    return {
        "post": post["post"],
        "post_url": f"https://t.me/{post['post']}",
        "posted_at": post["posted_at"],
        "channel": channel_cfg["username"],
        "category_slug": channel_cfg["default_category"],
        "cover_url": post["photo"],
        "notes": notes,
        "source_text": text,
        "event_input": body,
    }, None


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--channel", help="только один канал")
    ap.add_argument("--pages", type=int, default=3, help="страниц истории на канал (по 10-20 постов)")
    ap.add_argument("--days", type=int, default=45, help="горизонт: насколько вперёд берём события")
    ap.add_argument("--pause", type=float, default=2.0, help="пауза между запросами, сек")
    ap.add_argument("--all", action="store_true", help="не пропускать уже импортированные")
    args = ap.parse_args()

    cfg = json.load(open(f"{HERE}/channels.json", encoding="utf-8"))
    state = json.load(open(f"{HERE}/state.json", encoding="utf-8"))
    chans = [c for c in cfg["channels"] if not args.channel or c["username"] == args.channel]
    if not chans:
        print("канал не найден в channels.json", file=sys.stderr)
        return 2

    candidates, rejected = [], []
    for ch in chans:
        print(f"→ {ch['username']}: читаю превью…")
        for post in collect(ch["username"], args.pages, args.pause):
            if not args.all and post["post"] in state["imported"]:
                continue
            item, reason = analyse(post, cfg, ch, args.days)
            if item:
                candidates.append(item)
            else:
                rejected.append({"post": post["post"], "reason": reason,
                                 "first_line": post["text"].split("\n")[0][:70]})
        print(f"  найдено кандидатов: {sum(1 for c in candidates if c['channel'] == ch['username'])}")

    candidates.sort(key=lambda c: c["event_input"]["starts_at"])
    stamp = datetime.now(MSK).strftime("%Y%m%d-%H%M")
    out_json = f"{HERE}/out/candidates-{stamp}.json"
    json.dump(candidates, open(out_json, "w", encoding="utf-8"), ensure_ascii=False, indent=1)

    lines = [f"# Кандидаты из Telegram — {datetime.now(MSK):%d.%m.%Y %H:%M}", "",
             f"Каналы: {', '.join(c['username'] for c in chans)}. "
             f"Кандидатов: {len(candidates)}, отброшено: {len(rejected)}.", "",
             "Всё проверяет редактор: описание пишется своими словами, цена и площадка сверяются с постом.", ""]
    for c in candidates:
        ei = c["event_input"]
        price = {"free": "бесплатно", "fixed": f"{ei.get('price_min')} ₽",
                 "from": f"от {ei.get('price_min')} ₽"}[ei["price_type"]]
        lines += [f"## {ei['title']}",
                  f"- когда: **{datetime.fromisoformat(ei['starts_at']):%d.%m %H:%M}**",
                  f"- канал: @{c['channel']} · [пост]({c['post_url']})",
                  f"- цена: {price}",
                  f"- обложка: {'есть' if c['cover_url'] else 'нет'}",
                  ("- на что смотреть: " + "; ".join(c["notes"])) if c["notes"] else "",
                  "", "```", c["source_text"][:700], "```", ""]
    lines += ["## Отброшено", ""]
    for r in rejected:
        lines.append(f"- {r['post']} — {r['reason']} — {r['first_line']}")
    out_md = f"{HERE}/out/review-{stamp}.md"
    open(out_md, "w", encoding="utf-8").write("\n".join(lines))

    print(f"\nкандидатов: {len(candidates)}, отброшено: {len(rejected)}")
    print(f"  {out_json}\n  {out_md}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
