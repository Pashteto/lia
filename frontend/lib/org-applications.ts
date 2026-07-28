import { decideApplication } from "./api";

export async function decideMany(
  eventId: string,
  ids: string[],
  decision: "accept" | "decline",
  decide: typeof decideApplication = decideApplication,
): Promise<{ ok: string[]; failed: string[] }> {
  const ok: string[] = [];
  const failed: string[] = [];
  for (const id of ids) {
    try {
      await decide(eventId, id, decision);
      ok.push(id);
    } catch {
      failed.push(id);
    }
  }
  return { ok, failed };
}

/** Meta caption under applicant name — never invent attendance history. */
export function applicationMeta(answer?: string | null): string {
  const trimmed = answer?.trim();
  return trimmed || "Первая заявка";
}
