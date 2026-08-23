import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { OwnerEventActions } from "@/components/OwnerEventActions";
import { ownerPanelMode } from "@/lib/owner-actions";
import { showSignupControls } from "@/lib/signup-availability";
import { ssrFallbackEvents } from "@/lib/mock-events";
import type { LiaEvent } from "@/lib/types";

function render(event: LiaEvent): string {
  const qc = new QueryClient();
  return renderToStaticMarkup(
    <QueryClientProvider client={qc}>
      <OwnerEventActions event={event} />
    </QueryClientProvider>,
  );
}

const base = ssrFallbackEvents()[0];

describe("ownerPanelMode", () => {
  it("maps draft and pending_review to panel modes, everything else to null", () => {
    expect(ownerPanelMode("draft")).toBe("draft");
    expect(ownerPanelMode("pending_review")).toBe("pending");
    expect(ownerPanelMode("published")).toBeNull();
    expect(ownerPanelMode("cancelled")).toBeNull();
    expect(ownerPanelMode("rejected")).toBeNull();
  });

  it("published + isOwner → published mode; ownerless published stays null", () => {
    expect(ownerPanelMode("published", true)).toBe("published");
    expect(ownerPanelMode("published", false)).toBeNull();
    expect(ownerPanelMode("cancelled", true)).toBeNull();
  });
});

describe("OwnerEventActions", () => {
  it("offers publish + edit on a draft", () => {
    const html = render({ ...base, status: "draft" });
    expect(html).toContain("Черновик");
    expect(html).toContain("Опубликовать");
    expect(html).toContain(`/events/${base.id}/edit`);
  });

  it("shows the pending state without a publish button", () => {
    const html = render({ ...base, status: "pending_review" });
    expect(html).toContain("На модерации");
    expect(html).not.toContain("Опубликовать");
    expect(html).toContain(`/events/${base.id}/edit`);
  });

  it("renders nothing for a published event of someone else", () => {
    expect(render({ ...base, status: "published" })).toBe("");
  });

  it("shows the management strip on the owner's published event", () => {
    const html = render({ ...base, status: "published", isOwner: true });
    expect(html).toContain("Ваше событие — опубликовано");
    expect(html).toContain(`/events/${base.id}/edit`);
    expect(html).toContain("Мои события");
    expect(html).not.toContain("Опубликовать");
  });
});

describe("showSignupControls", () => {
  it("hides signup controls from the owner", () => {
    expect(showSignupControls({ isOwner: true })).toBe(false);
    expect(showSignupControls({ isOwner: false })).toBe(true);
    expect(showSignupControls({})).toBe(true);
  });
});
