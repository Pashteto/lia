import { describe, expect, it } from "vitest";
import {
  complaintsCountRu,
  complaintsSignalRu,
  submittedAgoRu,
  testDataSignalRu,
} from "../admin-copy";

/**
 * Admin copy defects from the 2026-07-31 sweep: the overview signal line
 * hard-coded the «события» form («1 события с тестовыми данными»), and the
 * moderation detail appended «назад» to whatever the relative formatter
 * returned — including an absolute date, giving «ПОДАНО 05.07 НАЗАД».
 */
describe("testDataSignalRu", () => {
  it("uses the singular for 1 — the reported defect", () => {
    expect(testDataSignalRu(1)).toBe("1 событие с тестовыми данными");
  });
  it("uses the few form for 2–4", () => {
    expect(testDataSignalRu(3)).toBe("3 события с тестовыми данными");
  });
  it("uses the many form for 5+ and the teens", () => {
    expect(testDataSignalRu(7)).toBe("7 событий с тестовыми данными");
    expect(testDataSignalRu(11)).toBe("11 событий с тестовыми данными");
  });
  it("uses the singular for 21, not the teens form", () => {
    expect(testDataSignalRu(21)).toBe("21 событие с тестовыми данными");
  });
});

describe("complaintsSignalRu", () => {
  it("pluralizes the same way", () => {
    expect(complaintsSignalRu(1)).toBe("1 событие с жалобами");
    expect(complaintsSignalRu(2)).toBe("2 события с жалобами");
    expect(complaintsSignalRu(9)).toBe("9 событий с жалобами");
  });
});

describe("complaintsCountRu", () => {
  it("pluralizes жалоба, which the inbox row printed as a bare «жалоб»", () => {
    expect(complaintsCountRu(1)).toBe("1 жалоба");
    expect(complaintsCountRu(3)).toBe("3 жалобы");
    expect(complaintsCountRu(11)).toBe("11 жалоб");
    expect(complaintsCountRu(21)).toBe("21 жалоба");
  });
});

describe("submittedAgoRu", () => {
  const now = new Date("2026-07-31T12:00:00+03:00");

  it("says назад only with a relative value", () => {
    expect(submittedAgoRu("2026-07-29T12:00:00+03:00", now)).toBe("подано 2 д назад");
    expect(submittedAgoRu("2026-07-31T09:00:00+03:00", now)).toBe("подано 3 ч назад");
  });

  it("drops назад once the value is an absolute date — the reported defect", () => {
    expect(submittedAgoRu("2026-07-05T12:00:00+03:00", now)).toBe("подано 05.07");
  });

  it("drops назад for «сейчас», which does not take it", () => {
    expect(submittedAgoRu("2026-07-31T11:59:40+03:00", now)).toBe("подано сейчас");
  });

  it("is an em dash when the timestamp is missing", () => {
    expect(submittedAgoRu(undefined, now)).toBe("—");
    expect(submittedAgoRu("", now)).toBe("—");
  });
});
