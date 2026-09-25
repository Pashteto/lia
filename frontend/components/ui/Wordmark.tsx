import { BRAND } from "@/lib/brand";
import { cn } from "@/lib/cn";

/** The сообща wordmark. `admin` appends the small tracked ADMIN suffix.
 * `data-wordmark` is the landing target the intro splash measures. */
export function Wordmark({ admin, className }: { admin?: boolean; className?: string }) {
  return (
    <span data-wordmark className={cn("wordmark", className)}>
      {BRAND.wordmark}
      {admin ? <span className="wordmark-suffix">ADMIN</span> : null}
    </span>
  );
}
