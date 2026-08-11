import { EventDetailView } from "@/components/EventDetailView";
import { OwnerDraftFallback } from "@/components/OwnerDraftFallback";
import { fetchEvent, getCategories } from "@/lib/api";
import { ssrFallbackEvent } from "@/lib/mock-events";
import type { LiaEvent } from "@/lib/types";

// Event detail + registration — built from design/screens/event-detail.html.
// Detail/reading views use the plain `bg` base with grouped blocks on top
// (systemBackground vs systemGroupedBackground, per DESIGN.md).
//
// Fetches a single event from GET /api/v1/events/{id}; falls back to mock data
// when the backend is unreachable so the screen renders in frontend-only dev.
//
// This server fetch is anonymous (the session token lives in localStorage, out
// of reach here), so the backend hides non-published events from it. When it
// misses, we hand off to <OwnerDraftFallback>, which retries with the caller's
// token so an owner can view their own draft (e.g. right after creating it);
// only if that also misses do we render the real 404.
export default async function EventDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let event: LiaEvent | null = null;
  try {
    event = await fetchEvent(id);
  } catch {
    event = ssrFallbackEvent(id);
  }

  if (event) {
    // The ordered taxonomy drives the positional category numeral on the
    // coverless hero plate. Best-effort: the page renders fine without it.
    const categories = await getCategories().catch(() => undefined);
    return <EventDetailView event={event} categories={categories} />;
  }

  return <OwnerDraftFallback id={id} />;
}
