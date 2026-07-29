import { describe, expect, it } from "vitest";
import {
  hygieneIssueSource,
  hygieneIssueValue,
  hygieneKindLabel,
} from "@/lib/hygiene-labels";
import { priceLabel } from "@/lib/price-label";

describe("hygieneKindLabel", () => {
  it("names the two detected kinds", () => {
    expect(hygieneKindLabel("test_data")).toBe("Тестовые данные");
    expect(hygieneKindLabel("suspicious_price")).toBe("Подозрительная цена");
  });

  it("falls back for an unknown kind rather than printing the slug", () => {
    expect(hygieneKindLabel("something_new")).toBe("Требует проверки");
  });
});

describe("hygieneIssueValue", () => {
  it("quotes the event title for test data", () => {
    expect(
      hygieneIssueValue({ kind: "test_data", title: "QA Тур в Геленджик (Блок 8)" }),
    ).toBe("«QA Тур в Геленджик (Блок 8)»");
  });

  it("renders the offending price with the от prefix", () => {
    expect(
      hygieneIssueValue({ kind: "suspicious_price", title: "Лабораторная сцена", price_rub: 100500 }),
    ).toBe(priceLabel(100500, "from"));
  });

  it("falls back to the title when a price issue has no price", () => {
    expect(hygieneIssueValue({ kind: "suspicious_price", title: "Лабораторная сцена" })).toBe(
      "«Лабораторная сцена»",
    );
  });

  it("returns an em dash when there is nothing to show", () => {
    expect(hygieneIssueValue({ kind: "test_data", title: "" })).toBe("—");
  });
});

describe("hygieneIssueSource", () => {
  it("prefixes the organizer name", () => {
    expect(hygieneIssueSource({ kind: "test_data", title: "x", organizer_name: "QA Block8" })).toBe(
      "организатор QA Block8",
    );
  });

  it("returns null when the organizer is unknown", () => {
    expect(hygieneIssueSource({ kind: "test_data", title: "x" })).toBeNull();
    expect(hygieneIssueSource({ kind: "test_data", title: "x", organizer_name: "  " })).toBeNull();
  });
});
