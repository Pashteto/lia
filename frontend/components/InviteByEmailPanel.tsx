"use client";

import { useState } from "react";
import { sendInvitations, EMAIL_NOT_VERIFIED } from "@/lib/api";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Field";

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
    <div className="flex flex-col gap-[10px]">
      <Textarea
        label="Пригласить по email (через запятую)"
        rows={2}
        value={raw}
        onChange={(e) => setRaw(e.target.value)}
        placeholder="a@mail.ru, b@mail.ru"
        error={error || undefined}
      />
      {msg ? <p className="text-[11px] text-ink">{msg}</p> : null}
      <Button type="button" variant="primary" className="w-full" onClick={onSend} disabled={busy}>
        Отправить приглашения
      </Button>
      {showVerify ? <VerifyEmailInterstitial onClose={() => setShowVerify(false)} /> : null}
    </div>
  );
}
