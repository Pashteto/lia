import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { MOCK_EVENTS } from "@/lib/mock-events";

// U1 · Лента событий. SSR both the events and the ordered category taxonomy
// (numerals are positional in this list); mock fallback keeps local dev alive.
export default async function DiscoveryPage() {
  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents().catch(() => MOCK_EVENTS),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="МСК" />
      <DiscoveryFeed initialEvents={initialEvents} categories={categories} />
    </>
  );
}
