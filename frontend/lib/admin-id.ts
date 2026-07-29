const FALLBACK = "EV-————";

/** Display id for moderation rows: EV- + first 4 hex of UUID (no dashes), uppercased. */
export function adminShortId(id: string): string {
  if (!id) return FALLBACK;
  const hex = id.replace(/-/g, "").match(/^[0-9a-fA-F]{4}/);
  if (!hex) return FALLBACK;
  return `EV-${hex[0].toUpperCase()}`;
}

/** Display id for the A4 user registry: first 4 hex of the UUID, uppercased.
 * No `EV-` prefix — the column is 44px and the rows are users, not events. */
export function adminUserShortId(id: string): string {
  if (!id) return "—";
  const hex = id.replace(/-/g, "").match(/^[0-9a-fA-F]{4}/);
  return hex ? hex[0].toUpperCase() : "—";
}
