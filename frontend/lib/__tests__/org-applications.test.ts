import { describe, expect, it, vi } from "vitest";
import { applicationMeta, decideMany } from "../org-applications";

describe("applicationMeta", () => {
  it("uses trimmed answer when present — never invents history", () => {
    expect(applicationMeta("  была на 3  ")).toBe("была на 3");
    expect(applicationMeta("  была на 3  ", "accepted")).toBe("была на 3");
    expect(applicationMeta("ok", "declined")).toBe("ok");
  });

  it("returns Первая заявка only for applied + empty answer", () => {
    expect(applicationMeta("")).toBe("Первая заявка");
    expect(applicationMeta("   ")).toBe("Первая заявка");
    expect(applicationMeta(undefined)).toBe("Первая заявка");
    expect(applicationMeta(null)).toBe("Первая заявка");
    expect(applicationMeta("", "applied")).toBe("Первая заявка");
  });

  it("returns em dash for decided statuses with empty answer", () => {
    expect(applicationMeta("", "accepted")).toBe("—");
    expect(applicationMeta("   ", "declined")).toBe("—");
    expect(applicationMeta(undefined, "going")).toBe("—");
    expect(applicationMeta(null, "withdrawn")).toBe("—");
    expect(applicationMeta("", "cancelled")).toBe("—");
    expect(applicationMeta("", "waitlist")).toBe("—");
  });
});

describe("decideMany", () => {
  it("calls decide sequentially and partitions ok vs failed", async () => {
    const decide = vi.fn(async (_eventId: string, id: string) => {
      if (id === "b") throw new Error("nope");
    });

    const result = await decideMany("ev1", ["a", "b", "c"], "accept", decide);

    expect(decide.mock.calls.map((c) => c[1])).toEqual(["a", "b", "c"]);
    expect(decide).toHaveBeenCalledTimes(3);
    expect(decide.mock.calls[0][0]).toBe("ev1");
    expect(decide.mock.calls[0][2]).toBe("accept");
    expect(result).toEqual({ ok: ["a", "c"], failed: ["b"] });
  });

  it("returns empty partitions for empty ids", async () => {
    const decide = vi.fn();
    const result = await decideMany("ev1", [], "decline", decide);
    expect(decide).not.toHaveBeenCalled();
    expect(result).toEqual({ ok: [], failed: [] });
  });
});
