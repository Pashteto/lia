import type { LiaEvent } from "./types";

// A discovery filter chip. Special chips (all/today/weekend/nearby) are not
// real categories, so this is its own shape rather than EventCategory.
export interface DiscoveryFilter {
  slug: string;
  label: string;
}

// Mock data for the Discovery scaffold. Content (titles, organizers) is drawn
// from the curatorial copy in design/screens/discovery.html. The deployed demo
// (lia.pashteto.com) renders from this when the backend is unreachable.

export const FILTERS: DiscoveryFilter[] = [
  { slug: "all", label: "Все" },
  { slug: "today", label: "Сегодня" },
  { slug: "weekend", label: "Выходные" },
  { slug: "mediation", label: "Медиации" },
  { slug: "workshop", label: "Мастер-классы" },
  { slug: "lecture", label: "Лекции" },
  { slug: "nearby", label: "Рядом" },
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
