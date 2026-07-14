"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { useAuth } from "@/lib/auth-context";
import { getMyFeedback, submitFeedback } from "@/lib/api";

const STARS = [1, 2, 3, 4, 5];

/**
 * Participant-facing post-event feedback block: 5 tappable stars + optional
 * comment. Shown on ended events (EventDetailView) and on the /me/practices
 * past tab. Checks GET /me/feedback on mount to show the thank-you state if
 * the user already submitted — the server still enforces participant-only /
 * ended-only / one-per-user via distinct Russian error messages.
 */
export function FeedbackForm({ eventId }: { eventId: string }) {
  const { isAuthed } = useAuth();
  const [checking, setChecking] = useState(true);
  const [submitted, setSubmitted] = useState(false);
  const [rating, setRating] = useState(0);
  const [hoverRating, setHoverRating] = useState(0);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthed) {
      return;
    }
    let cancelled = false;
    getMyFeedback(eventId)
      .then((already) => {
        if (!cancelled) setSubmitted(already);
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });
    return () => {
      cancelled = true;
    };
  }, [eventId, isAuthed]);

  async function handleSubmit() {
    if (rating < 1) {
      setError("Поставьте оценку от 1 до 5 звёзд");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await submitFeedback(eventId, rating, comment.trim() || undefined);
      setSubmitted(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить отзыв");
    } finally {
      setBusy(false);
    }
  }

  if (!isAuthed) {
    return (
      <div className="rounded-card bg-bg-secondary p-4 text-center shadow-card-subtle">
        <p className="text-[15px] text-label-secondary">
          Войдите, чтобы оставить отзыв о встрече
        </p>
      </div>
    );
  }

  if (checking) {
    return null;
  }

  if (submitted) {
    return (
      <div className="rounded-card bg-bg-secondary p-4 text-center shadow-card-subtle">
        <p className="text-[15px] font-medium">Спасибо за отзыв 🙌</p>
      </div>
    );
  }

  const shown = hoverRating || rating;

  return (
    <div className="rounded-card bg-bg-secondary p-4 shadow-card-subtle">
      <h3 className="text-[15px] font-semibold">Как прошла встреча?</h3>

      <div className="mt-3 flex items-center gap-1" onMouseLeave={() => setHoverRating(0)}>
        {STARS.map((star) => (
          <button
            key={star}
            type="button"
            aria-label={`${star} из 5`}
            onClick={() => setRating(star)}
            onMouseEnter={() => setHoverRating(star)}
            className="text-[32px] leading-none transition active:scale-90"
          >
            <span className={star <= shown ? "text-accent" : "text-fill"}>★</span>
          </button>
        ))}
      </div>

      <textarea
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        placeholder="Комментарий (необязательно)"
        rows={3}
        className="mt-3 w-full rounded-control bg-bg-tertiary p-3 text-[15px]"
      />

      {error ? <p className="mt-2 text-[13px] text-red-500">{error}</p> : null}

      <div className="mt-3 flex justify-end">
        <Button variant="filled" onClick={handleSubmit} disabled={busy}>
          {busy ? "Отправка…" : "Отправить отзыв"}
        </Button>
      </div>
    </div>
  );
}
