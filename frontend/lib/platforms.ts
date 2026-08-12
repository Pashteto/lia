/** Client-side mirror of backend/internal/platforms matching — used only for
 * the live form hint; the backend check is authoritative. */
export interface TrustedPlatform {
  domainSuffix: string;
  displayName: string;
  category: string;
}

export function hostOf(rawUrl: string): string | null {
  try {
    // `new URL().hostname` is already punycode + lowercase in browsers/node.
    return new URL(rawUrl).hostname.toLowerCase();
  } catch {
    return null;
  }
}

export function matchPlatform(
  rawUrl: string,
  platforms: TrustedPlatform[],
): TrustedPlatform | null {
  const host = hostOf(rawUrl);
  if (!host) return null;
  return (
    platforms.find(
      (p) => host === p.domainSuffix || host.endsWith(`.${p.domainSuffix}`),
    ) ?? null
  );
}
