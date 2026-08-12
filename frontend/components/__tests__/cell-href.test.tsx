import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Cell } from "@/components/ui/Cell";

describe("Cell href affordance", () => {
  it("renders a link with arrow-suffixed caption when href is given", () => {
    const html = renderToStaticMarkup(
      <Cell caption="Черновики" value="01" mono href="/events/mine?status=draft" />,
    );
    expect(html).toContain("<a ");
    expect(html).toContain('href="/events/mine?status=draft"');
    expect(html).toContain("Черновики →");
    expect(html).toContain("hover-invert");
    expect(html).toContain("swiss-focus");
    expect(html).toContain("cursor-pointer");
  });

  it("stays an inert div without arrow when href is absent", () => {
    const html = renderToStaticMarkup(<Cell caption="Всего записей" value="01" mono />);
    expect(html).not.toContain("<a ");
    expect(html).not.toContain("→");
    expect(html).not.toContain("hover-invert");
  });

  it("keeps the invert styling on a linked cell", () => {
    const html = renderToStaticMarkup(
      <Cell caption="На модерации" value="00" invert href="/events/mine?status=pending_review" />,
    );
    expect(html).toContain("bg-on-surface");
    expect(html).toContain("На модерации →");
  });
});
