"use client";

import { Button } from "@/components/ui/Button";
import { useEffect, useState } from "react";

/**
 * Scaffold helper: flips the `.dark` class on <html> so both themes can be
 * eyeballed without changing OS settings. A real app would persist preference.
 */
export function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
  }, [dark]);

  return (
    <Button
      variant="plain"
      onClick={() => setDark((d) => !d)}
      aria-label="Переключить тему"
    >
      {dark ? "Светлая" : "Тёмная"}
    </Button>
  );
}
