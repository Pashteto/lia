import { cookies } from "next/headers";

import { DiscoverBrowse } from "@/components/DiscoverBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { CityCookieSync } from "@/components/ui/CityCookieSync";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { CITIES, CITY_COOKIE, cityBySlug } from "@/lib/city";
import { ssrFallbackEvents } from "@/lib/mock-events";

export const metadata = { title: "Подбор — PRESENCE" };

// U3 · AI-подбор. Public route — deterministic smart-filter (LLM deferred).
// City resolution mirrors the feed: cookie, with a shareable ?city= override.
export default async function SearchPage({
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
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="ПОДБОР" />
      <DiscoverBrowse initialEvents={initialEvents} categories={categories} city={city} />
    </>
  );
}
