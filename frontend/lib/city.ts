import type { LatLon } from "@/lib/geo";

/** City slugs shared with the backend (?city=, events.city, venues.city). */
export type CitySlug = "msk" | "spb";

/** Cookie that stores the visitor's chosen city slug (SSR-readable). */
export const CITY_COOKIE = "lia_city";

/** Single source of truth for the city: the header switcher, the feed copy,
 * the login caption and the default map center all derive from it, so adding
 * a real second city later is a data change, not a copy hunt (QA 14.08). */
export interface City {
  /** Backend slug, e.g. "msk" — what goes into cookies and ?city=. */
  slug: CitySlug;
  /** Header code, e.g. «МСК». */
  code: string;
  name: string;
  /** «События <genitive>». */
  genitive: string;
  /** «Живые события <culturalGenitive> — …» (adjective gender differs). */
  culturalGenitive: string;
  center: LatLon;
  /** SSR fallback only — the live value comes from GET /cities (the
   * cities.spb_available runtime setting). Unavailable cities render as
   * «Скоро» and stay disabled. */
  available: boolean;
}

export const CITIES: readonly City[] = [
  {
    slug: "msk",
    code: "МСК",
    name: "Москва",
    genitive: "Москвы",
    culturalGenitive: "культурной Москвы",
    center: [55.742, 37.618],
    available: true,
  },
  {
    slug: "spb",
    code: "СПБ",
    name: "Санкт-Петербург",
    genitive: "Петербурга",
    culturalGenitive: "культурного Петербурга",
    center: [59.938784, 30.314997],
    available: false,
  },
];

export const CURRENT_CITY = CITIES[0];

/** Resolves a slug (cookie / ?city=) to a City, silently falling back to the
 * default for unknown or missing values — a garbage cookie must never 500. */
export function cityBySlug(slug: string | undefined | null): City {
  return CITIES.find((c) => c.slug === slug) ?? CITIES[0];
}

export function cityTagline(city: City): string {
  return `Живые события ${city.culturalGenitive} — медиации, лекции и разговоры об искусстве. Участвуйте, а не только смотрите.`;
}

export function cityLoginCaption(city: City): string {
  return `События ${city.genitive}`;
}
