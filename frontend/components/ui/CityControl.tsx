"use client";

import { useEffect, useRef, useState } from "react";
import { CITIES } from "@/lib/city";
import { useCity } from "@/lib/city-context";
import { cn } from "@/lib/cn";

/** The city list lives in lib/city (single source shared with the feed copy,
 * login caption and map center); re-exported here for existing imports. */
export const CITY_OPTIONS = CITIES;

/** Dropdown list under the header control. Split out so the open-state markup
 * is testable without simulating a click. Availability comes from the city
 * context (GET /cities → the cities.spb_available runtime setting). */
export function CityMenu({ onClose }: { onClose: () => void }) {
  const { city: current, available, setCity } = useCity();
  return (
    <ul
      role="listbox"
      aria-label="Город"
      className="absolute right-0 top-[calc(100%+6px)] z-40 min-w-[196px] border border-ink bg-paper shadow-[0_2px_0_0_var(--color-ink)]"
    >
      {CITY_OPTIONS.map((city) => {
        const isAvailable = available(city.slug);
        return (
          <li key={city.code} role="option" aria-selected={city.slug === current.slug}>
            <button
              type="button"
              disabled={!isAvailable}
              onClick={() => {
                if (city.slug !== current.slug) setCity(city.slug);
                onClose();
              }}
              className={cn(
                "flex w-full items-baseline justify-between gap-[14px] px-[12px] py-[9px] text-left text-[12px]",
                isAvailable
                  ? "swiss-focus cursor-pointer font-bold hover-invert"
                  : "cursor-default text-text-dim",
              )}
            >
              <span>{city.name}</span>
              {city.slug === current.slug ? (
                <span aria-hidden className="font-mono text-[11px]">✓</span>
              ) : isAvailable ? null : (
                <span className="cap">Скоро</span>
              )}
            </button>
          </li>
        );
      })}
    </ul>
  );
}

/** Header city control: looks like the old «МСК» caption but is an honest
 * button — opens the city list; unavailable cities are announced as «Скоро»
 * instead of a dead label (QA 14.08, finding 3). Picking an available city
 * writes the lia_city cookie and re-renders server data (city-context). */
export function CityControl() {
  const { city: current } = useCity();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`Город: ${current.name}`}
        onClick={() => setOpen((v) => !v)}
        className="cap swiss-focus cursor-pointer whitespace-nowrap hover-invert"
      >
        {current.code} ↓
      </button>
      {open ? <CityMenu onClose={() => setOpen(false)} /> : null}
    </div>
  );
}
