import { describe, expect, it } from "vitest";

import { shouldShowSessionExpired } from "@/components/session-expired-visibility";

describe("shouldShowSessionExpired", () => {
  it("shows once hydration finished and the session was found expired", () => {
    expect(shouldShowSessionExpired({ ready: true, sessionExpired: true })).toBe(true);
  });

  it("stays hidden before hydration — SSR has no localStorage to judge from", () => {
    expect(shouldShowSessionExpired({ ready: false, sessionExpired: true })).toBe(false);
  });

  it("stays hidden for a live session", () => {
    expect(shouldShowSessionExpired({ ready: true, sessionExpired: false })).toBe(false);
  });
});
