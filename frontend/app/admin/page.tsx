"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { getAdminOverview } from "@/lib/api";
import { cn } from "@/lib/cn";

export default function AdminHome() {
  const [counts, setCounts] = useState<{
    events_total: number;
    events_published: number;
    events_removed: number;
  } | null>(null);

  useEffect(() => {
    getAdminOverview().then(setCounts).catch(() => setCounts(null));
  }, []);

  return (
    <div>
      <h1 className="mb-6 text-[28px] font-bold tracking-[-0.022em]">Админ</h1>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Stat label="Всего событий" value={counts?.events_total} />
        <Stat label="Опубликовано" value={counts?.events_published} />
        <Stat label="Снято" value={counts?.events_removed} />
      </div>

      <div className="mt-8">
        <Link
          href="/admin/moderation/events"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-control px-4 py-2.5",
            "bg-accent/12 text-accent text-[15px] font-semibold",
            "transition hover:bg-accent/20 active:scale-[0.97] motion-reduce:transform-none motion-reduce:transition-none",
          )}
        >
          Открыть очередь модерации →
        </Link>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value?: number }) {
  return (
    <div
      className={cn(
        "rounded-card bg-bg-secondary shadow-card-subtle",
        "flex flex-col gap-1.5 p-5",
      )}
    >
      <span className="text-[13px] text-label-secondary">{label}</span>
      <span className="text-[32px] font-bold leading-none tracking-[-0.022em]">
        {value ?? "—"}
      </span>
    </div>
  );
}
