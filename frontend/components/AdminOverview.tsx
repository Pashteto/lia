"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { complaintsSignalRu, testDataSignalRu } from "@/lib/admin-copy";
import { formatShortDate } from "@/lib/format";
import { isLikelyTestContent } from "@/lib/admin-test-heuristic";
import {
  getAdminOverview,
  listModerationEvents,
  listModerationOrganizers,
  type AdminEvent,
  type AdminOrganizer,
} from "@/lib/api";
import { cn } from "@/lib/cn";
import { padCount } from "@/lib/org-seats";
import { tileCount } from "@/lib/tile-count";

type OverviewCounts = Awaited<ReturnType<typeof getAdminOverview>>;

type OverviewData = {
  overview: OverviewCounts;
  published: AdminEvent[];
  pendingOrgs: AdminOrganizer[];
};

/** Mono tile hero: pad to 2 digits under 100; raw string at 100+. */
function tileHero(n: number): string {
  return n < 100 ? padCount(n) : String(n);
}

function OverviewSkeleton() {
  return (
    <div className="flex flex-col gap-[12px] px-[20px] py-[26px] max-sm:px-[14px]">
      <Skeleton className="h-[72px] w-full" />
      <div className="grid grid-cols-2 gap-[12px] max-sm:grid-cols-1">
        <Skeleton className="h-[160px] w-full" />
        <Skeleton className="h-[160px] w-full max-sm:hidden" />
      </div>
    </div>
  );
}

function OpenQueueCta({ className }: { className?: string }) {
  return (
    <Link
      href="/admin/moderation/events"
      className={cn(
        "swiss-focus inline-flex min-h-[44px] w-full items-center justify-center bg-paper px-[11px] py-[11px] text-center text-[10px] font-bold uppercase tracking-[0.07em] text-ink transition-colors duration-[120ms] ease-linear hover:opacity-90",
        className,
      )}
    >
      ОТКРЫТЬ ОЧЕРЕДЬ
    </Link>
  );
}

function SignalsFooter({
  complaintsOpen,
  published,
}: {
  complaintsOpen: number;
  published: AdminEvent[];
}) {
  if (complaintsOpen > 0) {
    return (
      <Link
        href="/admin/complaints"
        className="swiss-focus text-[11px] font-bold text-signal hover:underline"
      >
        {complaintsSignalRu(complaintsOpen)}
      </Link>
    );
  }

  const testCount = published.filter((e) => isLikelyTestContent(e.title)).length;
  if (testCount > 0) {
    return (
      <p className="text-[11px] font-bold text-signal">
        {testDataSignalRu(testCount)}
      </p>
    );
  }

  return <p className="cap">Сигналов нет</p>;
}

function DesktopOverview({ data }: { data: OverviewData }) {
  const { overview, published, pendingOrgs } = data;
  const queue = published.slice(0, 3);
  const orgs = pendingOrgs.slice(0, 3);
  const waiting = published.length;

  return (
    <div className="hidden min-h-[360px] flex-col sm:flex">
      <div className="grid flex-none grid-cols-4 border-b border-paper">
        <div className="border-r border-paper p-[14px]">
          <div className="cap">Событий всего</div>
          <div className="font-mono text-[26px] font-bold leading-none">
            {tileHero(overview.events_total)}
          </div>
        </div>
        <Link
          href="/admin/moderation/events"
          className="group swiss-focus cursor-pointer border-r border-paper bg-signal p-[14px] transition-colors duration-[120ms] hover:bg-paper"
        >
          <div className="cap text-signal-tint group-hover:text-signal">
            Ждут модерации →
          </div>
          <div className="font-mono text-[26px] font-bold leading-none text-white group-hover:text-signal">
            {tileHero(waiting)}
          </div>
        </Link>
        <Link
          href="/admin/organizers"
          className="group swiss-focus cursor-pointer border-r border-paper p-[14px] transition-colors duration-[120ms] hover:bg-paper"
        >
          <div className="cap group-hover:text-ink">Организаторов →</div>
          <div className="font-mono text-[26px] font-bold leading-none group-hover:text-ink">
            {tileCount(overview.organizers_total)}
          </div>
        </Link>
        <Link
          href="/admin/users"
          className="group swiss-focus cursor-pointer p-[14px] transition-colors duration-[120ms] hover:bg-paper"
        >
          <div className="cap group-hover:text-ink">Пользователей →</div>
          <div className="font-mono text-[26px] font-bold leading-none group-hover:text-ink">
            {tileCount(overview.users_total)}
          </div>
        </Link>
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-2">
        <div className="flex flex-col border-r border-paper">
          <div className="border-b border-rule-inner px-[16px] py-[10px]">
            <span className="cap">Очередь модерации</span>
          </div>
          {queue.length === 0 ? (
            <p className="cap px-[16px] py-[12px]">Очередь пуста</p>
          ) : (
            queue.map((event) => {
              const test = isLikelyTestContent(event.title);
              return (
                <Link
                  key={event.id}
                  href="/admin/moderation/events"
                  className="group swiss-focus flex cursor-pointer items-baseline justify-between gap-[12px] border-b border-rule-inner px-[16px] py-[9px] transition-colors hover:bg-paper"
                >
                  <span
                    className={cn(
                      "text-[11.5px] font-bold leading-[1.2] group-hover:text-ink",
                      test && "text-signal",
                    )}
                  >
                    {event.title}
                  </span>
                  <span className="cap shrink-0 font-mono group-hover:text-ink">
                    {formatShortDate(event.starts_at)}
                  </span>
                </Link>
              );
            })
          )}
          <div className="mt-auto px-[16px] py-[10px]">
            <OpenQueueCta />
          </div>
        </div>

        <div className="flex flex-col">
          <div className="border-b border-rule-inner px-[16px] py-[10px]">
            <span className="cap">
              Заявки на верификацию · {pendingOrgs.length}
            </span>
          </div>
          {orgs.length === 0 ? (
            <p className="cap px-[16px] py-[12px]">Заявок нет</p>
          ) : (
            orgs.map((org) => {
              const test = isLikelyTestContent(org.name, org.website_url);
              return (
                <Link
                  key={org.id}
                  href={`/admin/organizers/${org.id}`}
                  className="group swiss-focus block cursor-pointer border-b border-rule-inner px-[16px] py-[9px] transition-colors hover:bg-paper"
                >
                  <span
                    className={cn(
                      "text-[11.5px] font-bold leading-[1.2] group-hover:text-ink",
                      test && "text-signal",
                    )}
                  >
                    {org.name}
                  </span>
                </Link>
              );
            })
          )}
          <div className="mt-auto px-[16px] py-[10px]">
            <div className="cap mb-[5px]">Сигналы</div>
            <SignalsFooter
              complaintsOpen={overview.complaints_open ?? 0}
              published={published}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function DutyOverview({ data }: { data: OverviewData }) {
  const { published, pendingOrgs } = data;
  const queue = published.slice(0, 3);
  const waiting = published.length;

  return (
    <div className="flex min-h-[360px] flex-col sm:hidden">
      <div className="grid flex-none grid-cols-2 border-b border-paper">
        <Link
          href="/admin/moderation/events"
          className="group swiss-focus cursor-pointer border-r border-paper bg-signal p-[11px] transition-colors duration-[120ms] hover:bg-paper"
        >
          <div className="cap text-signal-tint group-hover:text-signal">
            Модерация →
          </div>
          <div className="font-mono text-[20px] font-bold leading-none text-white group-hover:text-signal">
            {tileHero(waiting)}
          </div>
        </Link>
        <Link
          href="/admin/organizers?filter=pending"
          className="group swiss-focus cursor-pointer p-[11px] transition-colors duration-[120ms] hover:bg-paper"
        >
          <div className="cap group-hover:text-ink">Верификация →</div>
          <div className="font-mono text-[20px] font-bold leading-none group-hover:text-ink">
            {tileHero(pendingOrgs.length)}
          </div>
        </Link>
      </div>

      <div className="min-h-0 flex-1">
        {queue.length === 0 ? (
          <p className="cap px-[14px] py-[12px]">Очередь пуста</p>
        ) : (
          queue.map((event, i) => {
            const test = isLikelyTestContent(event.title);
            const age = formatShortDate(event.starts_at);
            const org = event.organizer_name?.trim() || "—";
            return (
              <div
                key={event.id}
                className={cn(
                  "px-[14px] py-[10px]",
                  i < queue.length - 1 && "border-b border-rule-inner",
                )}
              >
                <div className="cap mb-[3px]">
                  {age} · {org}
                </div>
                <div
                  className={cn(
                    "text-[11.5px] font-bold leading-[1.1]",
                    test && "text-signal",
                  )}
                >
                  {event.title}
                </div>
              </div>
            );
          })
        )}
      </div>

      <div className="flex-none px-[14px] py-[11px]">
        <OpenQueueCta />
      </div>
    </div>
  );
}

/** A1 · Обзор — desktop 4-tile strip + queues; max-sm duty mode. */
export function AdminOverview() {
  const [data, setData] = useState<OverviewData | null>(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [overview, published, pendingOrgs] = await Promise.all([
          getAdminOverview(),
          listModerationEvents("published"),
          listModerationOrganizers("pending"),
        ]);
        if (!cancelled) {
          setData({ overview, published, pendingOrgs });
          setError(false);
        }
      } catch {
        if (!cancelled) {
          setData(null);
          setError(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tick]);

  if (loading) return <OverviewSkeleton />;

  if (error || !data) {
    return (
      <EmptyState
        title="Не удалось загрузить обзор"
        text="Обновите страницу или зайдите позже."
        actions={
          <Button
            variant="inverted"
            type="button"
            onClick={() => {
              setLoading(true);
              setTick((t) => t + 1);
            }}
          >
            Повторить
          </Button>
        }
      />
    );
  }

  return (
    <>
      <DesktopOverview data={data} />
      <DutyOverview data={data} />
    </>
  );
}
