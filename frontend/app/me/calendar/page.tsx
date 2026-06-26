"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { Segmented } from "@/components/ui/Segmented";
import { fetchCalendar } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { cn } from "@/lib/cn";
import {
  WEEKDAY_LABELS,
  addDays,
  civilKey,
  dayLabel,
  monthGrid,
  monthYearLabel,
  moscowDayKey,
  moscowTime,
  sameMonth,
  shiftMonth,
  todayCivil,
  weekGrid,
} from "@/lib/calendar";
import type { CalendarEvent } from "@/lib/types";

type View = "month" | "week" | "day";

const VIEW_OPTIONS = [
  { value: "month" as const, label: "Месяц" },
  { value: "week" as const, label: "Неделя" },
  { value: "day" as const, label: "День" },
];

/** Chip colour by source. attending → accent; from a followed organizer →
 * amber; both → accent with an amber ring. Kept in sync with the legend. */
function chipClasses(ev: CalendarEvent): string {
  if (ev.attending && ev.fromFollowed)
    return "bg-accent text-white ring-2 ring-amber-400";
  if (ev.attending) return "bg-accent text-white";
  return "bg-amber-500/20 text-amber-700 dark:text-amber-300";
}

function Legend() {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[12px] text-label-secondary">
      <span className="inline-flex items-center gap-1.5">
        <span className="h-3 w-3 rounded-full bg-accent" /> Вы участвуете
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="h-3 w-3 rounded-full bg-amber-500/70" /> От подписок
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="h-3 w-3 rounded-full bg-accent ring-2 ring-amber-400" /> И то, и
        другое
      </span>
    </div>
  );
}

/** A compact event chip (month cells, week columns). */
function EventChip({ ev, withTime }: { ev: CalendarEvent; withTime?: boolean }) {
  return (
    <Link
      href={`/events/${ev.id}`}
      title={ev.title}
      className={cn(
        "block truncate rounded-md px-1.5 py-0.5 text-[11px] font-medium leading-tight transition hover:opacity-90",
        chipClasses(ev),
      )}
    >
      {withTime && (
        <span className="tabular-nums opacity-80">{moscowTime(new Date(ev.startsAt))} </span>
      )}
      {ev.title}
    </Link>
  );
}

/** A roomier agenda row (day view). */
function AgendaRow({ ev }: { ev: CalendarEvent }) {
  return (
    <Link
      href={`/events/${ev.id}`}
      className="flex items-stretch gap-3 rounded-card bg-bg-secondary p-3 shadow-card-subtle transition hover:opacity-95"
    >
      <span className={cn("w-1.5 shrink-0 rounded-full", chipClasses(ev))} />
      <div className="min-w-0 flex-1">
        <div className="text-[13px] tabular-nums text-label-secondary">
          {moscowTime(new Date(ev.startsAt))}
        </div>
        <div className="truncate text-[15px] font-semibold leading-snug">{ev.title}</div>
        {ev.venue?.name && (
          <div className="truncate text-[13px] text-label-secondary">{ev.venue.name}</div>
        )}
      </div>
    </Link>
  );
}

export default function CalendarPage() {
  const { isAuthed, ready } = useAuth();
  const [view, setView] = useState<View>("month");
  const [anchor, setAnchor] = useState<Date>(() => todayCivil());

  // Visible civil-date range for the current view.
  const { cells, rangeStart, rangeEnd } = useMemo(() => {
    if (view === "month") {
      const g = monthGrid(anchor);
      return { cells: g, rangeStart: g[0], rangeEnd: addDays(g[g.length - 1], 1) };
    }
    if (view === "week") {
      const g = weekGrid(anchor);
      return { cells: g, rangeStart: g[0], rangeEnd: addDays(g[g.length - 1], 1) };
    }
    return { cells: [anchor], rangeStart: anchor, rangeEnd: addDays(anchor, 1) };
  }, [view, anchor]);

  const {
    data: events = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["calendar", view, civilKey(rangeStart), civilKey(rangeEnd)],
    // Widen ±1 day so no visible-day event is missed at the Moscow tz boundary;
    // exact placement is done client-side by Moscow civil day.
    queryFn: () => fetchCalendar(addDays(rangeStart, -1), addDays(rangeEnd, 1)),
    enabled: ready && isAuthed,
  });

  // Bucket events by their Europe/Moscow civil day.
  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const ev of events) {
      const key = moscowDayKey(new Date(ev.startsAt));
      const list = map.get(key);
      if (list) list.push(ev);
      else map.set(key, [ev]);
    }
    for (const list of map.values())
      list.sort((a, b) => a.startsAt.localeCompare(b.startsAt));
    return map;
  }, [events]);

  function go(delta: number) {
    if (view === "month") setAnchor((a) => shiftMonth(a, delta));
    else if (view === "week") setAnchor((a) => addDays(a, delta * 7));
    else setAnchor((a) => addDays(a, delta));
  }

  if (!ready) return <div className="min-h-screen bg-bg-grouped" />;

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <Link href="/" className="inline-flex items-center text-[17px] text-accent">
          ‹ События
        </Link>
        <div className="mt-8 text-center">
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Календарь</h1>
          <p className="mt-3 text-label-secondary">
            Войдите, чтобы видеть события подписок и свои записи в календаре.
          </p>
        </div>
      </main>
    );
  }

  const periodLabel =
    view === "day" ? dayLabel(anchor) : monthYearLabel(view === "week" ? cells[3] : anchor);

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>
      <h1 className="text-[28px] font-bold tracking-[-0.022em]">Календарь</h1>

      <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Segmented options={VIEW_OPTIONS} value={view} onChange={setView} className="sm:w-72" />
        <div className="flex items-center gap-2">
          <button
            onClick={() => go(-1)}
            aria-label="Назад"
            className="rounded-control px-3 py-1.5 text-label-secondary transition hover:text-label"
          >
            ‹
          </button>
          <button
            onClick={() => setAnchor(todayCivil())}
            className="rounded-control px-3 py-1.5 text-[14px] font-medium text-accent transition hover:opacity-70"
          >
            Сегодня
          </button>
          <button
            onClick={() => go(1)}
            aria-label="Вперёд"
            className="rounded-control px-3 py-1.5 text-label-secondary transition hover:text-label"
          >
            ›
          </button>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-[20px] font-semibold capitalize">{periodLabel}</h2>
        <Legend />
      </div>

      {isLoading ? (
        <p className="mt-8 text-label-secondary">Загрузка…</p>
      ) : isError ? (
        <p className="mt-8 text-label-secondary">Не удалось загрузить календарь.</p>
      ) : (
        <div className="mt-4">
          {view === "month" && <MonthView cells={cells} anchor={anchor} byDay={byDay} />}
          {view === "week" && <WeekView cells={cells} byDay={byDay} />}
          {view === "day" && <DayView anchor={anchor} byDay={byDay} />}
        </div>
      )}
    </main>
  );
}

function MonthView({
  cells,
  anchor,
  byDay,
}: {
  cells: Date[];
  anchor: Date;
  byDay: Map<string, CalendarEvent[]>;
}) {
  const todayKey = civilKey(todayCivil());
  return (
    <div className="overflow-hidden rounded-card bg-bg-secondary shadow-card-subtle">
      <div className="grid grid-cols-7 border-b border-black/5 dark:border-white/10">
        {WEEKDAY_LABELS.map((w) => (
          <div key={w} className="px-2 py-2 text-center text-[12px] font-medium text-label-secondary">
            {w}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7">
        {cells.map((cell) => {
          const key = civilKey(cell);
          const dayEvents = byDay.get(key) ?? [];
          const muted = !sameMonth(cell, anchor);
          const isToday = key === todayKey;
          return (
            <div
              key={key}
              className={cn(
                "min-h-[92px] border-b border-r border-black/5 p-1.5 dark:border-white/10",
                muted && "bg-bg-grouped/40",
              )}
            >
              <div
                className={cn(
                  "mb-1 inline-flex h-6 w-6 items-center justify-center rounded-full text-[12px]",
                  muted ? "text-label-tertiary" : "text-label-secondary",
                  isToday && "bg-accent font-semibold text-white",
                )}
              >
                {cell.getUTCDate()}
              </div>
              <div className="flex flex-col gap-1">
                {dayEvents.slice(0, 3).map((ev) => (
                  <EventChip key={ev.id} ev={ev} />
                ))}
                {dayEvents.length > 3 && (
                  <span className="px-1 text-[11px] text-label-secondary">
                    +{dayEvents.length - 3} ещё
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function WeekView({ cells, byDay }: { cells: Date[]; byDay: Map<string, CalendarEvent[]> }) {
  const todayKey = civilKey(todayCivil());
  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
      {cells.map((cell, i) => {
        const key = civilKey(cell);
        const dayEvents = byDay.get(key) ?? [];
        const isToday = key === todayKey;
        return (
          <div key={key} className="rounded-card bg-bg-secondary p-2 shadow-card-subtle">
            <div className="mb-2 flex items-baseline gap-1.5">
              <span className="text-[12px] font-medium text-label-secondary">
                {WEEKDAY_LABELS[i]}
              </span>
              <span
                className={cn(
                  "inline-flex h-6 min-w-6 items-center justify-center rounded-full px-1 text-[13px] font-semibold",
                  isToday ? "bg-accent text-white" : "text-label",
                )}
              >
                {cell.getUTCDate()}
              </span>
            </div>
            <div className="flex flex-col gap-1">
              {dayEvents.length === 0 ? (
                <span className="text-[12px] text-label-tertiary">—</span>
              ) : (
                dayEvents.map((ev) => <EventChip key={ev.id} ev={ev} withTime />)
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function DayView({ anchor, byDay }: { anchor: Date; byDay: Map<string, CalendarEvent[]> }) {
  const dayEvents = byDay.get(civilKey(anchor)) ?? [];
  if (dayEvents.length === 0)
    return <p className="text-label-secondary">В этот день ничего нет.</p>;
  return (
    <div className="flex flex-col gap-2">
      {dayEvents.map((ev) => (
        <AgendaRow key={ev.id} ev={ev} />
      ))}
    </div>
  );
}
