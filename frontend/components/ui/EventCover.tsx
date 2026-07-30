import { cn } from "@/lib/cn";
import Image from "next/image";
import type { ReactNode } from "react";

/**
 * Event cover — plain photograph, edge-to-edge. No gradients, no decorative
 * glyphs (Swiss Grid P7.3).
 *
 * Takes a resolved URL rather than a LiaEvent: cover-resolution policy lives in
 * `lib/covers.ts` and is applied by the callers' adapters, which keeps this
 * component reusable across the feed card, the detail hero and the wizard
 * preview. Missing photo → `fallback` on the blank-cell tone.
 */
export function EventCover({
  src,
  sizes,
  priority,
  className,
  aspect = "aspect-[16/9]",
  fallback,
}: {
  src?: string;
  sizes: string;
  priority?: boolean;
  className?: string;
  /** Tailwind aspect-ratio utility, e.g. "aspect-[5/2]" (feed) or "aspect-[3/1]" (detail). */
  aspect?: string;
  /** Rendered instead of the photo when `src` is undefined. */
  fallback?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "relative w-full overflow-hidden",
        src ? "bg-paper" : "bg-cell-blank",
        aspect,
        className,
      )}
    >
      {src ? (
        <Image
          src={src}
          alt=""
          fill
          sizes={sizes}
          priority={priority}
          className="object-cover"
        />
      ) : (
        fallback
      )}
    </div>
  );
}
