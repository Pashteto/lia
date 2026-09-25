import { cookies } from "next/headers";

import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { IntroSplash } from "@/components/IntroSplash";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { CityControl } from "@/components/ui/CityControl";
import { CityCookieSync } from "@/components/ui/CityCookieSync";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { CITIES, CITY_COOKIE, cityBySlug } from "@/lib/city";
import { INTRO_INLINE_SCRIPT } from "@/lib/intro-splash";
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
  // The splash sign only names a city once the page knows it for sure (a
  // city cookie, or a valid ?city= override) — the geo default resolves
  // client-side after paint, so guessing here would show the wrong city on
  // many first visits. The year is computed server-side to avoid hydration
  // drift with a client Date().
  const year = new Date().getFullYear();
  const cityKnown = Boolean(override) || CITIES.some((c) => c.slug === cookieSlug);
  const sign = cityKnown ? `${city.name.toUpperCase()} · ${year}` : String(year);

  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents(undefined, undefined, city.slug).catch(() => ssrFallbackEvents()),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      {/* First-visit splash. The pre-paint script sets html[data-intro]; it
          lives here (server component) rather than inside the client
          component so React never renders a <script> on the client. */}
      <script dangerouslySetInnerHTML={{ __html: INTRO_INLINE_SCRIPT }} />
      <IntroSplash sign={sign} />
      {override ? <CityCookieSync slug={override.slug} /> : null}
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption={<CityControl />} />
      <div className="intro-content">
        <DiscoveryFeed initialEvents={initialEvents} categories={categories} city={city} />
      </div>
    </>
  );
}
