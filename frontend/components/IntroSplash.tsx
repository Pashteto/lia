"use client";

import { useEffect, useRef, useState, type CSSProperties } from "react";
import { BRAND } from "@/lib/brand";
import { SCATTER_DESKTOP, SCATTER_MOBILE, landingTransform, markSeen, shouldStartIntro } from "@/lib/intro-splash";

function storage(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

/** Both offset sets go out as custom properties; CSS picks one by media
 * query, so the server and client markup are identical (no hydration branch). */
function letterStyle(i: number): CSSProperties {
  const d = SCATTER_DESKTOP[i];
  const m = SCATTER_MOBILE[i];
  return {
    "--i": i,
    "--xd": `${d.x}px`, "--yd": `${d.y}px`, "--rd": `${d.r}deg`,
    "--xm": `${m.x}px`, "--ym": `${m.y}px`, "--rm": `${m.r}deg`,
  } as CSSProperties;
}

/** First-visit splash (handoff INTRO-SPLASH.md): сообща gathers from scattered
 * letters and flies into the feed header. CSS keyframes only; this component
 * just measures the landing target and ends the show. The pre-paint script
 * that decides html[data-intro] is rendered by the page (INTRO_INLINE_SCRIPT). */
export function IntroSplash({ sign }: { sign: string }) {
  const root = useRef<HTMLDivElement>(null);
  const word = useRef<HTMLSpanElement>(null);
  const [state, setState] = useState<"idle" | "run">("idle");
  // Guards the unmount cleanup below: only true once this effect instance
  // actually started the show (state "run"). Without it, StrictMode's dev
  // double-invoke (mount → cleanup → mount) would have the first instance's
  // cleanup flip data-intro to "done" before fonts.ready ever resolves,
  // making the second (real) instance's early-return bail and killing the
  // splash in dev.
  const started = useRef(false);

  useEffect(() => {
    const html = document.documentElement;
    const el = root.current;
    if (html.dataset.intro !== "play" || !el) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let remeasure: ReturnType<typeof setTimeout> | undefined;
    const finish = () => {
      if (html.dataset.intro !== "play") return;
      markSeen(storage());
      html.dataset.intro = "done";
    };
    // Measures the landing target and writes --dx/--dy/--s from fresh rects.
    // Must run before .intro-flight gets its transform (the 2.1s flight
    // delay), so it can be called both at start and again just before the
    // flight to absorb any header reflow (e.g. a banner mounting) in between.
    const measure = () => {
      if (!word.current) return false;
      const target = document.querySelector("header [data-wordmark]");
      if (!target) return false;
      const t = landingTransform(word.current.getBoundingClientRect(), target.getBoundingClientRect());
      el.style.setProperty("--dx", `${t.dx}px`);
      el.style.setProperty("--dy", `${t.dy}px`);
      el.style.setProperty("--s", String(t.scale));
      return true;
    };
    document.fonts.ready.then(() => {
      if (cancelled || !word.current) return;
      // Too late to start (slow hydration/fonts): skip cleanly to the feed
      // rather than let the paint-relative CSS fail-safe cut the show.
      if (!shouldStartIntro(performance.now())) return finish();
      if (!measure()) return finish();
      started.current = true;
      setState("run");
      // Banners (VerifyEmailBanner, SessionExpiredBanner) can mount above
      // the header between now and the 2.1s flight start and shift it, so
      // re-measure right before the flight begins.
      remeasure = setTimeout(measure, 2000);
    });
    const onEnd = (e: AnimationEvent) => {
      if (e.target === el && e.animationName === "intro-lift") timer = setTimeout(finish, 350);
    };
    el.addEventListener("animationend", onEnd);
    el.addEventListener("click", finish);
    window.addEventListener("keydown", finish);
    return () => {
      cancelled = true;
      clearTimeout(timer);
      clearTimeout(remeasure);
      el.removeEventListener("animationend", onEnd);
      el.removeEventListener("click", finish);
      window.removeEventListener("keydown", finish);
      // Real unmount (route away) mid-show: leave the run in a finished
      // state so a later remount of the layout never finds a stale "play".
      if (started.current && html.dataset.intro === "play") html.dataset.intro = "done";
    };
  }, []);

  return (
    <div ref={root} className="intro" data-state={state} aria-hidden="true">
      <div className="intro-stage">
        <div className="intro-flight">
          <span ref={word} className="wordmark intro-word">
            {[...BRAND.wordmark].map((ch, i) => (
              <span key={i} className="intro-letter" style={letterStyle(i)}>
                {ch}
              </span>
            ))}
          </span>
        </div>
        <div className="intro-rule" />
        <div className="intro-sign">
          <span>{BRAND.domain.toUpperCase()}</span>
          <span>{sign}</span>
        </div>
      </div>
      <p className="intro-caption">Вместе · общими усилиями</p>
    </div>
  );
}
