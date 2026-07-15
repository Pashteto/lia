"use client";

import { useRef } from "react";

/**
 * Styled Russian image upload. Wraps a visually-hidden native file input behind
 * a themed button so the UI never shows the browser's English "Choose File".
 * Shows a thumbnail once a preview URL is available, plus progress/error text.
 * Upload itself is the caller's concern (via onFile → uploadFile).
 */
export function ImageUpload({
  label,
  previewUrl,
  uploading = false,
  error,
  onFile,
  disabled = false,
}: {
  label: string;
  previewUrl?: string;
  uploading?: boolean;
  error?: string;
  onFile: (file: File) => void;
  disabled?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <div className="rounded-card bg-bg-secondary p-4">
      {previewUrl && (
        <div className="relative mb-3 aspect-[3/2] w-full overflow-hidden rounded-[10px] bg-fill">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={previewUrl} alt="Предпросмотр" className="h-full w-full object-cover" />
        </div>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="sr-only"
        disabled={disabled || uploading}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) onFile(file);
          e.target.value = ""; // allow re-picking the same file
        }}
      />
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        disabled={disabled || uploading}
        className="rounded-control bg-fill px-4 py-2 text-[15px] font-medium text-label transition hover:bg-fill-secondary disabled:opacity-60"
      >
        {uploading ? "Загрузка…" : previewUrl ? `Заменить ${label.toLowerCase()}` : `Загрузить ${label.toLowerCase()}`}
      </button>
      {error && <p className="mt-2 text-[13px] text-red-500">{error}</p>}
    </div>
  );
}
