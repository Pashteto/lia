import { describe, expect, it } from "vitest";
import { categoryNumeral } from "../category-numerals";

const CATS = [{ slug: "festival" }, { slug: "mediation" }, { slug: "lecture" }];

describe("categoryNumeral", () => {
  it("is 1-based and zero-padded", () => {
    expect(categoryNumeral("festival", CATS)).toBe("01");
    expect(categoryNumeral("lecture", CATS)).toBe("03");
  });
  it("unknown slug → em dash", () => {
    expect(categoryNumeral("nope", CATS)).toBe("—");
  });
  it("pads to width 2 up to 99", () => {
    const many = Array.from({ length: 12 }, (_, i) => ({ slug: `c${i}` }));
    expect(categoryNumeral("c11", many)).toBe("12");
  });
});
