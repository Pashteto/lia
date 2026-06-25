import { cn } from "@/lib/cn";

/**
 * Minimal line glyph for an event category, drawn as a watermark on covers that
 * have no photograph. Hand-authored 24×24 stroke paths (not an icon dependency)
 * so the imageless state reads as a deliberate identity, not a missing image.
 * Unknown / missing slugs fall back to a calendar.
 */
const PATHS: Record<string, React.ReactNode> = {
  // Лекции — a presentation screen on a stand.
  lecture: (
    <>
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M2 20h20M12 16v4" />
    </>
  ),
  // Мастер-классы — a potter's wheel (on-theme for ceramics et al.).
  workshop: (
    <>
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="3.5" />
    </>
  ),
  // Медиации — a guided path between two points.
  mediation: (
    <>
      <circle cx="6" cy="6.5" r="2.2" />
      <circle cx="18" cy="17.5" r="2.2" />
      <path d="M8 7.5c6 1 8 4 8 8.5" />
    </>
  ),
  // Концерты — a music note.
  concert: (
    <>
      <path d="M9 18V5l10-2v11" />
      <circle cx="6.5" cy="18" r="2.5" />
      <circle cx="16.5" cy="16" r="2.5" />
    </>
  ),
  // Выставки — a framed landscape (a painting on the wall).
  exhibition: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="m6 16 4-5 3 3 2-2 3 4" />
    </>
  ),
  // Спектакли — a stage spotlight cone.
  performance: (
    <>
      <circle cx="12" cy="3.5" r="1.5" />
      <path d="M11 5 6 21M13 5l5 16M6 21h12" />
    </>
  ),
  // Кино — a film strip.
  film: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M7 3v18M17 3v18M3 8h4M3 16h4M17 8h4M17 16h4" />
    </>
  ),
  // Фестивали — a burst of light.
  festival: (
    <>
      <circle cx="12" cy="12" r="3.5" />
      <path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M5 19l2-2" />
    </>
  ),
};

const FALLBACK = (
  <>
    <rect x="3" y="4" width="18" height="17" rx="2" />
    <path d="M3 9h18M8 2v4M16 2v4" />
  </>
);

export function CategoryGlyph({
  slug,
  className,
}: {
  slug?: string;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.25}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      className={cn(className)}
    >
      {(slug && PATHS[slug]) || FALLBACK}
    </svg>
  );
}
