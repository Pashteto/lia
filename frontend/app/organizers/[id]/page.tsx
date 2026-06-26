"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getPublicOrganizer } from "@/lib/api";

export default function PublicOrganizerPage() {
  const params = useParams<{ id: string }>();
  const [org, setOrg] = useState<{
    id: string;
    name: string;
    description: string;
    website_url: string;
    verified: boolean;
  } | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    getPublicOrganizer(params.id)
      .then((o) => (o ? setOrg(o) : setNotFound(true)))
      .catch(() => setNotFound(true));
  }, [params.id]);

  if (notFound)
    return (
      <main className="mx-auto max-w-3xl px-4 py-12">
        <p>Организатор не найден.</p>
      </main>
    );
  if (!org) return null;

  return (
    <main className="mx-auto max-w-3xl px-4 py-12 space-y-4">
      <div className="flex items-center gap-2">
        <h1 className="text-3xl font-bold tracking-[-0.022em]">{org.name}</h1>
        {org.verified && (
          <span className="rounded-full bg-accent/10 px-2 py-0.5 text-sm font-medium text-accent">
            ✓ Проверен
          </span>
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
