import { describe, expect, it } from "vitest";
import { isLikelyTestContent } from "../admin-test-heuristic";

describe("isLikelyTestContent", () => {
  it("flags QA / test / bla bla", () => {
    expect(isLikelyTestContent("QA Block8")).toBe(true);
    expect(isLikelyTestContent("bla bla meet")).toBe(true);
    expect(isLikelyTestContent("Летний фестиваль")).toBe(false);
  });
});
