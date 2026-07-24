"use client";

import { EventCard } from "@/components/ui/EventCard";
import { FilterChip } from "@/components/ui/FilterChip";
import { SearchField } from "@/components/ui/SearchField";
import { fetchNearbyEvents, fetchPublishedEvents } from "@/lib/api";
import { FILTERS } from "@/lib/mock-events";
import type { LiaEvent } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

/**
 * Discovery feed: large title, search field, capsule filter row, event grid.
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
}: {
  initialEvents: LiaEvent[];
}) {
  const [active, setActive] = useState("all");
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
    const f = FILTERS.find((x) => x.slug === active);
    return f?.dateRange?.(new Date());
  }, [active]);

  const { data: allEvents = [], isError } = useQuery({
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
      } else if (dateRange) {
        // The backend already narrowed to the window; re-check client-side so
        // the offline mock fallback narrows too (same range → consistent).
        const t = new Date(e.startsAt).getTime();
        matchesFilter =
          t >= dateRange.from.getTime() && t < dateRange.to.getTime();
      } else {
        matchesFilter = e.categories.some((c) => c.slug === active);
      }
      const matchesQuery =
        query.trim() === "" ||
        e.title.toLowerCase().includes(query.toLowerCase()) ||
        (e.organizer?.name ?? "").toLowerCase().includes(query.toLowerCase());
      return matchesFilter && matchesQuery;
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
  }, [allEvents, active, dateRange, query, now]);

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

  return (
    <main className="mx-auto max-w-5xl px-5 pb-28 pt-6">
      <h1 className="mb-4 text-[34px] font-bold tracking-[-0.022em]">События</h1>

      <SearchField
        placeholder="Поиск по названию, месту, ведущему"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        className="mb-4"
      />

      {/* Filter row: category chips, then the location toggle set apart by a
          hairline — the divider encodes that "рядом со мной" filters by a
          different axis (distance) than the category chips beside it. */}
      <div className="-mx-5 mb-6 flex items-center gap-2 overflow-x-auto px-5 py-0.5 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {FILTERS.map((f) => (
          <FilterChip
            key={f.slug}
            label={f.label}
            active={active === f.slug}
            onClick={() => setActive(f.slug)}
          />
        ))}

        <span
          className="mx-1 h-6 w-px shrink-0 self-center bg-separator"
          aria-hidden
        />

        <FilterChip
          label={geoLoading ? "Определяем…" : "Рядом со мной"}
          active={isNearbyMode}
          disabled={geoLoading}
          onClick={isNearbyMode ? resetNearby : enableNearby}
          aria-label={isNearbyMode ? "Сбросить фильтр по расстоянию" : "Показать события рядом со мной"}
          icon={geoLoading ? <SpinnerGlyph /> : <PinGlyph />}
          trailing={isNearbyMode ? <ClearGlyph /> : undefined}
        />
      </div>

      {geoError && (
        <p className="-mt-4 mb-6 text-[13px] text-label-secondary">{geoError}</p>
      )}

      {isError && allEvents.length === 0 ? (
        <p className="py-16 text-center text-[15px] text-label-secondary">
          Не удалось загрузить события. Проверьте, что бэкенд запущен.
        </p>
      ) : displayEvents.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {displayEvents.map((e) => (
            <EventCard
              key={e.id}
              event={e}
              distanceBadge={
                isNearbyMode && e.distanceM != null ? (
                  <span className="text-[12px] text-label-secondary">
                    ≈ {(e.distanceM / 1000).toFixed(1)} км
                  </span>
                ) : undefined
              }
            />
          ))}
        </div>
      ) : (
        <p className="py-16 text-center text-[15px] text-label-secondary">
          {isNearbyMode
            ? "Событий рядом не найдено."
            : "Ничего не нашлось. Попробуйте другой фильтр."}
        </p>
      )}
    </main>
  );
}

/** Location pin — marks the near-me chip as a distance filter, not a category. */
function PinGlyph() {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      className="shrink-0"
      aria-hidden
    >
      <path
        d="M12 21s7-6.3 7-11a7 7 0 1 0-14 0c0 4.7 7 11 7 11Z"
        fill="currentColor"
        fillOpacity="0.18"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="10" r="2.5" fill="currentColor" />
    </svg>
  );
}

/** × affordance shown on the active near-me chip to signal it clears the filter. */
function ClearGlyph() {
  return (
    <svg
      width="13"
      height="13"
      viewBox="0 0 24 24"
      fill="none"
      className="-mr-1 shrink-0 opacity-80"
      aria-hidden
    >
      <path
        d="m7 7 10 10M17 7 7 17"
        stroke="currentColor"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
    </svg>
  );
}

/** Spinner shown while geolocation resolves. */
function SpinnerGlyph() {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      className="shrink-0 animate-spin motion-reduce:animate-none"
      aria-hidden
    >
      <circle
        cx="12"
        cy="12"
        r="9"
        stroke="currentColor"
        strokeWidth="2.4"
        strokeOpacity="0.25"
      />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
    </svg>
  );
}
