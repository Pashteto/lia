import { describe, expect, it } from "vitest";
import {
  civil,
  civilKey,
  monthCaption,
  monthGridTrimmed,
  monthTitle,
  sameMonth,
  selectedDayLabel,
  selectedDayLabelLong,
} from "../calendar";

describe("monthTitle / monthCaption", () => {
  it("nominative month + full year", () => {
    expect(monthTitle(civil(2026, 6, 1))).toBe("Июль 2026");
    expect(monthTitle(civil(2026, 0, 1))).toBe("Январь 2026");
  });
  it("mobile caption is uppercase with an apostrophed year", () => {
    expect(monthCaption(civil(2026, 6, 1))).toBe("ИЮЛЬ ’26");
  });
});

describe("selectedDayLabel", () => {
  it("day, genitive month, short weekday", () => {
    // 12 July 2026 is a Sunday.
    expect(selectedDayLabel(civil(2026, 6, 12))).toBe("12 июля, вс");
    // 13 July 2026 is a Monday.
    expect(selectedDayLabel(civil(2026, 6, 13))).toBe("13 июля, пн");
  });
  it("long form spells the weekday out", () => {
    expect(selectedDayLabelLong(civil(2026, 6, 12))).toBe("12 июля, воскресенье");
  });
});

describe("monthGridTrimmed", () => {
  it("keeps Monday-first alignment and covers the whole month", () => {
    const cells = monthGridTrimmed(civil(2026, 6, 1));
    expect(cells.length % 7).toBe(0);
    expect(cells[0].getUTCDay()).toBe(1); // Monday
    expect(cells.some((c) => civilKey(c) === "2026-07-01")).toBe(true);
    expect(cells.some((c) => civilKey(c) === "2026-07-31")).toBe(true);
  });
  it("drops a trailing week that is entirely in the next month", () => {
    // June 2026 starts on a Monday with 30 days → exactly 5 rows, so
    // monthGrid()'s 6th row is all July and must be dropped.
    const june = civil(2026, 5, 1);
    const juneCells = monthGridTrimmed(june);
    expect(juneCells.length).toBe(35);
    expect(juneCells.slice(28).some((c) => sameMonth(c, june))).toBe(true);
  });
  it("keeps the sixth row when the month needs it", () => {
    // March 2026 starts on a Sunday (Monday index 6) with 31 days → 6 rows.
    const march = civil(2026, 2, 1);
    const marchCells = monthGridTrimmed(march);
    expect(marchCells.length).toBe(42);
    expect(marchCells.slice(35).some((c) => sameMonth(c, march))).toBe(true);
  });
});
