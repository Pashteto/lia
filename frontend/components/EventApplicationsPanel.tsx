"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Skeleton } from "@/components/ui/Skeleton";
import { decideApplication, fetchEventApplications } from "@/lib/api";
import {
  applicationMeta,
  decideMany,
} from "@/lib/org-applications";
import { seatsFill } from "@/lib/org-seats";
import { formatRelativeRu } from "@/lib/relative-time";
import type { LiaEvent, Rsvp, RsvpStatus } from "@/lib/types";

type Tab = "applied" | "accepted" | "declined";

const TABS: { key: Tab; label: string }[] = [
  { key: "applied", label: "Новые" },
  { key: "accepted", label: "Принятые" },
  { key: "declined", label: "Отклонённые" },
];

function matchesTab(status: RsvpStatus, tab: Tab): boolean {
  if (tab === "applied") return status === "applied";
  if (tab === "accepted") return status === "accepted" || status === "going";
  return status === "declined";
}

const dayMonthFmt = new Intl.DateTimeFormat("ru-RU", {
  day: "numeric",
  month: "long",
  timeZone: "Europe/Moscow",
});

export interface EventApplicationsPanelProps {
  eventId: string;
  event: Pick<LiaEvent, "title" | "startsAt" | "capacity" | "seatsRemaining" | "signupMode">;
  /** Hide event context strip when parent already renders chrome. */
  hideContext?: boolean;
}

export function EventApplicationsPanel({
  eventId,
  event,
  hideContext = false,
}: EventApplicationsPanelProps) {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("applied");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [busyIds, setBusyIds] = useState<Set<string>>(new Set());
  const [bulkBusy, setBulkBusy] = useState(false);
  const [rowErrors, setRowErrors] = useState<Record<string, string>>({});
  const [optimisticFilled, setOptimisticFilled] = useState<number | null>(null);

  const baseFill = seatsFill(event);

  useEffect(() => {
    setOptimisticFilled(null);
  }, [event.capacity, event.seatsRemaining, eventId]);

  const displayFill = useMemo(() => {
    if (!baseFill) return null;
    const filled = Math.min(
      baseFill.capacity,
      Math.max(0, optimisticFilled ?? baseFill.filled),
    );
    return {
      filled,
      capacity: baseFill.capacity,
      label: `${filled} / ${baseFill.capacity}`,
      ratio: Math.min(1, Math.max(0, filled / baseFill.capacity)),
    };
  }, [baseFill, optimisticFilled]);

  const {
    data: applications = [],
    isLoading,
    isError,
  } = useQuery<Rsvp[]>({
    queryKey: ["event-applications", eventId],
    queryFn: () => fetchEventApplications(eventId),
  });

  const counts = useMemo(() => {
    let applied = 0;
    let accepted = 0;
    let declined = 0;
    for (const a of applications) {
      if (a.status === "applied") applied += 1;
      else if (a.status === "accepted" || a.status === "going") accepted += 1;
      else if (a.status === "declined") declined += 1;
    }
    return { applied, accepted, declined };
  }, [applications]);

  const filtered = useMemo(
    () => applications.filter((a) => matchesTab(a.status, tab)),
    [applications, tab],
  );

  const pendingInFilter = useMemo(
    () => filtered.filter((a) => a.status === "applied"),
    [filtered],
  );

  const allPendingSelected =
    pendingInFilter.length > 0 &&
    pendingInFilter.every((a) => selected.has(a.id));

  function toggleSelectAll() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allPendingSelected) {
        for (const a of pendingInFilter) next.delete(a.id);
      } else {
        for (const a of pendingInFilter) next.add(a.id);
      }
      return next;
    });
  }

  function toggleOne(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function bumpSeatsOnAccept(n: number) {
    if (!baseFill || n <= 0) return;
    setOptimisticFilled((prev) => {
      const current = prev ?? baseFill.filled;
      return Math.min(baseFill.capacity, current + n);
    });
  }

  async function refresh() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["event-applications", eventId] }),
      queryClient.invalidateQueries({ queryKey: ["my-events"] }),
    ]);
  }

  async function handleDecide(rsvp: Rsvp, decision: "accept" | "decline") {
    setBusyIds((prev) => new Set(prev).add(rsvp.id));
    setRowErrors((prev) => {
      const next = { ...prev };
      delete next[rsvp.id];
      return next;
    });
    try {
      await decideApplication(eventId, rsvp.id, decision);
      if (decision === "accept") bumpSeatsOnAccept(1);
      setSelected((prev) => {
        const next = new Set(prev);
        next.delete(rsvp.id);
        return next;
      });
      await refresh();
    } catch (err) {
      setRowErrors((prev) => ({
        ...prev,
        [rsvp.id]: err instanceof Error ? err.message : "Ошибка",
      }));
    } finally {
      setBusyIds((prev) => {
        const next = new Set(prev);
        next.delete(rsvp.id);
        return next;
      });
    }
  }

  async function handleBulkAccept() {
    const ids = [...selected].filter((id) =>
      applications.some((a) => a.id === id && a.status === "applied"),
    );
    if (ids.length === 0) return;
    setBulkBusy(true);
    try {
      const { ok, failed } = await decideMany(eventId, ids, "accept");
      if (ok.length) bumpSeatsOnAccept(ok.length);
      setSelected((prev) => {
        const next = new Set(prev);
        for (const id of ok) next.delete(id);
        return next;
      });
      if (failed.length) {
        setRowErrors((prev) => {
          const next = { ...prev };
          for (const id of failed) next[id] = "Не удалось принять";
          return next;
        });
      }
      await refresh();
    } finally {
      setBulkBusy(false);
    }
  }

  const dateCap = dayMonthFmt.format(new Date(event.startsAt));

  return (
    <div className="flex flex-col">
      {!hideContext ? (
        <>
          {/* Desktop context */}
          <div className="hidden grid-cols-[1fr_200px] border-b border-on-surface sm:grid">
            <div className="border-r border-on-surface px-[20px] py-[14px]">
              <p className="cap mb-[5px]">Событие · {dateCap}</p>
              <h2 className="text-[22px] font-black leading-[1.05] tracking-[-0.02em]">
                {event.title}
              </h2>
            </div>
            <div className="px-[18px] py-[14px]">
              <p className="cap">Заполнено</p>
              <p className="font-mono text-[22px] font-bold leading-none">
                {displayFill?.label ?? "—"}
              </p>
              {displayFill ? (
                <ProgressBar
                  value={displayFill.filled}
                  max={displayFill.capacity}
                  className="mt-[5px] h-[6px]"
                />
              ) : (
                <div className="mt-[5px] h-[6px] border border-on-surface" aria-hidden />
              )}
            </div>
          </div>
          {/* Mobile capacity */}
          <div className="border-b border-on-surface px-[14px] py-[10px] sm:hidden">
            <p className="cap mb-[4px]">
              {event.title} · {dateCap}
            </p>
            <div className="flex items-baseline justify-between">
              <span className="font-mono text-[15px] font-bold">
                {displayFill?.label ?? "—"}
              </span>
              <span className="cap">заполнено</span>
            </div>
            {displayFill ? (
              <ProgressBar
                value={displayFill.filled}
                max={displayFill.capacity}
                className="mt-[5px] h-[6px]"
              />
            ) : (
              <div className="mt-[5px] h-[6px] border border-on-surface" aria-hidden />
            )}
          </div>
        </>
      ) : null}

      {/* Tabs */}
      <div className="flex items-center justify-between gap-[12px] border-b border-on-surface px-[20px] py-[8px] max-sm:px-[14px]">
        <div className="flex min-w-0 flex-wrap gap-[6px]">
          {TABS.map((t) => (
            <Chip
              key={t.key}
              variant={tab === t.key ? "active" : "default"}
              onClick={() => setTab(t.key)}
            >
              {t.label} · {counts[t.key]}
            </Chip>
          ))}
        </div>
        {tab === "applied" && pendingInFilter.length > 0 ? (
          <button
            type="button"
            className="cap hidden shrink-0 swiss-focus hover:underline sm:inline"
            onClick={toggleSelectAll}
          >
            {allPendingSelected ? "Снять все" : "Выбрать все"}
          </button>
        ) : (
          <span className="cap hidden shrink-0 text-text-dim sm:inline">Выбрать все</span>
        )}
      </div>

      {isLoading ? (
        <div className="flex flex-col">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-[48px] w-full border-x-0 border-t-0" />
          ))}
        </div>
      ) : isError ? (
        <EmptyState
          numeral="—"
          title="Не удалось загрузить заявки"
          text="Обновите страницу или попробуйте позже."
        />
      ) : filtered.length === 0 ? (
        <EmptyState
          numeral="00"
          title={
            tab === "applied"
              ? "Нет новых заявок"
              : tab === "accepted"
                ? "Нет принятых"
                : "Нет отклонённых"
          }
          text={
            applications.length === 0
              ? "Пока никто не подал заявку на это событие."
              : undefined
          }
        />
      ) : (
        <ul className="flex flex-col">
          {filtered.map((rsvp) => {
            const isPending = rsvp.status === "applied";
            const isBusy = busyIds.has(rsvp.id) || bulkBusy;
            const checked = selected.has(rsvp.id);

            return (
              <li
                key={rsvp.id}
                className="border-b border-on-surface max-sm:px-[14px] max-sm:py-[9px]"
              >
                {/* Desktop row */}
                <div className="hidden grid-cols-[26px_1fr_120px_150px] items-center sm:grid">
                  <div className="py-[11px] text-center">
                    {isPending ? (
                      <button
                        type="button"
                        aria-label={checked ? "Снять выбор" : "Выбрать"}
                        aria-pressed={checked}
                        className="inline-flex h-[44px] w-[44px] items-center justify-center swiss-focus"
                        onClick={() => toggleOne(rsvp.id)}
                        disabled={isBusy}
                      >
                        <span
                          className={`inline-block h-[11px] w-[11px] border border-ink ${
                            checked ? "bg-ink" : "bg-transparent"
                          }`}
                        />
                      </button>
                    ) : (
                      <span className="inline-block h-[11px] w-[11px]" aria-hidden />
                    )}
                  </div>
                  <div className="min-w-0 px-[12px] py-[9px]">
                    <span className="block text-[12.5px] font-bold leading-tight">
                      {rsvp.applicant?.name || "Участник"}
                    </span>
                    <span className="cap mt-[2px] block">
                      {applicationMeta(rsvp.applicationAnswer, rsvp.status)}
                    </span>
                    {rowErrors[rsvp.id] ? (
                      <span className="mt-[2px] block text-[10px] text-signal">
                        {rowErrors[rsvp.id]}
                      </span>
                    ) : null}
                  </div>
                  <div className="border-l border-on-surface px-[12px] py-[9px]">
                    <span className="cap block">Заявка</span>
                    <span className="font-mono text-[11px]">
                      {formatRelativeRu(rsvp.createdAt)}
                    </span>
                  </div>
                  <div className="flex gap-[6px] px-[10px] py-[8px]">
                    {isPending ? (
                      <>
                        <Button
                          size="sm"
                          className="min-h-[44px] flex-1 sm:min-h-0"
                          disabled={isBusy}
                          onClick={() => handleDecide(rsvp, "accept")}
                        >
                          ПРИНЯТЬ
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="min-h-[44px] flex-1 sm:min-h-0"
                          disabled={isBusy}
                          onClick={() => handleDecide(rsvp, "decline")}
                        >
                          ОТКЛОНИТЬ
                        </Button>
                      </>
                    ) : null}
                  </div>
                </div>

                {/* Mobile card */}
                <div className="sm:hidden">
                  <div className="mb-[3px] flex items-baseline justify-between gap-[8px]">
                    <span className="text-[12px] font-bold">
                      {rsvp.applicant?.name || "Участник"}
                    </span>
                    <span className="cap font-mono shrink-0">
                      {formatRelativeRu(rsvp.createdAt)}
                    </span>
                  </div>
                  <p className="cap mb-[6px]">{applicationMeta(rsvp.applicationAnswer, rsvp.status)}</p>
                  {rowErrors[rsvp.id] ? (
                    <p className="mb-[6px] text-[10px] text-signal">{rowErrors[rsvp.id]}</p>
                  ) : null}
                  {isPending ? (
                    <div className="flex gap-[6px]">
                      <Button
                        size="sm"
                        className="min-h-[44px] flex-1"
                        disabled={isBusy}
                        onClick={() => handleDecide(rsvp, "accept")}
                      >
                        ПРИНЯТЬ
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="min-h-[44px] flex-1"
                        disabled={isBusy}
                        onClick={() => handleDecide(rsvp, "decline")}
                      >
                        ОТКЛОНИТЬ
                      </Button>
                    </div>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {/* Bulk bar — desktop only (O4 mobile mock: per-row actions, no multi-select) */}
      {tab === "applied" ? (
        <div className="sticky bottom-0 hidden items-center justify-between border-t border-on-surface bg-paper px-[20px] py-[9px] sm:flex">
          <span className="cap">Выбрано: {selected.size}</span>
          <Button
            size="sm"
            className="px-[16px]"
            disabled={selected.size === 0 || bulkBusy}
            onClick={handleBulkAccept}
          >
            ПРИНЯТЬ ВЫБРАННЫЕ
          </Button>
        </div>
      ) : null}
    </div>
  );
}
