"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { fetchCitiesFresh } from "@/lib/api";
import { CITIES, CITY_COOKIE } from "@/lib/city";
import { writeCityCookie } from "@/lib/city-context";

/** First-visit geo default: when the visitor has NO city cookie, ask the
 * backend which city their IP is in (GeoLite2 `suggested` on GET /cities) and
 * persist it. MUST run from the browser — a server-side fetch would geolocate
 * the box's IP, not the visitor's. Runs at most once per visitor: whatever we
 * decide (including the msk fallback) is written to the cookie, and a visitor
 * with a cookie skips the lookup entirely. An explicit later choice via the
 * header switcher simply overwrites the same cookie. */
export function CityGeoDefault() {
  const router = useRouter();
  useEffect(() => {
    const hasCookie = document.cookie
      .split("; ")
      .some((c) => c.startsWith(`${CITY_COOKIE}=`));
    if (hasCookie) return;
    let live = true;
    fetchCitiesFresh()
      .then((list) => {
        if (!live) return;
        const suggested = list.find((c) => c.suggested && c.available)?.code;
        const city = CITIES.find((c) => c.slug === suggested) ?? CITIES[0];
        writeCityCookie(city.slug);
        // Only re-render when the guess differs from what already rendered
        // (the default city) — otherwise the refresh is a no-op flash.
        if (city.slug !== CITIES[0].slug) router.refresh();
      })
      .catch(() => {
        // Geo is best-effort; the visitor stays on the default city.
      });
    return () => {
      live = false;
    };
  }, [router]);
  return null;
}
