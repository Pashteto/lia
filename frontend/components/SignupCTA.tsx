"use client";

import { useEffect, useRef, useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";
import { cancelRsvp, eventCalendarUrl, EMAIL_NOT_VERIFIED, fetchEventWithAuth, signUp } from "@/lib/api";
import { googleCalendarUrl } from "@/lib/calendar-links";
import { useAuth } from "@/lib/auth-context";
import { extractHttpStatus, rsvpErrorMessage } from "@/lib/rsvp-errors";
import { signupClosedLabel } from "@/lib/signup-availability";
import { hostOf } from "@/lib/platforms";
import type { LiaEvent, RsvpStatus } from "@/lib/types";

// ─── Local types ────────────────────────────────────────────────────────────

/** Active RSVP status held in local state after an action. */
type LocalStatus = RsvpStatus | "";

// ─── Helper: seats counter ───────────────────────────────────────────────────

function SeatsCounter({ event }: { event: LiaEvent }) {
  if (event.capacity == null) return null;
  const remaining = event.seatsRemaining ?? event.capacity;
  return (
    <span className="text-[11.5px] text-text-dim">
      Осталось мест: <span className="font-mono">{remaining}</span>
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
        className="w-full max-w-sm border border-ink bg-paper p-[20px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-1 text-[22px] font-black tracking-[-0.02em]">Подать заявку</h2>
        <p className="mb-3 text-[12.5px]">{question}</p>
        <textarea
          className="w-full border border-on-surface bg-transparent px-[11px] py-[9px] text-[12.5px] text-on-surface outline-none placeholder:text-field-text swiss-focus"
          rows={4}
          placeholder="Ваш ответ…"
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          disabled={busy}
        />
        {error && (
          <p className="mt-2 text-[11px] text-signal">{error}</p>
        )}
        <div className="mt-4 flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={busy}
          >
            Отмена
          </Button>
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
  const { isAuthed, ready, roleResolved, emailVerified } = useAuth();

  // Known-unverified viewer: explain the state up front instead of letting the
  // API call fail (design review P1 — "sign up failed: 401" at the CTA).
  const needsVerify = isAuthed && roleResolved && !emailVerified;

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
  const [showVerify, setShowVerify] = useState(false);

  // The user has taken an action this session (sign up / cancel / apply) — once
  // true, the authed refetch below must not overwrite what they just did.
  const acted = useRef(false);

  // The detail page SSR-fetches the event anonymously (shared revalidate cache),
  // so event.myRsvpStatus arrives as "" there. Once we know the caller is
  // authed, re-fetch THEIR status client-side (no-store, token) so a reload
  // shows the correct joined/applied state (design-review R4). Skipped if the
  // user has already acted this session.
  useEffect(() => {
    if (!ready || !isAuthed) return;
    let cancelled = false;
    void fetchEventWithAuth(event.id)
      .then((e) => {
        if (!cancelled && e && !acted.current) {
          setLocalStatus(e.myRsvpStatus ?? "");
        }
      })
      .catch(() => {
        // Best-effort enrichment; leave the mount-seeded status on failure.
      });
    return () => {
      cancelled = true;
    };
  }, [ready, isAuthed, event.id]);

  const calendarUrl = eventCalendarUrl(event.id);

  // ── Helpers ──────────────────────────────────────────────────────────────

  function handleAuthError() {
    setShowLoginModal(true);
  }

  async function handleSignUp(applicationAnswer?: string) {
    acted.current = true;
    if (!isAuthed) {
      handleAuthError();
      return;
    }
    if (needsVerify) {
      setShowVerify(true);
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
        if (err.message === EMAIL_NOT_VERIFIED) {
          setShowVerify(true);
          return;
        }
        if (err.message.startsWith("EXTERNAL:")) {
          const url = err.message.slice("EXTERNAL:".length);
          if (url) window.open(url, "_blank");
          return;
        }
        // A server-side 401 means the stored token is stale/invalid — that is
        // an auth problem, not a message to display.
        if (extractHttpStatus(err) === 401) {
          handleAuthError();
          return;
        }
        setError(rsvpErrorMessage(err, "signup"));
      } else {
        setError("Произошла ошибка. Попробуйте ещё раз.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleCancel() {
    acted.current = true;
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
      if (
        (err instanceof Error && err.message === "not authenticated") ||
        extractHttpStatus(err) === 401
      ) {
        handleAuthError();
        return;
      }
      setError(rsvpErrorMessage(err, "cancel"));
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
        className="swiss-focus text-[11.5px] text-text-dim underline underline-offset-2"
      >
        В календарь (.ics)
      </a>
      <a
        href={googleCalendarUrl(event)}
        target="_blank"
        rel="noopener noreferrer"
        className="swiss-focus text-[11.5px] text-text-dim underline underline-offset-2"
      >
        В Google-календарь
      </a>
      <SeatsCounter event={event} />
    </div>
  );

  // ── Not-ready (SSR placeholder) ─────────────────────────────────────────

  if (!ready) {
    return (
      <div className="flex flex-col items-end gap-2">
        <Button className="px-8" disabled>
          Записаться
        </Button>
        {footer}
      </div>
    );
  }

  // Non-published events are not signup-able. Render the status instead of an
  // active CTA so an unverified viewer isn't pushed into verification on a
  // cancelled/withdrawn event (QA 5b).
  const closed = signupClosedLabel(event.status);
  if (closed) {
    return (
      <div className="flex flex-col items-end gap-2">
        <span className="cap">{closed}</span>
        {footer}
      </div>
    );
  }

  // ── Render by signupMode ─────────────────────────────────────────────────

  const mode = event.signupMode;

  // ── EXTERNAL ─────────────────────────────────────────────────────────────

  if (mode === "external") {
    const isPaid = event.priceType !== "free";
    const name = event.externalPlatformName;
    const host = event.externalRegistrationUrl
      ? hostOf(event.externalRegistrationUrl)
      : null;

    return (
      <div className="flex flex-col items-end gap-2">
        <a
          href={event.externalRegistrationUrl ?? "#"}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center justify-center gap-1.5 whitespace-nowrap bg-ink px-8 py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white transition-colors duration-[120ms] ease-linear select-none hover:bg-black swiss-focus"
        >
          {name
            ? isPaid
              ? `Билеты на ${name}`
              : `Записаться через ${name}`
            : "Записаться на сайте организатора"}
        </a>
        <span className="text-[11.5px] text-text-dim">
          {name ? host : host ? `Переход на ${host}` : "Запись ведёт организатор"}
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
            <span className="cap">
              Вы записаны
            </span>
            <Button
              variant="ghost"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Отписаться"}
            </Button>
          </div>
          {error && <p className="text-[11px] text-signal">{error}</p>}
          {footer}
        </div>
      );
    }

    if (localStatus === "waitlist") {
      return (
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <span className="cap">
              Вы в листе ожидания
            </span>
            <Button
              variant="ghost"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Покинуть лист"}
            </Button>
          </div>
          {error && <p className="text-[11px] text-signal">{error}</p>}
          {footer}
        </div>
      );
    }

    // No active status — show sign-up button
    return (
      <div className="flex flex-col items-end gap-2">
        <Button
          variant={isFull ? "ghost" : "primary"}
          className="px-8"
          disabled={busy}
          onClick={() => handleSignUp()}
        >
          {busy ? "…" : needsVerify ? "Подтвердить почту" : isFull ? "В лист ожидания" : "Записаться"}
        </Button>
        {needsVerify && (
          <p className="text-[11px] text-text-dim">
            Записаться можно после подтверждения почты
          </p>
        )}
        {error && <p className="text-[11px] text-signal">{error}</p>}
        {footer}
        {showLoginModal && (
          <LoginModal onClose={() => setShowLoginModal(false)} />
        )}
        {showVerify && (
          <VerifyEmailInterstitial onClose={() => setShowVerify(false)} />
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
            <span className="cap">
              Заявка отправлена
            </span>
            <Button
              variant="ghost"
              onClick={handleCancel}
              disabled={busy}
              className="px-4"
            >
              {busy ? "…" : "Отозвать заявку"}
            </Button>
          </div>
          {error && <p className="text-[11px] text-signal">{error}</p>}
          {footer}
        </div>
      );
    }

    if (localStatus === "accepted") {
      return (
        <div className="flex flex-col items-end gap-2">
          <span className="cap">
            Заявка принята
          </span>
          {footer}
        </div>
      );
    }

    if (localStatus === "declined") {
      return (
        <div className="flex flex-col items-end gap-2">
          <span className="cap">
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
          className="px-8"
          disabled={busy}
          onClick={() => {
            if (!isAuthed) {
              setShowLoginModal(true);
              return;
            }
            if (needsVerify) {
              setShowVerify(true);
              return;
            }
            setShowApplicationSheet(true);
          }}
        >
          {needsVerify ? "Подтвердить почту" : "Подать заявку"}
        </Button>
        {needsVerify && (
          <p className="text-[11px] text-text-dim">
            Подать заявку можно после подтверждения почты
          </p>
        )}
        {error && <p className="text-[11px] text-signal">{error}</p>}
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
        {showVerify && (
          <VerifyEmailInterstitial onClose={() => setShowVerify(false)} />
        )}
      </div>
    );
  }

  // ── Fallback (unknown mode) ───────────────────────────────────────────────

  return footer;
}
