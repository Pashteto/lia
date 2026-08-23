import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { FormError } from "@/components/AuthForm";

describe("FormError", () => {
  it("renders a visible alert with the message", () => {
    const html = renderToStaticMarkup(
      <FormError error="Неверный email или пароль" />,
    );
    expect(html).toContain('role="alert"');
    expect(html).toContain("Неверный email или пароль");
    expect(html).toContain("text-signal");
  });

  it("renders nothing when there is no error", () => {
    expect(renderToStaticMarkup(<FormError error={null} />)).toBe("");
  });
});
