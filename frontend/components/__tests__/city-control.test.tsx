import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { CityControl, CityMenu, CITY_OPTIONS } from "@/components/ui/CityControl";

describe("CityControl", () => {
  it("renders a real button (not a bare caption) labelled with the current city code", () => {
    const html = renderToStaticMarkup(<CityControl />);
    expect(html).toContain("<button");
    expect(html).toContain("МСК");
    expect(html).toContain('aria-haspopup="listbox"');
    expect(html).toContain('aria-expanded="false"');
  });

  it("keeps the menu closed by default", () => {
    const html = renderToStaticMarkup(<CityControl />);
    expect(html).not.toContain("Санкт-Петербург");
  });
});

describe("CityMenu", () => {
  it("marks Moscow as the current city and Petersburg as coming soon (disabled)", () => {
    const html = renderToStaticMarkup(<CityMenu onClose={() => {}} />);
    expect(html).toContain("Москва");
    expect(html).toContain('aria-selected="true"');
    expect(html).toContain("Санкт-Петербург");
    expect(html).toContain("Скоро");
    expect(html).toContain("disabled");
  });
});

describe("CITY_OPTIONS", () => {
  it("has exactly one available city (Moscow)", () => {
    expect(CITY_OPTIONS.filter((c) => c.available).map((c) => c.name)).toEqual(["Москва"]);
  });
});
