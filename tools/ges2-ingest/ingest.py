#!/usr/bin/env python3
"""Сборщик дискуссионных и инклюзивных событий Дома культуры «ГЭС-2».

Источник — календарь сайта ges-2.org. Сайт сделан на Next.js и сам ходит за
данными в открытый GraphQL (https://ges-2.org/graphql): тот же запрос
calendarItems, что строит страницу календаря, отдаёт события по датам вместе
с форматом (subtype), ценой, кнопкой регистрации и пометками доступности.
Токен не нужен.

Что отбираем (config.json → subtypes):
- «talk» — разговорные форматы: беседы, лекции, кинопоказы с обсуждением,
  книжные клубы. Берутся всегда.
- «inclusive» — остальные форматы (туры, лаборатории, концерты, кино…), но
  только если у события есть пометка доступности ГЭС-2: «Доступно глухим и
  слабослышащим» / «Доступно незрячим и слабовидящим».
- «unsure» — публичная программа: там и разговоры, и йога; решает редактор.
Выставки, инсталляции и платные мастер-классы в «Сводах» не берём никогда.

Серии. Медиаторский тур или лаборатория идут по много раз в месяц, а в
данных у них старая дата старта. Время конкретной встречи берётся из
calendarHours на этот день. Серию (3+ встречи в окне) предлагаем ОДНИМ
событием — ближайшей встречей — в корзину спорных, остальные даты в отчёт;
иначе афиша зарастёт ежедневными турами.

Пометка доступности на сайте общая для материала, а точный формат (РЖЯ,
тифлокомментарий, субтитры, индукционная петля) написан в описании. Редактор
переносит его в название или первую фразу описания — отдельного поля
доступности у события нет (docs/research/2026-09-15-inclusive-events).

    python3 ingest.py               # горизонт 30 дней
    python3 ingest.py --days 60
    python3 ingest.py --all         # включая уже импортированные

Ничего не публикует: готовит черновики (status: draft). Описание не пишет —
текст ГЭС-2 идёт только в отчёт, редактор пересказывает своими словами.
Обложка — логотип ГЭС-2 (venue.cover_file_id), фото с сайта не перезаливаем.
"""
import argparse, html, json, os, re, sys, urllib.request
from datetime import date, datetime, timedelta, timezone

HERE = os.path.dirname(os.path.abspath(__file__))
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130 Safari/537.36"
MSK = timezone(timedelta(hours=3))
SERIES_MIN = 3

QUERY = """query calendarItems($dates: [Date!]) {
  calendarItems(pageFullPath: "ges2/calendar", dates: $dates) {
    forDate calendarHours customTitle
    buttons { title url }
    material {
      id ges2FullPath paid price ageRestriction
      subtype { slug title }
      accessibility { title description group { slug } }
      translation { title description shareDescription verstkaDesktopHtml }
    }
  }
}"""


def fetch(url: str, body: dict | None = None) -> str:
    data = json.dumps(body).encode() if body is not None else None
    headers = {"User-Agent": UA, "Accept-Language": "ru,en"}
    if data:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers)
    with urllib.request.urlopen(req, timeout=60) as r:
        return r.read().decode("utf-8", "ignore")


def load_categories(api_base: str) -> dict:
    """slug → id категорий из живого API (в теле события нужны id, не slug)."""
    try:
        return {c["slug"]: c["id"] for c in json.loads(fetch(api_base.rstrip("/") + "/api/v1/categories"))}
    except Exception as e:
        print(f"категории не загрузились ({e}) — черновики будут без рубрики", file=sys.stderr)
        return {}


def load_feed(api_base: str) -> set:
    """(название, дата) уже опубликованных событий — чтобы не предлагать дубль,
    заведённый руками или другим сборщиком (у них другой source_url)."""
    out = set()
    for city in ("msk", "spb"):
        try:
            items = json.loads(fetch(f"{api_base.rstrip('/')}/api/v1/events?city={city}&limit=500"))
        except Exception as e:
            print(f"афиша {city} не загрузилась ({e}) — дубли не проверены", file=sys.stderr)
            continue
        for e in items.get("items", items) if isinstance(items, dict) else items:
            out.add((norm(e["title"]), e["starts_at"][:10]))
    return out


def text_of(chunk: str | None) -> str:
    return " ".join(html.unescape(re.sub(r"<[^>]+>", " ", chunk or "")).split())


def norm(title: str) -> str:
    return re.sub(r"[^\w]+", " ", text_of(title).lower()).strip()


def in_feed(title: str, day: str, feed: set) -> bool:
    """Совпадение по дате и названию — точному или когда одно содержит другое:
    в афише название часто дополнено («Излучение голоса: вокальная лаборатория»)."""
    t = norm(title)
    return any(d == day and (t == f or t in f or f in t) for f, d in feed if len(min(t, f, key=len)) >= 6)


ACCESS_RE = re.compile(r"(?:[^.!?\n]|\.(?=\S)){0,140}(?:РЖЯ|жестов\w* язык|тифлокоммент\w*|субтитр\w*|индукционн\w* петл\w*|"
                       r"тактильн\w*|шрифт\w* Брайл\w*|для незрячих|для глухих|слабослыш\w*|слабовидящ\w*|"
                       r"нейроотлич\w*|ментальн\w* особенност\w*)(?:[^.!?\n]|\.(?=\S)){0,140}[.!?]?", re.I)


def access_details(page_html: str | None) -> list:
    """Фразы из текста страницы, где назван конкретный формат доступности."""
    seen, out = set(), []
    page = re.sub(r"<(style|script)\b.*?</\1>", " ", page_html or "", flags=re.S | re.I)
    for m in ACCESS_RE.finditer(text_of(page)):
        s = m.group(0).strip()
        if s.lower() not in seen:
            seen.add(s.lower())
            out.append(s)
    return out[:4]


def slots(hours: str):
    """'<time>13:00–14:30</time><br /><time>17:00–18:30</time>' → [('13:00','14:30'), …]"""
    return re.findall(r"(\d{1,2}:\d{2})\s*[–—-]\s*(\d{1,2}:\d{2})", text_of(hours))


def at(day: str, hm: str) -> datetime:
    h, m = map(int, hm.split(":"))
    return datetime.fromisoformat(day).replace(hour=h, minute=m, tzinfo=MSK)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--days", type=int, default=30, help="горизонт вперёд")
    ap.add_argument("--all", action="store_true", help="не пропускать уже импортированные")
    args = ap.parse_args()

    cfg = json.load(open(f"{HERE}/config.json", encoding="utf-8"))
    state = json.load(open(f"{HERE}/state.json", encoding="utf-8"))
    categories = load_categories(cfg["api_base"])
    feed = load_feed(cfg["api_base"])
    now = datetime.now(MSK)

    days = [(now.date() + timedelta(days=i)).isoformat() for i in range(args.days + 1)]
    print(f"→ ГЭС-2: календарь {days[0]} … {days[-1]}")
    resp = json.loads(fetch(cfg["graphql"], {"query": QUERY, "variables": {"dates": days}}))
    if resp.get("errors"):
        print(f"GraphQL ответил ошибкой: {resp['errors'][:2]}", file=sys.stderr)
        return 1
    items = resp["data"]["calendarItems"]
    print(f"  встреч в календаре: {len(items)}")

    # Встречи одного материала — вместе: так видно серии.
    by_material: dict = {}
    for it in items:
        by_material.setdefault(it["material"]["id"], []).append(it)

    candidates, unsure, rejected = [], [], []
    for mid, meets in by_material.items():
        m = meets[0]["material"]
        title = text_of(meets[0].get("customTitle") or m["translation"]["title"])
        sub = (m.get("subtype") or {}).get("slug") or ""
        sub_title = (m.get("subtype") or {}).get("title") or "без формата"
        rule = cfg["subtypes"].get(sub)
        access = [a for a in (m.get("accessibility") or []) if a.get("title")]
        key0 = f"ges2/{mid}"

        why_not = None
        if sub in cfg["never"]:
            why_not = f"формат не берём ({sub_title})"
        elif any(w in title.lower() for w in cfg["skip_titles"]):
            why_not = f"вне подборки ({sub_title})"
        elif not rule:
            why_not = f"формат вне подборки ({sub_title})"
        elif rule["bucket"] == "inclusive" and not access:
            why_not = f"{sub_title} без пометки доступности"
        if why_not:
            rejected.append({"key": key0, "reason": why_not, "title": title})
            continue

        # Ближайшая встреча, которой ещё нет ни в state, ни в афише.
        meets.sort(key=lambda it: it["forDate"])
        future, chosen = [], None
        for it in meets:
            sl = slots(it["calendarHours"])
            if not sl:
                continue
            start = at(it["forDate"], sl[0][0])
            if start < now:
                continue
            future.append((it, sl, start))
        if not future:
            rejected.append({"key": key0, "reason": "все встречи в окне уже прошли или без времени", "title": title})
            continue
        if len(future) >= SERIES_MIN:
            listed = [x["forDate"] for x, _, _ in future if in_feed(title, x["forDate"], feed)]
            done = [k for k in state["imported"] if k.startswith(key0 + "/") and k[-10:] >= days[0]]
            if (listed or done) and not args.all:
                when = (listed or [d[-10:] for d in done])[0]
                rejected.append({"key": key0, "reason": f"серия уже в афише ({when[8:10]}.{when[5:7]})", "title": title})
                continue
        for it, sl, start in future:
            key = f"{key0}/{it['forDate']}"
            if not args.all and key in state["imported"]:
                continue
            if in_feed(title, it["forDate"], feed):
                rejected.append({"key": key, "reason": "уже в афише", "title": title})
                continue
            chosen = (it, sl, start, key)
            break
        if not chosen:
            continue
        it, sl, start, key = chosen

        series = len(future) >= SERIES_MIN
        notes = []
        if series:
            notes.append(f"серия: {len(future)} встреч в окне — заводим одну ближайшую; даты: "
                         + ", ".join(f"{x['forDate'][8:10]}.{x['forDate'][5:7]}" for x, _, _ in future))
        if len(sl) > 1:
            notes.append("в этот день несколько сеансов: " + ", ".join(f"{a}–{b}" for a, b in sl)
                         + " — взят первый, остальные упомянуть в описании")
        details = access_details(m["translation"].get("verstkaDesktopHtml"))
        for a in access:
            notes.append(f"доступность: {a['title']} — точный формат вынести в название или первую фразу"
                         + ("" if details else "; на странице формат не назван — проверить"))
        price_cfg = {"price_type": "free"}
        if m.get("paid"):
            if m.get("price"):
                price_cfg = {"price_type": "fixed", "price_min": int(m["price"]), "price_max": int(m["price"])}
            else:
                price_cfg = {"price_type": "from", "price_min": None}
                notes.append("платно, цена на сайте не указана — уточнить на странице билетов и проставить")

        url = f"{cfg['site']}/{m['ges2FullPath']}"
        button = next((b for b in it.get("buttons") or [] if b.get("url")), None)
        body = {
            "title": title,
            "description": "",            # пишет редактор своими словами
            "city": cfg["city"],
            "venue_id": cfg["venue"]["id"],
            "category_ids": [categories[rule["category"]]] if rule["category"] in categories else [],
            "status": "draft",
            "format": "offline",
            "starts_at": start.isoformat(),
            "ends_at": at(it["forDate"], sl[0][1]).isoformat(),
            "source_url": url,
            "source_label": cfg["source_label"],
            "organizer_id": cfg["organizer_id"],
            "cover_file_id": cfg["venue"]["cover_file_id"],
            **price_cfg,
        }
        if button:
            body["signup_mode"] = "external"
            body["external_registration_url"] = button["url"].replace("{CURRENT_DATE}", it["forDate"])
        else:
            body["signup_mode"] = "open"
            notes.append("кнопки регистрации нет — вход свободный или уточнить")
        if not body["category_ids"]:
            notes.append("рубрика не подставилась — выбрать вручную")

        item = {
            "key": key, "url": url, "format": sub_title, "category_slug": rule["category"],
            "accessibility": [a["title"] for a in access],
            "access_details": details,
            "announcement": text_of(m["translation"].get("description")
                                    or m["translation"].get("shareDescription")),
            "notes": notes, "event_input": body,
        }
        if rule["bucket"] == "unsure" or series:
            unsure.append(item)
        else:
            candidates.append(item)

    candidates.sort(key=lambda c: c["event_input"]["starts_at"])
    unsure.sort(key=lambda c: c["event_input"]["starts_at"])
    os.makedirs(f"{HERE}/out", exist_ok=True)
    stamp = now.strftime("%Y%m%d-%H%M")
    for name, data in (("candidates", candidates), ("unsure", unsure)):
        json.dump(data, open(f"{HERE}/out/{name}-{stamp}.json", "w", encoding="utf-8"), ensure_ascii=False, indent=1)

    def block(item):
        ei = item["event_input"]
        price = {"free": "бесплатно", "fixed": f"{ei.get('price_min')} ₽",
                 "from": f"от {ei.get('price_min') or '?'} ₽"}[ei["price_type"]]
        when = datetime.fromisoformat(ei["starts_at"]).strftime("%d.%m %H:%M")
        when += "–" + datetime.fromisoformat(ei["ends_at"]).strftime("%H:%M")
        return ["### " + ei["title"],
                f"- когда: **{when}** · формат: {item['format']} · цена: {price}",
                f"- страница: {item['url']}",
                f"- регистрация: {ei.get('external_registration_url', '—')}",
                ("- доступность: " + "; ".join(item["accessibility"])) if item["accessibility"] else "",
                *[f"  - «{d}»" for d in item["access_details"]],
                ("- на что смотреть: " + "; ".join(item["notes"])) if item["notes"] else "",
                "", "> " + (item["announcement"][:500] or "—"), ""]

    lines = [f"# ГЭС-2 — кандидаты на {now:%d.%m.%Y %H:%M}", "",
             f"Окно {args.days} дн. Подходят: {len(candidates)}, спорные: {len(unsure)}, отброшено: {len(rejected)}.", "",
             "Описание пишет редактор своими словами: ниже текст ГЭС-2 только для чтения.", "",
             "## Подходят — разговорные и инклюзивные разовые события", ""]
    for c in candidates:
        lines += block(c)
    lines += ["## Спорные — серии и публичная программа, решает редактор", ""]
    for c in unsure:
        lines += block(c)
    lines += ["## Отброшено", ""]
    for r in sorted(rejected, key=lambda r: r["reason"]):
        lines.append(f"- {r['key']} — {r['reason']} — {r['title'][:70]}")
    open(f"{HERE}/out/review-{stamp}.md", "w", encoding="utf-8").write("\n".join(lines))

    print(f"\nподходят: {len(candidates)}, спорные: {len(unsure)}, отброшено: {len(rejected)}")
    print(f"  {HERE}/out/candidates-{stamp}.json\n  {HERE}/out/review-{stamp}.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
