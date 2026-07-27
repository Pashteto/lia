import { Chip, type ChipVariant } from "@/components/ui/Chip";
import { statusChipVariant } from "@/lib/status-chip";

const TONE_TO_VARIANT: Record<ReturnType<typeof statusChipVariant>, ChipVariant> = {
  active: "active",
  default: "default",
  signal: "signal",
};

/** Status chip per the handoff map (published→ink fill, draft→outline,
 * moderation/waiting/test→signal). Non-interactive. */
export function StatusChip({ status, className }: { status: string; className?: string }) {
  return (
    <Chip as="span" variant={TONE_TO_VARIANT[statusChipVariant(status)]} className={className}>
      {status}
    </Chip>
  );
}
