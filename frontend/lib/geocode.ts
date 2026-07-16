import { API_BASE } from "@/lib/api";
import { getToken } from "@/lib/auth";

export interface GeoResult {
  lat: number;
  lon: number;
  label: string;
}

// Forward geocoding via the auth-gated backend Yandex proxy.
// The backend never exposes the geocoder key to the browser.
export async function geocodeAddress(q: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const token = getToken();
  if (!token) throw new Error("not authenticated");
  const res = await fetch(
    `${API_BASE}/api/v1/geocode?q=${encodeURIComponent(query)}`,
    {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    },
  );
  if (!res.ok) throw new Error(`geocode failed: ${res.status}`);
  return (await res.json()) as GeoResult[];
}
