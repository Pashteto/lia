import { describe, expect, it } from "vitest";
import { statusChipVariant } from "../status-chip";

describe("statusChipVariant", () => {
  it("published family → active", () => {
    for (const s of ["Опубликовано", "Подтверждено", "Верифицирован"])
      expect(statusChipVariant(s)).toBe("active");
  });
  it("draft/past → default", () => {
    for (const s of ["Черновик", "Прошедшее"])
      expect(statusChipVariant(s)).toBe("default");
  });
  it("attention family → signal", () => {
    for (const s of ["На модерации", "Ожидает", "На проверке", "Тестовый"])
      expect(statusChipVariant(s)).toBe("signal");
  });
  it("unknown → default", () => {
    expect(statusChipVariant("Что-то ещё")).toBe("default");
  });
});
