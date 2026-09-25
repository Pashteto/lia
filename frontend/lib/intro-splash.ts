/** Pure logic for the first-visit splash (handoff INTRO-SPLASH.md).
 * Anything that can fail degrades to "skip" — the splash must never trap
 * the page behind an overlay. */

export const INTRO_SEEN_KEY = "soobscha:intro-seen";

type StorageGet = Pick<Storage, "getItem">;
type StorageSet = Pick<Storage, "setItem">;

export function readSeen(storage: StorageGet | null): boolean {
  if (!storage) return true;
  try {
    return storage.getItem(INTRO_SEEN_KEY) === "1";
  } catch {
    return true;
  }
}

export function markSeen(storage: StorageSet | null): void {
  try {
    storage?.setItem(INTRO_SEEN_KEY, "1");
  } catch {
    /* private mode / blocked storage: nothing to remember, nothing to break */
  }
}

export function shouldPlayIntro(o: { storage: StorageGet | null; reducedMotion: boolean }): boolean {
  return !o.reducedMotion && !readSeen(o.storage);
}

export type Offset = { x: number; y: number; r: number };
const o = (x: number, y: number, r: number): Offset => ({ x, y, r });

/** Start offsets (px, deg), letters с о о б щ а. */
export const SCATTER_DESKTOP: Offset[] = [o(-560, -300, -18), o(-260, 330, 12), o(80, -380, 22), o(380, 300, -14), o(620, -200, 9), o(-140, 380, -24)];
export const SCATTER_MOBILE: Offset[] = [o(-150, -300, -18), o(-110, 280, 12), o(40, -360, 22), o(150, 260, -14), o(160, -180, 9), o(-40, 340, -24)];

export type Box = { left: number; top: number; width: number; height: number };

/** FLIP: translate + uniform scale (transform-origin top left) taking the
 * big centred word onto the header wordmark. Letter-spacing stays -0.04em
 * throughout (em-relative), so a uniform scale is exact. */
export function landingTransform(from: Box, to: Box): { dx: number; dy: number; scale: number } {
  return { dx: to.left - from.left, dy: to.top - from.top, scale: to.height / from.height };
}

/** Runs before first paint (inline <script> in the overlay) so returning
 * visitors never see a flash of the overlay. Mirrors shouldPlayIntro. */
export const INTRO_INLINE_SCRIPT = `(function(){var d=document.documentElement,p="skip";try{if(!matchMedia("(prefers-reduced-motion: reduce)").matches&&sessionStorage.getItem("${INTRO_SEEN_KEY}")!=="1")p="play"}catch(e){}d.dataset.intro=p})();`;
