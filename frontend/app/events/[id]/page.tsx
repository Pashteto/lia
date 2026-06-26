import { ReportButton } from "@/components/ReportButton";
import { SignupCTA } from "@/components/SignupCTA";
import { VerifiedBadge } from "@/components/VerifiedBadge";
import { EventCover } from "@/components/ui/EventCover";
import { VenueMap } from "@/components/VenueMap";
import { fetchEvent } from "@/lib/api";
import {
  formatAttendance,
  formatEventDate,
  formatPrice,
} from "@/lib/format";
import { MOCK_EVENTS } from "@/lib/mock-events";
import type { LiaEvent } from "@/lib/types";
import Link from "next/link";
import { notFound } from "next/navigation";

// Event detail + registration — built from design/screens/event-detail.html.
// Detail/reading views use the plain `bg` base with grouped blocks on top
// (systemBackground vs systemGroupedBackground, per DESIGN.md).
//
// Fetches a single event from GET /api/v1/events/{id}; falls back to mock data
// when the backend is unreachable so the screen renders in frontend-only dev.
export default async function EventDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let event: LiaEvent | null;
  try {
    event = await fetchEvent(id);
  } catch {
    event = MOCK_EVENTS.find((e) => e.id === id) ?? null;
  }
  if (!event) {
    notFound();
  }

  const attendance = formatAttendance(event);
  const where =
    event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—");

  return (
    <div className="min-h-screen bg-bg pb-28">
      {/* Top nav — solid (opaque) bg, not translucent glass. */}
      <header className="sticky top-0 z-10 border-b border-separator bg-bg">
        <div className="mx-auto flex max-w-2xl items-center gap-3 px-5 py-3">
          <Link href="/" className="text-[17px] text-accent">
            ‹ События
          </Link>
        </div>
      </header>

      <article className="mx-auto max-w-2xl px-5">
        {/* Cover */}
        <EventCover
          event={event}
          aspect="aspect-[16/9]"
          rounded="rounded-card"
          sizes="(max-width: 768px) 100vw, 640px"
          priority
          className="mt-4"
        />

        {/* Title */}
        <div className="mt-5">
          {event.categories.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {event.categories.map((c) => (
                <span
                  key={c.id}
                  className="rounded-full bg-fill px-2.5 py-1 text-[12px] font-medium uppercase tracking-[0.03em] text-label-secondary"
                >
                  {c.label}
                </span>
              ))}
            </div>
          )}
          <h1 className="mt-2 text-[28px] font-bold leading-tight tracking-[-0.022em]">
            {event.title}
          </h1>
        </div>

        {/* Key facts */}
        <dl className="mt-5 grid grid-cols-2 gap-3">
          <Fact label="Когда" value={formatEventDate(event.startsAt)} />
          <Fact label="Где" value={where} />
          <Fact label="Участников" value={attendance ?? "—"} />
          <Fact label="Формат" value={event.format === "online" ? "Онлайн" : "Очно"} />
        </dl>

        {/* Sections */}
        {event.description && (
          <Section title="О встрече">
            <p className="text-[17px] leading-relaxed text-label">
              {event.description}
            </p>
          </Section>
        )}

        {event.organizer && (
          <Section title="Ведущий">
            <div className="flex items-center gap-3">
              <div className="size-11 shrink-0 rounded-full bg-fill" aria-hidden />
              <div>
                <p className="flex items-center gap-1.5 text-[17px] font-medium">
                  {event.organizer.name || "Организатор"}
                  {event.organizer.verified && (
                    <VerifiedBadge profileId={event.organizer.profile_id} />
                  )}
                </p>
                {event.organizer.affiliation && (
                  <p className="text-[13px] text-label-secondary">
                    {event.organizer.affiliation}
                  </p>
                )}
              </div>
            </div>
          </Section>
        )}

        {event.venue && (
          <Section title="Место">
            <p className="text-[17px] text-label">{event.venue.name}</p>
            {event.venue.metro && (
              <p className="text-[13px] text-label-secondary">
                м. {event.venue.metro}
              </p>
            )}
            {event.venue.lat != null && event.venue.lon != null && (
              <div className="mt-3">
                <VenueMap lat={event.venue.lat} lon={event.venue.lon} />
              </div>
            )}
          </Section>
        )}

        <div className="mt-6">
          <ReportButton eventId={event.id} />
        </div>
      </article>

      {/* Sticky registration bar — solid (opaque) bg, not translucent glass. */}
      <div className="fixed inset-x-0 bottom-0 z-10 border-t border-separator bg-bg">
        <div className="mx-auto flex max-w-2xl items-center justify-between gap-4 px-5 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
          <span className="text-[17px] font-semibold">{formatPrice(event)}</span>
          <SignupCTA event={event} />
        </div>
      </div>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-fact bg-bg-secondary p-4 shadow-card-subtle">
      <dt className="text-[12px] font-semibold uppercase tracking-[0.03em] text-label-secondary">
        {label}
      </dt>
      <dd className="mt-1 text-[15px] font-medium text-label">{value}</dd>
    </div>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mt-7">
      <h2 className="mb-2 text-[20px] font-bold tracking-[-0.01em]">{title}</h2>
      {children}
    </section>
  );
}
