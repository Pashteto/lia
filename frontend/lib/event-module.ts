import { categoryNumeral } from "./category-numerals";
import { formatModuleDate } from "./format";
import { priceLabel } from "./price-label";
import type { LiaEvent } from "./types";

export interface EventModuleData {
  numeral: string;
  category: string;
  title: string;
  venue: string;
  date: string;
  price: string;
  href: string;
}

/** Adapts a backend event to the EventModule props. Numerals are positional
 * in the backend's ordered category list (Swiss rule: numerals, not colours). */
export function eventToModuleProps(
  event: LiaEvent,
  categories: ReadonlyArray<{ slug: string }>,
): EventModuleData {
  const cat = event.categories[0];
  return {
    numeral: cat ? categoryNumeral(cat.slug, categories) : "—",
    category: cat?.label ?? "—",
    title: event.title,
    venue: event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—"),
    date: formatModuleDate(event.startsAt, event.endsAt),
    price: priceLabel(event.priceMin, event.priceType),
    href: `/events/${event.id}`,
  };
}
