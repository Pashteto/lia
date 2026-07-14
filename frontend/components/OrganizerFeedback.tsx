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
    return <p className="mt-2 text-[13px] text-label-secondary">Загрузка отзывов…</p>;
  }

  if (isError || !data) {
    return null;
  }

  if (data.count === 0) {
    return (
      <p className="mt-2 text-[13px] text-label-secondary">
        Отзывы появятся после завершения события
      </p>
    );
  }

  return (
    <div className="mt-3 space-y-2">
      <p className="text-[13px] font-medium text-label-secondary">
        <span className="text-accent">★ {data.average.toFixed(1)}</span>
        {" · "}
        {data.count} {data.count === 1 ? "отзыв" : "отзывов"}
      </p>
      {data.items.map((item, idx) => (
        <div
          key={idx}
          className="rounded-card bg-bg-secondary p-3 shadow-card-subtle"
        >
          <div className="flex items-center justify-between gap-3">
            <span className="text-[14px] font-medium text-label">{item.author_name}</span>
            <span className="text-[13px] text-accent">{"★".repeat(item.rating)}</span>
          </div>
          {item.comment && (
            <p className="mt-1 text-[14px] leading-snug text-label">{item.comment}</p>
          )}
          <p className="mt-1 text-[12px] text-label-secondary">
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
