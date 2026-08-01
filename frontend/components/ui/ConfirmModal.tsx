"use client";

import { Button } from "@/components/ui/Button";

/**
 * Centered, dimmed confirmation modal — the styled Russian replacement for the
 * native window.confirm(). Backdrop click and Отмена both dismiss.
 *
 * Swiss Grid: square, hairline ink border on paper. No radius, no shadow, and
 * the danger state uses the signal token rather than a raw Tailwind red.
 */
export function ConfirmModal({
  title,
  body,
  confirmLabel,
  cancelLabel = "Отмена",
  danger = false,
  onConfirm,
  onClose,
}: {
  title: string;
  body?: string;
  confirmLabel: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm border border-ink bg-paper p-[14px]"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-[6px] text-[14px] font-black leading-[1.05] tracking-[-0.02em]">
          {title}
        </h2>
        {body && (
          <p className="mb-[12px] max-w-[52ch] text-[11.5px] leading-[1.45] text-text-dim">
            {body}
          </p>
        )}
        <div className="mt-[12px] flex items-center justify-end gap-[8px]">
          <Button variant="ghost" size="sm" type="button" onClick={onClose}>
            {cancelLabel}
          </Button>
          <Button
            type="button"
            size="sm"
            variant={danger ? "destructive" : "primary"}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
