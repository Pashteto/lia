"use client";

import { useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { getToken } from "@/lib/auth";

// Self-contained so it does not depend on the (concurrently edited) lib/api.ts.
const API_V1 = `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1`;

/**
 * Publishes a draft event via PATCH /events/{id} with {status:"published"},
 * behind a styled confirmation modal (never native confirm()). Publishing is
 * one-way: the backend locks a published event from further edits. On success,
 * invalidates the "my-events" query so the card re-renders without its badge.
 */
export function PublishEventButton({
  eventId,
  onPublished,
}: {
  eventId: string;
  /** Extra success hook for hosts outside «Мои события» (event detail strip). */
  onPublished?: () => void;
}) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);

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
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["my-events"] });
      onPublished?.();
    },
  });

  return (
    <div>
      <Button
        type="button"
        variant="primary"
        className="w-full"
        onClick={() => setConfirming(true)}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? "Публикация…" : "Опубликовать"}
      </Button>
      {mutation.isError ? (
        <p className="mt-[6px] text-[11px] text-signal">Не удалось опубликовать.</p>
      ) : null}
      {confirming ? (
        <ConfirmModal
          title="Опубликовать событие?"
          body="Событие сразу появится в ленте. Его можно будет отредактировать или вернуть в черновики."
          confirmLabel="Опубликовать"
          onConfirm={() => {
            setConfirming(false);
            mutation.mutate();
          }}
          onClose={() => setConfirming(false)}
        />
      ) : null}
    </div>
  );
}
