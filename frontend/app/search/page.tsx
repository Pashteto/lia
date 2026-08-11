import { DiscoverBrowse } from "@/components/DiscoverBrowse";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { fetchPublishedEvents, getCategories } from "@/lib/api";
import { ssrFallbackEvents } from "@/lib/mock-events";

export const metadata = { title: "Подбор — PRESENCE" };

// U3 · AI-подбор. Public route — deterministic smart-filter (LLM deferred).
export default async function SearchPage() {
  const [initialEvents, categories] = await Promise.all([
    fetchPublishedEvents().catch(() => ssrFallbackEvents()),
    getCategories().catch(() => []),
  ]);

  return (
    <>
      <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="ПОДБОР" />
      <DiscoverBrowse initialEvents={initialEvents} categories={categories} />
    </>
  );
}
