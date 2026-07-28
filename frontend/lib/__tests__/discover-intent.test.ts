import { describe, expect, it } from "vitest";
import {
  DISCOVER_CHIPS,
  intentFromChip,
  parseDiscoverQuery,
} from "../discover-intent";

describe("DISCOVER_CHIPS", () => {
  it("lists the four handoff labels in order", () => {
    expect(DISCOVER_CHIPS.map((c) => c.label)).toEqual([
      "Тихое в выходные",
      "Бесплатно рядом",
      "Для двоих вечером",
      "С детьми",
    ]);
  });
});

describe("intentFromChip", () => {
  it("maps each chip id", () => {
    expect(intentFromChip("quiet_weekend")).toMatchObject({
      id: "quiet_weekend",
      wantsNearby: false,
      keyword: "",
    });
    expect(intentFromChip("free_nearby").wantsNearby).toBe(true);
    expect(intentFromChip("evening_pair").id).toBe("evening_pair");
    expect(intentFromChip("with_kids").id).toBe("with_kids");
  });
});

describe("parseDiscoverQuery", () => {
  it("returns null for blank input", () => {
    expect(parseDiscoverQuery("")).toBeNull();
    expect(parseDiscoverQuery("   ")).toBeNull();
  });
  it("detects free / weekend / evening / kids / quiet stems", () => {
    expect(parseDiscoverQuery("что-нибудь бесплатное").id).toBe("free_nearby");
    expect(parseDiscoverQuery("на выходные").id).toBe("quiet_weekend");
    expect(parseDiscoverQuery("вечером вдвоём").id).toBe("evening_pair");
    expect(parseDiscoverQuery("с детьми в музей").id).toBe("with_kids");
    expect(parseDiscoverQuery("тихое место").id).toBe("quiet_any");
  });
  it("falls back to keyword with trimmed sourceLabel", () => {
    const i = parseDiscoverQuery("  медиация гараж  ");
    expect(i).toEqual({
      id: "keyword",
      sourceLabel: "медиация гараж",
      keyword: "медиация гараж",
      wantsNearby: false,
    });
  });
});
