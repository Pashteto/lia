export const ORG_VERIFY_STEPS = [
  "Заявка подана",
  "Документы проверены",
  "Верифицирован ✓",
] as const;

/** 0-based current step for Stepper fillMode="exclusive". */
export function orgVerificationStep(
  status: "draft" | "pending" | "verified" | "rejected",
): number {
  switch (status) {
    case "pending":
      return 1;
    case "verified":
      return 2;
    default:
      return 0;
  }
}
