import { padCount } from "./org-seats";

/**
 * A count tile's value: the padded mono form, or «—» when the backend omitted
 * the field. `undefined` and `0` are different things — a missing total is
 * unknown, not zero — so the dash is reserved for the former.
 */
export function tileCount(n: number | undefined): string {
  if (n == null) return "—";
  return n < 100 ? padCount(n) : String(n);
}
