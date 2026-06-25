"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { getToken } from "@/lib/auth";

// Self-contained so it does not depend on the (concurrently edited) lib/api.ts.
const API_V1 = `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1`;

/**
 * Publishes a draft event via PATCH /events/{id} with {status:"published"},
 * behind a confirm. Publishing is one-way: the backend locks a published event
 * from further edits, so we warn before committing. On success, invalidates the
 * "my-events" query so the card re-renders without its draft badge.
 *
 * Render only for events the caller owns and that are in `draft` status
 * (the backend rejects publishing any non-draft with 409).
 */
export function PublishEventButton({ eventId }: { eventId: string }) {
  const qc = useQueryClient();

  const mutation = useMutation({
    mutationFn: async () => {
      const token = getToken();
      if (!token) throw new Error("not authenticated");
      const res = await fetch(`${API_V1}/events/${eventId}`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ status: "published" }),
      });
      if (!res.ok) {
        throw new Error(`publish failed: ${res.status} ${await res.text().catch(() => "")}`);
      }
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-events"] }),
  });

  const onClick = () => {
    if (window.confirm("Опубликовать событие? После публикации изменить его будет нельзя.")) {
      mutation.mutate();
    }
  };

  return (
    <div className="mt-1">
      <button
        type="button"
        onClick={onClick}
        disabled={mutation.isPending}
        className="flex w-full items-center justify-center gap-1 rounded-control px-2 py-1.5 text-[13px] font-medium text-accent hover:bg-accent/8 transition disabled:opacity-50"
      >
        {mutation.isPending ? "Публикация…" : "Опубликовать"}
      </button>
      {mutation.isError && (
        <p className="px-2 text-[12px] text-red-500">Не удалось опубликовать.</p>
      )}
    </div>
  );
}
