import { CategoryGlyph } from "@/components/ui/CategoryGlyph";
import { cn } from "@/lib/cn";
import { coverGradient, coverPhoto } from "@/lib/covers";
import type { LiaEvent } from "@/lib/types";
import Image from "next/image";

/**
 * Event cover image with a designed fallback. A real photo (uploaded cover or a
 * curated demo photo) renders over a category-tinted gradient; when there is no
 * photo, the gradient plus a corner category glyph stands in — so an imageless
 * event still has a stable, recognizable identity instead of a blank box.
 *
 * The gradient also sits behind every photo, so it doubles as the load-in
 * background (no flash of empty fill while the image streams in).
 */
export function EventCover({
  event,
  sizes,
  priority,
  className,
  aspect = "aspect-[16/9]",
  rounded,
}: {
  event: LiaEvent;
  sizes: string;
  priority?: boolean;
  className?: string;
  /** Tailwind aspect-ratio utility, e.g. "aspect-[16/9]" (detail) or "aspect-[3/2]" (card). */
  aspect?: string;
  /** Tailwind radius utility, e.g. "rounded-card". Cards clip via their own overflow. */
  rounded?: string;
}) {
  const photo = coverPhoto(event);
  const { from, to } = coverGradient(event);
  const slug = event.categories[0]?.slug;

  return (
    <div
      className={cn("relative w-full overflow-hidden", aspect, rounded, className)}
      style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
    >
      {/* Soft top-left light for depth. */}
      <div
        aria-hidden
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(120% 120% at 18% 12%, rgba(255,255,255,0.22), transparent 55%)",
        }}
      />

      {photo ? (
        <Image
          src={photo}
          alt=""
          fill
          sizes={sizes}
          priority={priority}
          className="object-cover transition duration-300 group-hover:scale-[1.02]"
        />
      ) : (
        <CategoryGlyph
          slug={slug}
          className="absolute -bottom-6 -right-5 h-2/3 w-auto text-white/15"
        />
      )}
    </div>
  );
}
