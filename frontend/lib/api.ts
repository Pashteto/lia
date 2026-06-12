import type { ApiEvent, EventFormat, LiaEvent, PriceType } from "./types";

// Base URL of the Go backend. Overridable via env; defaults to local compose.
// NEXT_PUBLIC_ prefix so the value is available in both server and client code.
export const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const API_V1 = `${API_BASE}/api/v1`;

/** Maps a backend Event to the frontend LiaEvent shape. */
export function apiEventToLia(e: ApiEvent): LiaEvent {
  return {
    id: e.id,
    title: e.title,
    description: e.description,
    category: e.category ? { slug: e.category, label: e.category } : undefined,
    format: (e.format as EventFormat) ?? "offline",
    status: e.status,
    startsAt: e.starts_at,
    endsAt: e.ends_at,
    priceType: (e.price_type as PriceType) ?? "free",
    priceMin: e.price_min,
    organizer: e.organizer_id ? { id: e.organizer_id, name: "" } : undefined,
    venue: e.venue_name
      ? { id: e.venue_id ?? "", name: e.venue_name, metro: e.venue_metro }
      : undefined,
    // cover image is not yet provided by the backend.
  };
}

/**
 * Fetches published events from the backend. Works on both server (SSR) and
 * client. Throws on network/HTTP error so callers can decide how to degrade.
 */
export async function fetchPublishedEvents(): Promise<LiaEvent[]> {
  const res = await fetch(`${API_V1}/events?status=published`, {
    // Revalidate every 30s on the server; always fresh enough for discovery.
    next: { revalidate: 30 },
  });
  if (!res.ok) {
    throw new Error(`fetch events failed: ${res.status}`);
  }
  const data = (await res.json()) as ApiEvent[];
  return data.map(apiEventToLia);
}

/**
 * Fetches a single event by id. Returns null on 404; throws on other errors.
 */
export async function fetchEvent(id: string): Promise<LiaEvent | null> {
  const res = await fetch(`${API_V1}/events/${id}`, { next: { revalidate: 30 } });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error(`fetch event failed: ${res.status}`);
  }
  return apiEventToLia((await res.json()) as ApiEvent);
}
