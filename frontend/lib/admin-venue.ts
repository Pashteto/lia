import { CITIES } from "@/lib/city";

export interface VenueSummary {
  /** Name · address · metro, or the reason there is nothing to show. */
  text: string;
  /** Russian city name, or "" when the event carries no known city. */
  city: string;
  /** An offline event with no venue — something for the moderator to weigh. */
  missing: boolean;
}

/**
 * What the moderation card says about where an event happens.
 *
 * Kept out of the component so the three states — a real venue, an online
 * event, and a venue nobody filled in — are checkable without rendering.
 */
export function venueSummary(input: {
  venue?: { name?: string; address?: string; metro?: string } | null;
  format?: string;
  city?: string;
}): VenueSummary {
  // cityBySlug() falls back to Moscow for unknown slugs so a garbage cookie
  // cannot break the public site. Here that would quietly label an event with
  // a city nobody assigned it — the gap is the honest answer.
  const city = CITIES.find((c) => c.slug === input.city)?.name ?? "";

  if (input.format === "online") {
    return { text: "Онлайн, без площадки", city, missing: false };
  }

  const parts = [input.venue?.name, input.venue?.address, input.venue?.metro]
    .map((p) => p?.trim())
    .filter((p): p is string => !!p);

  if (parts.length === 0) {
    return { text: "Площадка не указана", city, missing: true };
  }
  return { text: parts.join(" · "), city, missing: false };
}
