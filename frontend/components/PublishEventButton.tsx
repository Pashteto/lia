"use client";

import { useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/Button";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { getMyOrganizer } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { publishDialogCopy } from "@/lib/publish-copy";

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
  // Cache-shared with the /me hub; decides which truth the dialog tells.
  const myOrganizer = useQuery({
    queryKey: ["my-organizer"],
    queryFn: () => getMyOrganizer().catch(() => null),
  });
  const verified = myOrganizer.data?.verification_status === "verified";
  const copy = publishDialogCopy(verified);

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
        {mutation.isPending ? "Отправляем…" : "Опубликовать"}
      </Button>
      {mutation.isError ? (
        <p className="mt-[6px] text-[11px] text-signal">Не удалось опубликовать.</p>
      ) : null}
      {confirming ? (
        <ConfirmModal
          title={copy.title}
          body={copy.body}
          confirmLabel={copy.confirmLabel}
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
