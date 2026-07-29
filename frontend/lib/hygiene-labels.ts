import { priceLabel } from "./price-label";

export type HygieneIssueLike = {
  kind: string;
  title: string;
  organizer_name?: string;
  price_rub?: number;
};

const KIND_LABEL: Record<string, string> = {
  test_data: "Тестовые данные",
  suspicious_price: "Подозрительная цена",
};

/** Issue-type caption of an A4 hygiene block. Unknown kinds get a neutral
 * label rather than leaking a backend slug into the UI. */
export function hygieneKindLabel(kind: string): string {
  return KIND_LABEL[kind] ?? "Требует проверки";
}

/** The offending value, 11px/700 in the block: the price for a price issue,
 * the quoted event title otherwise. */
export function hygieneIssueValue(issue: HygieneIssueLike): string {
  if (issue.kind === "suspicious_price" && issue.price_rub) {
    return priceLabel(issue.price_rub, "from");
  }
  const title = issue.title?.trim();
  return title ? `«${title}»` : "—";
}

/** Source caption under the value. Null when the organizer is unknown — the
 * caller drops the line rather than printing a placeholder. */
export function hygieneIssueSource(issue: HygieneIssueLike): string | null {
  const name = issue.organizer_name?.trim();
  return name ? `организатор ${name}` : null;
}
