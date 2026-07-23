import { describe, expect, it } from "vitest";
import { shouldShowVerifyBanner } from "@/components/verify-banner-visibility";

describe("shouldShowVerifyBanner", () => {
  const base = { ready: true, isAuthed: true, roleResolved: true, emailVerified: false };
  it("shows for an authed, resolved, unverified user", () => {
    expect(shouldShowVerifyBanner(base)).toBe(true);
  });
  it("hides while /auth/me is still in flight (not resolved) — the reported bug", () => {
    expect(shouldShowVerifyBanner({ ...base, roleResolved: false })).toBe(false);
  });
  it("hides for a verified user", () => {
    expect(shouldShowVerifyBanner({ ...base, emailVerified: true })).toBe(false);
  });
  it("hides before hydration and when signed out", () => {
    expect(shouldShowVerifyBanner({ ...base, ready: false })).toBe(false);
    expect(shouldShowVerifyBanner({ ...base, isAuthed: false })).toBe(false);
  });
});
