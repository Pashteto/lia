export type DiscoverIntentId =
  | "quiet_weekend"
  | "quiet_any"
  | "free_nearby"
  | "evening_pair"
  | "with_kids"
  | "keyword";

export interface DiscoverIntent {
  id: DiscoverIntentId;
  sourceLabel: string;
  keyword: string;
  wantsNearby: boolean;
}

export const DISCOVER_CHIPS: readonly {
  id: DiscoverIntentId;
  label: string;
  /** Compact label for ≤430px (HTML U3 mobile frame). */
  shortLabel: string;
}[] = [
  { id: "quiet_weekend", label: "Тихое в выходные", shortLabel: "Тихое" },
  { id: "free_nearby", label: "Бесплатно рядом", shortLabel: "Бесплатно" },
  { id: "evening_pair", label: "Для двоих вечером", shortLabel: "Вечер" },
  { id: "with_kids", label: "С детьми", shortLabel: "С детьми" },
];

export function intentFromChip(chipId: DiscoverIntentId): DiscoverIntent {
  const chip = DISCOVER_CHIPS.find((c) => c.id === chipId);
  const label = chip?.label ?? chipId;
  return {
    id: chipId,
    sourceLabel: label,
    keyword: "",
    wantsNearby: chipId === "free_nearby",
  };
}

export function parseDiscoverQuery(raw: string): DiscoverIntent | null {
  const sourceLabel = raw.trim().replace(/\s+/g, " ");
  if (!sourceLabel) return null;
  const q = sourceLabel.toLowerCase();

  if (/бесплат/.test(q)) {
    return { id: "free_nearby", sourceLabel, keyword: "", wantsNearby: true };
  }
  if (/выходн/.test(q)) {
    return { id: "quiet_weekend", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/вечер/.test(q)) {
    return { id: "evening_pair", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/дет|семь/.test(q)) {
    return { id: "with_kids", sourceLabel, keyword: "", wantsNearby: false };
  }
  if (/тих|спокой/.test(q)) {
    return { id: "quiet_any", sourceLabel, keyword: "", wantsNearby: false };
  }
  return { id: "keyword", sourceLabel, keyword: sourceLabel, wantsNearby: false };
}
