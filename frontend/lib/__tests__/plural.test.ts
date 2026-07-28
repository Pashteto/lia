import { describe, expect, it } from "vitest";
import { pluralRu } from "../plural";

const FORMS: [string, string, string] = ["событие", "события", "событий"];

describe("pluralRu", () => {
  it("singular", () => {
    expect(pluralRu(1, FORMS)).toBe("событие");
    expect(pluralRu(21, FORMS)).toBe("событие");
  });
  it("few", () => {
    expect(pluralRu(2, FORMS)).toBe("события");
    expect(pluralRu(42, FORMS)).toBe("события");
  });
  it("many + teens", () => {
    expect(pluralRu(0, FORMS)).toBe("событий");
    expect(pluralRu(5, FORMS)).toBe("событий");
    expect(pluralRu(11, FORMS)).toBe("событий");
    expect(pluralRu(14, FORMS)).toBe("событий");
    expect(pluralRu(111, FORMS)).toBe("событий");
  });
});
