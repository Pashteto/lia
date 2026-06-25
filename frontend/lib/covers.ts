import type { LiaEvent } from "@/lib/types";

// Visual identity for an event's cover when it has no uploaded photo.
//
// Resolution order (see <EventCover>):
//   1. event.coverUrl        — a real uploaded cover, always wins.
//   2. a curated photo        — only for the seeded demo events, matched by category.
//   3. a per-category gradient + line glyph — the genuine fallback for everything
//      else (e.g. events created in the app), so an imageless event still looks
//      designed rather than blank.

/** Diagonal gradient stops per category slug. Same category → same look. */
const CATEGORY_GRADIENT: Record<string, { from: string; to: string }> = {
  lecture: { from: "#2b3a8f", to: "#5b6fd6" },
  workshop: { from: "#b5552e", to: "#e0935f" },
  mediation: { from: "#1f6f6b", to: "#48b3a4" },
  concert: { from: "#5e2bb0", to: "#a85ad9" },
  exhibition: { from: "#9c7a1f", to: "#e0b24a" },
  performance: { from: "#9c1f3d", to: "#d64f6b" },
  film: { from: "#2a3340", to: "#52647a" },
  festival: { from: "#c0356e", to: "#f0883a" },
};

const DEFAULT_GRADIENT = { from: "#3a3a40", to: "#5e5e66" };

/**
 * Curated, license-checked photos per category — used to fill the seeded demo
 * events so the live demo doesn't show empty covers. Restricted to seed events
 * on purpose: real / user-created events get the gradient instead of a repeated
 * stock photo.
 *
 * Self-hosted under /public/covers (sourced from Unsplash, free license). The
 * box has no outbound egress to images.unsplash.com, so next/image's
 * server-side optimizer can't proxy remote covers there — local files are read
 * from disk and served same-origin, which works regardless of network.
 */
const CATEGORY_PHOTO: Record<string, string> = {
  lecture: "/covers/lecture.jpg",
  workshop: "/covers/workshop.jpg",
  mediation: "/covers/mediation.jpg",
  concert: "/covers/concert.jpg",
  exhibition: "/covers/exhibition.jpg",
  performance: "/covers/performance.jpg",
  film: "/covers/film.jpg",
  festival: "/covers/festival.jpg",
};

/** Seed events share the id prefix b0000000-…; only they get curated photos. */
const SEED_EVENT_PREFIX = "b0000000-0000-0000-0000-";

export function coverGradient(event: LiaEvent): { from: string; to: string } {
  const slug = event.categories[0]?.slug;
  return (slug && CATEGORY_GRADIENT[slug]) || DEFAULT_GRADIENT;
}

/** The photo to show over the gradient, or undefined to show the gradient alone. */
export function coverPhoto(event: LiaEvent): string | undefined {
  if (event.coverUrl) return event.coverUrl;
  if (!event.id.startsWith(SEED_EVENT_PREFIX)) return undefined;
  const slug = event.categories[0]?.slug;
  return slug ? CATEGORY_PHOTO[slug] : undefined;
}
