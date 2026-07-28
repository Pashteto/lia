import { decideApplication } from "./api";
import type { RsvpStatus } from "./types";

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

/**
 * Meta caption under applicant name — never invent attendance history.
 * «Первая заявка» only for pending (applied) rows with an empty answer;
 * decided statuses fall back to an em dash.
 */
export function applicationMeta(
  answer?: string | null,
  status: RsvpStatus = "applied",
): string {
  const trimmed = answer?.trim();
  if (trimmed) return trimmed;
  if (status === "applied") return "Первая заявка";
  return "—";
}
