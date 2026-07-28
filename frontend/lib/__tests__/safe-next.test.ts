import { describe, expect, it } from "vitest";
import { safeNextPath } from "@/lib/safe-next";

const ORIGIN = "https://presence.example";

describe("safeNextPath", () => {
  it("rejects protocol-relative paths", () => {
    expect(safeNextPath("//evil.example", ORIGIN)).toBeNull();
  });

  it("rejects backslash bypass paths", () => {
    expect(safeNextPath("/\\evil.example", ORIGIN)).toBeNull();
  });

  it("rejects encoded backslash paths", () => {
    expect(safeNextPath("/%5Cevil.example", ORIGIN)).toBeNull();
    expect(safeNextPath("/%5cevil.example", ORIGIN)).toBeNull();
  });

  it("allows internal paths with query strings", () => {
    expect(safeNextPath("/me/calendar?view=week", ORIGIN)).toBe(
      "/me/calendar?view=week",
    );
  });

  it("allows root path", () => {
    expect(safeNextPath("/", ORIGIN)).toBe("/");
  });

  it("rejects absolute external URLs", () => {
    expect(safeNextPath("https://evil.example", ORIGIN)).toBeNull();
  });

  it("rejects empty and null values", () => {
    expect(safeNextPath("", ORIGIN)).toBeNull();
    expect(safeNextPath("   ", ORIGIN)).toBeNull();
    expect(safeNextPath(null, ORIGIN)).toBeNull();
  });
});
