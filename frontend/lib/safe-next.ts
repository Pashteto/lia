/** App-internal ?next= paths only — blocks open redirects (incl. backslash bypass). */
export function safeNextPath(raw: string | null, origin: string): string | null {
  if (raw == null) return null;

  const trimmed = raw.trim();
  if (trimmed === "") return null;

  if (trimmed.includes("\\")) return null;
  if (/%5[cC]/i.test(trimmed)) return null;

  let decoded: string;
  try {
    decoded = decodeURIComponent(trimmed);
  } catch {
    return null;
  }

  if (decoded.includes("\\")) return null;
  if (/%5[cC]/i.test(decoded)) return null;

  if (!decoded.startsWith("/") || decoded.startsWith("//")) return null;

  let resolved: URL;
  try {
    resolved = new URL(decoded, origin);
  } catch {
    return null;
  }

  if (resolved.origin !== origin) return null;

  return `${resolved.pathname}${resolved.search}${resolved.hash}`;
}
