import Link from "next/link";
import { cn } from "@/lib/cn";
import { EventCover } from "@/components/ui/EventCover";

/** Cover `sizes` are px-only by design. Next builds the srcset ladder from
 * whether `sizes` carries a viewport unit: with any `vw` the smallest candidate
 * is 256w, with px-only values it starts at 32w. Since the desktop and mobile
 * layouts are separate DOM subtrees toggled by `hidden`/`sm:flex`, and a hidden
 * next/image still downloads, each string resolves to `1px` in the breakpoint
 * where its subtree is invisible — so the browser fetches a 32w placeholder
 * instead of a real image. */
const BAND_SIZES = "(max-width: 639px) 1px, (max-width: 1023px) 340px, 460px";
const THUMB_SIZES = "(min-width: 640px) 1px, 44px";

export interface EventModuleProps {
  numeral: string; // "01" — from categoryNumeral()
  category: string; // "Фестивали"
  title: string;
  venue: string;
  date: string; // preformatted, renders in mono
  price: string; // from priceLabel() — "FREE" | "800 ₽"
  href: string;
  /** Resolved cover URL from coverPhoto(); undefined → numeral plate. */
  cover?: string;
  /** U3 AI variant: match reason on a hairline top rule. */
  matchReason?: string;
  className?: string;
}

/** Swiss Grid feed card.
 * Desktop: 5:2 cover band (or numeral plate) → numeral/category row, 15px/900
 * title, venue caption, footer date(mono 700)+price.
 * Mobile: grid 44px 1fr auto — 44x44 cover / title / price, then the caption
 * carrying numeral · venue · date.
 * Single <Link> wrapper — never nest an <a> inside (React #418 fix). */
export function EventModule({
  numeral, category, title, venue, date, price, href, cover, matchReason, className,
}: EventModuleProps) {
  return (
    <Link
      href={href}
      className={cn("block h-full min-w-0 swiss-focus hover-invert", className)}
    >
      {/* Desktop — handoff U1 catalogue cell */}
      <div className="hidden h-full flex-col sm:flex">
        <EventCover
          src={cover}
          sizes={BAND_SIZES}
          aspect="aspect-[5/2]"
          className="flex-none border-b border-rule-inner"
          fallback={
            <div className="flex h-full items-center justify-between px-[14px]">
              <span className="font-mono text-[26px] font-bold tracking-[-0.02em] text-ink">
                {numeral}
              </span>
              <span className="cap">{category}</span>
            </div>
          }
        />
        <div className="flex flex-1 flex-col px-[14px] py-[11px]">
          <div className="flex items-baseline justify-between">
            <span className="font-mono text-[11px] font-bold">{numeral}</span>
            <span className="cap">{category}</span>
          </div>
          <h3 className="mt-[7px] text-[15px] font-black leading-[1.02] tracking-[-0.02em]">
            {title}
          </h3>
          <p className="cap mt-[6px]">{venue}</p>
          <div className="mt-auto flex items-baseline justify-between pt-[9px]">
            <span className="font-mono text-[11px] font-bold">{date}</span>
            <span className="text-[12px] font-black">{price}</span>
          </div>
          {matchReason ? (
            <p className="mt-[8px] border-t border-rule-inner pt-[8px] text-[10px]">
              Совпало: {matchReason}
            </p>
          ) : null}
        </div>
      </div>

      {/* Mobile — handoff U1 row: 44px 1fr auto */}
      <div className="grid grid-cols-[44px_1fr_auto] items-baseline gap-x-[10px] gap-y-[2px] px-[14px] py-[9px] sm:hidden">
        <EventCover
          src={cover}
          sizes={THUMB_SIZES}
          aspect="aspect-square"
          className="row-span-2 self-start border border-ink"
          fallback={
            <div className="flex h-full items-center justify-center">
              <span className="font-mono text-[10px] font-bold text-ink">{numeral}</span>
            </div>
          }
        />
        <h3 className="text-[12.5px] font-bold leading-[1.1] tracking-normal">
          {title}
        </h3>
        <span className="text-[10px] font-black">{price}</span>
        <span className="cap col-span-2">
          {numeral} · {venue} · {date}
        </span>
        {matchReason ? (
          <p className="col-span-3 mt-[6px] border-t border-rule-inner pt-[6px] text-[10px]">
            Совпало: {matchReason}
          </p>
        ) : null}
      </div>
    </Link>
  );
}
