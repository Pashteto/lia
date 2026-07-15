import { describe, expect, it } from "vitest";
import { joinLocal, splitLocal } from "@/components/ui/DateTimeField";

describe("datetime-local split/join", () => {
  it("splits a datetime-local string", () => {
    expect(splitLocal("2026-08-15T18:30")).toEqual({ date: "2026-08-15", time: "18:30" });
  });
  it("handles empty", () => {
    expect(splitLocal("")).toEqual({ date: "", time: "" });
  });
  it("joins date + time", () => {
    expect(joinLocal("2026-08-15", "18:30")).toBe("2026-08-15T18:30");
  });
  it("join returns empty when date missing", () => {
    expect(joinLocal("", "18:30")).toBe("");
  });
  it("join defaults a missing time to 00:00 when date present", () => {
    expect(joinLocal("2026-08-15", "")).toBe("2026-08-15T00:00");
  });
});
