import { describe, expect, it } from "vitest";
import {
  REJECT_REASON_CHIPS,
  concatenateReasons,
  revisionReason,
} from "../admin-reject-reasons";

describe("reject reasons", () => {
  it("lists four handoff chips", () => {
    expect([...REJECT_REASON_CHIPS]).toEqual([
      "Тестовые данные",
      "Нет описания",
      "Обложка низкого качества",
      "Дубликат",
    ]);
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
