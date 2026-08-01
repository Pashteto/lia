import { describe, expect, it } from "vitest";
import { tileCount } from "../tile-count";

/**
 * The «ОРГАНИЗАТОРОВ» and «ПОЛЬЗОВАТЕЛЕЙ» tiles were hard-coded to «—» because
 * the payload carried no totals. Now that it does, an absent field must still
 * degrade to the dash rather than rendering 0 — which would be a lie.
 */
describe("tileCount", () => {
  it("pads small counts to the tile's two-digit mono form", () => {
    expect(tileCount(9)).toBe("09");
    expect(tileCount(28)).toBe("28");
  });

  it("leaves three-digit counts alone", () => {
    expect(tileCount(128)).toBe("128");
  });

  it("renders zero as a count, not a dash", () => {
    expect(tileCount(0)).toBe("00");
  });

  it("falls back to a dash when the backend omitted the field", () => {
    expect(tileCount(undefined)).toBe("—");
  });
});
