"use client";

import { useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useMemo } from "react";

import { CITY_COOKIE, cityBySlug, CITIES, type City, type CitySlug } from "@/lib/city";

/** Availability by slug, as reported by GET /cities (msk is always true). */
export type CityAvailability = Partial<Record<CitySlug, boolean>>;

interface CityContextValue {
  /** The visitor's current city (cookie-resolved server-side per request). */
  city: City;
  /** Live availability; falls back to the static flags in lib/city. */
  available: (slug: CitySlug) => boolean;
  /** Persist a new city choice (cookie, 1 year) and re-render server data. */
  setCity: (slug: CitySlug) => void;
}

const CityContext = createContext<CityContextValue | null>(null);

/** Writes the city cookie. Kept module-level so tests can spy the format. */
export function writeCityCookie(slug: CitySlug): void {
  document.cookie = `${CITY_COOKIE}=${slug};path=/;max-age=31536000;samesite=lax`;
}

export function CityProvider({
  initialSlug,
  availability,
  children,
}: {
  initialSlug: string | undefined;
  /** null = /cities fetch failed; static fallback applies. */
  availability: CityAvailability | null;
  children: React.ReactNode;
}) {
  const router = useRouter();

  const city = cityBySlug(initialSlug);

  const available = useCallback(
    (slug: CitySlug) =>
      availability?.[slug] ?? CITIES.find((c) => c.slug === slug)?.available ?? false,
    [availability],
  );

  const setCity = useCallback(
    (slug: CitySlug) => {
      writeCityCookie(slug);
      // A ?city= override in the URL would beat the fresh cookie — strip it,
      // otherwise just re-render server components against the new cookie.
      // window is read inside the callback (client-only), not during render.
      const params = new URLSearchParams(window.location.search);
      if (params.get("city")) {
        params.delete("city");
        const qs = params.toString();
        router.replace(qs ? `${window.location.pathname}?${qs}` : window.location.pathname);
      }
      router.refresh();
    },
    [router],
  );

  const value = useMemo(() => ({ city, available, setCity }), [city, available, setCity]);
  return <CityContext.Provider value={value}>{children}</CityContext.Provider>;
}

/** Falls back to a static msk-only view when no provider is mounted (tests). */
export function useCity(): CityContextValue {
  const ctx = useContext(CityContext);
  if (ctx) return ctx;
  return {
    city: CITIES[0],
    available: (slug) => CITIES.find((c) => c.slug === slug)?.available ?? false,
    setCity: () => {},
  };
}
