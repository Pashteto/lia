"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { EventApplicationsPanel } from "@/components/EventApplicationsPanel";
import { InviteByEmailPanel } from "@/components/InviteByEmailPanel";
import { OrganizerFeedback } from "@/components/OrganizerFeedback";
import { PublishEventButton } from "@/components/PublishEventButton";
import { CancelEventButton } from "@/components/CancelEventButton";
import { AuthGate } from "@/components/ui/AuthGate";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import { fetchMyEvents } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { formatShortDate } from "@/lib/format";
import { duplicateAsDraft } from "@/lib/org-duplicate-event";
import { orgEventStatusLabel } from "@/lib/org-event-status";
import { seatsFill } from "@/lib/org-seats";
import type { LiaEvent } from "@/lib/types";

type Filter = "all" | "published" | "pending_review" | "draft" | "cancelled";

/** Maps the ?status= query value to a Filter; anything unknown → "all". */
export function parseStatusParam(raw: string | null): Filter {
  const valid: Filter[] = ["all", "published", "pending_review", "draft", "cancelled"];
  return valid.includes(raw as Filter) ? (raw as Filter) : "all";
}

const FILTERS: {
  key: Filter;
  label: string;
  shortLabel: string;
  signal?: boolean;
}[] = [
  { key: "all", label: "Все", shortLabel: "Все" },
  { key: "published", label: "Опубликовано", shortLabel: "Опубл" },
  { key: "pending_review", label: "Модерация", shortLabel: "Мод", signal: true },
  { key: "draft", label: "Черновики", shortLabel: "Черн" },
  { key: "cancelled", label: "Отменённые", shortLabel: "Отм" },
];

function matchesFilter(event: LiaEvent, filter: Filter): boolean {
  if (filter === "all") return true;
  return event.status === filter;
}

function venueCaption(event: LiaEvent): string {
  if (event.venue?.name) return event.venue.name;
  if (event.format === "online") return "Онлайн";
  return "—";
}

function CreateLink({ className }: { className?: string }) {
  return (
    <Link
      href="/events/new"
      className={`swiss-focus inline-flex items-center justify-center bg-ink px-[16px] py-[8px] text-[9px] font-bold uppercase tracking-[0.07em] text-white transition-colors duration-[120ms] ease-linear hover:bg-black ${className ?? ""}`}
    >
      + СОЗДАТЬ
    </Link>
  );
}

function RowSkeleton() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 4 }, (_, i) => (
        <Skeleton key={i} className="h-[56px] w-full border-x-0 border-t-0" />
      ))}
    </div>
  );
}

function SeatsCell({ event, compact }: { event: LiaEvent; compact?: boolean }) {
  const fill = seatsFill(event);
  return (
    <div className={compact ? "flex min-w-0 flex-1 items-center gap-[8px]" : undefined}>
      <span className={`font-mono font-bold ${compact ? "text-[9.5px]" : "text-[11px]"}`}>
        {fill?.label ?? "—"}
      </span>
      {fill ? (
        <ProgressBar
          value={fill.filled}
          max={fill.capacity}
          thin
          className={compact ? "mt-0 flex-1" : "mt-[4px]"}
        />
      ) : compact ? (
        <div className="h-[5px] flex-1 border border-on-surface" aria-hidden />
      ) : (
        <div className="mt-[4px] h-[5px] border border-on-surface" aria-hidden />
      )}
    </div>
  );
}

function OverflowExtras({
  event,
  onDuplicate,
  duplicating,
}: {
  event: LiaEvent;
  onDuplicate?: () => void;
  duplicating?: boolean;
}) {
  return (
    <div className="border-b border-on-surface bg-paper px-[20px] py-[12px] max-sm:px-[14px]">
      <div className="flex flex-col gap-[10px] border border-ink p-[12px]">
        <p className="cap">Дополнительно</p>
        <div className="cap flex gap-[14px] sm:hidden">
          <Link href={`/events/${event.id}/edit`} className="swiss-focus hover-invert">
            Ред.
          </Link>
          {onDuplicate ? (
            <button
              type="button"
              onClick={onDuplicate}
              disabled={duplicating}
              className="swiss-focus cursor-pointer uppercase tracking-[0.13em] hover-invert disabled:opacity-50"
            >
              {duplicating ? "…" : "Копия"}
            </button>
          ) : null}
        </div>
        {event.status === "draft" ? <PublishEventButton eventId={event.id} /> : null}
        {event.signupMode === "application" ? (
          <div>
            <p className="cap mb-[6px]">Заявки</p>
            <EventApplicationsPanel
              eventId={event.id}
              event={{
                title: event.title,
                startsAt: event.startsAt,
                capacity: event.capacity,
                seatsRemaining: event.seatsRemaining,
                signupMode: event.signupMode,
              }}
              hideContext
            />
          </div>
        ) : null}
        {event.status === "published" ? (
          <div>
            <p className="cap mb-[6px]">Пригласить</p>
            <InviteByEmailPanel eventId={event.id} />
          </div>
        ) : null}
        {event.status === "published" ? (
          <div>
            <p className="cap mb-[6px]">Отзывы</p>
            <OrganizerFeedback eventId={event.id} />
          </div>
        ) : null}
        {event.status === "draft" || event.status === "published" ? (
          <CancelEventButton eventId={event.id} />
        ) : null}
        {event.status !== "draft" &&
        event.status !== "published" &&
        event.signupMode !== "application" ? (
          <p className="text-[11px] text-text-dim">Нет дополнительных действий для этого статуса.</p>
        ) : null}
      </div>
    </div>
  );
}

function EventRow({
  event,
  expanded,
  onToggleExpand,
  onDuplicate,
  duplicating,
}: {
  event: LiaEvent;
  expanded: boolean;
  onToggleExpand: () => void;
  onDuplicate: () => void;
  duplicating: boolean;
}) {
  const statusLabel = orgEventStatusLabel(event.status);

  return (
    <div className="border-b border-on-surface">
      {/* Desktop — 56px 1fr 96px 110px 92px */}
      <div className="hidden grid-cols-[56px_1fr_96px_110px_92px] items-center sm:grid">
        <span className="border-r border-on-surface px-[6px] py-[10px] text-center font-mono text-[10px] font-bold">
          {formatShortDate(event.startsAt)}
        </span>
        <span className="border-r border-on-surface px-[12px] py-[9px]">
          <span className="block text-[12.5px] font-bold leading-[1.1]">{event.title}</span>
          <span className="cap mt-[2px] block">{venueCaption(event)}</span>
        </span>
        <span className="border-r border-on-surface px-[8px] py-[9px]">
          <SeatsCell event={event} />
        </span>
        <span className="flex items-center justify-start border-r border-on-surface px-[8px] py-[9px]">
          <StatusChip status={statusLabel} className="px-[6px] py-[3px] text-[8px]" />
        </span>
        <span className="cap flex flex-col items-start gap-[2px] px-[8px] py-[9px] leading-[1.7]">
          <Link href={`/events/${event.id}/edit`} className="swiss-focus hover-invert">
            Ред.
          </Link>
          <button
            type="button"
            onClick={onDuplicate}
            disabled={duplicating}
            className="swiss-focus cursor-pointer uppercase tracking-[0.13em] hover-invert disabled:opacity-50"
          >
            {duplicating ? "…" : "Копия"}
          </button>
          <button
            type="button"
            onClick={onToggleExpand}
            aria-expanded={expanded}
            aria-label="Дополнительные действия"
            className="swiss-focus min-h-[28px] cursor-pointer tracking-[0.2em] hover-invert"
          >
            ···
          </button>
        </span>
      </div>

      {/* Mobile — stacked */}
      <div className="px-[14px] py-[10px] sm:hidden">
        <div className="mb-[4px] flex items-baseline justify-between gap-[8px]">
          <span className="font-mono text-[10px] font-bold">{formatShortDate(event.startsAt)}</span>
          <StatusChip status={statusLabel} className="px-[5px] py-[2px] text-[7px]" />
        </div>
        <p className="mb-[5px] text-[12px] font-bold leading-[1.1]">{event.title}</p>
        <SeatsCell event={event} compact />
        {/* Mobile mock is date/status/title/seats; actions via ··· only */}
        <div className="cap mt-[8px] flex min-h-[44px] items-center">
          <button
            type="button"
            onClick={onToggleExpand}
            aria-expanded={expanded}
            aria-label="Действия"
            className="swiss-focus cursor-pointer tracking-[0.2em] hover-invert"
          >
            ···
          </button>
        </div>
      </div>

      {expanded ? (
        <OverflowExtras
          event={event}
          onDuplicate={onDuplicate}
          duplicating={duplicating}
        />
      ) : null}
    </div>
  );
}

export function MyEventsBrowse() {
  const { isAuthed, ready } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const qc = useQueryClient();
  const filter = parseStatusParam(searchParams.get("status"));
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [dupId, setDupId] = useState<string | null>(null);

  const authed = ready && isAuthed;

  const events = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: authed,
  });

  const duplicate = useMutation({
    mutationFn: duplicateAsDraft,
    onSuccess: (id) => {
      void qc.invalidateQueries({ queryKey: ["my-events"] });
      router.push(`/events/${id}/edit`);
    },
    onSettled: () => setDupId(null),
  });

  if (!ready) {
    return (
      <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
        <Skeleton className="h-[56px] w-full border-x-0 border-t-0" />
        <RowSkeleton />
      </main>
    );
  }

  if (!isAuthed) {
    return (
      <AuthGate
        title="Войдите, чтобы увидеть созданные вами события"
        reassurance="Лента и карта доступны без входа."
      />
    );
  }

  if (events.isLoading) {
    return (
      <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
        <Skeleton className="h-[56px] w-full border-x-0 border-t-0" />
        <Skeleton className="h-[44px] w-full border-x-0 border-t-0" />
        <RowSkeleton />
      </main>
    );
  }

  const list = events.data ?? [];
  const counts: Record<Filter, number> = {
    all: list.length,
    published: list.filter((e) => e.status === "published").length,
    pending_review: list.filter((e) => e.status === "pending_review").length,
    draft: list.filter((e) => e.status === "draft").length,
    cancelled: list.filter((e) => e.status === "cancelled").length,
  };
  const filtered = list.filter((e) => matchesFilter(e, filter));

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      {/* Title bar — desktop only; mobile uses AppHeader caption + actions «+» */}
      <div className="hidden items-baseline justify-between border-b border-ink px-[20px] py-[14px] sm:flex">
        <h1 className="text-[26px] font-black leading-[0.94] tracking-[-0.03em]">
          Мои события
        </h1>
        <CreateLink />
      </div>

      {/* Filter chips */}
      <div className="flex gap-[6px] overflow-x-auto border-b border-ink px-[20px] py-[9px] max-sm:gap-[5px] max-sm:px-[14px] max-sm:py-[8px] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {FILTERS.map((f) => {
          const active = filter === f.key;
          const variant = active ? "active" : f.signal ? "signal" : "default";
          return (
            <Chip
              key={f.key}
              variant={variant}
              onClick={() =>
                router.replace(f.key === "all" ? "/events/mine" : `/events/mine?status=${f.key}`)
              }
              aria-pressed={active}
              className="min-h-[44px] max-sm:text-[8px]"
            >
              <span className="max-sm:hidden">{f.label}</span>
              <span className="sm:hidden">{f.shortLabel}</span>
              {" · "}
              <span className="font-mono">{counts[f.key]}</span>
            </Chip>
          );
        })}
      </div>

      {events.isError ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить события"
          text="Проверьте соединение и попробуйте обновить страницу."
        />
      ) : list.length === 0 ? (
        <EmptyState
          numeral="00"
          title="Событий пока нет"
          text="Создайте первое событие — оно появится в этой таблице."
          actions={
            <Link
              href="/events/new"
              className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black"
            >
              СОЗДАТЬ СОБЫТИЕ
            </Link>
          }
        />
      ) : (
        <>
          {/* Table head — desktop */}
          <div className="hidden grid-cols-[56px_1fr_96px_110px_92px] border-b border-on-surface bg-table-head sm:grid">
            <span className="cap border-r border-on-surface px-[8px] py-[6px] text-center">Дата</span>
            <span className="cap border-r border-on-surface px-[12px] py-[6px]">Событие</span>
            <span className="cap border-r border-on-surface px-[8px] py-[6px]">Записи</span>
            <span className="cap border-r border-on-surface px-[8px] py-[6px]">Статус</span>
            <span className="cap px-[8px] py-[6px]">Действия</span>
          </div>

          {filtered.length === 0 ? (
            <EmptyState
              numeral="00"
              title="Нет событий в этом фильтре"
              text="Переключите вкладку или создайте новое событие."
            />
          ) : (
            <div className="flex flex-col">
              {filtered.map((event) => (
                <EventRow
                  key={event.id}
                  event={event}
                  expanded={expandedId === event.id}
                  onToggleExpand={() =>
                    setExpandedId((id) => (id === event.id ? null : event.id))
                  }
                  duplicating={dupId === event.id && duplicate.isPending}
                  onDuplicate={() => {
                    setDupId(event.id);
                    duplicate.mutate(event);
                  }}
                />
              ))}
            </div>
          )}

          {duplicate.isError ? (
            <p className="border-t border-on-surface px-[20px] py-[10px] text-[11px] text-signal">
              Не удалось создать копию. Попробуйте ещё раз.
            </p>
          ) : null}
        </>
      )}
    </main>
  );
}
