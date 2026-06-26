"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import {
  COMPLAINT_CATEGORIES,
  submitComplaint,
  type ComplaintCategory,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export function ReportButton({ eventId }: { eventId: string }) {
  const { isAuthed } = useAuth();
  const [open, setOpen] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
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
      setError("Не удалось отправить жалобу");
    } finally {
      setBusy(false);
    }
  }

  if (done) {
    return (
      <p className="text-[13px] text-label-secondary">Жалоба отправлена. Спасибо.</p>
    );
  }

  return (
    <>
      <button
        type="button"
        onClick={openModal}
        className="text-[13px] text-label-secondary underline-offset-2 hover:underline"
      >
        Пожаловаться
      </button>

      {open ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
          onClick={() => setOpen(false)}
        >
          <div
            className="w-full max-w-md rounded-card bg-bg-secondary p-5 shadow-card"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="mb-3 text-[18px] font-semibold">Пожаловаться на событие</h2>

            <div className="space-y-2">
              {COMPLAINT_CATEGORIES.map((c) => (
                <label key={c.value} className="flex items-center gap-2 text-[15px]">
                  <input
                    type="radio"
                    name="complaint-category"
                    value={c.value}
                    checked={category === c.value}
                    onChange={() => setCategory(c.value)}
                  />
                  {c.label}
                </label>
              ))}
            </div>

            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Комментарий (необязательно)"
              rows={3}
              className="mt-3 w-full rounded-control bg-bg-tertiary p-3 text-[15px]"
            />

            {error ? <p className="mt-2 text-[13px] text-red-500">{error}</p> : null}

            <div className="mt-4 flex justify-end gap-2">
              <Button variant="tinted" onClick={() => setOpen(false)}>
                Отмена
              </Button>
              <Button variant="filled" onClick={submit} disabled={busy}>
                {busy ? "Отправка…" : "Отправить"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {showLogin ? <LoginModal onClose={() => setShowLogin(false)} /> : null}
    </>
  );
}
