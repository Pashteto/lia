"use client";

import { useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { getToken } from "@/lib/auth";

// Mirrors PublishEventButton: self-contained fetch so it does not depend on the
// (concurrently edited) lib/api.ts.
const API_V1 = `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1`;

/**
 * Cancels an event via PATCH /events/{id} with {status:"cancelled"} — the
 * organizer-facing "delete". A true delete is deliberately not offered: people
 * may already be registered, and their copy of the event has to survive so they
 * can see it was called off. Cancelling drops the event out of the public feed
 * and leaves it visible, marked «Отменено», to the organizer and to anyone
 * holding a registration.
 */
export function CancelEventButton({ eventId }: { eventId: string }) {
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
        body: JSON.stringify({ status: "cancelled" }),
      });
      if (!res.ok) {
        throw new Error(`cancel failed: ${res.status} ${await res.text().catch(() => "")}`);
      }
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["my-events"] }),
  });

  return (
    <div>
      <Button
        type="button"
        variant="ghost"
        className="w-full"
        onClick={() => setConfirming(true)}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? "Отмена…" : "Отменить событие"}
      </Button>
      {mutation.isError ? (
        <p className="mt-[6px] text-[11px] text-signal">Не удалось отменить.</p>
      ) : null}
      {confirming ? (
        <ConfirmModal
          title="Отменить событие?"
          body="Событие пропадёт из ленты и поиска. Те, кто уже записался, увидят его помеченным как отменённое. Вернуть событие в ленту после отмены нельзя."
          confirmLabel="Отменить событие"
          danger
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
