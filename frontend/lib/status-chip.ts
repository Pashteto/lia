export type ChipTone = "active" | "default" | "signal";

const ACTIVE = new Set(["Опубликовано", "Подтверждено", "Вы идёте", "Верифицирован"]);
const SIGNAL = new Set(["На модерации", "Ожидает", "Ждёт ответа", "На проверке", "Тестовый"]);

/** Handoff status→chip map. Red strictly means "needs attention". */
export function statusChipVariant(status: string): ChipTone {
  if (ACTIVE.has(status)) return "active";
  if (SIGNAL.has(status)) return "signal";
  return "default";
}
