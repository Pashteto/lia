const NOMINATIM = "https://nominatim.openstreetmap.org/search";

export interface GeoResult {
  lat: number;
  lon: number;
  label: string;
}

// Browser-side forward geocoding via OSM Nominatim. Backend never geocodes.
// Honors Nominatim usage policy: identify the app, request JSON, cap results.
export async function geocodeAddress(q: string): Promise<GeoResult[]> {
  const query = q.trim();
  if (query === "") return [];
  const params = new URLSearchParams({
    q: query,
    format: "jsonv2",
    limit: "5",
    "accept-language": "ru",
  });
  const res = await fetch(`${NOMINATIM}?${params.toString()}`, {
    headers: { "Accept": "application/json" },
  });
  if (!res.ok) throw new Error(`geocode failed: ${res.status}`);
  const data = (await res.json()) as { lat: string; lon: string; display_name: string }[];
  return data.map((d) => ({
    lat: parseFloat(d.lat),
    lon: parseFloat(d.lon),
    label: d.display_name,
  }));
}
