import { cookies } from "next/headers";

import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { CityControl } from "@/components/ui/CityControl";
import { CityCookieSync } from "@/components/ui/CityCookieSync";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { CITIES, CITY_COOKIE, cityBySlug } from "@/lib/city";
import { ssrFallbackEvents } from "@/lib/mock-events";

// U1 · Лента событий. SSR both the events and the ordered category taxonomy
// (numerals are positional in this list); the fallback serves mocks only in
// API-less local dev — in prod a failed fetch degrades to an empty list and
// the client-side query recovers.
//
// City: cookie-driven, with a shareable ?city= override that then persists
// itself into the cookie (CityCookieSync).
export default async function DiscoveryPage({
  searchParams,
}: {
  searchParams: Promise<{ city?: string }>;
}) {
  const { city: cityParam } = await searchParams;
  const cookieSlug = (await cookies()).get(CITY_COOKIE)?.value;
  const override = CITIES.find((c) => c.slug === cityParam);
  const city = override ?? cityBySlug(cookieSlug);

  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents(undefined, undefined, city.slug).catch(() => ssrFallbackEvents()),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      {override ? <CityCookieSync slug={override.slug} /> : null}
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption={<CityControl />} />
      <DiscoveryFeed initialEvents={initialEvents} categories={categories} city={city} />
    </>
  );
}
