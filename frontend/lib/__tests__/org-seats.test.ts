import { describe, expect, it } from "vitest";
import { padCount, seatsFill } from "../org-seats";

describe("seatsFill", () => {
  it("derives filled from capacity - seatsRemaining", () => {
    expect(seatsFill({ capacity: 40, seatsRemaining: 12 })).toEqual({
      filled: 28,
      capacity: 40,
      label: "28 / 40",
      ratio: 0.7,
    });
  });
  it("returns null when capacity or seatsRemaining missing", () => {
    expect(seatsFill({ capacity: 40 })).toBeNull();
    expect(seatsFill({ seatsRemaining: 3 })).toBeNull();
    expect(seatsFill({})).toBeNull();
  });
});

describe("padCount", () => {
  it("pads to two digits", () => {
    expect(padCount(3)).toBe("03");
    expect(padCount(86)).toBe("86");
  });
});
