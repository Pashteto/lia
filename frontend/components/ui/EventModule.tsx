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

/** Swiss Grid feed card. Desktop: numeral/category row, 15px/900 title,
 * venue caption, footer date(mono)+price pinned bottom. Hover inverts.
 * Single <Link> wrapper — never nest an <a> inside (React #418 fix). */
export function EventModule({
  numeral, category, title, venue, date, price, href, matchReason, className,
}: EventModuleProps) {
  return (
    <Link
      href={href}
      className={cn(
        "flex min-w-0 flex-col px-[14px] py-[12px] swiss-focus hover-invert",
        className,
      )}
    >
      <div className="flex items-baseline justify-between">
        <span className="font-mono text-[11px] font-bold">{numeral}</span>
        <span className="cap">{category}</span>
      </div>
      <h3 className="mt-[8px] text-[15px] font-black leading-[1.02] tracking-[-0.02em] max-sm:text-[12.5px] max-sm:font-bold">
        {title}
      </h3>
      <p className="cap mt-[6px]">{venue}</p>
      <div className="mt-auto flex items-baseline justify-between pt-[10px]">
        <span className="font-mono text-[11px]">{date}</span>
        <span className="text-[12px] font-black">{price}</span>
      </div>
      {matchReason ? (
        <p className="mt-[8px] border-t border-rule-inner pt-[8px] text-[10px]">
          Совпало: {matchReason}
        </p>
      ) : null}
    </Link>
  );
}
