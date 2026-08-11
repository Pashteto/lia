import { describe, expect, it } from "vitest";

import { todayRange, tonightRange, weekendRange, weekRange } from "../mock-events";

// 2026-08-11 is a Tuesday.
const tue = new Date(2026, 7, 11, 14, 30);

describe("tonightRange", () => {
  it("spans [17:00 today, 00:00 tomorrow)", () => {
    const { from, to } = tonightRange(tue);
    expect(from).toEqual(new Date(2026, 7, 11, 17, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 12, 0, 0, 0, 0));
  });
  it("is stable when opened late in the evening", () => {
    const { from, to } = tonightRange(new Date(2026, 7, 11, 23, 45));
    expect(from).toEqual(new Date(2026, 7, 11, 17, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 12, 0, 0, 0, 0));
  });
});

describe("weekRange", () => {
  it("runs from today 00:00 to next Monday 00:00", () => {
    const { from, to } = weekRange(tue);
    expect(from).toEqual(new Date(2026, 7, 11, 0, 0, 0, 0));
    // Next Monday is 17 August 2026.
    expect(to).toEqual(new Date(2026, 7, 17, 0, 0, 0, 0));
  });
  it("on Monday covers the full week ahead", () => {
    const { from, to } = weekRange(new Date(2026, 7, 10, 9, 0)); // Mon 10 Aug
    expect(from).toEqual(new Date(2026, 7, 10, 0, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 17, 0, 0, 0, 0));
  });
  it("on Sunday covers just the remaining day", () => {
    const { from, to } = weekRange(new Date(2026, 7, 16, 9, 0)); // Sun 16 Aug
    expect(from).toEqual(new Date(2026, 7, 16, 0, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 17, 0, 0, 0, 0));
  });
});

describe("todayRange / weekendRange (regression)", () => {
  it("today is a single civil day", () => {
    const { from, to } = todayRange(tue);
    expect(from).toEqual(new Date(2026, 7, 11, 0, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 12, 0, 0, 0, 0));
  });
  it("weekend from a Tuesday is the coming Sat–Sun", () => {
    const { from, to } = weekendRange(tue);
    expect(from).toEqual(new Date(2026, 7, 15, 0, 0, 0, 0));
    expect(to).toEqual(new Date(2026, 7, 17, 0, 0, 0, 0));
  });
});
