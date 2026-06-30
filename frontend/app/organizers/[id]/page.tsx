"use client";

import { useEffect, useState, useMemo } from "react";
import { useParams } from "next/navigation";
import { followOrganizer, getPublicOrganizer, unfollowOrganizer, fetchEventsByOrganizer } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/Button";
import { VerifiedBadge } from "@/components/VerifiedBadge";
import { EventCard } from "@/components/ui/EventCard";
import type { LiaEvent } from "@/lib/types";

export default function PublicOrganizerPage() {
  const params = useParams<{ id: string }>();
  const { isAuthed, ready } = useAuth();
  const [org, setOrg] = useState<{
    id: string;
    name: string;
    description: string;
    website_url: string;
    verified: boolean;
    is_following: boolean;
  } | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [following, setFollowing] = useState(false);
  const [pending, setPending] = useState(false);
  const [events, setEvents] = useState<LiaEvent[]>([]);

  useEffect(() => {
    getPublicOrganizer(params.id)
      .then((o) => {
        if (o) {
          setOrg(o);
          setFollowing(o.is_following);
        } else {
          setNotFound(true);
        }
      })
      .catch(() => setNotFound(true));
  }, [params.id]);

  useEffect(() => {
    fetchEventsByOrganizer(params.id)
      .then(setEvents)
      .catch(() => setEvents([]));
  }, [params.id]);

  const { upcoming, past } = useMemo(() => {
    const now = new Date().getTime();
    const upcoming = events
      .filter((e) => new Date(e.startsAt).getTime() >= now)
      .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime());
    const past = events
      .filter((e) => new Date(e.startsAt).getTime() < now)
      .sort((a, b) => new Date(b.startsAt).getTime() - new Date(a.startsAt).getTime())
      .slice(0, 10);
    return { upcoming, past };
  }, [events]);

  async function toggleFollow() {
    if (!org || pending) return;
    const next = !following;
    setFollowing(next); // optimistic
    setPending(true);
    try {
      if (next) await followOrganizer(org.id);
      else await unfollowOrganizer(org.id);
    } catch {
      setFollowing(!next); // revert on failure
    } finally {
      setPending(false);
    }
  }

  if (notFound)
    return (
      <main className="mx-auto max-w-3xl px-4 py-12">
        <p>Организатор не найден.</p>
      </main>
    );
  if (!org) return null;

  return (
    <main className="mx-auto max-w-3xl px-4 py-12 space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-3xl font-bold tracking-[-0.022em]">{org.name}</h1>
        {org.verified && <VerifiedBadge />}
        {ready && isAuthed && (
          <Button
            variant={following ? "tinted" : "filled"}
            onClick={toggleFollow}
            disabled={pending}
            className="ml-auto"
          >
            {following ? "Вы подписаны" : "Подписаться"}
          </Button>
        )}
      </div>
      {org.description && (
        <p className="text-label-secondary">{org.description}</p>
      )}
      {org.website_url && (
        <a
          href={org.website_url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-accent"
        >
          {org.website_url}
        </a>
      )}
      <section className="space-y-3 pt-4">
        <h2 className="text-xl font-semibold tracking-[-0.022em]">Предстоящие мероприятия</h2>
        {upcoming.length === 0 ? (
          <p className="text-label-secondary">Пока нет предстоящих мероприятий.</p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {upcoming.map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        )}
      </section>

      {past.length > 0 && (
        <section className="space-y-3 pt-4">
          <h2 className="text-xl font-semibold tracking-[-0.022em]">Прошедшие мероприятия</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {past.map((e) => (
              <EventCard key={e.id} event={e} />
            ))}
          </div>
        </section>
      )}
    </main>
  );
}
