import { FeedbackForm } from "@/components/FeedbackForm";
import { ReportButton } from "@/components/ReportButton";
import { SignupCTA } from "@/components/SignupCTA";
import { VerifiedBadge } from "@/components/VerifiedBadge";
import { EventCover } from "@/components/ui/EventCover";
import { VenueMap } from "@/components/VenueMap";
import {
  formatAttendance,
  formatEventDate,
  formatPrice,
} from "@/lib/format";
import type { LiaEvent } from "@/lib/types";
import Link from "next/link";

// Presentational event-detail view — the markup extracted from
// app/events/[id]/page.tsx so it can render from both the server page (anonymous
// SSR fetch, common published case) and the client-side owner-draft fallback.
// Uses only client-safe children, so it works in either context.
export function EventDetailView({ event }: { event: LiaEvent }) {
  const attendance = formatAttendance(event);
  const where =
    event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—");
  const ended = new Date(event.endsAt ?? event.startsAt) < new Date();

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
            {event.organizer.profile_id ? (
              <Link
                href={`/organizers/${event.organizer.profile_id}`}
                className="flex items-center gap-3 transition hover:opacity-70"
              >
                <div className="size-11 shrink-0 rounded-full bg-fill" aria-hidden />
                <div>
                  <p className="flex items-center gap-1.5 text-[17px] font-medium">
                    {event.organizer.name || "Организатор"}
                    {event.organizer.verified && <VerifiedBadge />}
                  </p>
                  {event.organizer.affiliation && (
                    <p className="text-[13px] text-label-secondary">
                      {event.organizer.affiliation}
                    </p>
                  )}
                </div>
              </Link>
            ) : (
              <div className="flex items-center gap-3">
                <div className="size-11 shrink-0 rounded-full bg-fill" aria-hidden />
                <div>
                  <p className="flex items-center gap-1.5 text-[17px] font-medium">
                    {event.organizer.name || "Организатор"}
                    {event.organizer.verified && <VerifiedBadge />}
                  </p>
                  {event.organizer.affiliation && (
                    <p className="text-[13px] text-label-secondary">
                      {event.organizer.affiliation}
                    </p>
                  )}
                </div>
              </div>
            )}
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

        {ended && (
          <Section title="Отзыв">
            <FeedbackForm eventId={event.id} />
          </Section>
        )}

        <div className="mt-6">
          <ReportButton eventId={event.id} />
        </div>
      </article>

      {/* Sticky registration bar — solid (opaque) bg, not translucent glass.
          Once the event has ended, the RSVP CTA is replaced by the feedback
          block above, so the bar is dropped entirely. */}
      {!ended && (
        <div className="fixed inset-x-0 bottom-0 z-10 border-t border-separator bg-bg">
          <div className="mx-auto flex max-w-2xl items-center justify-between gap-4 px-5 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
            <span className="text-[17px] font-semibold">{formatPrice(event)}</span>
            <SignupCTA event={event} />
          </div>
        </div>
      )}
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
