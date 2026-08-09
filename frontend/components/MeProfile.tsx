"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthGate } from "@/components/ui/AuthGate";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { Cell } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import {
  fetchFollowedOrganizers,
  fetchMyApplications,
  fetchMyPractices,
  getMe,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatShortDate, formatStartTime } from "@/lib/format";
import { memberSince } from "@/lib/member-since";
import { rsvpStatusLabel } from "@/lib/rsvp-labels";
import type { FollowedOrganizer, Rsvp } from "@/lib/types";

type Tab = "upcoming" | "past" | "applications" | "follows";

const TABS: { key: Tab; label: string }[] = [
  { key: "upcoming", label: "Предстоящие" },
  { key: "past", label: "Прошедшие" },
  { key: "applications", label: "Заявки" },
  { key: "follows", label: "Подписки" },
];

function isTab(value: string | null): value is Tab {
  return value === "upcoming" || value === "past" || value === "applications" || value === "follows";
}

/** Count for a tab chip: "—" until the list has loaded (numbers never guess). */
function countLabel(n: number | undefined): string {
  return n == null ? "—" : String(n);
}

/** U6 registration row: mono date · title+context · organizer · status.
 * The whole row is ONE <Link> — nothing inside may be an anchor (React #418). */
function RegistrationRow({ rsvp }: { rsvp: Rsvp }) {
  const event = rsvp.event;
  const status = rsvpStatusLabel(rsvp.status);
  // A cancelled event keeps its row here on purpose — the registration is what
  // the user remembers, so the row has to say it is off rather than vanish.
  const cancelled = event?.status === "cancelled";
  const context = event
    ? [event.venue?.name ?? (event.format === "online" ? "Онлайн" : "—"), formatStartTime(event.startsAt)]
        .filter(Boolean)
        .join(" · ")
    : "Событие недоступно";

  return (
    <Link
      href={`/events/${rsvp.eventId}`}
      className="block border-b border-on-surface swiss-focus hover-invert"
    >
      {/* Desktop — 56px 1fr 118px 134px */}
      <div className="hidden grid-cols-[56px_1fr_118px_134px] items-center sm:grid">
        <span className="border-r border-on-surface px-[8px] py-[10px] text-center font-mono text-[11px] font-bold">
          {event ? formatShortDate(event.startsAt) : "—"}
        </span>
        <span className="border-r border-on-surface px-[14px] py-[10px]">
          <span className="block text-[13px] font-bold leading-[1.1]">
            {cancelled && <span className="text-signal">ОТМЕНЕНО · </span>}
            {event?.title ?? `Событие #${rsvp.eventId.slice(0, 8)}`}
          </span>
          <span className="cap mt-[3px] block">{context}</span>
        </span>
        <span className="border-r border-on-surface px-[14px] py-[10px]">
          <span className="cap block">Организатор</span>
          <span className="mt-[2px] block truncate text-[11px] font-bold">
            {event?.organizer?.name || "—"}
            {event?.organizer?.verified ? " ✓" : ""}
          </span>
        </span>
        <span className="flex items-center justify-center px-[8px] py-[10px]">
          <StatusChip status={status.long} className="px-[7px] py-[3px] text-[8px]" />
        </span>
      </div>

      {/* Mobile — compact row, ≥44px tall */}
      <div className="flex min-h-[56px] flex-col justify-center gap-[3px] px-[14px] py-[10px] sm:hidden">
        <span className="flex items-baseline justify-between gap-[8px]">
          <span className="font-mono text-[10px] font-bold">
            {event ? `${formatShortDate(event.startsAt)} · ${formatStartTime(event.startsAt)}` : "—"}
          </span>
          <StatusChip status={status.long} className="px-[5px] py-[2px] text-[7px]">
            {status.short}
          </StatusChip>
        </span>
        <span className="text-[12px] font-bold leading-[1.1]">
          {cancelled && <span className="text-signal">ОТМЕНЕНО · </span>}
          {event?.title ?? `Событие #${rsvp.eventId.slice(0, 8)}`}
        </span>
      </div>
    </Link>
  );
}

/** Subscriptions row: organizer name + «Открыть →», one <Link> per row. */
function FollowRow({ org }: { org: FollowedOrganizer }) {
  return (
    <Link
      href={`/organizers/${org.profileId}`}
      className="flex items-center justify-between gap-[10px] border-b border-on-surface px-[14px] py-[12px] swiss-focus hover-invert"
    >
      <span className="min-w-0">
        <span className="cap block">Организатор</span>
        <span className="mt-[2px] block truncate text-[13px] font-bold leading-[1.1]">{org.name}</span>
      </span>
      <span className="lbl shrink-0">Открыть →</span>
    </Link>
  );
}

function RowSkeletons() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 3 }, (_, i) => (
        <Skeleton key={i} className="h-[56px] w-full border-x-0 border-t-0" />
      ))}
    </div>
  );
}

export function MeProfile() {
  const { isAuthed, ready } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const requested = searchParams.get("tab");
  const tab: Tab = isTab(requested) ? requested : "upcoming";

  const authed = ready && isAuthed;

  const me = useQuery({ queryKey: ["me"], queryFn: getMe, enabled: authed });
  const upcoming = useQuery({
    queryKey: ["my-practices", "upcoming"],
    queryFn: () => fetchMyPractices("upcoming"),
    enabled: authed,
  });
  const past = useQuery({
    queryKey: ["my-practices", "past"],
    queryFn: () => fetchMyPractices("past"),
    enabled: authed,
  });
  const applications = useQuery({
    queryKey: ["my-applications", "all"],
    queryFn: () => fetchMyApplications(),
    enabled: authed,
  });
  const follows = useQuery({
    queryKey: ["my-follows"],
    queryFn: fetchFollowedOrganizers,
    enabled: authed,
  });

  const header = <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption="ПРОФИЛЬ" />;

  if (!ready) {
    return (
      <>
        {header}
        <main className="mx-auto max-w-[1360px] px-[20px] py-[26px]">
          <Skeleton className="h-[120px] w-full" />
        </main>
      </>
    );
  }

  if (!isAuthed) {
    return (
      <>
        {header}
        <AuthGate
          title="Войдите, чтобы видеть свои записи"
          reassurance="Лента и карта доступны без входа."
        />
      </>
    );
  }

  const active =
    tab === "upcoming" ? upcoming
    : tab === "past" ? past
    : tab === "applications" ? applications
    : follows;

  const counts: Record<Tab, number | undefined> = {
    upcoming: upcoming.data?.length,
    past: past.data?.length,
    applications: applications.data?.length,
    follows: follows.data?.length,
  };

  const since = memberSince(me.data?.createdAt);

  return (
    <>
      {header}
      <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
        {/* Identity strip — 1fr 200px desktop */}
        <div className="grid grid-cols-[1fr_200px] border-b border-ink max-md:grid-cols-1">
          <div className="border-r border-on-surface px-[20px] py-[16px] max-md:border-r-0 max-md:px-[14px] max-md:py-[13px]">
            {since ? <p className="cap mb-[6px] max-md:mb-[5px]">{since}</p> : null}
            <h1 className="text-[30px] font-black leading-[0.94] tracking-[-0.03em] max-md:text-[20px]">
              {me.data?.name?.trim() || me.data?.email || "Профиль"}
            </h1>
          </div>
          {/* Desktop: two stacked cells, 15px values. Mobile: three-cell strip
              below at 12px / 8px-10px padding (reference :423-426 / :454-458). */}
          <div className="grid grid-rows-2 max-md:hidden">
            <Cell
              caption="Посещено"
              value={countLabel(counts.past)}
              mono
              valueClassName="text-[15px]"
              className="border-b border-on-surface"
            />
            <Cell caption="Подписки" value={countLabel(counts.follows)} mono valueClassName="text-[15px]" />
          </div>
          <div className="grid grid-cols-3 border-b border-ink md:hidden [&>*+*]:border-l [&>*+*]:border-on-surface">
            <Cell caption="Посещено" value={countLabel(counts.past)} mono className="px-[10px] py-[8px]" />
            <Cell caption="Заявки" value={countLabel(counts.applications)} mono className="px-[10px] py-[8px]" />
            <Cell caption="Подписки" value={countLabel(counts.follows)} mono className="px-[10px] py-[8px]" />
          </div>
        </div>

        {/* Tab chips with counts — 9px/20px gap 6 desktop, 8px/14px gap 5 mobile */}
        <div className="flex gap-[6px] overflow-x-auto border-b border-ink px-[20px] py-[9px] max-sm:gap-[5px] max-sm:px-[14px] max-sm:py-[8px] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          {TABS.map((t) => (
            <Chip
              key={t.key}
              variant={tab === t.key ? "active" : "default"}
              onClick={() => router.replace(t.key === "upcoming" ? "/me" : `/me?tab=${t.key}`)}
              aria-pressed={tab === t.key}
              className="min-h-[44px] max-sm:text-[8px]"
            >
              {t.label} · <span className="font-mono">{countLabel(counts[t.key])}</span>
            </Chip>
          ))}
        </div>

        {/* Rows */}
        {active.isPending ? (
          <RowSkeletons />
        ) : active.isError ? (
          <EmptyState
            numeral="!"
            title="Не удалось загрузить данные"
            text="Проверьте соединение и попробуйте обновить страницу."
          />
        ) : tab === "follows" ? (
          follows.data && follows.data.length > 0 ? (
            <div className="flex flex-col">
              {follows.data.map((org) => (
                <FollowRow key={org.profileId} org={org} />
              ))}
            </div>
          ) : (
            <EmptyState
              title="Подписок пока нет"
              text="Подпишитесь на организатора — его события появятся в вашем календаре."
              actions={
                <Link
                  href="/"
                  className="swiss-focus border border-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover-invert"
                >
                  Найти события
                </Link>
              }
            />
          )
        ) : (active.data as Rsvp[] | undefined)?.length ? (
          <>
            <div className="flex flex-col">
              {(active.data as Rsvp[]).map((rsvp) => (
                <RegistrationRow key={rsvp.id} rsvp={rsvp} />
              ))}
            </div>
            <div className="px-[20px] py-[16px] text-center">
              <p className="cap mb-[8px]">Больше записей пока нет</p>
              <Link
                href="/"
                className="swiss-focus inline-flex items-center border border-ink px-[9px] py-[4px] text-[9px] uppercase tracking-[0.12em] hover-invert"
              >
                Найти события →
              </Link>
            </div>
          </>
        ) : (
          <EmptyState
            title={
              tab === "upcoming" ? "Записей пока нет"
              : tab === "past" ? "Прошедших событий пока нет"
              : "Заявок пока нет"
            }
            text={
              tab === "upcoming"
                ? "Когда вы запишетесь на событие, оно появится здесь."
                : tab === "past"
                  ? "Здесь появятся события, на которых вы побывали."
                  : "Заявки на события с отбором участников появятся здесь."
            }
            actions={
              <Link
                href="/"
                className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
              >
                Найти событие
              </Link>
            }
          />
        )}
      </main>
    </>
  );
}
