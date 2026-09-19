import { describe, expect, it } from "vitest";

import { isTokenExpired } from "../auth";

/** Builds a JWT-shaped string whose payload carries `exp` (seconds). */
function tokenWithExp(exp: number | null): string {
  const payload = exp === null ? {} : { exp };
  const b64 = Buffer.from(JSON.stringify(payload)).toString("base64url");
  return `header.${b64}.signature`;
}

const now = () => Math.floor(Date.now() / 1000);

describe("isTokenExpired", () => {
  // The case that stranded the admin page: a 72h token whose exp passed 16
  // hours earlier. /auth/me answered 401 and the UI waited forever.
  it("reports an expired token as expired", () => {
    expect(isTokenExpired(tokenWithExp(now() - 58_199))).toBe(true);
  });

  it("reports a live token as not expired", () => {
    expect(isTokenExpired(tokenWithExp(now() + 3600))).toBe(false);
  });

  it("treats the moment of expiry as expired", () => {
    expect(isTokenExpired(tokenWithExp(now() - 1))).toBe(true);
  });

  // Anything we cannot read must NOT be called expired: a false positive here
  // signs a working user out, which is worse than the stall we are fixing.
  it("does not claim expiry for tokens it cannot read", () => {
    expect(isTokenExpired(tokenWithExp(null))).toBe(false);
    expect(isTokenExpired("not-a-jwt")).toBe(false);
    expect(isTokenExpired("a.b")).toBe(false);
    expect(isTokenExpired("header.@@not-base64@@.sig")).toBe(false);
    expect(isTokenExpired("")).toBe(false);
    expect(isTokenExpired(null)).toBe(false);
  });

  it("ignores a non-numeric exp", () => {
    const b64 = Buffer.from(JSON.stringify({ exp: "soon" })).toString("base64url");
    expect(isTokenExpired(`header.${b64}.sig`)).toBe(false);
  });
});

/**
 * Regression guard for the QA-23-aug fix: the backend answers 401 on /auth/me
 * for a session it refuses for reasons other than expiry, and tearing that
 * session down logged people out mid-verification. Expiry is judged from the
 * token alone, so a live token is never mistaken for that case — whatever the
 * server said.
 */
describe("a live token is never torn down over a 401", () => {
  it("stays valid for a session the server refuses", () => {
    const live = tokenWithExp(now() + 72 * 3600);
    expect(isTokenExpired(live)).toBe(false);
  });
});
