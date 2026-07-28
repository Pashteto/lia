"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

/** U8-2: auth-required surface. Names the situation, one sentence, two actions. */
export function AuthGate({
  title = "Войдите, чтобы продолжить",
  reassurance = "Лента и карта доступны без входа.",
}: {
  title?: string;
  reassurance?: string;
}) {
  const pathname = usePathname();
  const loginHref = `/login?next=${encodeURIComponent(pathname)}`;
  return (
    <div className="flex flex-col items-start gap-[10px] px-[20px] py-[40px]">
      <span className="cap">Доступ</span>
      <h2 className="text-[17px] font-black leading-[1.05] tracking-[-0.02em]">{title}</h2>
      <p className="max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">{reassurance}</p>
      <div className="mt-[6px] flex gap-[8px]">
        <Link href={loginHref} className="swiss-focus bg-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-white hover:bg-black">
          Войти
        </Link>
        <Link href="/" className="swiss-focus border border-ink px-[11px] py-[11px] text-[11px] font-bold uppercase tracking-[0.07em] text-ink hover-invert">
          К ленте
        </Link>
      </div>
    </div>
  );
}
