import type { LiaEvent } from "./types";

// A discovery filter chip. The "all" chip is special (shows everything). A chip
// with `dateRange` filters by the event's start time over a half-open window
// (today / weekend); the rest filter by category slug.
export interface DiscoveryFilter {
  slug: string;
  label: string;
  // When present, the chip filters by start time over [from, to) instead of by
  // category slug. The SAME range is sent to the backend (so events beyond the
  // list cap are still found) and applied client-side (so the offline mock
  // fallback narrows too). Returns the window for the given "now".
  dateRange?: (now: Date) => { from: Date; to: Date };
}

// [start of today, start of tomorrow) in local time.
export function todayRange(now: Date): { from: Date; to: Date } {
  const from = new Date(now);
  from.setHours(0, 0, 0, 0);
  const to = new Date(from);
  to.setDate(from.getDate() + 1);
  return { from, to };
}

// The current week's weekend: [Sat 00:00, Mon 00:00) in local time. On Mon–Fri
// this is the upcoming weekend; on Sat/Sun it is the weekend in progress.
export function weekendRange(now: Date): { from: Date; to: Date } {
  const day = now.getDay(); // 0=Sun … 6=Sat
  const from = new Date(now);
  from.setHours(0, 0, 0, 0);
  if (day === 0) {
    // Sunday: the weekend's Saturday was yesterday.
    from.setDate(from.getDate() - 1);
  } else {
    // 0 days when today is Saturday, otherwise the coming Saturday.
    from.setDate(from.getDate() + ((6 - day + 7) % 7));
  }
  const to = new Date(from);
  to.setDate(from.getDate() + 2); // Monday 00:00, exclusive
  return { from, to };
}

// Mock data for the Discovery scaffold. Content (titles, organizers) is drawn
// from the curatorial copy in design/screens/discovery.html. The deployed demo
// (lia.pashteto.com) renders from this when the backend is unreachable.

// The "Рядом" chip was removed — it duplicated the dedicated geolocation
// "рядом со мной" button and filtered by a non-existent "nearby" category, so
// it always showed nothing. The date chips (today/weekend) filter by the
// event's start time via `dateRange` (applied server-side AND client-side);
// everything else filters by category slug.
export const FILTERS: DiscoveryFilter[] = [
  { slug: "all", label: "Все" },
  { slug: "today", label: "Сегодня", dateRange: todayRange },
  { slug: "weekend", label: "Выходные", dateRange: weekendRange },
  { slug: "mediation", label: "Медиации" },
  { slug: "workshop", label: "Мастер-классы" },
  { slug: "lecture", label: "Лекции" },
];

export const MOCK_EVENTS: LiaEvent[] = [
  {
    id: "evt-pamyat-arkhiv",
    title: "Память и архив: разговор у работ",
    description:
      "Совместная медиация в залах постоянной экспозиции. Смотрим, говорим, замечаем — без экскурсионного монолога.",
    categories: [{ id: "cat-mediation", slug: "mediation", label: "Медиации" }],
    format: "offline",
    status: "published",
    startsAt: "2026-06-13T19:00:00+03:00",
    priceType: "free",
    capacity: 18,
    attendeeCount: 11,
    coverUrl:
      "https://images.unsplash.com/photo-1518998053901-5348d3961a04?w=1200&q=80&auto=format&fit=crop",
    organizer: { id: "org-garage", name: "Музей «Гараж»", affiliation: "Медиатека" },
    venue: { id: "v-gorky", name: "Парк Горького", metro: "Парк культуры" },
  },
  {
    id: "evt-bumaga",
    title: "Бумага ручного отлива",
    description:
      "Мастерская: делаем лист бумаги из вторсырья и растительных волокон. Уносим с собой.",
    categories: [{ id: "cat-workshop", slug: "workshop", label: "Мастер-классы" }],
    format: "offline",
    status: "published",
    startsAt: "2026-06-14T14:00:00+03:00",
    priceType: "from",
    priceMin: 2500,
    capacity: 10,
    attendeeCount: 7,
    coverUrl:
      "https://images.unsplash.com/photo-1607344645866-009c320b63e0?w=1200&q=80&auto=format&fit=crop",
    organizer: { id: "org-tsekh", name: "Цех бумаги", affiliation: "Независимая мастерская" },
    venue: { id: "v-winzavod", name: "Винзавод", metro: "Чкаловская" },
  },
  {
    id: "evt-smotret-vmeste",
    title: "Что значит смотреть вместе",
    description:
      "Открытая лекция о практиках совместного просмотра и о том, как зритель становится участником.",
    categories: [{ id: "cat-lecture", slug: "lecture", label: "Лекции" }],
    format: "online",
    status: "published",
    startsAt: "2026-06-15T18:30:00+03:00",
    priceType: "free",
    attendeeCount: 64,
    coverUrl:
      "https://images.unsplash.com/photo-1531058020387-3be344556be6?w=1200&q=80&auto=format&fit=crop",
    organizer: { id: "org-inst", name: "Институт «База»", affiliation: "Лекторий" },
  },
  {
    id: "evt-zebald",
    title: "Читаем Зебальда",
    description:
      "Читательская группа: медленное чтение и разговор. В этот раз — «Кольца Сатурна».",
    categories: [
      { id: "cat-mediation", slug: "mediation", label: "Медиации" },
      { id: "cat-lecture", slug: "lecture", label: "Лекции" },
    ],
    format: "offline",
    status: "published",
    startsAt: "2026-06-16T20:00:00+03:00",
    priceType: "from",
    priceMin: 500,
    capacity: 12,
    attendeeCount: 9,
    coverUrl:
      "https://images.unsplash.com/photo-1524995997946-a1c2e315a42f?w=1200&q=80&auto=format&fit=crop",
    organizer: { id: "org-poryadok", name: "Порядок слов", affiliation: "Книжный магазин" },
    venue: { id: "v-poryadok", name: "Порядок слов", metro: "Маяковская" },
  },
];
