"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import { SquareRadio } from "@/components/ui/SquareRadio";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";
import {
  COMPLAINT_CATEGORIES,
  EMAIL_NOT_VERIFIED,
  submitComplaint,
  type ComplaintCategory,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export function ReportButton({ eventId }: { eventId: string }) {
  const { isAuthed } = useAuth();
  const [open, setOpen] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const [showVerify, setShowVerify] = useState(false);
  const [category, setCategory] = useState<ComplaintCategory>(COMPLAINT_CATEGORIES[0].value);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function openModal() {
    if (!isAuthed) {
      setShowLogin(true);
      return;
    }
    setOpen(true);
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      await submitComplaint(eventId, category, note.trim());
      setOpen(false);
      setDone(true);
      setNote("");
    } catch (err) {
      if (err instanceof Error && err.message === "not authenticated") {
        setOpen(false);
        setShowLogin(true);
        return;
      }
      if (err instanceof Error && err.message === EMAIL_NOT_VERIFIED) {
        setOpen(false);
        setShowVerify(true);
        return;
      }
      setError("Не удалось отправить жалобу");
    } finally {
      setBusy(false);
    }
  }

  if (done) {
    return (
      <p className="cap text-muted-2">Жалоба отправлена. Спасибо.</p>
    );
  }

  return (
    <>
      <button
        type="button"
        onClick={openModal}
        className="swiss-focus min-h-[44px] text-[11px] font-bold uppercase tracking-[0.07em] text-muted-2 underline-offset-2 hover:underline"
      >
        Пожаловаться
      </button>

      {open ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
          onClick={() => setOpen(false)}
        >
          <div
            className="w-full max-w-md border border-ink bg-paper p-[14px]"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-[10px] text-[14px] font-black leading-[1.05] tracking-[-0.02em]">
              Пожаловаться на событие
            </h2>

            <div className="border-t border-rule-inner">
              {COMPLAINT_CATEGORIES.map((c) => (
                <div key={c.value} className="border-b border-rule-inner">
                  <SquareRadio
                    name="complaint-category"
                    value={c.value}
                    label={c.label}
                    checked={category === c.value}
                    onChange={() => setCategory(c.value)}
                  />
                </div>
              ))}
            </div>

            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Комментарий (необязательно)"
              rows={3}
              className="swiss-focus mt-[12px] w-full border border-muted-2 bg-transparent px-[10px] py-[8px] text-[12px] outline-none"
            />

            {error ? <p className="mt-[8px] text-[11px] text-signal">{error}</p> : null}

            <div className="mt-[12px] flex justify-end gap-[8px]">
              <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
                Отмена
              </Button>
              <Button size="sm" onClick={submit} disabled={busy}>
                {busy ? "Отправка…" : "Отправить"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {showLogin ? <LoginModal onClose={() => setShowLogin(false)} /> : null}
      {showVerify ? <VerifyEmailInterstitial onClose={() => setShowVerify(false)} /> : null}
    </>
  );
}
