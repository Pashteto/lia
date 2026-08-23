import { cookies } from "next/headers";

import { MapBrowse } from "@/components/MapBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { CityCookieSync } from "@/components/ui/CityCookieSync";
import { CITIES, CITY_COOKIE, cityBySlug } from "@/lib/city";

export const metadata = { title: "Карта — PRESENCE" };

// U5 · Карта. Public route — no auth gate. City (cookie / ?city= override)
// sets the initial map center; the search itself is coordinate-driven.
export default async function MapPage({
  searchParams,
}: {
  searchParams: Promise<{ city?: string }>;
}) {
  const { city: cityParam } = await searchParams;
  const cookieSlug = (await cookies()).get(CITY_COOKIE)?.value;
  const override = CITIES.find((c) => c.slug === cityParam);
  const city = override ?? cityBySlug(cookieSlug);

  return (
    <>
      {override ? <CityCookieSync slug={override.slug} /> : null}
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="КАРТА" />
      <MapBrowse city={city} />
    </>
  );
}
