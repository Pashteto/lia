import type { LatLon } from "@/lib/geo";

/** Single source of truth for the city: the header switcher, the feed copy,
 * the login caption and the default map center all derive from it, so adding
 * a real second city later is a data change, not a copy hunt (QA 14.08). */
export interface City {
  /** Header code, e.g. «МСК». */
  code: string;
  name: string;
  /** «События <genitive>». */
  genitive: string;
  /** «Живые события <culturalGenitive> — …» (adjective gender differs). */
  culturalGenitive: string;
  center: LatLon;
  /** Only Moscow is live; the rest render as «Скоро» and stay disabled. */
  available: boolean;
}

export const CITIES: readonly City[] = [
  {
    code: "МСК",
    name: "Москва",
    genitive: "Москвы",
    culturalGenitive: "культурной Москвы",
    center: [55.742, 37.618],
    available: true,
  },
  {
    code: "СПБ",
    name: "Санкт-Петербург",
    genitive: "Петербурга",
    culturalGenitive: "культурного Петербурга",
    center: [59.938784, 30.314997],
    available: false,
  },
];

export const CURRENT_CITY = CITIES[0];

export function cityTagline(city: City): string {
  return `Живые события ${city.culturalGenitive} — медиации, лекции и разговоры об искусстве. Участвуйте, а не только смотрите.`;
}

export function cityLoginCaption(city: City): string {
  return `События ${city.genitive}`;
}
