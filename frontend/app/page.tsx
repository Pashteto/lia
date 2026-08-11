import { DiscoveryFeed } from "@/components/DiscoveryFeed";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { ssrFallbackEvents } from "@/lib/mock-events";

// U1 · Лента событий. SSR both the events and the ordered category taxonomy
// (numerals are positional in this list); the fallback serves mocks only in
// API-less local dev — in prod a failed fetch degrades to an empty list and
// the client-side query recovers.
export default async function DiscoveryPage() {
  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents().catch(() => ssrFallbackEvents()),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="МСК" />
      <DiscoveryFeed initialEvents={initialEvents} categories={categories} />
    </>
  );
}
