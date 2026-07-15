"use client";

import { useState } from "react";
import { sendInvitations, EMAIL_NOT_VERIFIED } from "@/lib/api";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";

const inputClass =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export function InviteByEmailPanel({ eventId }: { eventId: string }) {
  const [raw, setRaw] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [showVerify, setShowVerify] = useState(false);

  async function onSend() {
    const emails = raw.split(/[\s,;]+/).map((s) => s.trim()).filter(Boolean);
    if (emails.length === 0) return;
    setBusy(true); setError(""); setMsg("");
    try {
      const n = await sendInvitations(eventId, emails);
      setMsg(`Приглашения отправлены: ${n}`);
      setRaw("");
    } catch (e) {
      if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) { setShowVerify(true); }
      else { setError("Не удалось отправить приглашения."); }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-card bg-bg-secondary p-3">
      <p className="mb-1.5 block text-[13px] text-label-secondary">Пригласить по email (через запятую)</p>
      <textarea className={inputClass} rows={2} value={raw}
        onChange={(e) => setRaw(e.target.value)} placeholder="a@mail.ru, b@mail.ru" />
      {error && <p className="mt-2 text-[14px] text-red-500">{error}</p>}
      {msg && <p className="mt-2 text-[14px] text-green-600">{msg}</p>}
      <button onClick={onSend} disabled={busy}
        className="mt-2 rounded-capsule bg-accent px-4 py-2 text-white disabled:opacity-50">
        Отправить приглашения
      </button>
      {showVerify && <VerifyEmailInterstitial onClose={() => setShowVerify(false)} />}
    </div>
  );
}
