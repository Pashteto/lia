"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  fetchEventsByOrganizer,
  followOrganizer,
  getPublicOrganizer,
  unfollowOrganizer,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatShortDate } from "@/lib/format";
import { padCount } from "@/lib/org-seats";
import type { LiaEvent } from "@/lib/types";

function EventRow({ event }: { event: LiaEvent }) {
  const category = event.categories[0]?.label ?? "—";
  return (
    <Link
      href={`/events/${event.id}`}
      className="block border-b border-on-surface px-[14px] py-[9px] swiss-focus hover-invert"
    >
      <span className="mb-[3px] flex items-baseline justify-between gap-[8px]">
        <span className="font-mono text-[9.5px] font-bold">{formatShortDate(event.startsAt)}</span>
        <span className="cap">{category}</span>
      </span>
      <span className="block text-[12px] font-bold leading-[1.1]">{event.title}</span>
    </Link>
  );
}

function PublicSkeleton() {
  return (
    <main className="mx-auto max-w-[1360px]">
      <Skeleton className="h-[120px] w-full border-x-0 border-t-0" />
      <Skeleton className="h-[52px] w-full border-x-0 border-t-0" />
      <Skeleton className="h-[56px] w-full border-x-0 border-t-0" />
      <Skeleton className="h-[56px] w-full border-x-0 border-t-0" />
    </main>
  );
}

export function PublicOrganizerView() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const { isAuthed, ready } = useAuth();
  const [followOverride, setFollowOverride] = useState<{ id: string; value: boolean } | null>(
    null,
  );
  const [pending, setPending] = useState(false);
  // Keep Date.now out of render (react-hooks/purity) — same pattern as DiscoveryFeed.
  const [now] = useState(() => Date.now());

  const orgQuery = useQuery({
    queryKey: ["public-organizer", id],
    queryFn: () => getPublicOrganizer(id),
    enabled: Boolean(id),
  });

  const eventsQuery = useQuery({
    queryKey: ["organizer-events", id],
    queryFn: () => fetchEventsByOrganizer(id),
    enabled: Boolean(id),
  });

  const following =
    followOverride?.id === id ? followOverride.value : (orgQuery.data?.is_following ?? false);

  const upcoming = useMemo(() => {
    return (eventsQuery.data ?? [])
      .filter((e) => new Date(e.startsAt).getTime() >= now)
      .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime());
  }, [eventsQuery.data, now]);

  const past = useMemo(() => {
    return (eventsQuery.data ?? [])
      .filter((e) => new Date(e.startsAt).getTime() < now)
      .sort((a, b) => new Date(b.startsAt).getTime() - new Date(a.startsAt).getTime())
      .slice(0, 10);
  }, [eventsQuery.data, now]);

  async function toggleFollow() {
    if (!orgQuery.data || pending || !isAuthed) return;
    const next = !following;
    setFollowOverride({ id, value: next });
    setPending(true);
    try {
      if (next) await followOrganizer(orgQuery.data.id);
      else await unfollowOrganizer(orgQuery.data.id);
    } catch {
      setFollowOverride({ id, value: !next });
    } finally {
      setPending(false);
    }
  }

  if (orgQuery.isLoading) return <PublicSkeleton />;

  const org = orgQuery.data;
  if (orgQuery.isError || !org) {
    return (
      <main className="mx-auto max-w-[1360px]">
        <EmptyState
          title="Организатор не найден"
          text="Проверьте ссылку или вернитесь к ленте событий."
          actions={
            <Link
              href="/"
              className="swiss-focus inline-flex min-h-[44px] items-center justify-center bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white"
            >
              К событиям
            </Link>
          }
        />
      </main>
    );
  }
  const eventCount = eventsQuery.data?.length;
  const eventsValue =
    eventCount == null ? "—" : eventCount < 100 ? padCount(eventCount) : String(eventCount);

  return (
    <main className="mx-auto flex max-w-[1360px] flex-col pb-[88px] sm:pb-[64px]">
      {/* Identity */}
      <section className="border-b border-on-surface px-[14px] py-[14px] sm:px-[20px] sm:py-[16px]">
        <div className="mb-[10px] flex items-center gap-[10px]">
          {org.logo_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={org.logo_url}
              alt=""
              className="h-[40px] w-[40px] shrink-0 object-cover sm:h-[40px] sm:w-[40px]"
            />
          ) : (
            <span className="h-[40px] w-[40px] shrink-0 bg-ink" aria-hidden />
          )}
          <span className="min-w-0">
            <span className="block text-[16px] font-black leading-[1.05] tracking-[-0.02em]">
              {org.name}
              {org.verified ? " ✓" : ""}
            </span>
            <span className="cap">
              {org.verified ? "Проверенный организатор" : "Организатор"}
            </span>
          </span>
        </div>
        {org.description ? (
          <p className="m-0 text-[11px] leading-[1.45] text-on-surface">{org.description}</p>
        ) : null}
        {org.website_url ? (
          <a
            href={org.website_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-[8px] inline-block text-[11px] text-text-dim underline swiss-focus"
          >
            {org.website_url}
          </a>
        ) : null}
      </section>

      {/* Stats */}
      <CellStrip cols={2}>
        <Cell
          caption="Событий"
          value={eventsValue}
          mono
          valueClassName="text-[15px] font-bold"
          className="px-[12px] py-[9px]"
        />
        <Cell
          caption="Подписчиков"
          value="—"
          mono
          valueClassName="text-[15px] font-bold"
          className="px-[12px] py-[9px]"
        />
      </CellStrip>

      {/* Upcoming rows */}
      <section className="min-h-0 flex-1">
        {eventsQuery.isLoading ? (
          <>
            <Skeleton className="h-[52px] w-full border-x-0 border-t-0" />
            <Skeleton className="h-[52px] w-full border-x-0 border-t-0" />
          </>
        ) : upcoming.length === 0 && past.length === 0 ? (
          <p className="px-[14px] py-[14px] text-[11.5px] text-text-dim">Пока нет событий.</p>
        ) : (
          <>
            {upcoming.map((e) => (
              <EventRow key={e.id} event={e} />
            ))}
            {upcoming.length === 0 && past.length > 0
              ? past.map((e) => <EventRow key={e.id} event={e} />)
              : null}
          </>
        )}
      </section>

      {/* Follow CTA */}
      {ready && isAuthed ? (
        <div className="sticky bottom-0 border-t border-on-surface bg-paper px-[14px] py-[11px] sm:static">
          <Button
            type="button"
            variant={following ? "ghost" : "primary"}
            onClick={toggleFollow}
            disabled={pending}
            className="w-full min-h-[44px]"
          >
            {following ? "ВЫ ПОДПИСАНЫ" : "ПОДПИСАТЬСЯ"}
          </Button>
        </div>
      ) : ready ? (
        <div className="border-t border-on-surface px-[14px] py-[11px]">
          <Link
            href={`/login?next=${encodeURIComponent(`/organizers/${id}`)}`}
            className="swiss-focus inline-flex min-h-[44px] w-full items-center justify-center bg-ink px-[11px] py-[11px] text-center text-[11px] font-bold uppercase tracking-[0.07em] text-white transition-colors duration-[120ms] ease-linear hover:bg-black"
          >
            ВОЙТИ, ЧТОБЫ ПОДПИСАТЬСЯ
          </Link>
        </div>
      ) : null}
    </main>
  );
}
