import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

// KEY is read at module load, so each case stubs the env and re-imports.
async function loadMap(key: string | undefined) {
  vi.resetModules();
  if (key === undefined) {
    vi.stubEnv("NEXT_PUBLIC_YANDEX_MAPS_KEY", "");
  } else {
    vi.stubEnv("NEXT_PUBLIC_YANDEX_MAPS_KEY", key);
  }
  const mod = await import("@/components/map/YandexMap");
  return mod.YandexMap;
}

describe("YandexMap loading placeholder", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
  });

  it("shows «Загружаем карту…» while ymaps has not initialised (SSR/first paint)", async () => {
    const YandexMap = await loadMap("test-key");
    const html = renderToStaticMarkup(<YandexMap center={[55.7, 37.6]} />);
    expect(html).toContain("Загружаем карту…");
  });

  it("keeps the keyless fallback text", async () => {
    const YandexMap = await loadMap(undefined);
    const html = renderToStaticMarkup(<YandexMap center={[55.7, 37.6]} />);
    expect(html).toContain("Карта недоступна");
    expect(html).not.toContain("Загружаем карту…");
  });
});
