"use client";

import { Button } from "@/components/ui/Button";
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
 * Theme switch. Applies an explicit `.light`/`.dark` class on <html> so the
 * choice wins over the OS `prefers-color-scheme` (see globals.css) and persists
 * it to localStorage. `useSyncExternalStore` reads the class set by the
 * pre-hydration script in layout.tsx without a hydration mismatch.
 */
export function ThemeToggle() {
  const theme = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  return (
    <Button
      variant="plain"
      onClick={() => applyTheme(theme === "dark" ? "light" : "dark")}
      aria-label="Переключить тему"
    >
      {theme === "dark" ? "Светлая" : "Тёмная"}
    </Button>
  );
}
