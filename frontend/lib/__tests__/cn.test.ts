import { describe, expect, it } from "vitest";
import { cn } from "../cn";

describe("cn", () => {
  it("joins and drops falsy", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });
  it("later tailwind class wins on conflict", () => {
    expect(cn("px-4 text-ink", "px-2")).toBe("text-ink px-2");
  });
});
