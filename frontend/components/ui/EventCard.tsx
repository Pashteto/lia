import { cn } from "@/lib/cn";
import { formatAttendance, formatEventDate, formatPrice } from "@/lib/format";
import type { LiaEvent } from "@/lib/types";
import Image from "next/image";
import Link from "next/link";
import { Kicker } from "./Kicker";

/**
 * Event content card: 18px radius, subtle system shadow, no hairline border
 * (DESIGN.md). Content surface — no Liquid Glass.
 */
export function EventCard({
  event,
  className,
}: {
  event: LiaEvent;
  className?: string;
}) {
  const attendance = formatAttendance(event);
  return (
    <Link
      href={`/events/${event.id}`}
      className={cn(
        "group block overflow-hidden rounded-card bg-bg-secondary shadow-card-subtle transition hover:shadow-card",
        className,
      )}
    >
      {event.coverUrl && (
        <div className="relative aspect-[3/2] w-full overflow-hidden bg-fill">
          <Image
            src={event.coverUrl}
            alt=""
            fill
            sizes="(max-width: 768px) 100vw, 360px"
            className="object-cover transition duration-300 group-hover:scale-[1.02]"
          />
        </div>
      )}
      <div className="space-y-2 p-4">
        <div className="flex items-center justify-between gap-2">
          <Kicker>{event.category.label}</Kicker>
          <span className="text-[13px] text-label-secondary">
            {event.format === "online" ? "Онлайн" : "Очно"}
          </span>
        </div>
        <h3 className="text-[17px] font-semibold leading-snug">{event.title}</h3>
        <p className="text-[13px] text-label-secondary">
          {formatEventDate(event.startsAt)}
          {event.venue ? ` · ${event.venue.name}` : ""}
        </p>
        <div className="flex items-center justify-between pt-1">
          <span className="text-[15px] font-medium">{formatPrice(event)}</span>
          {attendance && (
            <span className="text-[13px] text-label-secondary">{attendance}</span>
          )}
        </div>
      </div>
    </Link>
  );
}
