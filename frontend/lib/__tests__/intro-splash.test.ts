import { describe, expect, it } from "vitest";
import {
  INTRO_INLINE_SCRIPT, INTRO_SEEN_KEY, INTRO_START_BUDGET_MS, SCATTER_DESKTOP, SCATTER_MOBILE,
  landingTransform, markSeen, readSeen, shouldPlayIntro, shouldStartIntro,
} from "../intro-splash";

const mem = (init: Record<string, string> = {}) => {
  const m = new Map(Object.entries(init));
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), m };
};
const throwing = {
  getItem: () => { throw new DOMException("denied", "SecurityError"); },
  setItem: () => { throw new DOMException("denied", "SecurityError"); },
};

describe("shouldPlayIntro", () => {
  it("plays on a fresh session", () => {
    expect(shouldPlayIntro({ storage: mem(), reducedMotion: false })).toBe(true);
  });
  it("skips once seen this session", () => {
    expect(shouldPlayIntro({ storage: mem({ [INTRO_SEEN_KEY]: "1" }), reducedMotion: false })).toBe(false);
  });
  it("skips under prefers-reduced-motion", () => {
    expect(shouldPlayIntro({ storage: mem(), reducedMotion: true })).toBe(false);
  });
  it("skips (never blocks the page) when storage throws", () => {
    expect(shouldPlayIntro({ storage: throwing, reducedMotion: false })).toBe(false);
  });
  it("skips when storage is missing", () => {
    expect(shouldPlayIntro({ storage: null, reducedMotion: false })).toBe(false);
  });
});

describe("seen flag", () => {
  it("round-trips", () => {
    const s = mem();
    expect(readSeen(s)).toBe(false);
    markSeen(s);
    expect(readSeen(s)).toBe(true);
  });
  it("markSeen swallows storage errors", () => {
    expect(() => markSeen(throwing)).not.toThrow();
    expect(() => markSeen(null)).not.toThrow();
  });
});

describe("scatter offsets (INTRO-SPLASH.md)", () => {
  it("has one offset per letter of сообща", () => {
    expect(SCATTER_DESKTOP).toHaveLength(6);
    expect(SCATTER_MOBILE).toHaveLength(6);
  });
  it("matches the spec's first and last letters", () => {
    expect(SCATTER_DESKTOP[0]).toEqual({ x: -560, y: -300, r: -18 });
    expect(SCATTER_DESKTOP[5]).toEqual({ x: -140, y: 380, r: -24 });
    expect(SCATTER_MOBILE[3]).toEqual({ x: 150, y: 260, r: -14 });
  });
});

describe("landingTransform", () => {
  it("maps the 220px centred word onto a 15px header word", () => {
    const from = { left: 400, top: 340, width: 640, height: 220 };
    const to = { left: 20, top: 13, width: 44, height: 15 };
    const t = landingTransform(from, to);
    expect(t.scale).toBeCloseTo(15 / 220, 6);
    expect(from.left + t.dx).toBe(to.left);
    expect(from.top + t.dy).toBe(to.top);
  });
  it("is the identity when boxes coincide", () => {
    const b = { left: 1, top: 2, width: 3, height: 4 };
    expect(landingTransform(b, b)).toEqual({ dx: 0, dy: 0, scale: 1 });
  });
});

describe("shouldStartIntro (start budget)", () => {
  it("is 1500 ms", () => {
    expect(INTRO_START_BUDGET_MS).toBe(1500);
  });
  it("starts within the budget, skips past it", () => {
    expect(shouldStartIntro(0)).toBe(true);
    expect(shouldStartIntro(1500)).toBe(true);
    expect(shouldStartIntro(1501)).toBe(false);
  });
});

describe("INTRO_INLINE_SCRIPT", () => {
  /** Runs the pre-paint script against a fake document/window. */
  const run = (o: { reduced?: boolean; storage: ReturnType<typeof mem> | typeof throwing }) => {
    const documentElement = { dataset: {} as Record<string, string> };
    new Function("document", "matchMedia", "sessionStorage", INTRO_INLINE_SCRIPT)(
      { documentElement },
      () => ({ matches: !!o.reduced }),
      o.storage,
    );
    return documentElement.dataset.intro;
  };
  it("plays on a fresh session and burns the flag at once", () => {
    const s = mem();
    expect(run({ storage: s })).toBe("play");
    expect(s.m.get(INTRO_SEEN_KEY)).toBe("1");
  });
  it("skips (and does not write) once seen", () => {
    const s = mem({ [INTRO_SEEN_KEY]: "1" });
    expect(run({ storage: s })).toBe("skip");
  });
  it("skips without burning the flag under reduced motion", () => {
    const s = mem();
    expect(run({ storage: s, reduced: true })).toBe("skip");
    expect(s.m.has(INTRO_SEEN_KEY)).toBe(false);
  });
  it("skips when storage throws", () => {
    expect(run({ storage: throwing })).toBe("skip");
  });
});
