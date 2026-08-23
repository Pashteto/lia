import { API_BASE } from "@/lib/api";
import { getToken } from "@/lib/auth";

export interface GeoResult {
  lat: number;
  lon: number;
  label: string;
}

// Forward geocoding via the auth-gated backend Yandex proxy.
// The backend never exposes the geocoder key to the browser.
export async function geocodeAddress(q: string, city?: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const cityParam = city ? `&city=${encodeURIComponent(city)}` : "";
  const res = await fetch(
    `${API_BASE}/api/v1/geocode?q=${encodeURIComponent(query)}${cityParam}`,
    {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    },
  );
  if (!res.ok) throw new Error(`geocode failed: ${res.status}`);
  return (await res.json()) as GeoResult[];
}

/** Venue/organization NAME search via the auth-gated backend Yandex Places proxy. */
export async function searchPlaces(q: string, city?: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const cityParam = city ? `&city=${encodeURIComponent(city)}` : "";
  const res = await fetch(`${API_BASE}/api/v1/places?q=${encodeURIComponent(query)}${cityParam}`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`places failed: ${res.status}`);
  return (await res.json()) as GeoResult[];
}

/** Outcome of the paired name + address lookup behind the venue search box. */
export interface VenueLookup {
  /** Name hits first, then address hits; duplicate labels dropped. */
  results: GeoResult[];
  /** The Places proxy errored — name search is not running at all. */
  nameSearchDown: boolean;
  /** The geocoder proxy errored — address search is not running at all. */
  addressSearchDown: boolean;
}

/**
 * Merges the two settled lookups and says which of them is down.
 *
 * Both run in parallel and either can fail on its own (`/places` 503s whenever
 * YANDEX_PLACES_KEY is unprovisioned). Silently keeping whatever came back made
 * a dead name search indistinguishable from one that found nothing, so the
 * caller has to be told.
 */
export function mergeVenueLookups(
  places: PromiseSettledResult<GeoResult[]>,
  addresses: PromiseSettledResult<GeoResult[]>,
): VenueLookup {
  const p = places.status === "fulfilled" ? places.value : [];
  const a = addresses.status === "fulfilled" ? addresses.value : [];
  const seen = new Set<string>();
  const results = [...p, ...a].filter((r) => {
    if (seen.has(r.label)) return false;
    seen.add(r.label);
    return true;
  });
  return {
    results,
    nameSearchDown: places.status === "rejected",
    addressSearchDown: addresses.status === "rejected",
  };
}
