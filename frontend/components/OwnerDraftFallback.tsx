"use client";

import { EventDetailView } from "@/components/EventDetailView";
import { fetchEventWithAuth } from "@/lib/api";
import type { LiaEvent } from "@/lib/types";
import { notFound } from "next/navigation";
import { useEffect, useState } from "react";

// Client-side fallback for /events/[id] when the anonymous server fetch misses.
//
// The detail page fetches GET /events/{id} server-side WITHOUT auth (the session
// token lives in localStorage, unreachable from a server component). The backend
// hides non-published events from anonymous callers, so an owner viewing their
// own draft — right after creating it, or from "Мои события" — would 404.
//
// This retries with the bearer token. If the event resolves, it's the owner's
// draft and we render it. If it's still missing (truly gone, or not ours), we
// fall through to the real 404.
export function OwnerDraftFallback({ id }: { id: string }) {
  const [state, setState] = useState<
    { kind: "loading" } | { kind: "found"; event: LiaEvent } | { kind: "missing" }
  >({ kind: "loading" });

  useEffect(() => {
    let active = true;
    fetchEventWithAuth(id)
      .then((event) => {
        if (!active) return;
        setState(event ? { kind: "found", event } : { kind: "missing" });
      })
      .catch(() => {
        if (active) setState({ kind: "missing" });
      });
    return () => {
      active = false;
    };
  }, [id]);

  if (state.kind === "found") {
    return <EventDetailView event={state.event} />;
  }

  if (state.kind === "missing") {
    notFound();
  }

  // Loading: a minimal placeholder that matches the detail page background so
  // there's no flash of the 404 screen while the authenticated retry resolves.
  return <div className="min-h-screen bg-bg" />;
}
