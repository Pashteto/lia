/**
 * Human-facing rules and messages for image uploads.
 *
 * The cover uploader used to surface raw failures, and the one users actually
 * hit read «Failed to fetch» — a browser CORS artefact, not a real diagnosis:
 * the request was rejected upstream by a proxy body-size limit whose response
 * carries no CORS headers, so fetch() rejects instead of returning the status.
 * Checking the size before sending catches the common case outright, and every
 * remaining failure gets translated rather than shown raw.
 */

/** Matches the server cap (internal/http/uploads.maxBytes). */
export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

/** Formats bytes as a short Russian size, e.g. «12,4 МБ». */
function mb(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1).replace(".", ",")} МБ`;
}

/**
 * Returns a Russian complaint about the picked file, or null when it is worth
 * sending. Format is intentionally NOT checked here — the server decodes and
 * re-encodes, so it is the authority on what it can read, and a client-side
 * whitelist would only reject things that actually work (iPhone HEIC arrives
 * with an empty or unexpected `type` often enough to matter).
 */
export function validateImageFile(file: { size: number }): string | null {
  if (file.size > MAX_UPLOAD_BYTES) {
    return `Файл слишком большой: ${mb(file.size)}. Максимум ${mb(MAX_UPLOAD_BYTES)}.`;
  }
  if (file.size === 0) {
    return "Файл пустой — выберите другое изображение.";
  }
  return null;
}

/** Turns an upload failure into something a person can act on. */
export function uploadErrorMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err ?? "");

  // Status codes are checked first: our own thrown messages start with
  // "upload failed: <status>", and Safari's network error text is "Load failed"
  // — which is a substring of "upload failed", so order and word boundaries
  // both matter here.
  if (/\b413\b/.test(raw)) {
    return `Файл слишком большой. Максимум ${mb(MAX_UPLOAD_BYTES)}.`;
  }
  if (/\b415\b/.test(raw)) {
    return "Этот формат не поддерживается. Подойдут JPEG, PNG, WebP или HEIC.";
  }
  if (/\b401\b|\b403\b/.test(raw)) {
    return "Войдите заново — сессия истекла.";
  }
  // fetch() rejects with a TypeError for network-level failures, including a
  // proxy 413 whose error page carries no CORS headers.
  if (/failed to fetch|networkerror|(^|\W)load failed/i.test(raw)) {
    return `Не удалось отправить файл. Проверьте соединение и размер файла (максимум ${mb(MAX_UPLOAD_BYTES)}).`;
  }
  return "Не удалось загрузить изображение. Попробуйте ещё раз.";
}
