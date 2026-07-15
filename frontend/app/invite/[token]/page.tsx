"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getInvitationPreview, acceptInvitation, EMAIL_NOT_VERIFIED, type InvitationPreview } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

export default function InvitePage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const { isAuthed, ready, emailVerified } = useAuth();
  const [preview, setPreview] = useState<InvitationPreview | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    getInvitationPreview(token).then(setPreview).catch(() => setError("Приглашение не найдено или устарело."));
  }, [token]);

  async function onAccept() {
    setBusy(true); setError("");
    try {
      await acceptInvitation(token);
      router.push(`/events/${preview?.event_id ?? ""}`);
    } catch (e) {
      if (e instanceof Error && e.message === EMAIL_NOT_VERIFIED) { router.push("/auth/verify"); return; }
      setError("Не удалось принять приглашение.");
    } finally {
      setBusy(false);
    }
  }

  if (error) return <main className="mx-auto max-w-md px-4 py-10"><p className="text-red-500">{error}</p></main>;
  if (!preview || !ready) return <main className="min-h-screen bg-bg-grouped" />;

  return (
    <main className="mx-auto max-w-md px-4 py-10">
      <h1 className="mb-2 text-[24px] font-bold tracking-[-0.022em]">Вас пригласили</h1>
      <p className="mb-6 text-[17px] text-label">«{preview.event_title}»</p>

      {!isAuthed ? (
        <div className="rounded-card bg-bg-secondary p-4">
          <p className="mb-3 text-[15px] text-label-secondary">Войдите или зарегистрируйтесь, чтобы принять приглашение.</p>
          {/* Reuse the app's auth modal; the header AuthButton also exposes it.
              Simplest: link to home where the login modal lives, or mount <LoginModal/> here. */}
          <Link href={`/?next=/invite/${token}`} className="rounded-capsule bg-accent px-4 py-2 text-white">Войти</Link>
        </div>
      ) : !emailVerified ? (
        <div className="rounded-card bg-bg-secondary p-4">
          <p className="mb-3 text-[15px] text-label-secondary">Подтвердите почту, чтобы принять приглашение.</p>
          <Link href="/auth/verify" className="rounded-capsule bg-accent px-4 py-2 text-white">Подтвердить почту</Link>
        </div>
      ) : (
        <button onClick={onAccept} disabled={busy}
          className="rounded-capsule bg-accent px-5 py-2.5 text-white disabled:opacity-50">
          Принять приглашение
        </button>
      )}
    </main>
  );
}
