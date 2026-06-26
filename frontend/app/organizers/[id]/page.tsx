"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { followOrganizer, getPublicOrganizer, unfollowOrganizer } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/Button";
import { VerifiedBadge } from "@/components/VerifiedBadge";

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
      {/* Published events for this organizer are listed via the existing events
          list filtered to this organizer; wire once the event-list-by-organizer
          public filter is confirmed (spec §7.3). */}
    </main>
  );
}
