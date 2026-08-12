import { describe, expect, it } from "vitest";
import { hostOf, matchPlatform, type TrustedPlatform } from "@/lib/platforms";

const LIST: TrustedPlatform[] = [
  { domainSuffix: "timepad.ru", displayName: "TimePad", category: "ticketing" },
  { domainSuffix: "xn--80atdujec4e.xn--p1ai", displayName: "Культура.РФ", category: "gov" },
];

describe("hostOf", () => {
  it("extracts lowercased host", () => {
    expect(hostOf("https://Org.TimePad.ru/event/1")).toBe("org.timepad.ru");
  });
  it("returns null for garbage", () => {
    expect(hostOf("not a url")).toBeNull();
  });
});

describe("matchPlatform", () => {
  it("matches subdomains", () => {
    expect(matchPlatform("https://org.timepad.ru/e/1", LIST)?.displayName).toBe("TimePad");
  });
  it("rejects suffix-bypass hosts", () => {
    expect(matchPlatform("https://timepad.ru.evil.com/e", LIST)).toBeNull();
  });
  it("matches punycode idn (browser URL punycodes the host)", () => {
    expect(matchPlatform("https://культура.рф/afisha", LIST)?.displayName).toBe("Культура.РФ");
  });
  it("returns null for unknown domains", () => {
    expect(matchPlatform("https://unknown.ru/e", LIST)).toBeNull();
  });
});
