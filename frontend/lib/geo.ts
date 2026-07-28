/** [latitude, longitude] — the argument order Yandex Maps 2.1 uses. */
export type LatLon = [number, number];
/** [[latSW, lonSW], [latNE, lonNE]] — the shape of ymaps.Map#getBounds(). */
export type MapBounds = [LatLon, LatLon];

const EARTH_RADIUS_KM = 6371;
const rad = (deg: number) => (deg * Math.PI) / 180;

/** Great-circle distance in kilometres. */
export function haversineKm(a: LatLon, b: LatLon): number {
  const dLat = rad(b[0] - a[0]);
  const dLon = rad(b[1] - a[1]);
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(rad(a[0])) * Math.cos(rad(b[0])) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** The radius a "search this area" at the viewport centre would need to cover
 * the visible corners. Reported in the U5 «Радиус» cell. */
export function radiusKmFromBounds(bounds: MapBounds): number {
  const [[latSW, lonSW], [latNE, lonNE]] = bounds;
  const centre: LatLon = [(latSW + latNE) / 2, (lonSW + lonNE) / 2];
  return haversineKm(centre, [latNE, lonNE]);
}

/** "12 км" | "5.0 км" | "—". Coarse above 10 km, precise below it. */
export function radiusLabel(km: number | null | undefined): string {
  if (km == null || !Number.isFinite(km)) return "—";
  return km >= 10 ? `${Math.round(km)} км` : `${km.toFixed(1)} км`;
}

/** "0.8 км" for a nearby-result distance; null when the backend sent none. */
export function distanceLabel(distanceM: number | null | undefined): string | null {
  if (distanceM == null || !Number.isFinite(distanceM)) return null;
  return `${(distanceM / 1000).toFixed(1)} км`;
}
