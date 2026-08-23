import Link from "next/link";

/** «Организую» block on the /me hub (QA-23-aug №11): on mobile the organizer
 * cabinet was unreachable — the bottom nav is user-layer only and /organizer
 * lived behind a desktop-only menu item. Presentational; MeProfile feeds it. */
export function OrganizerSection({
  hasOrganizer,
  eventsCount,
  pendingApplications,
}: {
  hasOrganizer: boolean;
  eventsCount: number;
  pendingApplications: number;
}) {
  return (
    <section className="border-b border-ink">
      <p className="cap border-b border-on-surface px-[20px] py-[9px] max-sm:px-[14px]">
        Организую
      </p>
      {hasOrganizer ? (
        <div className="grid grid-cols-3 border-b border-on-surface [&>*+*]:border-l [&>*+*]:border-on-surface max-sm:grid-cols-1 max-sm:[&>*+*]:border-l-0 max-sm:[&>*+*]:border-t">
          <Link
            href="/organizer"
            className="swiss-focus hover-invert px-[20px] py-[12px] text-[12px] font-bold uppercase tracking-[0.06em] max-sm:px-[14px]"
          >
            Кабинет →
          </Link>
          <Link
            href="/events/mine"
            className="swiss-focus hover-invert px-[20px] py-[12px] text-[12px] font-bold uppercase tracking-[0.06em] max-sm:px-[14px]"
          >
            Мои события · <span className="font-mono">{eventsCount}</span>
          </Link>
          <Link
            href="/organizer/applications"
            className={`swiss-focus hover-invert px-[20px] py-[12px] text-[12px] font-bold uppercase tracking-[0.06em] max-sm:px-[14px] ${
              pendingApplications > 0 ? "text-signal" : ""
            }`}
          >
            Заявки · <span className="font-mono">{pendingApplications}</span>
          </Link>
        </div>
      ) : null}
      <div className="px-[20px] py-[12px] max-sm:px-[14px]">
        <Link
          href="/events/new"
          className="swiss-focus block w-full bg-ink px-[11px] py-[12px] text-center text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
        >
          + Создать событие
        </Link>
      </div>
    </section>
  );
}
