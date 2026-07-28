"use client";

import { useQuery } from "@tanstack/react-query";

import { getEventFeedback } from "@/lib/api";

/**
 * Private organizer view of post-event feedback: average rating + count and
 * the raw list (rating, comment, author name, date). The backend gates this
 * to the event owner/admin (403 for everyone else) — on 403 (or any error)
 * this renders nothing rather than an error state, since it's an optional
 * expander most viewers won't have access to.
 */
export function OrganizerFeedback({ eventId }: { eventId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["event-feedback", eventId],
    queryFn: () => getEventFeedback(eventId),
  });

  if (isLoading) {
    return <p className="text-[11px] text-text-dim">Загрузка отзывов…</p>;
  }

  if (isError || !data) {
    return null;
  }

  if (data.count === 0) {
    return (
      <p className="text-[11px] text-text-dim">
        Отзывы появятся после завершения события
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-[10px]">
      <p className="cap">
        <span className="font-bold text-ink">★ {data.average.toFixed(1)}</span>
        {" · "}
        {data.count} {data.count === 1 ? "отзыв" : "отзывов"}
      </p>
      {data.items.map((item, idx) => (
        <div key={idx} className="border border-ink p-[12px]">
          <div className="flex items-baseline justify-between gap-[8px]">
            <span className="text-[12.5px] font-bold text-ink">{item.author_name}</span>
            <span className="cap shrink-0 text-ink">{"★".repeat(item.rating)}</span>
          </div>
          {item.comment ? (
            <p className="mt-[6px] text-[12.5px] leading-snug text-ink">{item.comment}</p>
          ) : null}
          <p className="mt-[6px] font-mono text-[11px] text-text-dim">
            {new Date(item.created_at).toLocaleDateString("ru-RU", {
              day: "numeric",
              month: "short",
              year: "numeric",
            })}
          </p>
        </div>
      ))}
    </div>
  );
}
