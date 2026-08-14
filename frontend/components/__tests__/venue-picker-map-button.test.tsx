import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { MapPinButton } from "@/components/VenuePicker";

describe("VenuePicker «Указать на карте»", () => {
  it("renders as a bordered uppercase button, not link-styled text", () => {
    const html = renderToStaticMarkup(<MapPinButton onClick={() => {}} />);
    expect(html).toContain("Указать на карте");
    expect(html).toContain("border");
    expect(html).toContain("uppercase");
  });
});
