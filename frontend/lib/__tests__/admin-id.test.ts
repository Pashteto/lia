import { describe, expect, it } from "vitest";
import { adminShortId, adminUserShortId } from "../admin-id";

/**
 * The short id used to be the first 4 hex of the UUID. Every seeded event is
 * b0000000-…, so seven different moderation rows all displayed EV-B000 and the
 * id was useless for telling rows apart. It has to depend on the WHOLE uuid.
 */
const SEEDED = [
  "b0000000-0000-4000-8000-000000000001",
  "b0000000-0000-4000-8000-000000000002",
  "b0000000-0000-4000-8000-000000000003",
  "b0000000-0000-4000-8000-000000000004",
  "b0000000-0000-4000-8000-000000000005",
  "b0000000-0000-4000-8000-000000000006",
  "b0000000-0000-4000-8000-000000000007",
];

describe("adminShortId", () => {
  it("gives every seeded event its own id — the reported collision", () => {
    expect(new Set(SEEDED.map(adminShortId)).size).toBe(SEEDED.length);
  });

  it("differs on a change anywhere in the uuid, not just the head", () => {
    expect(adminShortId("b0000000-0000-4000-8000-000000000001")).not.toBe(
      adminShortId("b0000000-0000-4000-8000-000000000002"),
    );
  });

  it("is stable across calls, so a row keeps its id", () => {
    expect(adminShortId(SEEDED[0])).toBe(adminShortId(SEEDED[0]));
  });

  it("ignores dash placement — the same uuid is the same id", () => {
    expect(adminShortId("b0000000-0000-4000-8000-000000000001")).toBe(
      adminShortId("b0000000000040008000000000000001"),
    );
  });

  it("keeps the EV- prefix and a 4-character body for the 44px column", () => {
    for (const id of SEEDED) {
      expect(adminShortId(id)).toMatch(/^EV-[0-9A-Z]{4}$/);
    }
  });

  it("returns the fallback for empty or malformed ids", () => {
    expect(adminShortId("")).toBe("EV-————");
    expect(adminShortId("zzzz-nope")).toBe("EV-————");
  });
});

describe("adminUserShortId", () => {
  it("separates users sharing a uuid prefix", () => {
    const ids = new Set(
      [
        "c0000000-0000-4000-8000-000000000001",
        "c0000000-0000-4000-8000-000000000002",
        "c0000000-0000-4000-8000-000000000003",
      ].map(adminUserShortId),
    );
    expect(ids.size).toBe(3);
  });

  it("stays a bare 4-character id with no prefix", () => {
    expect(adminUserShortId("2f1a9c40-1111-2222-3333-444455556666")).toMatch(
      /^[0-9A-Z]{4}$/,
    );
  });

  it("returns an em dash for empty or malformed ids", () => {
    expect(adminUserShortId("")).toBe("—");
    expect(adminUserShortId("zzzz-nope")).toBe("—");
  });
});
