"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { verifyHref } from "@/lib/verify-link";

export function VerifyEmailInterstitial({ onClose }: { onClose?: () => void }) {
  const pathname = usePathname();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="rounded-card bg-bg p-5 max-w-sm">
        <h2 className="mb-2 text-[19px] font-semibold text-label">Подтвердите почту</h2>
        <p className="mb-4 text-[15px] text-label-secondary">
          Чтобы выполнить это действие, подтвердите свою электронную почту.
        </p>
        <div className="flex gap-2">
          <Link href={verifyHref(pathname)} className="rounded-capsule bg-accent px-4 py-2 text-white">
            Подтвердить сейчас
          </Link>
          {onClose && (
            <button onClick={onClose} className="rounded-capsule bg-fill px-4 py-2 text-label">
              Позже
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
