"use client";

import { cn } from "@/lib/cn";
import { useSyncExternalStore } from "react";

type Theme = "light" | "dark";

const listeners = new Set<() => void>();

// Subscribe to theme changes: in-page toggles notify via `listeners`, and other
// tabs notify via the `storage` event.
function subscribe(callback: () => void) {
  listeners.add(callback);
  window.addEventListener("storage", callback);
  return () => {
    listeners.delete(callback);
    window.removeEventListener("storage", callback);
  };
}

// The pre-hydration inline script in layout.tsx resolves the class on <html>;
// we read it back as the source of truth.
function getSnapshot(): Theme {
  return document.documentElement.classList.contains("dark") ? "dark" : "light";
}

// SSR has no DOM; default to light to match the server-rendered markup.
function getServerSnapshot(): Theme {
  return "light";
}

function applyTheme(next: Theme) {
  const root = document.documentElement;
  root.classList.toggle("dark", next === "dark");
  root.classList.toggle("light", next === "light");
  try {
    localStorage.setItem("theme", next);
  } catch {
    // Private mode / storage disabled — the in-page toggle still works.
  }
  listeners.forEach((cb) => cb());
}

/**
 * Sliding sun/moon theme switch — an iOS-style pill (same mechanics as
 * components/ui/Switch). The knob carries the current mode's icon and slides
 * over the matching edge; the destination icon stays faintly visible on the
 * opposite side. Track tint shifts warm-day ↔ deep-night so the control reads
 * the mode at a glance.
 *
 * Applies an explicit `.light`/`.dark` class on <html> so the choice wins over
 * the OS `prefers-color-scheme` (see globals.css) and persists it to
 * localStorage. `useSyncExternalStore` reads the class set by the pre-hydration
 * script in layout.tsx without a hydration mismatch.
 */
export function ThemeSwitch() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const dark = theme === "dark";

  return (
    <button
      type="button"
      role="switch"
      aria-checked={dark}
      aria-label="Переключить тему"
      onClick={() => applyTheme(dark ? "light" : "dark")}
      className={cn(
        "relative inline-flex h-[30px] w-[58px] shrink-0 items-center rounded-full px-[3px]",
        "transition-colors duration-300 motion-reduce:transition-none",
        "ring-1 ring-inset ring-black/5 dark:ring-white/10",
        dark
          ? "bg-[#26263a]"
          : "bg-[#ffe2a8]",
      )}
    >
      {/* Faint edge icons — the one under the knob is hidden by it. */}
      <span
        aria-hidden
        className="pointer-events-none absolute left-[7px] text-[12px] leading-none opacity-50"
      >
        ☀️
      </span>
      <span
        aria-hidden
        className="pointer-events-none absolute right-[7px] text-[12px] leading-none opacity-50"
      >
        🌙
      </span>
      {/* Knob carries the current mode's icon and slides to its side. */}
      <span
        className={cn(
          "z-10 inline-flex size-[24px] items-center justify-center rounded-full bg-white text-[13px] leading-none",
          "shadow-[0_1px_3px_rgba(0,0,0,0.3)]",
          "transition-transform duration-300 ease-out motion-reduce:transition-none",
          dark ? "translate-x-[28px]" : "translate-x-0",
        )}
      >
        {dark ? "🌙" : "☀️"}
      </span>
    </button>
  );
}
