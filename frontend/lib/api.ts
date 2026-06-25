import { getToken } from "./auth";
import type {
  ApiEvent,
  EventFormat,
  EventStatus,
  LiaEvent,
  PriceType,
} from "./types";

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
    categories: (e.categories ?? []).map((c) => ({
      id: c.id,
      slug: c.slug,
      label: c.label,
    })),
    format: (e.format as EventFormat) ?? "offline",
    status: e.status,
    startsAt: e.starts_at,
    endsAt: e.ends_at,
    priceType: (e.price_type as PriceType) ?? "free",
    priceMin: e.price_min,
    organizer: e.organizer_id ? { id: e.organizer_id, name: "" } : undefined,
    venue: e.venue
      ? {
          id: e.venue.id,
          name: e.venue.name,
          metro: e.venue.metro,
          address: e.venue.address,
          district: e.venue.district,
          lat: e.venue.lat,
          lon: e.venue.lon,
        }
      : undefined,
    distanceM: e.distance_m,
    coverUrl: e.cover_url,
  };
}

/** A venue from the backend. */
export interface ApiVenue {
  id: string;
  name: string;
  address?: string;
  metro?: string;
  district?: string;
  lat?: number;
  lon?: number;
}

/** Searches venues by name substring. Throws on network/HTTP error. */
export async function searchVenues(q: string, limit = 20): Promise<ApiVenue[]> {
  const params = new URLSearchParams();
  if (q.trim()) params.set("q", q.trim());
  params.set("limit", String(limit));
  const res = await fetch(`${API_V1}/venues?${params.toString()}`);
  if (!res.ok) {
    throw new Error(`search venues failed: ${res.status}`);
  }
  return (await res.json()) as ApiVenue[];
}

/** Creates (find-or-create) a venue. Throws on network/HTTP error. */
export async function createVenue(input: {
  name: string;
  address?: string;
  metro?: string;
  district?: string;
  lat?: number;
  lon?: number;
}): Promise<ApiVenue> {
  const res = await fetch(`${API_V1}/venues`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`create venue failed: ${res.status} ${detail}`);
  }
  return (await res.json()) as ApiVenue;
}

/** Updates a venue via PATCH /venues/{id}. Throws on network/HTTP error. */
export async function updateVenue(
  id: string,
  input: { name?: string; address?: string; metro?: string; district?: string; lat?: number; lon?: number },
): Promise<ApiVenue> {
  const res = await fetch(`${API_V1}/venues/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`update venue failed: ${res.status} ${detail}`);
  }
  return (await res.json()) as ApiVenue;
}

/** A category from the curated taxonomy. */
export interface ApiCategory {
  id: string;
  slug: string;
  label: string;
}

/** Fetches the curated category taxonomy. Throws on network/HTTP error. */
export async function getCategories(): Promise<ApiCategory[]> {
  const res = await fetch(`${API_V1}/categories`, { next: { revalidate: 300 } });
  if (!res.ok) {
    throw new Error(`fetch categories failed: ${res.status}`);
  }
  return (await res.json()) as ApiCategory[];
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
 * Fetches events near a given coordinate via `GET /events/nearby`.
 * The backend returns events within 50 km, pre-sorted nearest-first, each with
 * `distance_m`. Events without a venue / coordinates are excluded server-side.
 */
export async function fetchNearbyEvents(
  lat: number,
  lon: number,
  limit = 50,
): Promise<LiaEvent[]> {
  const params = new URLSearchParams({
    lat: String(lat),
    lon: String(lon),
    limit: String(limit),
  });
  const res = await fetch(`${API_V1}/events/nearby?${params.toString()}`);
  if (!res.ok) throw new Error(`fetch nearby failed: ${res.status}`);
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

/** Payload for creating an event (mirrors the backend EventInput). */
export interface CreateEventInput {
  title: string;
  description?: string;
  category_ids?: string[];
  venue_id?: string;
  status?: EventStatus;
  format?: EventFormat;
  price_type?: PriceType;
  price_min?: number;
  starts_at: string; // ISO 8601
  ends_at?: string;
  cover_file_id?: string;
}

/**
 * Creates an event via POST /events; returns the created event.
 * Requires authentication — attaches the demo-login bearer token. The backend
 * sets the organizer to the authenticated principal, so no organizer_id is sent.
 */
export async function createEvent(input: CreateEventInput): Promise<LiaEvent> {
  const token = getToken();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API_V1}/events`, {
    method: "POST",
    headers,
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`create event failed: ${res.status} ${detail}`);
  }
  return apiEventToLia((await res.json()) as ApiEvent);
}

/**
 * Uploads a file via POST /api/v1/uploads.
 * Returns the file id (uuid) and its publicly fetchable URL.
 * Requires authentication — attaches the demo-login bearer token.
 */
export async function uploadFile(file: File): Promise<{ id: string; url: string }> {
  const token = getToken();
  const fd = new FormData();
  fd.append("file", file);
  const res = await fetch(`${API_V1}/uploads`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: fd,
  });
  if (!res.ok) throw new Error(`upload failed: ${res.status} ${await res.text().catch(() => "")}`);
  return (await res.json()) as { id: string; url: string };
}

/** Response from POST /auth/demo-login. */
interface DemoLoginResponse {
  token: string;
}

/**
 * DEMO-ONLY login: mints a GateGuard session token for an email (no password).
 * Returns the bearer token; callers persist it via lib/auth.setSession.
 */
export async function demoLogin(email: string, name?: string): Promise<string> {
  const res = await fetch(`${API_V1}/auth/demo-login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, name: name || email.split("@")[0] }),
  });
  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(`login failed: ${res.status} ${detail}`);
  }
  const data = (await res.json()) as DemoLoginResponse;
  if (!data.token) throw new Error("login failed: empty token");
  return data.token;
}
