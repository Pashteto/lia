"use client";

import { useEffect, useState } from "react";

import { FeedbackForm } from "@/components/FeedbackForm";
import { isActiveParticipant } from "@/components/feedback-visibility";
import { fetchEventWithAuth } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent } from "@/lib/types";

/**
 * The «Отзыв» block on an ended event — rendered only for someone who actually
 * attended. Without the gate the form appeared for any signed-in visitor and
 * the server answered 403 ErrNotParticipant on submit.
 *
 * The detail page SSR-fetches the event anonymously (shared revalidate cache),
 * so `event.myRsvpStatus` arrives as "" even for an attendee. Once the caller
 * is known to be authed we re-fetch THEIR status with the token, same as
 * SignupCTA does for the join/apply state.
 */
export function EventFeedbackSection({ event }: { event: LiaEvent }) {
  const { isAuthed, ready } = useAuth();
  const [status, setStatus] = useState(event.myRsvpStatus ?? "");

  useEffect(() => {
    if (!ready || !isAuthed) return;
    let cancelled = false;
    void fetchEventWithAuth(event.id)
      .then((e) => {
        if (!cancelled && e) setStatus(e.myRsvpStatus ?? "");
      })
      .catch(() => {
        // Best-effort: leave the mount-seeded status, which hides the form.
      });
    return () => {
      cancelled = true;
    };
  }, [ready, isAuthed, event.id]);

  if (!isActiveParticipant(status)) return null;

  return (
    <section className="border-b border-rule-inner px-[20px] py-[16px]">
      <p className="cap mb-[10px]">Отзыв</p>
      <FeedbackForm eventId={event.id} />
    </section>
  );
}
