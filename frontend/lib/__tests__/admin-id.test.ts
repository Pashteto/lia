import { describe, expect, it } from "vitest";
import { adminShortId, adminUserShortId } from "../admin-id";

describe("adminShortId", () => {
  it("prefixes EV- and uses first 4 hex", () => {
    expect(adminShortId("a1b2c3d4-5678-4012-8000-000000000001")).toBe("EV-A1B2");
  });
});

describe("adminUserShortId", () => {
  it("returns the first four hex chars, uppercased, without a prefix", () => {
    expect(adminUserShortId("2f1a9c40-1111-2222-3333-444455556666")).toBe("2F1A");
  });

  it("returns an em dash for empty or malformed ids", () => {
    expect(adminUserShortId("")).toBe("—");
    expect(adminUserShortId("zzzz-nope")).toBe("—");
  });
});
