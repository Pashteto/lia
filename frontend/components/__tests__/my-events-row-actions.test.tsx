import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { EventRow } from "@/components/MyEventsBrowse";
import { ssrFallbackEvents } from "@/lib/mock-events";

const noop = () => {};

describe("MyEventsBrowse EventRow (mobile)", () => {
  it("labels the overflow toggle with a visible «Действия», not a bare «···»", () => {
    const html = renderToStaticMarkup(
      <EventRow
        event={{ ...ssrFallbackEvents()[0], status: "draft" }}
        expanded={false}
        onToggleExpand={noop}
        onDuplicate={noop}
        duplicating={false}
      />,
    );
    expect(html).toContain("Действия");
    expect(html).toContain("···");
  });
});
