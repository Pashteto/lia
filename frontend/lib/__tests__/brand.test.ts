import { describe, expect, it } from "vitest";
import { BRAND, pageTitle } from "../brand";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (name === "node_modules" || name === "__tests__" || name.startsWith(".")) return [];
    return statSync(p).isDirectory() ? sourceFiles(p) : /\.(tsx?|css)$/.test(name) ? [p] : [];
  });
}

describe("brand", () => {
  it("carries the Сообща identity from the handoff tokens", () => {
    expect(BRAND.name).toBe("Сообща");
    expect(BRAND.wordmark).toBe("сообща");
    expect(BRAND.monogram).toBe("сб");
    expect(BRAND.domain).toBe("soobscha.ru");
  });
  it("titles a section page as «<section> — Сообща»", () => {
    expect(pageTitle("Вход")).toBe("Вход — Сообща");
  });
  it("titles the root as «Сообща — События»", () => {
    expect(pageTitle()).toBe("Сообща — События");
  });
});

describe("rebrand guard", () => {
  it("no source file under app/, components/, lib/ says Presence", () => {
    const root = join(__dirname, "..", "..");
    const offenders = ["app", "components", "lib"]
      .flatMap((d) => sourceFiles(join(root, d)))
      .filter((f) => /presence/i.test(readFileSync(f, "utf8")));
    expect(offenders).toEqual([]);
  });
});
