"use client";

import { useEffect, useRef, useState } from "react";
import { CITIES, CURRENT_CITY } from "@/lib/city";
import { cn } from "@/lib/cn";

/** The city list lives in lib/city (single source shared with the feed copy,
 * login caption and map center); re-exported here for existing imports. */
export const CITY_OPTIONS = CITIES;

const CURRENT = CURRENT_CITY;

/** Dropdown list under the header control. Split out so the open-state markup
 * is testable without simulating a click. */
export function CityMenu({ onClose }: { onClose: () => void }) {
  return (
    <ul
      role="listbox"
      aria-label="Город"
      className="absolute right-0 top-[calc(100%+6px)] z-40 min-w-[196px] border border-ink bg-paper shadow-[0_2px_0_0_var(--color-ink)]"
    >
      {CITY_OPTIONS.map((city) => (
        <li key={city.code} role="option" aria-selected={city === CURRENT}>
          <button
            type="button"
            disabled={!city.available}
            onClick={onClose}
            className={cn(
              "flex w-full items-baseline justify-between gap-[14px] px-[12px] py-[9px] text-left text-[12px]",
              city.available
                ? "swiss-focus cursor-pointer font-bold hover-invert"
                : "cursor-default text-text-dim",
            )}
          >
            <span>{city.name}</span>
            {city === CURRENT ? (
              <span aria-hidden className="font-mono text-[11px]">✓</span>
            ) : (
              <span className="cap">Скоро</span>
            )}
          </button>
        </li>
      ))}
    </ul>
  );
}

/** Header city control: looks like the old «МСК» caption but is an honest
 * button — opens a one-real-option list so «СПб скоро» is announced instead
 * of a dead label (QA 14.08, finding 3). */
export function CityControl() {
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
        aria-label={`Город: ${CURRENT.name}`}
        onClick={() => setOpen((v) => !v)}
        className="cap swiss-focus cursor-pointer whitespace-nowrap hover-invert"
      >
        {CURRENT.code} ↓
      </button>
      {open ? <CityMenu onClose={() => setOpen(false)} /> : null}
    </div>
  );
}
