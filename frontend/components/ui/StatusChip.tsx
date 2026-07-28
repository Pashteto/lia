import { Chip, type ChipVariant } from "@/components/ui/Chip";
import { statusChipVariant } from "@/lib/status-chip";
import type { ReactNode } from "react";

const TONE_TO_VARIANT: Record<ReturnType<typeof statusChipVariant>, ChipVariant> = {
  active: "active",
  default: "default",
  signal: "signal",
};

/** Status chip per the handoff map (published→ink fill, draft→outline,
 * moderation/waiting/test→signal). Non-interactive. `children` overrides the
 * visible text (U6 mobile shows «ОК» / «ЖДЁМ») while `status` still picks the tone. */
export function StatusChip({
  status,
  className,
  children,
}: {
  status: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <Chip as="span" variant={TONE_TO_VARIANT[statusChipVariant(status)]} className={className}>
      {children ?? status}
    </Chip>
  );
}
