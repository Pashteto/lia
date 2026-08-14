"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { adminShortId } from "@/lib/admin-id";
import { nextQueueIndex } from "@/lib/admin-queue";
import {
  REJECT_REASON_CHIPS,
  concatenateReasons,
  revisionReason,
} from "@/lib/admin-reject-reasons";
import { submittedAgoRu } from "@/lib/admin-copy";
import { formatShortDate } from "@/lib/format";
import { isLikelyTestContent } from "@/lib/admin-test-heuristic";
import {
  type AdminEvent,
  approveEvent,
  fetchEventWithAuth,
  listModerationEvents,
  reinstateEvent,
  takedownEvent,
} from "@/lib/api";
import { cn } from "@/lib/cn";
import { formatModuleDate, formatPrice } from "@/lib/format";
import type { LiaEvent } from "@/lib/types";

type Filter = "waiting" | "all" | "links";

function mergeQueue(published: AdminEvent[], rejected: AdminEvent[]): AdminEvent[] {
  const seen = new Set<string>();
  const out: AdminEvent[] = [];
  for (const e of [...published, ...rejected]) {
    if (seen.has(e.id)) continue;
    seen.add(e.id);
    out.push(e);
  }
  return out;
}

/** Best-effort hostname for the domain caption above the raw URL text. Never
 * throws on a malformed URL — the admin still gets the raw text either way. */
function externalUrlDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return "";
  }
}

function ModerationSkeleton() {
  return (
    <div className="grid min-h-[360px] grid-cols-[250px_1fr] max-[899px]:grid-cols-1">
      <div className="flex flex-col gap-[8px] border-r border-paper p-[14px] max-[899px]:border-r-0 max-[899px]:border-b">
        <Skeleton className="h-[28px] w-full" />
        <Skeleton className="h-[64px] w-full" />
        <Skeleton className="h-[64px] w-full" />
        <Skeleton className="h-[64px] w-full" />
      </div>
      <div className="flex flex-col gap-[8px] p-[14px]">
        <Skeleton className="h-[120px] w-full" />
        <Skeleton className="h-[48px] w-full" />
        <Skeleton className="h-[80px] w-full" />
      </div>
    </div>
  );
}

/** A2 · Модерация событий — queue + record conveyor over published/rejected APIs. */
export function AdminModeration() {
  const [filter, setFilter] = useState<Filter>("waiting");
  const [queue, setQueue] = useState<AdminEvent[]>([]);
  const [waitingCount, setWaitingCount] = useState(0);
  const [linksCount, setLinksCount] = useState(0);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [reasons, setReasons] = useState<Set<string>>(new Set());
  const [detail, setDetail] = useState<LiaEvent | null>(null);
  const [detailForId, setDetailForId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        if (filter === "waiting") {
          const published = await listModerationEvents("published");
          if (cancelled) return;
          setWaitingCount(published.length);
          setQueue(published);
          setSelectedId((prev) => {
            if (prev && published.some((e) => e.id === prev)) return prev;
            return published[0]?.id ?? null;
          });
        } else if (filter === "links") {
          const pending = await listModerationEvents("pending_review");
          if (cancelled) return;
          setLinksCount(pending.length);
          setQueue(pending);
          setSelectedId((prev) => {
            if (prev && pending.some((e) => e.id === prev)) return prev;
            return pending[0]?.id ?? null;
          });
        } else {
          const [published, rejected] = await Promise.all([
            listModerationEvents("published"),
            listModerationEvents("rejected"),
          ]);
          if (cancelled) return;
          const merged = mergeQueue(published, rejected);
          setWaitingCount(published.length);
          setQueue(merged);
          setSelectedId((prev) => {
            if (prev && merged.some((e) => e.id === prev)) return prev;
            return merged[0]?.id ?? null;
          });
        }
        setError(false);
        setReasons(new Set());
        setActionError("");
      } catch {
        if (!cancelled) {
          setQueue([]);
          setSelectedId(null);
          setError(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [filter, tick]);

  useEffect(() => {
    if (!selectedId) return;
    let cancelled = false;
    fetchEventWithAuth(selectedId)
      .then((ev) => {
        if (!cancelled) {
          setDetail(ev);
          setDetailForId(selectedId);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setDetail(null);
          setDetailForId(selectedId);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  const detailReady = selectedId != null && detailForId === selectedId;
  const detailLoading = selectedId != null && !detailReady;
  const shownDetail = detailReady ? detail : null;
  function selectRow(id: string) {
    setSelectedId(id);
    setReasons(new Set());
    setActionError("");
  }

  function toggleReason(label: string) {
    setReasons((prev) => {
      const next = new Set(prev);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
    setActionError("");
  }

  function advanceAfterRemoval(removedId: string) {
    const idx = queue.findIndex((e) => e.id === removedId);
    const next = queue.filter((e) => e.id !== removedId);
    const nextIdx = nextQueueIndex(idx < 0 ? 0 : idx, next.length);
    const removed = queue.find((e) => e.id === removedId);
    setQueue(next);
    if (filter === "waiting") setWaitingCount(next.length);
    else if (filter === "links") setLinksCount(next.length);
    else if (removed?.status === "published") setWaitingCount((c) => Math.max(0, c - 1));
    setSelectedId(nextIdx >= 0 ? next[nextIdx]!.id : null);
    setReasons(new Set());
    setActionError("");
  }

  function advanceWithoutRemoval(currentId: string) {
    const idx = queue.findIndex((e) => e.id === currentId);
    setReasons(new Set());
    setActionError("");
    if (idx < 0) return;
    if (idx + 1 < queue.length) {
      setSelectedId(queue[idx + 1]!.id);
    }
  }

  async function onApprove() {
    const row = queue.find((e) => e.id === selectedId);
    if (!row || busy) return;
    setBusy(true);
    setActionError("");
    try {
      if (row.status === "pending_review") {
        await approveEvent(row.id);
        advanceAfterRemoval(row.id);
      } else if (row.status === "rejected") {
        await reinstateEvent(row.id);
        advanceAfterRemoval(row.id);
      } else {
        advanceWithoutRemoval(row.id);
      }
    } catch {
      setActionError("Не удалось одобрить");
    } finally {
      setBusy(false);
    }
  }

  async function onReject() {
    if (!selectedId || busy) return;
    if (reasons.size < 1) {
      setActionError("Выберите причину");
      return;
    }
    setBusy(true);
    try {
      await takedownEvent(selectedId, concatenateReasons([...reasons]));
      advanceAfterRemoval(selectedId);
    } catch {
      setActionError("Не удалось отклонить");
    } finally {
      setBusy(false);
    }
  }

  async function onRevision() {
    if (!selectedId || busy) return;
    if (reasons.size < 1) {
      setActionError("Выберите причину");
      return;
    }
    setBusy(true);
    try {
      await takedownEvent(selectedId, revisionReason([...reasons]));
      advanceAfterRemoval(selectedId);
    } catch {
      setActionError("Не удалось отправить на доработку");
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <ModerationSkeleton />;

  if (error) {
    return (
      <EmptyState
        title="Не удалось загрузить очередь"
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

  const selected = queue.find((e) => e.id === selectedId) ?? null;
  const isPendingReview = selected?.status === "pending_review";
  const coverSrc = shownDetail?.coverUrl || selected?.cover_url;
  // When the event was PUBLISHED — the queue is post-hoc, so that is what
  // «подано» means here. It used to read starts_at, i.e. when the event happens.
  const submittedIso = selected?.published_at;
  const orgName =
    shownDetail?.organizer?.name?.trim() || selected?.organizer_name?.trim() || "—";
  const categoryLabel = shownDetail?.categories?.[0]?.label ?? "—";
  const titleText = shownDetail?.title || selected?.title || "—";
  const testTitle = isLikelyTestContent(titleText);

  return (
    <div className="grid min-h-[360px] grid-cols-[250px_1fr] max-[899px]:grid-cols-1">
      {/* Queue — filter chips stay mounted even when empty so staff can switch to «Все» */}
      <div className="flex flex-col border-r border-paper max-[899px]:border-r-0 max-[899px]:border-b">
        <div className="flex gap-[5px] border-b border-paper px-[14px] py-[9px]">
          <Chip
            variant={filter === "waiting" ? "dark-active" : "dark-muted"}
            onClick={() => {
              if (filter === "waiting") return;
              setLoading(true);
              setFilter("waiting");
            }}
          >
            Ждут · {waitingCount}
          </Chip>
          <Chip
            variant={filter === "all" ? "dark-active" : "dark-muted"}
            onClick={() => {
              if (filter === "all") return;
              setLoading(true);
              setFilter("all");
            }}
          >
            Все
          </Chip>
          <Chip
            variant={filter === "links" ? "dark-active" : "dark-muted"}
            onClick={() => {
              if (filter === "links") return;
              setLoading(true);
              setFilter("links");
            }}
          >
            Ссылки · {linksCount}
          </Chip>
        </div>
        {/* Capped on mobile so the record card stays within one screen. */}
        <div className="min-h-0 flex-1 overflow-y-auto max-[899px]:max-h-[220px]">
          {queue.map((event) => {
            const selectedRow = event.id === selectedId;
            const test = isLikelyTestContent(event.title);
            return (
              <button
                key={event.id}
                type="button"
                title={event.id}
                data-id={event.id}
                onClick={() => selectRow(event.id)}
                className={cn(
                  "swiss-focus block w-full border-b border-rule-inner px-[14px] py-[10px] text-left transition-colors duration-[120ms] ease-linear",
                  selectedRow
                    ? "bg-paper text-ink"
                    : "bg-transparent text-on-surface hover:bg-surface-head",
                )}
              >
                <div className="mb-[3px] flex items-baseline justify-between gap-[8px]">
                  <span
                    className={cn(
                      "font-mono text-[9.5px]",
                      selectedRow ? "text-ink/55" : "text-muted-2",
                    )}
                  >
                    {adminShortId(event.id)}
                  </span>
                  <span
                    className={cn(
                      "cap shrink-0 font-mono",
                      selectedRow ? "text-ink/55" : undefined,
                    )}
                  >
                    {formatShortDate(event.starts_at)}
                  </span>
                </div>
                <div
                  className={cn(
                    "text-[11.5px] font-bold leading-[1.1]",
                    test && "text-signal",
                  )}
                >
                  {event.title}
                </div>
                <div
                  className={cn(
                    "cap mt-[3px]",
                    selectedRow ? "text-ink/55" : undefined,
                  )}
                >
                  {event.organizer_name?.trim() || "—"}
                  {filter === "all" && event.status === "rejected" ? " · снято" : null}
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Record — EmptyState only replaces this pane when queue is empty */}
      <div className="flex min-w-0 flex-col">
        {queue.length === 0 ? (
          <EmptyState
            numeral="00"
            title="Очередь пуста"
            text="Сейчас нет событий на проверку."
            actions={
              <Link
                href="/admin"
                className="swiss-focus inline-flex min-h-[44px] items-center justify-center bg-paper px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink"
              >
                К обзору
              </Link>
            }
          />
        ) : !selected ? (
          <p className="cap px-[16px] py-[14px]">Выберите событие</p>
        ) : (
          <>
            <div className="grid flex-none grid-cols-2 border-b border-paper">
              <div className="relative h-[120px] overflow-hidden border-r border-paper bg-surface-head">
                {coverSrc ? (
                  // eslint-disable-next-line @next/next/no-img-element -- admin cover URLs are API-hosted arbitrary keys
                  <img src={coverSrc} alt="" className="h-full w-full object-cover" />
                ) : (
                  <span className="cap absolute inset-0 flex items-center justify-center px-[12px] text-center">
                    Обложка на проверке
                  </span>
                )}
              </div>
              <div className="px-[16px] py-[14px]">
                <div className="cap mb-[5px]">
                  {adminShortId(selected.id)} · {submittedAgoRu(submittedIso)}
                </div>
                <h2
                  className={cn(
                    "mb-[6px] text-[20px] font-black leading-[1.05] tracking-[-0.02em]",
                    testTitle && "text-signal",
                  )}
                >
                  {titleText}
                </h2>
                <div className="cap">
                  {orgName}
                  {shownDetail?.organizer?.verified ? " ✓" : ""} · {categoryLabel}
                </div>
              </div>
            </div>

            {detailLoading ? (
              <div className="flex flex-col gap-[8px] p-[14px]">
                <Skeleton className="h-[48px] w-full" />
                <Skeleton className="h-[64px] w-full" />
              </div>
            ) : (
              <>
                <CellStrip
                  cols={4}
                  className="flex-none [&>*+*]:border-rule-inner"
                >
                  <Cell
                    caption="Дата"
                    value={
                      shownDetail
                        ? formatModuleDate(shownDetail.startsAt, shownDetail.endsAt)
                        : formatShortDate(selected.starts_at)
                    }
                    valueClassName="text-[11px]"
                  />
                  <Cell
                    caption="Цена"
                    value={shownDetail ? formatPrice(shownDetail) : "—"}
                    valueClassName="text-[11px]"
                  />
                  <Cell
                    caption="Мест"
                    value={
                      shownDetail?.capacity != null
                        ? String(shownDetail.capacity)
                        : "—"
                    }
                    mono
                    valueClassName="text-[11px]"
                  />
                  <Cell
                    caption="Формат"
                    value={
                      shownDetail
                        ? shownDetail.format === "online"
                          ? "Онлайн"
                          : "Очно"
                        : "—"
                    }
                    valueClassName="text-[11px]"
                  />
                </CellStrip>

                <div className="flex-none border-b border-rule-inner px-[16px] py-[12px]">
                  <div className="cap mb-[5px]">Описание</div>
                  <p className="line-clamp-4 text-[11.5px] leading-[1.45] text-text-dim">
                    {shownDetail?.description?.trim() || "—"}
                  </p>
                </div>
              </>
            )}
            <div className="flex-1 px-[16px] py-[12px]">
              {isPendingReview ? (
                <>
                  <p className="cap mb-[6px] text-muted-2">
                    Ссылка на регистрацию не в белом списке — событие ждёт публикации.
                  </p>
                  {selected.external_registration_url ? (
                    <div className="border border-rule-inner px-[10px] py-[8px]">
                      <div className="cap mb-[3px] text-muted-2">
                        {externalUrlDomain(selected.external_registration_url) || "домен неизвестен"}
                      </div>
                      {/* Deliberately not a link — this is the admin's judgment
                          target, not a place to click through to (avoids an
                          accidental visit to an unvetted, possibly untrusted URL). */}
                      <p className="break-all text-[11px] leading-[1.4] text-ink">
                        {selected.external_registration_url}
                      </p>
                    </div>
                  ) : null}
                </>
              ) : (
                <>
                  <div className="cap mb-[6px]">Причина отклонения</div>
                  <div className="flex flex-wrap gap-[5px]">
                    {REJECT_REASON_CHIPS.map((label) => {
                      const on = reasons.has(label);
                      return (
                        <Chip
                          key={label}
                          variant={on ? "dark-active" : "dark-muted"}
                          onClick={() => toggleReason(label)}
                          aria-pressed={on}
                        >
                          {label}
                        </Chip>
                      );
                    })}
                  </div>
                </>
              )}
              {actionError ? (
                <p className="mt-[8px] text-[11.5px] font-bold text-signal">
                  {actionError}
                </p>
              ) : null}
            </div>

            <div className="flex flex-none gap-[8px] border-t border-paper px-[16px] py-[11px]">
              <Button
                variant="inverted"
                type="button"
                disabled={busy}
                onClick={onApprove}
                className="min-h-[44px] flex-1 text-[10px]"
              >
                {isPendingReview ? "ОПУБЛИКОВАТЬ" : "ОДОБРИТЬ"}
              </Button>
              {isPendingReview ? null : (
                <>
                  <Button
                    variant="destructive"
                    type="button"
                    disabled={busy}
                    onClick={onReject}
                    className="min-h-[44px] flex-1 text-[10px]"
                  >
                    ОТКЛОНИТЬ
                  </Button>
                  <Button
                    variant="dark-ghost"
                    type="button"
                    disabled={busy}
                    onClick={onRevision}
                    className="min-h-[44px] flex-none px-[16px] text-[10px]"
                  >
                    НА ДОРАБОТКУ
                  </Button>
                </>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
