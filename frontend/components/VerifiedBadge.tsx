import Link from "next/link";

export function VerifiedBadge({ profileId }: { profileId?: string }) {
  const badge = (
    <span
      title="Подтверждённый организатор"
      className="inline-flex items-center gap-0.5 rounded-full bg-accent/10 px-1.5 py-0.5 text-xs font-medium text-accent"
    >
      ✓ Проверен
    </span>
  );
  if (!profileId) return badge;
  return (
    <Link href={`/organizers/${profileId}`} className="hover:opacity-70">
      {badge}
    </Link>
  );
}
