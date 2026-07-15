"use client";

import { Button } from "@/components/ui/Button";

/**
 * Centered, dimmed confirmation modal — the styled Russian replacement for the
 * native window.confirm(). Backdrop click and Отмена both dismiss.
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
        className="w-full max-w-sm rounded-card bg-bg p-5 shadow-card"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-1 text-[17px] font-semibold">{title}</h2>
        {body && <p className="mb-3 text-[15px] text-label-secondary">{body}</p>}
        <div className="mt-4 flex items-center justify-end gap-2">
          <button
            type="button"
            className="px-3 py-2 text-[15px] text-label"
            onClick={onClose}
          >
            {cancelLabel}
          </button>
          <Button
            type="button"
            variant="filled"
            onClick={onConfirm}
            className={danger ? "bg-red-500 hover:opacity-90" : undefined}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
