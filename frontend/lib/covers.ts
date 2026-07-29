import type { LiaEvent } from "@/lib/types";

/**
 * Cover photo resolution for an event (Swiss Grid — no gradients).
 *
 *   1. event.coverUrl — uploaded cover, always wins.
 *   2. curated seed photo — only for seeded demo events (id prefix b0000000-…),
 *      so the live demo isn't blank; user-created events stay paper until upload.
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

const SEED_EVENT_PREFIX = "b0000000-0000-0000-0000-";

/** The photo to show, or undefined for a blank paper cover. */
export function coverPhoto(event: LiaEvent): string | undefined {
  if (event.coverUrl) return event.coverUrl;
  if (!event.id.startsWith(SEED_EVENT_PREFIX)) return undefined;
  const slug = event.categories[0]?.slug;
  return slug ? CATEGORY_PHOTO[slug] : undefined;
}
