"use client";

import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { useAuth } from "@/lib/auth-context";

// Nav auth control: "Войти" when signed out (opens a demo-login modal), or the
// signed-in email + "Выйти" when authed. Demo-login takes just an email — no
// password (see lib/auth.ts).
export function AuthButton() {
  const { email, isAuthed, ready, logout } = useAuth();
  const [open, setOpen] = useState(false);

  // Before the stored session is read, render a stable placeholder so the
  // button doesn't flicker between states on hydration.
  if (!ready) {
    return (
      <Button variant="plain" disabled>
        Войти
      </Button>
    );
  }

  if (isAuthed) {
    return (
      <div className="flex items-center gap-2">
        <Link
          href="/events/mine"
          className="text-[14px] font-medium text-accent"
        >
          Мои события
        </Link>
        <span
          className="hidden max-w-[12rem] truncate text-[14px] text-label-secondary sm:block"
          title={email ?? undefined}
        >
          {email}
        </span>
        <Button variant="plain" onClick={logout}>
          Выйти
        </Button>
      </div>
    );
  }

  return (
    <>
      <Button variant="plain" onClick={() => setOpen(true)}>
        Войти
      </Button>
      {open && <LoginModal onClose={() => setOpen(false)} />}
    </>
  );
}

export function LoginModal({ onClose }: { onClose: () => void }) {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await login(email.trim(), name.trim() || undefined);
      onClose();
    } catch {
      setError("Не удалось войти. Попробуйте ещё раз.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <form
        className="w-full max-w-sm rounded-card bg-bg p-5"
        onClick={(e) => e.stopPropagation()}
        onSubmit={submit}
      >
        <h2 className="mb-1 text-[17px] font-semibold">Вход</h2>
        <p className="mb-4 text-[13px] text-label-secondary">
          Демо-вход: укажите email, чтобы создавать события. Пароль не нужен.
        </p>
        <label className="mb-3 block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Email</span>
          <input
            type="email"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent"
          />
        </label>
        <label className="mb-4 block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">
            Имя (необязательно)
          </span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Ваше имя"
            className="w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent"
          />
        </label>
        {error && <p className="mb-3 text-[14px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            className="px-3 py-2 text-[15px] text-label"
            onClick={onClose}
          >
            Отмена
          </button>
          <Button type="submit" disabled={busy || !email.trim()}>
            {busy ? "Вход…" : "Войти"}
          </Button>
        </div>
      </form>
    </div>
  );
}
