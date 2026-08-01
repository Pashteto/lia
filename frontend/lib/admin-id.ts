const FALLBACK = "EV-————";
const ALPHABET = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ";
const SPACE = 36 ** 4;

/** Dash-stripped lowercase hex of a uuid, or null when the input isn't one. */
function uuidHex(id: string): string | null {
  const hex = id.replace(/-/g, "").toLowerCase();
  return /^[0-9a-f]{4,}$/.test(hex) ? hex : null;
}

/**
 * FNV-1a, 32-bit. Cheap and stable across sessions, and — unlike the leading
 * hex chars this used to slice — it depends on the WHOLE uuid. Every seeded
 * event is `b0000000-…`, so a prefix gave seven moderation rows the same id.
 */
function hash32(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** Four base36 chars — same column width as before, 25× the id space. */
function shortCode(id: string): string | null {
  const hex = uuidHex(id);
  if (!hex) return null;
  let v = hash32(hex) % SPACE;
  let out = "";
  for (let i = 0; i < 4; i++) {
    out = ALPHABET[v % 36] + out;
    v = Math.floor(v / 36);
  }
  return out;
}

/** Display id for moderation rows: `EV-` + a 4-char code derived from the uuid. */
export function adminShortId(id: string): string {
  const code = shortCode(id);
  return code ? `EV-${code}` : FALLBACK;
}

/** Display id for the A4 user registry: the bare 4-char code.
 * No `EV-` prefix — the column is 44px and the rows are users, not events. */
export function adminUserShortId(id: string): string {
  return shortCode(id) ?? "—";
}
