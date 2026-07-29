const FALLBACK = "EV-————";

/** Display id for moderation rows: EV- + first 4 hex of UUID (no dashes), uppercased. */
export function adminShortId(id: string): string {
  if (!id) return FALLBACK;
  const hex = id.replace(/-/g, "").match(/^[0-9a-fA-F]{4}/);
  if (!hex) return FALLBACK;
  return `EV-${hex[0].toUpperCase()}`;
}
