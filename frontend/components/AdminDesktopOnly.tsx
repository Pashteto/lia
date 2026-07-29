"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";

const NARROW_QUERY = "(max-width: 899px)";

/** A2/A3 gate: desktop tools need ≥900px; narrow viewports get a notice + link to A1. */
export function AdminDesktopOnly({ children }: { children: ReactNode }) {
  const [narrow, setNarrow] = useState<boolean | null>(null);

  useEffect(() => {
    const mql = window.matchMedia(NARROW_QUERY);
    const sync = () => setNarrow(mql.matches);
    sync();
    mql.addEventListener("change", sync);
    return () => mql.removeEventListener("change", sync);
  }, []);

  if (narrow === null) return null;

  if (narrow) {
    return (
      <div className="flex min-h-[50vh] flex-col items-start justify-center gap-[14px] px-[20px] py-[40px]">
        <p className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">
          Админ-инструменты — с экрана от 900px
        </p>
        <Link href="/admin" className="swiss-focus text-[11.5px] underline underline-offset-2">
          К обзору
        </Link>
      </div>
    );
  }

  return <>{children}</>;
}
