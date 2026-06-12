import { Button } from "@/components/ui/Button";
import { Kicker } from "@/components/ui/Kicker";
import { fetchEvent } from "@/lib/api";
import {
  formatAttendance,
  formatEventDate,
  formatPrice,
} from "@/lib/format";
import { MOCK_EVENTS } from "@/lib/mock-events";
import type { LiaEvent } from "@/lib/types";
import Image from "next/image";
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
      {/* Glass nav */}
      <header className="glass sticky top-0 z-10 border-b border-separator">
        <div className="mx-auto flex max-w-2xl items-center gap-3 px-5 py-3">
          <Link href="/" className="text-[17px] text-accent">
            ‹ События
          </Link>
        </div>
      </header>

      <article className="mx-auto max-w-2xl px-5">
        {/* Cover */}
        <div className="relative mt-4 aspect-[16/9] w-full overflow-hidden rounded-card bg-fill">
          {event.coverUrl && (
            <Image
              src={event.coverUrl}
              alt=""
              fill
              sizes="(max-width: 768px) 100vw, 640px"
              className="object-cover"
            />
          )}
        </div>

        {/* Title */}
        <div className="mt-5">
          {event.category && <Kicker>{event.category.label}</Kicker>}
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
                <p className="text-[17px] font-medium">
                  {event.organizer.name || "Организатор"}
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
          </Section>
        )}
      </article>

      {/* Sticky registration bar */}
      <div className="glass fixed inset-x-0 bottom-0 z-10 border-t border-separator">
        <div className="mx-auto flex max-w-2xl items-center justify-between gap-4 px-5 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
          <span className="text-[17px] font-semibold">{formatPrice(event)}</span>
          {/* TODO: wire RSVP (POST /events/{id}/rsvp) — needs auth + rsvp module */}
          <Button variant="filled" className="px-8">
            Записаться
          </Button>
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
