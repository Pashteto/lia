import { describe, expect, it } from "vitest";
import {
  REJECT_REASON_CHIPS,
  concatenateReasons,
  revisionReason,
} from "../admin-reject-reasons";

describe("reject reasons", () => {
  it("lists the handoff chips first, then the ones the queue asked for", () => {
    expect([...REJECT_REASON_CHIPS]).toEqual([
      "Тестовые данные",
      "Нет описания",
      "Обложка низкого качества",
      "Дубликат",
      "Не наша тема",
      "Неверная дата или время",
      "Нет источника",
      "Реклама или продажа",
    ]);
  });

  // The joined string is the only record of why an event was taken down, and
  // "; " is what splits it back apart. A chip carrying the separator would
  // silently become two reasons in the history.
  it("keeps every chip free of the separator", () => {
    for (const chip of REJECT_REASON_CHIPS) {
      expect(chip).not.toContain(";");
      expect(chip.trim()).toBe(chip);
      expect(chip).not.toBe("");
    }
  });

  it("has no duplicates", () => {
    expect(new Set(REJECT_REASON_CHIPS).size).toBe(REJECT_REASON_CHIPS.length);
  });
  it("concatenates with semicolon", () => {
    expect(concatenateReasons(["Тестовые данные", "Дубликат"])).toBe(
      "Тестовые данные; Дубликат",
    );
  });
  it("builds revision prefix", () => {
    expect(revisionReason(["Нет описания"])).toBe("На доработку: Нет описания");
  });
});
