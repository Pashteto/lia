"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import { cancelRsvp, eventCalendarUrl, signUp } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { LiaEvent, RsvpStatus } from "@/lib/types";

// ─── Local types ────────────────────────────────────────────────────────────

/** Active RSVP status held in local state after an action. */
type LocalStatus = RsvpStatus | "";

// ─── Helper: seats counter ───────────────────────────────────────────────────

function SeatsCounter({ event }: { event: LiaEvent }) {
  if (event.capacity == null) return null;
  const remaining = event.seatsRemaining ?? event.capacity;
  return (
    <span className="text-[13px] text-label-secondary">
      Осталось мест: {remaining}
    </span>
  );
}

// ─── Application sheet (inline) ──────────────────────────────────────────────

function ApplicationSheet({
  question,
  onSubmit,
  onClose,
  busy,
  error,
}: {
  question: string;
  onSubmit: (answer: string) => void;
  onClose: () => void;
  busy: boolean;
  error: string | null;
}) {
  const [answer, setAnswer] = useState("");

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded-card bg-bg p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-1 text-[17px] font-semibold">Подать заявку</h2>
        <p className="mb-3 text-[15px] text-label">{question}</p>
        <textarea
          className="w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent"
          rows={4}
          placeholder="Ваш ответ…"
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          disabled={busy}
        />
        {error && (
          <p className="mt-2 text-[14px] text-red-500">{error}</p>
        )}
        <div className="mt-4 flex items-center justify-end gap-2">
          <button
            type="button"
            className="px-3 py-2 text-[15px] text-label"
            onClick={onClose}
            disabled={busy}
          >
            Отмена
          </button>
          <Button
            type="button"
            onClick={() => onSubmit(answer.trim())}
            disabled={busy || answer.trim().length === 0}
          >
            {busy ? "Отправляем…" : "Отправить"}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export function SignupCTA({ event }: { event: LiaEvent }) {
  const { isAuthed, ready } = useAuth();

  // Local state — seeded from the server's my_rsvp_status (populated on
  // GET /events/{id}) and authoritative after any user action this session.
  // A reload therefore renders the correct joined/applied state.
  const [localStatus, setLocalStatus] = useState<LocalStatus>(
    event.myRsvpStatus ?? "",
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showApplicationSheet, setShowApplicationSheet] = useState(false);

  const calendarUrl = eventCalendarUrl(event.id);

  // ── Helpers ──────────────────────────────────────────────────────────────

  function handleAuthError() {
    setShowLoginModal(true);
  }

  async function handleSignUp(applicationAnswer?: string) {
    if (!isAuthed) {
      handleAuthError();
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const rsvp = await signUp(event.id, applicationAnswer);
      setLocalStatus(rsvp.status);
      setShowApplicationSheet(false);
    } catch (err) {
      if (err instanceof Error) {
        if (err.message === "not authenticated") {
          handleAuthError();
          return;
        }
        if (err.message.startsWith("EXTERNAL:")) {
          const url = err.message.slice("EXTERNAL:".length);
          if (url) window.open(url, "_blank");
          return;
        }
        setError(err.message);
      } else {
        setError("Произошла ошибка. Попробуйте ещё раз.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleCancel() {
    if (!isAuthed) {
      handleAuthError();
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await cancelRsvp(event.id);
      setLocalStatus("");
    } catch (err) {
      if (err instanceof Error && err.message === "not authenticated") {
        handleAuthError();
        return;
      }
      setError(
        err instanceof Error ? err.message : "Не удалось отменить запись.",
      );
    } finally {
      setBusy(false);
    }
  }

  // ── Common footer: calendar link + seats ────────────────────────────────

  const footer = (
    <div className="flex items-center gap-3">
      <a
        href={calendarUrl}
        download
        className="text-[13px] font-medium text-accent hover:opacity-70"
      >
        В календарь
      </a>
      <SeatsCounter event={event} />
    </div>
  );

  // ── Not-ready (SSR placeholder) ─────────────────────────────────────────

  if (!ready) {
    return (
      <div className="flex flex-col items-end gap-2">
        <Button variant="filled" className="px-8" disabled>
          Записаться
        </Button>
        {footer}
      </div>
    );
  }

  // ── Render by signupMode ─────────────────────────────────────────────────

  const mode = event.signupMode;

  // ── EXTERNAL ─────────────────────────────────────────────────────────────

  if (mode === "external") {
    return (
      <div className="flex flex-col items-end gap-2">
        <a
          href={event.externalRegistrationUrl ?? "#"}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-control bg-accent px-8 py-2.5 text-[15px] font-semibold text-white transition select-none hover:opacity-90 active:scale-[0.97] active:opacity-80"
        >
          Записаться на сайте организатора
        </a>
        <span className="text-[13px] text-label-secondary">
          Запись ведёт организатор
        </span>
        {footer}
      </div>
    );
  }

  // ── OPEN ──────────────────────────────────────────────────────────────────

  if (mode === "open" || mode == null) {
    const isFull =
      event.capacity != null && (event.seatsRemaining ?? 1) === 0;

    if (localStatus === "going") {
      return (
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <span className="text-[15px] font-semibold text-accent">
              Вы записаны
            </span>
            <Button
              variant="tinted"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Отписаться"}
            </Button>
          </div>
          {error && <p className="text-[13px] text-red-500">{error}</p>}
          {footer}
        </div>
      );
    }

    if (localStatus === "waitlist") {
      return (
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <span className="text-[15px] font-semibold text-label-secondary">
              Вы в листе ожидания
            </span>
            <Button
              variant="tinted"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Покинуть лист"}
            </Button>
          </div>
          {error && <p className="text-[13px] text-red-500">{error}</p>}
          {footer}
        </div>
      );
    }

    // No active status — show sign-up button
    return (
      <div className="flex flex-col items-end gap-2">
        <Button
          variant="filled"
          className="px-8"
          disabled={busy}
          onClick={() => handleSignUp()}
        >
          {busy ? "…" : isFull ? "В лист ожидания" : "Записаться"}
        </Button>
        {error && <p className="text-[13px] text-red-500">{error}</p>}
        {footer}
        {showLoginModal && (
          <LoginModal onClose={() => setShowLoginModal(false)} />
        )}
      </div>
    );
  }

  // ── APPLICATION ───────────────────────────────────────────────────────────

  if (mode === "application") {
    if (localStatus === "applied") {
      return (
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <span className="text-[15px] font-semibold text-accent">
              Заявка отправлена
            </span>
            <Button
              variant="tinted"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Отозвать заявку"}
            </Button>
          </div>
          {error && <p className="text-[13px] text-red-500">{error}</p>}
          {footer}
        </div>
      );
    }

    if (localStatus === "accepted") {
      return (
        <div className="flex flex-col items-end gap-2">
          <span className="text-[15px] font-semibold text-accent">
            Заявка принята
          </span>
          {footer}
        </div>
      );
    }

    if (localStatus === "declined") {
      return (
        <div className="flex flex-col items-end gap-2">
          <span className="text-[15px] font-semibold text-label-secondary">
            Заявка отклонена
          </span>
          {footer}
        </div>
      );
    }

    // No active status — show "Подать заявку"
    return (
      <div className="flex flex-col items-end gap-2">
        <Button
          variant="filled"
          className="px-8"
          disabled={busy}
          onClick={() => {
            if (!isAuthed) {
              setShowLoginModal(true);
              return;
            }
            setShowApplicationSheet(true);
          }}
        >
          Подать заявку
        </Button>
        {error && <p className="text-[13px] text-red-500">{error}</p>}
        {footer}
        {showLoginModal && (
          <LoginModal onClose={() => setShowLoginModal(false)} />
        )}
        {showApplicationSheet && (
          <ApplicationSheet
            question={event.curatorQuestion ?? "Расскажите о себе"}
            onSubmit={handleSignUp}
            onClose={() => setShowApplicationSheet(false)}
            busy={busy}
            error={error}
          />
        )}
      </div>
    );
  }

  // ── Fallback (unknown mode) ───────────────────────────────────────────────

  return footer;
}
