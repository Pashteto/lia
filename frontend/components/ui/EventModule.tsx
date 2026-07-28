import Link from "next/link";
import { cn } from "@/lib/cn";

export interface EventModuleProps {
  numeral: string; // "01" — from categoryNumeral()
  category: string; // "Фестивали"
  title: string;
  venue: string;
  date: string; // preformatted, renders in mono
  price: string; // from priceLabel() — "FREE" | "800 ₽"
  href: string;
  /** U3 AI variant: match reason on a hairline top rule. */
  matchReason?: string;
  className?: string;
}

/** Swiss Grid feed card.
 * Desktop: numeral/category row, 15px/900 title, venue caption, footer date(mono 700)+price.
 * Mobile: grid 22px 1fr auto — numeral / title / price, then venue · date.
 * Single <Link> wrapper — never nest an <a> inside (React #418 fix). */
export function EventModule({
  numeral, category, title, venue, date, price, href, matchReason, className,
}: EventModuleProps) {
  return (
    <Link
      href={href}
      className={cn("block h-full min-w-0 swiss-focus hover-invert", className)}
    >
      {/* Desktop — handoff U1 catalogue cell */}
      <div className="hidden h-full flex-col px-[14px] py-[12px] sm:flex">
        <div className="flex items-baseline justify-between">
          <span className="font-mono text-[11px] font-bold">{numeral}</span>
          <span className="cap">{category}</span>
        </div>
        <h3 className="mt-[8px] text-[15px] font-black leading-[1.02] tracking-[-0.02em]">
          {title}
        </h3>
        <p className="cap mt-[6px]">{venue}</p>
        <div className="mt-auto flex items-baseline justify-between pt-[10px]">
          <span className="font-mono text-[11px] font-bold">{date}</span>
          <span className="text-[12px] font-black">{price}</span>
        </div>
        {matchReason ? (
          <p className="mt-[8px] border-t border-rule-inner pt-[8px] text-[10px]">
            Совпало: {matchReason}
          </p>
        ) : null}
      </div>

      {/* Mobile — handoff U1 row: 22px 1fr auto */}
      <div className="grid grid-cols-[22px_1fr_auto] items-baseline gap-x-[10px] gap-y-[2px] px-[14px] py-[10px] sm:hidden">
        <span className="font-mono text-[10px] font-bold">{numeral}</span>
        <h3 className="text-[12.5px] font-bold leading-[1.1] tracking-normal">
          {title}
        </h3>
        <span className="text-[10px] font-black">{price}</span>
        <span aria-hidden />
        <span className="cap">
          {venue} · {date}
        </span>
        <span aria-hidden />
        {matchReason ? (
          <p className="col-span-3 mt-[6px] border-t border-rule-inner pt-[6px] text-[10px]">
            Совпало: {matchReason}
          </p>
        ) : null}
      </div>
    </Link>
  );
}
