import { EventFeedbackSection } from "@/components/EventFeedbackSection";
import { ReportButton } from "@/components/ReportButton";
import { SignupCTA } from "@/components/SignupCTA";
import { VerifiedBadge } from "@/components/VerifiedBadge";
import { EventCover } from "@/components/ui/EventCover";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { VenueMap } from "@/components/VenueMap";
import {
  attendanceShort,
  formatEventRange,
  formatStartTime,
} from "@/lib/format";
import { categoryNumeral } from "@/lib/category-numerals";
import { coverPhoto } from "@/lib/covers";
import { priceLabel } from "@/lib/price-label";
import type { LiaEvent } from "@/lib/types";
import Link from "next/link";

/** U2 · Страница события. Paper, hairline blocks, price+CTA rail on desktop,
 * sticky price+CTA footer on mobile. */
export function EventDetailView({
  event,
  categories,
}: {
  event: LiaEvent;
  /** Ordered taxonomy, for the positional category numeral. Optional: the
   * owner-draft path has no server fetch to piggyback on, and the plate still
   * reads without it. */
  categories?: ReadonlyArray<{ slug: string }>;
}) {
  const ended = new Date(event.endsAt ?? event.startsAt) < new Date();
  const cancelled = event.status === "cancelled";
  const price = priceLabel(event.priceMin, event.priceType);
  const cat = event.categories[0];
  const numeral = cat && categories ? categoryNumeral(cat.slug, categories) : "—";
  const routeHref =
    event.venue?.lat != null && event.venue?.lon != null
      ? `https://yandex.ru/maps/?rtext=~${event.venue.lat},${event.venue.lon}`
      : null;

  return (
    <div className="mx-auto min-h-screen max-w-[1360px] pb-[150px] md:pb-[40px]">
      {/* Breadcrumb header */}
      <header className="flex items-baseline justify-between border-b border-ink px-[20px] py-[13px]">
        <Link
          href="/"
          className="swiss-focus font-alt text-[9px] uppercase tracking-[0.14em]"
        >
          ← События
        </Link>
        <span className="cap">{cat?.label ?? "Событие"}</span>
      </header>

      {/* Cover strip (single cover — we have one image per event) */}
      <div className="border-b border-ink">
        <EventCover
          src={coverPhoto(event)}
          aspect="aspect-[3/1] max-md:aspect-[3/2]"
          sizes="(max-width: 768px) 100vw, 1360px"
          priority
          // Without this the coverless event rendered ~430px of empty paper.
          // The feed already answers this case with a numeral plate; the
          // detail hero uses the same one at hero scale.
          fallback={
            <div className="flex h-full items-end justify-between px-[20px] py-[16px]">
              <span className="font-mono text-[64px] font-bold leading-none tracking-[-0.02em] text-ink max-md:text-[44px]">
                {numeral}
              </span>
              <span className="cap">{cat?.label ?? "Событие"}</span>
            </div>
          }
        />
      </div>

      {/* Cancelled events stay reachable for people who already registered —
       * the backend keeps serving them the page — so the cancellation has to be
       * the loudest thing on it, above the title. */}
      {cancelled && (
        <div className="border-b border-ink bg-signal px-[20px] py-[12px]">
          <p className="text-[16px] font-black uppercase tracking-[0.08em] text-white max-md:text-[13px]">
            Событие отменено
          </p>
          <p className="mt-[2px] text-[12px] text-white/90">
            Организатор отменил это событие. Ваша запись больше не действует.
          </p>
        </div>
      )}

      {/* Title block: text left, price+CTA rail right */}
      <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
        <div className="flex flex-col gap-[10px] px-[20px] py-[16px]">
          <p className="cap">
            {[cat?.label, event.format === "online" ? "Онлайн" : "Очно"]
              .filter(Boolean)
              .join(" · ")}
          </p>
          <h1 className="max-w-[22ch] text-[30px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[21px]">
            {event.title}
          </h1>
          {event.description && (
            <p className="max-w-[52ch] text-[12px] leading-[1.45] text-text-dim">
              {event.description}
            </p>
          )}
        </div>
        <div className="flex flex-col border-l border-rule-inner px-[14px] py-[10px] max-md:hidden">
          <span className="cap">Цена</span>
          <span className="mt-[4px] font-mono text-[26px] font-bold leading-none">
            {price}
          </span>
          {!ended && !cancelled && (
            <div className="mt-auto pt-[10px]">
              <SignupCTA event={event} />
            </div>
          )}
        </div>
      </div>

      {/* Fact strip */}
      <CellStrip cols={4} className="max-md:grid-cols-2!">
        <Cell caption="Когда" value={formatEventRange(event)} />
        <Cell caption="Начало" value={formatStartTime(event.startsAt)} mono />
        <Cell caption="Места" value={attendanceShort(event)} mono />
        <Cell
          caption="Организатор"
          value={
            event.organizer ? (
              event.organizer.profile_id ? (
                <Link
                  href={`/organizers/${event.organizer.profile_id}`}
                  className="swiss-focus underline-offset-2 hover:underline"
                >
                  {event.organizer.name || "Организатор"}
                  {event.organizer.verified ? (
                    <>
                      {" "}
                      <VerifiedBadge />
                    </>
                  ) : null}
                </Link>
              ) : (
                <>
                  {event.organizer.name || "Организатор"}
                  {event.organizer.verified ? (
                    <>
                      {" "}
                      <VerifiedBadge />
                    </>
                  ) : null}
                </>
              )
            ) : (
              "—"
            )
          }
        />
      </CellStrip>

      {/* Venue block: map left, address rail right */}
      {event.venue && (
        <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
          <div className="p-[14px]">
            {event.venue.lat != null && event.venue.lon != null ? (
              <VenueMap
                lat={event.venue.lat}
                lon={event.venue.lon}
                numeral={numeral === "—" ? undefined : numeral}
              />
            ) : (
              <p className="text-[12px] text-text-dim">{event.venue.name}</p>
            )}
          </div>
          <div className="flex flex-col border-l border-rule-inner px-[14px] py-[10px]">
            <span className="cap">Адрес</span>
            <span className="mt-[4px] text-[12px] font-bold leading-[1.25]">
              {event.venue.name}
            </span>
            {event.venue.address && (
              <span className="mt-[2px] text-[11px] text-field-text">
                {event.venue.address}
              </span>
            )}
            {event.venue.metro && (
              <span className="mt-[2px] text-[11px] text-field-text">
                м. {event.venue.metro}
              </span>
            )}
            {routeHref && (
              <a
                href={routeHref}
                target="_blank"
                rel="noopener noreferrer"
                className="swiss-focus hover-invert mt-auto border border-ink px-[11px] py-[7px] text-center text-[9px] font-bold uppercase tracking-[0.07em]"
              >
                Маршрут
              </a>
            )}
          </div>
        </div>
      )}

      {/* Description / feedback / report */}
      {ended && <EventFeedbackSection event={event} />}
      <div className="px-[20px] py-[16px]">
        <ReportButton eventId={event.id} />
      </div>

      {/* Mobile sticky footer: price + CTA */}
      {!ended && !cancelled && (
        <div className="fixed inset-x-0 bottom-[49px] z-20 border-t border-ink bg-paper md:hidden">
          <div className="flex items-center justify-between gap-[10px] px-[14px] py-[9px] pb-[calc(9px+env(safe-area-inset-bottom))]">
            <span className="font-mono text-[17px] font-bold">{price}</span>
            <SignupCTA event={event} />
          </div>
        </div>
      )}
    </div>
  );
}
