"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";

import { ResponsiveMapRegion } from "@/components/map/ResponsiveMapRegion";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { Chip } from "@/components/ui/Chip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { fetchNearbyEvents } from "@/lib/api";
import { cn } from "@/lib/cn";
import { distanceLabel, haversineKm, type LatLon } from "@/lib/geo";
import { createLatestRequestGate } from "@/lib/latest-request-gate";
import { mapAreaStats } from "@/lib/map-stats";
import { priceLabel } from "@/lib/price-label";
import type { LiaEvent } from "@/lib/types";
import type { MapPin, MapViewport } from "@/components/map/YandexMap";

const YandexMap = dynamic(() => import("@/components/map/YandexMap").then((m) => m.YandexMap), {
  ssr: false,
});

// Handoff U5 default view.
const MOSCOW: LatLon = [55.742, 37.618];
const SEARCH_LIMIT = 200;
const PIN_CAP = 100;
const NEAR_KM = 5;
const MOBILE_QUERY = "(max-width: 639px)";

type Filter = "all" | "near" | "free";

const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "Все" },
  { key: "near", label: "Рядом" },
  { key: "free", label: "Free" },
];

/** Positional numeral: the list index IS the pin number (handoff U5). */
function numeralAt(index: number): string {
  return String(index + 1).padStart(2, "0");
}

function subscribeMobileViewport(onChange: () => void): () => void {
  const media = window.matchMedia(MOBILE_QUERY);
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

function getMobileViewportSnapshot(): boolean {
  return window.matchMedia(MOBILE_QUERY).matches;
}

function getServerMobileViewportSnapshot(): boolean {
  return false;
}

export function MapBrowse() {
  const mobile = useSyncExternalStore(
    subscribeMobileViewport,
    getMobileViewportSnapshot,
    getServerMobileViewportSnapshot,
  );
  const requestGateRef = useRef(createLatestRequestGate());
  const [center, setCenter] = useState<LatLon>(MOSCOW);
  const [searchCenter, setSearchCenter] = useState<LatLon>(MOSCOW);
  const [viewport, setViewport] = useState<MapViewport | null>(null);
  const [events, setEvents] = useState<LiaEvent[]>([]);
  const [truncated, setTruncated] = useState(false);
  const [filter, setFilter] = useState<Filter>("all");
  const [activeId, setActiveId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (at: LatLon) => {
    const requestId = requestGateRef.current.begin();
    setLoading(true);
    setError(null);
    try {
      const all = await fetchNearbyEvents(at[0], at[1], SEARCH_LIMIT);
      if (!requestGateRef.current.isLatest(requestId)) return;
      const withCoords = all.filter((e) => e.venue?.lat != null && e.venue?.lon != null);
      setTruncated(withCoords.length > PIN_CAP);
      setEvents(withCoords.slice(0, PIN_CAP));
      setSearchCenter(at);
      setActiveId(null);
    } catch {
      if (!requestGateRef.current.isLatest(requestId)) return;
      setError("Не удалось загрузить события в этой области.");
    } finally {
      if (requestGateRef.current.isLatest(requestId)) setLoading(false);
    }
  }, []);

  // Open on the user's position when they allow it, Moscow otherwise.
  useEffect(() => {
    if (!navigator.geolocation) {
      queueMicrotask(() => void load(MOSCOW));
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const at: LatLon = [pos.coords.latitude, pos.coords.longitude];
        setCenter(at);
        void load(at);
      },
      () => void load(MOSCOW),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const visible = useMemo(() => {
    if (filter === "free") return events.filter((e) => e.priceType === "free");
    if (filter === "near")
      return events.filter(
        (e) => haversineKm(searchCenter, [e.venue!.lat!, e.venue!.lon!]) <= NEAR_KM,
      );
    return events;
  }, [events, filter, searchCenter]);

  // Memoized so YandexMap does not rebuild the placemark layer on every render.
  const pins = useMemo<MapPin[]>(
    () =>
      visible.map((e, i) => ({
        id: e.id,
        lat: e.venue!.lat!,
        lon: e.venue!.lon!,
        label: e.title,
        href: `/events/${e.id}`,
        numeral: numeralAt(i),
      })),
    [visible],
  );

  // Stats describe the last search; only «Радиус» tracks the live viewport.
  const stats = useMemo(
    () => mapAreaStats(visible, viewport?.radiusKm ?? null),
    [visible, viewport?.radiusKm],
  );

  const activeIndex = visible.findIndex((e) => e.id === activeId);
  const active = activeIndex >= 0 ? visible[activeIndex] : visible[0];
  const activeNumeral = activeIndex >= 0 ? numeralAt(activeIndex) : visible.length ? "01" : "—";

  const filterRow = (
    <div className="flex gap-[5px] border-b border-on-surface px-[14px] py-[8px]">
      {FILTERS.map((f) => (
        <Chip
          key={f.key}
          variant={filter === f.key ? "active" : "default"}
          onClick={() => setFilter(f.key)}
          aria-pressed={filter === f.key}
          className="text-[8px]"
        >
          {f.label}
        </Chip>
      ))}
    </div>
  );

  const listRail = (
    <>
      {filterRow}
      {loading ? (
        <div className="flex flex-col">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-[52px] border-x-0 border-t-0" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <p className="cap px-[14px] py-[16px]">Событий в этой области нет</p>
      ) : (
        <ul className="flex flex-col">
          {visible.map((e, i) => {
            const dist = distanceLabel(e.distanceM);
            return (
              <li key={e.id}>
                <Link
                  href={`/events/${e.id}`}
                  onMouseEnter={() => setActiveId(e.id)}
                  onFocus={() => setActiveId(e.id)}
                  className={cn(
                    "flex min-h-[44px] items-start gap-[8px] border-b border-on-surface px-[14px] py-[9px] swiss-focus hover-invert",
                    e.id === activeId && "bg-ink text-paper",
                  )}
                >
                  <span className="font-mono text-[10px] font-bold leading-[1.4]">{numeralAt(i)}</span>
                  <span className="min-w-0">
                    <span className="block text-[12px] font-bold leading-[1.1]">{e.title}</span>
                    <span className="cap mt-[3px] block truncate">
                      {[e.venue?.name ?? "—", dist].filter(Boolean).join(" · ")}
                    </span>
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
      {truncated ? (
        <p className="cap border-b border-on-surface px-[14px] py-[8px]">
          Показаны первые <span className="font-mono">{PIN_CAP}</span> событий
        </p>
      ) : null}
    </>
  );

  const mapPane = (mobile: boolean) => (
    <div className="relative min-h-0">
      <div className="h-full [filter:grayscale(1)_contrast(1.05)]">
        <YandexMap
          center={center}
          zoom={mobile ? 11 : 12}
          pins={pins}
          hideControls
          activePinId={activeId}
          onPinClick={setActiveId}
          onViewportChange={setViewport}
          className="h-full w-full"
        />
      </div>
      <button
        type="button"
        onClick={() => {
          const at = viewport?.center ?? center;
          setCenter(at);
          void load(at);
        }}
        className="swiss-focus hover-invert absolute left-1/2 top-[12px] z-[500] w-max -translate-x-1/2 border border-ink bg-paper px-[12px] py-[6px] text-[9px] font-bold uppercase tracking-[0.1em] text-ink"
      >
        Искать в этой области
      </button>
    </div>
  );

  if (error && events.length === 0) {
    return (
      <EmptyState
        numeral="!"
        title="Не удалось загрузить карту"
        text="Проверьте соединение и попробуйте обновить страницу."
      />
    );
  }

  return (
    <main className="mx-auto flex max-w-[1360px] flex-col max-sm:pb-[57px]">
      {/* Stat strip — four cells desktop at `.cell`/`.val` defaults, two on
          mobile at 8px/12px padding and an 11px value (reference :369-374, :394-397) */}
      <CellStrip cols={4} className="max-sm:hidden">
        <Cell caption="Всего" value={stats.total} mono />
        <Cell caption="Радиус" value={stats.radius} mono />
        <Cell caption="Бесплатно" value={stats.free} mono />
        <Cell caption="Сегодня" value={stats.today} mono />
      </CellStrip>
      <CellStrip cols={2} className="sm:hidden">
        <Cell caption="Всего" value={stats.total} mono valueClassName="text-[11px]" className="px-[12px] py-[8px]" />
        <Cell caption="Радиус" value={stats.radius} mono valueClassName="text-[11px]" className="px-[12px] py-[8px]" />
      </CellStrip>

      {/* Only the active breakpoint's map is mounted; hidden maps would race
          viewport callbacks and duplicate the Yandex API instance. */}
      <ResponsiveMapRegion mobile={mobile} listRail={listRail} renderMap={mapPane} />
      {active ? (
        <Link
          href={`/events/${active.id}`}
          className="block border-b border-ink px-[14px] py-[11px] swiss-focus hover-invert sm:hidden"
        >
          <span className="mb-[4px] flex items-baseline justify-between">
            <span className="font-mono text-[10px] font-bold uppercase">
              {activeNumeral}
              {distanceLabel(active.distanceM) ? ` · ${distanceLabel(active.distanceM)!.toUpperCase()}` : ""}
            </span>
            <span className="text-[11px] font-black">
              {priceLabel(active.priceMin, active.priceType)}
            </span>
          </span>
          <span className="block text-[13px] font-bold leading-[1.05]">{active.title}</span>
        </Link>
      ) : null}
      {error ? (
        <p className="border-b border-rule-inner px-[20px] py-[9px] text-[11.5px] text-signal">{error}</p>
      ) : null}
      {/* The list is the primary affordance on mobile too — below the card. */}
      <div className="flex flex-col sm:hidden">{listRail}</div>
    </main>
  );
}
