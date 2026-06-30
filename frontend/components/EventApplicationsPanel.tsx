"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { decideApplication, fetchEventApplications } from "@/lib/api";
import type { Rsvp, RsvpStatus } from "@/lib/types";

const STATUS_LABEL: Record<RsvpStatus, string> = {
  going: "Подтверждено",
  waitlist: "В очереди",
  applied: "Ожидает решения",
  accepted: "Принята",
  declined: "Отклонена",
  withdrawn: "Отозвана",
  cancelled: "Отменена",
};

const STATUS_CLASS: Partial<Record<RsvpStatus, string>> = {
  accepted: "text-green-600",
  declined: "text-red-500",
  applied: "text-label-secondary",
};

interface DecidingState {
  [rsvpId: string]: "accept" | "decline" | null;
}

interface Props {
  eventId: string;
  eventTitle?: string;
}

/**
 * Organizer panel: lists all applications for an event and allows
 * accepting or declining each pending ("applied") one.
 * Only intended for events with signupMode === "application".
 */
export function EventApplicationsPanel({ eventId, eventTitle }: Props) {
  const queryClient = useQueryClient();
  const [deciding, setDeciding] = useState<DecidingState>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  const {
    data: applications = [],
    isLoading,
    isError,
  } = useQuery<Rsvp[]>({
    queryKey: ["event-applications", eventId],
    queryFn: () => fetchEventApplications(eventId),
  });

  async function handleDecide(rsvp: Rsvp, decision: "accept" | "decline") {
    setDeciding((prev) => ({ ...prev, [rsvp.id]: decision }));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[rsvp.id];
      return next;
    });
    try {
      await decideApplication(eventId, rsvp.id, decision);
      await queryClient.invalidateQueries({ queryKey: ["event-applications", eventId] });
    } catch (err) {
      setErrors((prev) => ({
        ...prev,
        [rsvp.id]: err instanceof Error ? err.message : "Ошибка",
      }));
    } finally {
      setDeciding((prev) => {
        const next = { ...prev };
        delete next[rsvp.id];
        return next;
      });
    }
  }

  if (isLoading) {
    return (
      <p className="mt-2 text-[13px] text-label-secondary">Загрузка заявок…</p>
    );
  }

  if (isError) {
    return (
      <p className="mt-2 text-[13px] text-red-500">Не удалось загрузить заявки.</p>
    );
  }

  if (applications.length === 0) {
    return (
      <p className="mt-2 text-[13px] text-label-secondary">Заявок пока нет.</p>
    );
  }

  return (
    <div className="mt-3 space-y-2">
      {eventTitle && (
        <p className="text-[13px] font-medium text-label-secondary">{eventTitle}</p>
      )}
      {applications.map((rsvp) => {
        const isPending = rsvp.status === "applied";
        const isActing = deciding[rsvp.id] != null;
        const statusClass = STATUS_CLASS[rsvp.status] ?? "text-label-secondary";

        return (
          <div
            key={rsvp.id}
            className="rounded-card bg-bg-secondary p-3 shadow-card-subtle"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1 space-y-1">
                <p className="text-[14px] font-medium text-label">
                  {rsvp.applicant?.name || "Участник"}
                </p>
                {rsvp.applicationAnswer ? (
                  <p className="text-[14px] leading-snug">{rsvp.applicationAnswer}</p>
                ) : (
                  <p className="text-[13px] italic text-label-secondary">Ответ не указан</p>
                )}
                <p className="text-[12px] text-label-secondary">
                  {new Date(rsvp.createdAt).toLocaleDateString("ru-RU", {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                  })}
                  {" · "}
                  <span className={statusClass}>{STATUS_LABEL[rsvp.status]}</span>
                </p>
                {errors[rsvp.id] && (
                  <p className="text-[12px] text-red-500">{errors[rsvp.id]}</p>
                )}
              </div>

              {isPending && (
                <div className="flex shrink-0 gap-2">
                  <Button
                    variant="tinted"
                    className="py-1.5 text-[13px]"
                    disabled={isActing}
                    onClick={() => handleDecide(rsvp, "accept")}
                  >
                    {deciding[rsvp.id] === "accept" ? "…" : "Принять"}
                  </Button>
                  <Button
                    variant="plain"
                    className="py-1.5 text-[13px] text-red-500 hover:text-red-400"
                    disabled={isActing}
                    onClick={() => handleDecide(rsvp, "decline")}
                  >
                    {deciding[rsvp.id] === "decline" ? "…" : "Отклонить"}
                  </Button>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
