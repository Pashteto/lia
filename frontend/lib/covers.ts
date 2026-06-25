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
 * Curated, license-checked Unsplash photos per category — used to fill the
 * seeded demo events so the live demo doesn't show empty covers. Restricted to
 * seed events on purpose: real / user-created events get the gradient instead of
 * a repeated stock photo. Hosts are allow-listed in next.config.ts.
 */
const CATEGORY_PHOTO: Record<string, string> = {
  lecture:
    "https://images.unsplash.com/photo-1747674148491-51f8a5c723db?w=1200&q=70&auto=format&fit=crop",
  workshop:
    "https://images.unsplash.com/photo-1753164725860-ffcd260b7b32?w=1200&q=70&auto=format&fit=crop",
  mediation:
    "https://images.unsplash.com/photo-1637578035851-c5b169722de1?w=1200&q=70&auto=format&fit=crop",
  concert:
    "https://images.unsplash.com/photo-1745328597533-3df3e5db2dec?w=1200&q=70&auto=format&fit=crop",
  exhibition:
    "https://images.unsplash.com/photo-1740598307395-3ccc0ec28a28?w=1200&q=70&auto=format&fit=crop",
  performance:
    "https://images.unsplash.com/photo-1576514129883-2f1d47a65da6?w=1200&q=70&auto=format&fit=crop",
  film: "https://images.unsplash.com/photo-1717915604557-94283edbcc1b?w=1200&q=70&auto=format&fit=crop",
  festival:
    "https://images.unsplash.com/photo-1489100517551-92a468b736f0?w=1200&q=70&auto=format&fit=crop",
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
