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

  const { data: allEvents = [], isError } = useQuery({
    queryKey: ["events", "published"],
    queryFn: fetchPublishedEvents,
    initialData: initialEvents,
  });

  const events = useMemo(() => {
    return allEvents.filter((e) => {
      const matchesFilter =
        active === "all" || e.categories.some((c) => c.slug === active);
      const matchesQuery =
        query.trim() === "" ||
        e.title.toLowerCase().includes(query.toLowerCase()) ||
        (e.organizer?.name ?? "").toLowerCase().includes(query.toLowerCase());
      return matchesFilter && matchesQuery;
    });
  }, [allEvents, active, query]);

  const enableNearby = () => {
    if (!navigator.geolocation) {
      setGeoError("Геолокация недоступна");
      return;
    }
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
        }
      },
      () => setGeoError("Доступ к геолокации отклонён"),
    );
  };

  const resetNearby = () => {
    setNearby(null);
    setGeoError(null);
  };

  // Which list to render and whether to show a distance badge per card.
  const displayEvents = nearby ?? events;
  const isNearbyMode = nearby !== null;

  return (
    <main className="mx-auto max-w-3xl px-5 pb-28 pt-6">
      <h1 className="mb-4 text-[34px] font-bold tracking-[-0.022em]">События</h1>

      <SearchField
        placeholder="Поиск по названию, месту, ведущему"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        className="mb-4"
      />

      <div className="-mx-5 mb-4 flex gap-2 overflow-x-auto px-5 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {FILTERS.map((f) => (
          <FilterChip
            key={f.slug}
            label={f.label}
            active={active === f.slug}
            onClick={() => setActive(f.slug)}
          />
        ))}
      </div>

      {/* Near-me control row */}
      <div className="mb-6 flex items-center gap-3">
        {isNearbyMode ? (
          <button
            type="button"
            onClick={resetNearby}
            className="rounded-full bg-fill px-4 py-1.5 text-[14px] font-medium text-label-primary transition hover:bg-fill-secondary"
          >
            сбросить
          </button>
        ) : (
          <button
            type="button"
            onClick={enableNearby}
            className="rounded-full bg-fill px-4 py-1.5 text-[14px] font-medium text-label-primary transition hover:bg-fill-secondary"
          >
            рядом со мной
          </button>
        )}
        {geoError && (
          <span className="text-[13px] text-label-secondary">{geoError}</span>
        )}
      </div>

      {isError && allEvents.length === 0 ? (
        <p className="py-16 text-center text-[15px] text-label-secondary">
          Не удалось загрузить события. Проверьте, что бэкенд запущен.
        </p>
      ) : displayEvents.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
