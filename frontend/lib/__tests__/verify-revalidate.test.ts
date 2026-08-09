import { describe, expect, it } from "vitest";
import { shouldRevalidateVerification } from "../verify-revalidate";

describe("shouldRevalidateVerification", () => {
  it("re-reads /auth/me when an unverified tab comes back to the front", () => {
    expect(
      shouldRevalidateVerification({
        hasToken: true,
        emailVerified: false,
        visibility: "visible",
      }),
    ).toBe(true);
  });

  it("stays quiet once the session is verified", () => {
    expect(
      shouldRevalidateVerification({
        hasToken: true,
        emailVerified: true,
        visibility: "visible",
      }),
    ).toBe(false);
  });

  it("stays quiet while the tab is hidden", () => {
    expect(
      shouldRevalidateVerification({
        hasToken: true,
        emailVerified: false,
        visibility: "hidden",
      }),
    ).toBe(false);
  });

  it("stays quiet when signed out", () => {
    expect(
      shouldRevalidateVerification({
        hasToken: false,
        emailVerified: false,
        visibility: "visible",
      }),
    ).toBe(false);
  });
});
