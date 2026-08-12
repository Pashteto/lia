"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { AuthGate } from "@/components/ui/AuthGate";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Skeleton } from "@/components/ui/Skeleton";
import { fetchEventApplications, fetchMyEvents, getMyOrganizer } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatStartTime } from "@/lib/format";
import { nextOrganizerEvent, orgDashboardStats } from "@/lib/org-dashboard";
import { padCount, seatsFill } from "@/lib/org-seats";
import type { LiaEvent } from "@/lib/types";

const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});

function stripHero(n: number): string {
  return n < 100 ? padCount(n) : String(n);
}

function regsHero(n: number | null): string {
  if (n == null) return "—";
  return n < 100 ? padCount(n) : String(n);
}

/** "12 июля · 16:00" (+ venue on desktop). */
function nextEventMeta(event: LiaEvent, includeVenue: boolean): string {
  const parts = [dayMonthFmt.format(new Date(event.startsAt)), formatStartTime(event.startsAt)];
  if (includeVenue) {
    const place = event.venue?.name ?? (event.format === "online" ? "Онлайн" : null);
    if (place) parts.push(place);
  }
  return parts.join(" · ");
}

function CreateEventCta({ className }: { className?: string }) {
  return (
    <Link
      href="/events/new"
      className={`swiss-focus inline-flex min-h-[44px] w-full items-center justify-center bg-ink px-[11px] py-[11px] text-center text-[11px] font-bold uppercase tracking-[0.07em] text-white transition-colors duration-[120ms] ease-linear hover:bg-black ${className ?? ""}`}
    >
      + СОЗДАТЬ СОБЫТИЕ
    </Link>
  );
}

function HubSkeleton() {
  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      <Skeleton className="h-[78px] w-full border-x-0 border-t-0" />
      <Skeleton className="h-[70px] w-full border-x-0 border-t-0" />
      <Skeleton className="h-[48px] w-full border-x-0 border-t-0" />
      <div className="grid grid-cols-2 max-sm:grid-cols-1">
        <Skeleton className="h-[160px] w-full border-x-0 border-t-0" />
        <Skeleton className="h-[160px] w-full border-x-0 border-t-0 max-sm:hidden" />
      </div>
    </main>
  );
}

export function OrganizerHub() {
  const { isAuthed, ready } = useAuth();
  const authed = ready && isAuthed;

  const organizer = useQuery({
    queryKey: ["my-organizer"],
    queryFn: getMyOrganizer,
    enabled: authed,
  });
  const events = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: authed,
  });

  const eventList = events.data ?? [];
  const applicationIds = eventList
    .filter((e) => e.signupMode === "application")
    .map((e) => e.id);

  const pendingApps = useQuery({
    queryKey: ["org-pending-applications", applicationIds],
    queryFn: async () => {
      const lists = await Promise.all(applicationIds.map((id) => fetchEventApplications(id)));
      return lists.reduce((sum, list) => sum + list.filter((r) => r.status === "applied").length, 0);
    },
    enabled: authed && applicationIds.length > 0,
  });

  if (!ready) return <HubSkeleton />;

  if (!isAuthed) {
    return (
      <AuthGate
        title="Войдите, чтобы открыть кабинет и создавать события"
        reassurance="Лента и карта доступны без входа."
      />
    );
  }

  if (organizer.isLoading || events.isLoading) return <HubSkeleton />;

  const stats = orgDashboardStats(eventList);
  const next = nextOrganizerEvent(eventList);
  const nextFill = next ? seatsFill(next) : null;
  const orgName = organizer.data?.name?.trim() || "Организатор";
  const verified = organizer.data?.verification_status === "verified";
  const nameLine = `${orgName}${verified ? " ✓" : ""}`;
  const pendingCount =
    applicationIds.length === 0 ? 0 : (pendingApps.data ?? (pendingApps.isLoading ? null : 0));

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      {/* Identity — desktop 1fr 190px; mobile stacked title only */}
      <div className="grid grid-cols-[1fr_190px] border-b border-ink max-sm:grid-cols-1">
        <div className="border-r border-on-surface px-[20px] py-[16px] max-sm:border-r-0 max-sm:px-[14px] max-sm:py-[12px]">
          <p className="cap mb-[6px] max-sm:mb-[5px]">
            <span className="max-sm:hidden">Организатор · </span>
            {nameLine}
          </p>
          <h1 className="text-[30px] font-black leading-[0.94] tracking-[-0.03em] max-sm:text-[20px]">
            Кабинет
          </h1>
        </div>
        <div className="flex items-end px-[18px] py-[16px] max-sm:hidden">
          <CreateEventCta />
        </div>
      </div>

      {/* Status strip — 4 cells desktop / 3 mobile (drop «Всего записей») */}
      <CellStrip cols={4} className="max-sm:hidden">
        <Cell
          caption="Опубликовано"
          value={stripHero(stats.published)}
          mono
          href="/events/mine?status=published"
          valueClassName="text-[26px]"
          className="px-[14px] py-[14px]"
        />
        <Cell
          caption="На модерации"
          value={stripHero(stats.pendingReview)}
          mono
          invert
          href="/events/mine?status=pending_review"
          valueClassName="text-[26px]"
          className="px-[14px] py-[14px]"
        />
        <Cell
          caption="Черновики"
          value={stripHero(stats.drafts)}
          mono
          href="/events/mine?status=draft"
          valueClassName="text-[26px]"
          className="px-[14px] py-[14px]"
        />
        <Cell
          caption="Всего записей"
          value={regsHero(stats.totalRegistrations)}
          mono
          valueClassName="text-[26px]"
          className="px-[14px] py-[14px]"
        />
      </CellStrip>
      <CellStrip cols={3} className="sm:hidden">
        <Cell
          caption="Опубл."
          value={stripHero(stats.published)}
          mono
          href="/events/mine?status=published"
          valueClassName="text-[16px]"
          className="px-[10px] py-[9px]"
        />
        <Cell
          caption="Модер."
          value={stripHero(stats.pendingReview)}
          mono
          invert
          href="/events/mine?status=pending_review"
          valueClassName="text-[16px]"
          className="px-[10px] py-[9px]"
        />
        <Cell
          caption="Черн."
          value={stripHero(stats.drafts)}
          mono
          href="/events/mine?status=draft"
          valueClassName="text-[16px]"
          className="px-[10px] py-[9px]"
        />
      </CellStrip>

      {/* Pending applications banner — hidden when 0; signal red on border + count only */}
      {pendingCount != null && pendingCount > 0 ? (
        <Link
          href="/organizer/applications"
          className="flex items-center gap-[14px] border-b border-on-surface border-l-4 border-l-signal px-[20px] py-[12px] swiss-focus hover-invert max-sm:gap-[10px] max-sm:px-[14px] max-sm:py-[10px]"
        >
          <span className="font-mono text-[22px] font-bold text-signal max-sm:text-[17px]">
            {stripHero(pendingCount)}
          </span>
          <span className="flex-1 text-[12.5px] font-bold leading-[1.2] max-sm:text-[11px]">
            <span className="max-sm:hidden">новых заявок ждут подтверждения</span>
            <span className="sm:hidden">заявок ждут ответа</span>
          </span>
          <Chip as="span" className="max-sm:hidden">
            Смотреть →
          </Chip>
          <span className="cap sm:hidden">→</span>
        </Link>
      ) : null}

      {/* Bottom: next event + activity stub (desktop); next event only (mobile) */}
      <div className="grid grid-cols-2 border-b border-ink max-sm:grid-cols-1">
        {next ? (
          <Link
            href={`/events/${next.id}`}
            className="flex min-h-[160px] flex-col border-r border-on-surface px-[20px] py-[14px] swiss-focus hover-invert max-sm:min-h-0 max-sm:border-r-0 max-sm:px-[14px] max-sm:py-[12px]"
          >
            <p className="cap mb-[8px] max-sm:mb-[6px]">
              <span className="max-sm:hidden">Ближайшее событие</span>
              <span className="sm:hidden">Ближайшее</span>
            </p>
            <p className="mb-[6px] text-[17px] font-black leading-[1.05] tracking-[-0.02em] max-sm:mb-[5px] max-sm:text-[13px] max-sm:font-bold">
              {next.title}
            </p>
            <p className="cap max-sm:mb-[10px]">
              <span className="max-sm:hidden">{nextEventMeta(next, true)}</span>
              <span className="sm:hidden">{nextEventMeta(next, false)}</span>
            </p>
            <div className="mt-auto max-sm:mt-[0]">
              <div className="mb-[4px] flex justify-between text-[10px] max-sm:mb-[3px] max-sm:text-[9px]">
                <span className="cap">Заполнено</span>
                <span className="font-mono font-bold">{nextFill?.label ?? "—"}</span>
              </div>
              {nextFill ? (
                <ProgressBar value={nextFill.filled} max={nextFill.capacity} className="max-sm:h-[7px]" />
              ) : (
                <div className="h-[8px] border border-on-surface max-sm:h-[7px]" aria-hidden />
              )}
            </div>
          </Link>
        ) : (
          <div className="flex min-h-[160px] flex-col border-r border-on-surface px-[20px] py-[14px] max-sm:min-h-0 max-sm:border-r-0 max-sm:px-[14px] max-sm:py-[12px]">
            <p className="cap mb-[8px]">
              <span className="max-sm:hidden">Ближайшее событие</span>
              <span className="sm:hidden">Ближайшее</span>
            </p>
            <p className="text-[13px] font-bold text-text-dim">Нет предстоящих событий</p>
          </div>
        )}

        {/* Activity rail — stub only (no fake rows) */}
        <div className="px-[20px] py-[14px] max-sm:hidden">
          <p className="cap mb-[8px]">Последнее</p>
          <p className="cap">Лента активности появится позже</p>
        </div>
      </div>

      {/* Mobile sticky create CTA */}
      <div className="sticky bottom-0 border-t border-ink bg-paper px-[14px] py-[12px] sm:hidden">
        <CreateEventCta />
      </div>
    </main>
  );
}
