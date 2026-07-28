const RUB = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 0 });

export type PriceKind = "free" | "from" | "fixed";

/** Swiss Grid price string: free is the literal FREE; otherwise grouped
 * rubles + narrow space + ₽; "from" prices get the «от» prefix. Whole RUB in. */
export function priceLabel(priceRub: number | null | undefined, kind: PriceKind = "fixed"): string {
  if (kind === "free" || !priceRub || !Number.isFinite(priceRub) || priceRub <= 0) return "FREE";
  const amount = `${RUB.format(Math.round(priceRub))} ₽`;
  return kind === "from" ? `от ${amount}` : amount;
}
