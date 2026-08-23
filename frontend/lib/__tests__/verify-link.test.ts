import { describe, expect, it } from "vitest";
import { verifyHref } from "@/lib/verify-link";

describe("verifyHref", () => {
  it("carries the current page into ?next=", () => {
    expect(verifyHref("/events/abc")).toBe("/auth/verify?next=%2Fevents%2Fabc");
    expect(verifyHref("/me")).toBe("/auth/verify?next=%2Fme");
  });

  it("drops auth pages and the feed root", () => {
    expect(verifyHref("/")).toBe("/auth/verify");
    expect(verifyHref("/login")).toBe("/auth/verify");
    expect(verifyHref("/signup")).toBe("/auth/verify");
    expect(verifyHref("/auth/verify")).toBe("/auth/verify");
    expect(verifyHref(null)).toBe("/auth/verify");
  });

  it("an explicit next wins over the pathname", () => {
    expect(verifyHref("/login", "/events/abc")).toBe(
      "/auth/verify?next=%2Fevents%2Fabc",
    );
  });
});
