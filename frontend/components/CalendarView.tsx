"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { AuthGate } from "@/components/ui/AuthGate";
import { AuthNavControl } from "@/components/ui/AuthNavControl";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { eventCalendarUrl, fetchCalendar, getCategories } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import {
  WEEKDAY_LABELS,
  addDays,
  civilKey,
  eventDayKeys,
  monthCaption,
  monthGridTrimmed,
  monthTitle,
  moscowTime,
  sameMonth,
  selectedDayLabel,
  selectedDayLabelLong,
  shiftMonth,
  todayCivil,
} from "@/lib/calendar";
import { categoryNumeral } from "@/lib/category-numerals";
import { cn } from "@/lib/cn";
import type { CalendarEvent } from "@/lib/types";

type Mode = "month" | "list";

/** One agenda block: mono time, «Записан» chip when the user attends, title, venue. */
/** Legend under the month grid: the black day fill was unexplained (QA 14.08,
 * finding 8). Exported for tests. */
export function CalendarLegend() {
  return (
    <p className="cap flex items-center gap-[12px] border-t border-rule-grid px-[14px] py-[6px]">
      <span className="flex items-center gap-[5px]">
        <span aria-hidden className="inline-block h-[8px] w-[8px] bg-ink" />
        есть события
      </span>
      <span className="flex items-center gap-[5px]">
        <span aria-hidden className="inline-block h-[8px] w-[8px] outline outline-2 -outline-offset-1 outline-ink" />
        выбранный день
      </span>
    </p>
  );
}

function AgendaBlock({
  event,
  categories,
  dense,
}: {
  event: CalendarEvent;
  categories: ReadonlyArray<{ slug: string }>;
  dense?: boolean;
}) {
  const cat = event.categories[0];
  return (
    <div className={cn("border-b border-on-surface", dense ? "px-[14px] py-[9px]" : "px-[14px] py-[11px]")}>
      <div className="mb-[4px] flex items-baseline justify-between gap-[8px] max-sm:mb-[3px]">
        <span className="font-mono text-[10px] font-bold max-sm:text-[9px]">
          {moscowTime(new Date(event.startsAt))}
        </span>
        {event.attending ? (
          <Chip as="span" className="px-[6px] py-[2px] text-[8px]">
            Записан
          </Chip>
        ) : cat ? (
          <span className="font-mono text-[9px] font-bold">{categoryNumeral(cat.slug, categories)}</span>
        ) : null}
      </div>
      <Link
        href={`/events/${event.id}`}
        className="swiss-focus block text-[12.5px] font-bold leading-[1.1] underline-offset-2 hover:underline max-sm:text-[11.5px]"
      >
        {event.title}
      </Link>
      {event.venue?.name ? <p className="cap mt-[4px]">{event.venue.name}</p> : null}
    </div>
  );
}

export function CalendarView() {
  const { isAuthed, ready } = useAuth();
  const [mode, setMode] = useState<Mode>("month");
  const [anchor, setAnchor] = useState<Date>(() => todayCivil());
  const [selected, setSelected] = useState<Date>(() => todayCivil());

  const authed = ready && isAuthed;
  const cells = useMemo(() => monthGridTrimmed(anchor), [anchor]);
  const rangeStart = cells[0];
  const rangeEnd = addDays(cells[cells.length - 1], 1);

  const categoriesQuery = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
    staleTime: 60 * 60 * 1000,
  });
  const categories = categoriesQuery.data ?? [];

  const {
    data: events = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ["calendar", "month", civilKey(rangeStart), civilKey(rangeEnd)],
    // Widen ±1 day so no visible-day event is missed at the Moscow tz boundary;
    // exact placement is done client-side by Moscow civil day.
    queryFn: () => fetchCalendar(addDays(rangeStart, -1), addDays(rangeEnd, 1)),
    enabled: authed,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    placeholderData: (prev) => prev,
  });

  // Bucket events by every Europe/Moscow civil day they span.
  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const ev of events) {
      for (const key of eventDayKeys(ev.startsAt, ev.endsAt)) {
        const list = map.get(key) ?? [];
        list.push(ev);
        map.set(key, list);
      }
    }
    for (const list of map.values()) list.sort((a, b) => a.startsAt.localeCompare(b.startsAt));
    return map;
  }, [events]);

  const selectedKey = civilKey(selected);
  const dayEvents = byDay.get(selectedKey) ?? [];
  const icsEvent = dayEvents.find((e) => e.attending) ?? dayEvents[0];
  const monthEventDays = cells.filter(
    (cell) => sameMonth(cell, anchor) && (byDay.get(civilKey(cell))?.length ?? 0) > 0,
  );

  function go(delta: number) {
    const next = shiftMonth(anchor, delta);
    setAnchor(next);
    // Keep the selection inside the visible month so the rail is never empty
    // for an invisible day.
    setSelected(next);
  }

  const header = (
    <AppHeader nav={USER_NAV} actions={<AuthNavControl />} mobileCaption={monthCaption(anchor)} />
  );

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
          title="Войдите, чтобы видеть свой календарь"
          reassurance="Лента и карта доступны без входа."
        />
      </>
    );
  }

  return (
    <>
      {header}
      <main className="mx-auto flex max-w-[1360px] flex-col pb-[64px] max-sm:pb-[88px]">
        {/* Month bar */}
        <div className="flex items-baseline justify-between gap-[10px] border-b border-ink px-[20px] py-[14px] max-sm:px-[14px] max-sm:py-[11px]">
          <h1 className="text-[26px] font-black leading-[0.94] tracking-[-0.03em] max-sm:text-[20px]">
            {monthTitle(anchor)}
          </h1>
          <div className="flex shrink-0 gap-[6px]">
            <Chip variant={mode === "month" ? "active" : "default"} onClick={() => setMode("month")}>
              Месяц
            </Chip>
            <Chip variant={mode === "list" ? "active" : "default"} onClick={() => setMode("list")}>
              Список
            </Chip>
            <Chip onClick={() => go(-1)} aria-label="Предыдущий месяц">
              ←
            </Chip>
            <Chip onClick={() => go(1)} aria-label="Следующий месяц">
              →
            </Chip>
          </div>
        </div>

        {isError ? (
          <EmptyState
            numeral="!"
            title="Не удалось загрузить календарь"
            text="Проверьте соединение и попробуйте обновить страницу."
          />
        ) : isPending && events.length === 0 ? (
          <div className="grid grid-cols-7 border-b border-ink">
            {Array.from({ length: 35 }, (_, i) => (
              <Skeleton key={i} className="h-[56px] max-sm:h-[44px]" />
            ))}
          </div>
        ) : mode === "list" ? (
          /* «Список» — flat month agenda */
          monthEventDays.length === 0 ? (
            <EmptyState
              title="В этом месяце ничего нет"
              text="Записи и события ваших подписок появятся здесь."
            />
          ) : (
            <div className="flex flex-col">
              {monthEventDays.map((c) => (
                <section key={civilKey(c)}>
                  <p className="cap border-b border-on-surface bg-surface-head px-[14px] py-[6px]">
                    {selectedDayLabel(c)}
                  </p>
                  {(byDay.get(civilKey(c)) ?? []).map((ev) => (
                    <AgendaBlock key={`${civilKey(c)}-${ev.id}`} event={ev} categories={categories} dense />
                  ))}
                </section>
              ))}
            </div>
          )
        ) : (
          /* «Месяц» — grid + agenda rail */
          <div className="grid grid-cols-[1fr_230px] border-b border-ink max-md:grid-cols-1">
            <div className="flex flex-col border-r border-on-surface max-md:border-r-0">
              {/* Weekday strip */}
              <div className="grid grid-cols-7 border-b border-ink">
                {WEEKDAY_LABELS.map((w) => (
                  <span key={w} className="cap py-[6px] text-center max-sm:py-[5px] max-sm:text-[7.5px]">
                    {w}
                  </span>
                ))}
              </div>
              {/* Day grid */}
              <div className="grid grid-cols-7 auto-rows-[minmax(56px,1fr)] max-sm:auto-rows-[44px]">
                {cells.map((cell) => {
                  const key = civilKey(cell);
                  const inMonth = sameMonth(cell, anchor);
                  const cellClass =
                    "flex min-w-0 flex-col items-start border-r border-b border-rule-grid px-[7px] py-[5px] text-left";

                  // Leading/trailing cells are blanks (#ECEAE4) in the reference:
                  // context, not content. They carry no fill, no category tag and
                  // no interaction — the arrows move between months.
                  if (!inMonth) {
                    return (
                      <div key={key} className={cn(cellClass, "bg-cell-blank")} aria-hidden>
                        <span className="font-mono text-[10px] font-bold text-muted-2 max-sm:text-[9px]">
                          {cell.getUTCDate()}
                        </span>
                      </div>
                    );
                  }

                  const cellEvents = byDay.get(key) ?? [];
                  const filled = cellEvents.length > 0;
                  const cat = cellEvents[0]?.categories[0];
                  const isSelected = key === selectedKey;
                  return (
                    <button
                      key={key}
                      type="button"
                      onClick={() => setSelected(cell)}
                      aria-pressed={isSelected}
                      aria-label={`${selectedDayLabelLong(cell)}, событий: ${cellEvents.length}`}
                      className={cn(
                        cellClass,
                        "swiss-focus",
                        filled && !isSelected && "bg-ink text-paper",
                        isSelected &&
                          "bg-paper text-ink outline outline-2 -outline-offset-2 outline-ink",
                      )}
                    >
                      <span className="font-mono text-[10px] font-bold max-sm:text-[9px]">
                        {cell.getUTCDate()}
                      </span>
                      {filled && cat ? (
                        <span className="mt-auto font-mono text-[8px] font-bold tracking-[0.06em] max-sm:hidden">
                          {categoryNumeral(cat.slug, categories)}
                        </span>
                      ) : null}
                    </button>
                  );
                })}
              </div>
              <CalendarLegend />
            </div>

            {/* Agenda rail. Desktop header cell = «Выбрано» + 16px/900 short date
                (reference :329). Mobile is a single caption strip carrying the
                long weekday, with no «Выбрано» label (reference :349). */}
            <div className="flex flex-col max-md:border-t max-md:border-ink">
              <div className="border-b border-on-surface px-[14px] py-[11px] max-md:py-[9px]">
                <p className="cap max-md:hidden">Выбрано</p>
                <p className="mt-[4px] text-[16px] font-black leading-[1.25] max-md:hidden">
                  {selectedDayLabel(selected)}
                </p>
                <p className="cap md:hidden">{selectedDayLabelLong(selected)}</p>
              </div>
              {dayEvents.length === 0 ? (
                <p className="cap px-[14px] py-[16px]">В этот день ничего нет</p>
              ) : (
                dayEvents.map((ev) => (
                  <AgendaBlock key={ev.id} event={ev} categories={categories} />
                ))
              )}
              {icsEvent ? (
                <div className="mt-auto px-[14px] py-[11px]">
                  <a
                    href={eventCalendarUrl(icsEvent.id)}
                    download
                    title={icsEvent.title}
                    className="swiss-focus hover-invert block border border-ink px-[11px] py-[11px] text-center text-[11px] font-bold uppercase tracking-[0.07em]"
                  >
                    Добавить в календарь
                  </a>
                </div>
              ) : null}
            </div>
          </div>
        )}
      </main>
    </>
  );
}
