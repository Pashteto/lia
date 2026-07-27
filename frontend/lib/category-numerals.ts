/** Positional category numeral per the Swiss Grid rule: categories are
 * numerals, never colours. Order comes from GET /api/v1/categories. */
export function categoryNumeral(
  slug: string,
  ordered: ReadonlyArray<{ slug: string }>,
): string {
  const i = ordered.findIndex((c) => c.slug === slug);
  if (i === -1) return "—";
  return String(i + 1).padStart(2, "0");
}
