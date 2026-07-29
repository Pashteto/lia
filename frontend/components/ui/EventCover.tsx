import { cn } from "@/lib/cn";
import { coverPhoto } from "@/lib/covers";
import type { LiaEvent } from "@/lib/types";
import Image from "next/image";

/**
 * Event cover — plain photograph, edge-to-edge. No gradients, no decorative
 * glyphs (Swiss Grid P7.3). Missing photo → paper fill.
 */
export function EventCover({
  event,
  sizes,
  priority,
  className,
  aspect = "aspect-[16/9]",
}: {
  event: LiaEvent;
  sizes: string;
  priority?: boolean;
  className?: string;
  /** Tailwind aspect-ratio utility, e.g. "aspect-[16/9]" (detail) or "aspect-[3/2]" (card). */
  aspect?: string;
}) {
  const photo = coverPhoto(event);

  return (
    <div className={cn("relative w-full overflow-hidden bg-paper", aspect, className)}>
      {photo ? (
        <Image
          src={photo}
          alt=""
          fill
          sizes={sizes}
          priority={priority}
          className="object-cover"
        />
      ) : null}
    </div>
  );
}
