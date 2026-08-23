import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { ChipRow } from "@/components/ui/ChipRow";

describe("ChipRow", () => {
  it("scrolls horizontally and shows a right-edge fade hint", () => {
    const html = renderToStaticMarkup(
      <ChipRow>
        <span>Один</span>
        <span>Два</span>
      </ChipRow>,
    );
    expect(html).toContain("overflow-x-auto");
    expect(html).toContain("data-fade");
    expect(html).toContain("Один");
  });
});
