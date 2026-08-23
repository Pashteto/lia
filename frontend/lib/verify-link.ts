/** Builds the /auth/verify link so the user returns to where they started
 * after entering the code (QA-23-aug №7: verification used to dump everyone
 * on the feed). An explicit next (e.g. from the signup flow) wins; otherwise
 * the current pathname is carried, except pages it makes no sense to return
 * to (root feed, auth pages). */
const SKIP = [/^\/$/, /^\/login/, /^\/signup/, /^\/auth\//];

export function verifyHref(
  pathname: string | null,
  explicitNext?: string | null,
): string {
  const next =
    explicitNext ??
    (pathname && !SKIP.some((re) => re.test(pathname)) ? pathname : null);
  if (!next) return "/auth/verify";
  return `/auth/verify?next=${encodeURIComponent(next)}`;
}
