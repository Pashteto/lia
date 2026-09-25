"use client";

import { useEffect, useRef, useState, type CSSProperties } from "react";
import { BRAND } from "@/lib/brand";
import { SCATTER_DESKTOP, SCATTER_MOBILE, landingTransform, markSeen } from "@/lib/intro-splash";

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
export function IntroSplash() {
  const root = useRef<HTMLDivElement>(null);
  const word = useRef<HTMLSpanElement>(null);
  const [state, setState] = useState<"idle" | "run">("idle");

  useEffect(() => {
    const html = document.documentElement;
    const el = root.current;
    if (html.dataset.intro !== "play" || !el) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const finish = () => {
      markSeen(storage());
      html.dataset.intro = "done";
    };
    document.fonts.ready.then(() => {
      if (cancelled || !word.current) return;
      const target = document.querySelector("header [data-wordmark]");
      if (!target) return finish();
      const t = landingTransform(word.current.getBoundingClientRect(), target.getBoundingClientRect());
      el.style.setProperty("--dx", `${t.dx}px`);
      el.style.setProperty("--dy", `${t.dy}px`);
      el.style.setProperty("--s", String(t.scale));
      setState("run");
    });
    const onEnd = (e: AnimationEvent) => {
      if (e.target === el && e.animationName === "intro-lift") timer = setTimeout(finish, 350);
    };
    el.addEventListener("animationend", onEnd);
    el.addEventListener("click", finish);
    return () => {
      cancelled = true;
      clearTimeout(timer);
      el.removeEventListener("animationend", onEnd);
      el.removeEventListener("click", finish);
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
          <span>МОСКВА · 2026</span>
        </div>
      </div>
      <p className="intro-caption">Вместе · общими усилиями</p>
    </div>
  );
}
