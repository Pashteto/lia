"use client";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { ChipRow } from "@/components/ui/ChipRow";
import { EmptyState } from "@/components/ui/EmptyState";
import { EventModule } from "@/components/ui/EventModule";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  fetchNearbyEvents,
  fetchPublishedEvents,
  type ApiCategory,
} from "@/lib/api";
import { categoryNumeral } from "@/lib/category-numerals";
import { eventToModuleProps } from "@/lib/event-module";
import { useAuth } from "@/lib/auth-context";
import { todayRange, tonightRange, weekendRange, weekRange } from "@/lib/mock-events";
import { pluralRu } from "@/lib/plural";
import { CURRENT_CITY, cityTagline } from "@/lib/city";
import type { LiaEvent } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

const TIME_FILTERS = [
  { slug: "all", label: "Все" },
  { slug: "today", label: "Сегодня", dateRange: todayRange },
  // Social frames (design review P3): «куда сегодня вечером / на этой неделе».
  { slug: "tonight", label: "Сегодня вечером", dateRange: tonightRange },
  { slug: "week", label: "На этой неделе", dateRange: weekRange },
  { slug: "weekend", label: "Выходные", dateRange: weekendRange },
  { slug: "free", label: "Бесплатно" },
] as const;

/**
 * Discovery feed: title block, Swiss filter bar, search field, event grid.
 *
 * Data comes from the backend `GET /api/v1/events?status=published`. The server
 * component fetches the initial list (SSR) and passes it as `initialEvents`;
 * TanStack Query then owns client-side refetching. Filtering/search is applied
 * client-side over the fetched list.
 *
 * When the "рядом со мной" toggle is enabled, the normal filtered list is
 * replaced with a distance-sorted list from `GET /events/nearby`. Each card
 * shows a distance badge when `distanceM` is present. Geolocation errors are
 * shown as a hint; the normal list stays visible on failure (graceful fallback).
 */
export function DiscoveryFeed({
  initialEvents,
  categories,
}: {
  initialEvents: LiaEvent[];
  categories: ApiCategory[];
}) {
  const { ready, isAuthed } = useAuth();
  const [active, setActive] = useState("all");
  const [activeCat, setActiveCat] = useState<string | null>(null);
  const [query, setQuery] = useState("");

  // Nearby state — null means "normal mode", array means "near-me mode".
  const [nearby, setNearby] = useState<LiaEvent[] | null>(null);
  const [geoError, setGeoError] = useState<string | null>(null);
  const [geoLoading, setGeoLoading] = useState(false);
  // "Now" for the upcoming/past split, captured once at mount (a lazy
  // initializer keeps Date.now() out of render — the feed doesn't need to
  // re-partition as the clock ticks).
  const [now] = useState(() => Date.now());

  // Date chips (today/weekend) resolve to a [from, to) window, computed once per
  // active-chip change so the server query and the client filter agree on it.
  const dateRange = useMemo(() => {
    const f = TIME_FILTERS.find((x) => x.slug === active);
    return f && "dateRange" in f ? f.dateRange(new Date()) : undefined;
  }, [active]);

  const { data: allEvents = [], isError, isPending } = useQuery({
    // Each window is its own cache entry; the unfiltered list keeps its SSR seed.
    queryKey: [
      "events",
      "published",
      dateRange?.from.toISOString() ?? null,
      dateRange?.to.toISOString() ?? null,
    ],
    queryFn: () => fetchPublishedEvents(dateRange?.from, dateRange?.to),
    initialData: dateRange ? undefined : initialEvents,
  });

  const events = useMemo(() => {
    const filtered = allEvents.filter((e) => {
      let matchesFilter: boolean;
      if (active === "all") {
        matchesFilter = true;
      } else if (active === "free") {
        matchesFilter = e.priceType === "free";
      } else if (dateRange) {
        // The backend already narrowed to the window; re-check client-side so
        // the offline mock fallback narrows too (same range → consistent).
        const t = new Date(e.startsAt).getTime();
        matchesFilter =
          t >= dateRange.from.getTime() && t < dateRange.to.getTime();
      } else matchesFilter = true;
      const matchesCategory =
        activeCat === null ||
        e.categories.some((c) => c.slug === activeCat);
      const matchesQuery =
        query.trim() === "" ||
        e.title.toLowerCase().includes(query.toLowerCase()) ||
        (e.organizer?.name ?? "").toLowerCase().includes(query.toLowerCase());
      return matchesFilter && matchesCategory && matchesQuery;
    });
    // Lead with upcoming events (soonest first); demote past events below them
    // (most-recent past first). The backend returns no particular order, so the
    // feed would otherwise open on stale/past events. .filter() already returned
    // a fresh array, so sorting in place doesn't mutate allEvents.
    return filtered.sort((a, b) => {
      const ta = new Date(a.startsAt).getTime();
      const tb = new Date(b.startsAt).getTime();
      const aUpcoming = ta >= now;
      const bUpcoming = tb >= now;
      if (aUpcoming !== bUpcoming) return aUpcoming ? -1 : 1;
      return aUpcoming ? ta - tb : tb - ta;
    });
  }, [allEvents, active, activeCat, dateRange, query, now]);

  const enableNearby = () => {
    if (!navigator.geolocation) {
      setGeoError("Геолокация не поддерживается этим браузером");
      return;
    }
    setGeoError(null);
    setGeoLoading(true);
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        try {
          setNearby(
            await fetchNearbyEvents(
              pos.coords.latitude,
              pos.coords.longitude,
            ),
          );
          setGeoError(null);
        } catch {
          setGeoError("Не удалось загрузить события рядом");
        } finally {
          setGeoLoading(false);
        }
      },
      (err) => {
        // Distinguish the three GeolocationPositionError codes — collapsing them
        // all to "доступ отклонён" mislabels a timeout / unavailable position
        // (e.g. OS location services off) as a permission denial.
        setGeoError(
          err.code === err.PERMISSION_DENIED
            ? "Доступ к геолокации отклонён. Разрешите его в настройках сайта."
            : err.code === err.TIMEOUT
              ? "Не удалось определить местоположение (тайм-аут). Попробуйте ещё раз."
              : "Местоположение недоступно. Проверьте, включены ли службы геолокации.",
        );
        setGeoLoading(false);
      },
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
  };

  const resetNearby = () => {
    setNearby(null);
    setGeoError(null);
    setGeoLoading(false);
  };

  // Which list to render and whether to show a distance badge per card.
  const displayEvents = nearby ?? events;
  const isNearbyMode = nearby !== null;
  const count = displayEvents.length;

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]">
      {/* Title block */}
      <div className="border-b border-ink px-[20px] py-[18px]">
        <p className="cap">
          {CURRENT_CITY.name} · <span className="font-mono">{count}</span>{" "}
          {pluralRu(count, ["событие", "события", "событий"])}
        </p>
        <h1 className="mt-[8px] max-w-[14ch] text-[38px] font-black leading-[0.94] tracking-[-0.03em] max-sm:text-[22px]">
          Лента событий
        </h1>
        {/* One-line manifest for the cold visitor (design review P3): a guest
            landing from a link should get the vibe in two seconds — the brand
            voice otherwise hides behind the login screen. */}
        {ready && !isAuthed && (
          <p className="mt-[8px] max-w-[52ch] text-[12.5px] leading-[1.45] text-text-dim">
            {cityTagline(CURRENT_CITY)}
          </p>
        )}
      </div>

      {/* Filter bar: two labelled axes — when/where left, topic right — split by
          a full-height ink rule. Topic chips carry their positional numerals
          (the Swiss rule: categories are numerals), which is what visually
          separates them from the when/where chips. */}
      <ChipRow className="border-b border-ink" innerClassName="justify-between">
        <div className="flex shrink-0 items-center gap-[6px]">
          <span aria-hidden className="mr-[4px] text-[9px] uppercase tracking-[0.12em] text-muted-2">
            Когда и где
          </span>
          {TIME_FILTERS.map((f) => (
            <Chip
              key={f.slug}
              variant={active === f.slug ? "active" : "default"}
              onClick={() => setActive(f.slug)}
            >
              {f.label}
            </Chip>
          ))}
          <Chip
            variant={isNearbyMode ? "active" : "default"}
            disabled={geoLoading}
            onClick={isNearbyMode ? resetNearby : enableNearby}
            aria-label={isNearbyMode ? "Сбросить фильтр по расстоянию" : "Показать события рядом со мной"}
          >
            {geoLoading ? "Определяем…" : "Рядом со мной"}
          </Chip>
        </div>
        <div aria-hidden className="w-px shrink-0 self-stretch bg-ink" />
        <div className="flex shrink-0 items-center gap-[6px]">
          <span aria-hidden className="mr-[4px] text-[9px] uppercase tracking-[0.12em] text-muted-2">
            Тема
          </span>
          {categories.map((c) => (
            <Chip
              key={c.id}
              variant={activeCat === c.slug ? "active" : "default"}
              onClick={() => setActiveCat(activeCat === c.slug ? null : c.slug)}
            >
              <span className="mr-[5px] font-mono">{categoryNumeral(c.slug, categories)}</span>
              {c.label}
            </Chip>
          ))}
        </div>
      </ChipRow>

      {/* Search (retained feature, Swiss field) */}
      <div className="border-b border-rule-inner px-[20px] py-[9px]">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Поиск по названию, месту, ведущему"
          aria-label="Поиск событий"
          className="swiss-focus w-full bg-transparent text-[12.5px] text-on-surface placeholder:text-field-text"
        />
      </div>

      {geoError && <p className="border-b border-rule-inner px-[20px] py-[9px] text-[11.5px] text-signal">{geoError}</p>}

      {/* Catalogue grid */}
      {isPending && allEvents.length === 0 ? (
        <div className="grid grid-cols-3 border-b border-ink max-sm:grid-cols-1">
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} className="h-[290px] max-sm:h-[66px]" />
          ))}
        </div>
      ) : isError && allEvents.length === 0 ? (
        <EmptyState
          numeral="!"
          title="Не удалось загрузить события"
          text="Проверьте соединение и попробуйте обновить страницу."
        />
      ) : count > 0 ? (
        <div className="grid auto-rows-fr grid-cols-3 border-b border-ink max-sm:grid-cols-1 [&>a]:border-b [&>a]:border-r [&>a]:border-rule-inner max-sm:[&>a]:border-r-0">
          {displayEvents.map((e) => {
            const m = eventToModuleProps(e, categories);
            return (
              <EventModule
                key={e.id}
                {...m}
                venue={
                  isNearbyMode && e.distanceM != null
                    ? `${m.venue} · ${(e.distanceM / 1000).toFixed(1)} км`
                    : m.venue
                }
              />
            );
          })}
        </div>
      ) : (
        <EmptyState
          title={isNearbyMode ? "Событий рядом нет" : "Ничего не нашлось"}
          text={isNearbyMode ? "Попробуйте расширить поиск или сбросить фильтр." : "Попробуйте другой фильтр или сбросьте поиск."}
          actions={
            <Button
              variant="ghost"
              onClick={() => {
                setActive("all");
                setActiveCat(null);
                setQuery("");
                resetNearby();
              }}
            >
              Сбросить фильтры
            </Button>
          }
        />
      )}
    </main>
  );
}
