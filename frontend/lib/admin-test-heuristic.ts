const TEST_PATTERN = /QA|тест|test|\.test\b|bla\s*bla/i;

export function isLikelyTestContent(
  ...parts: Array<string | null | undefined>
): boolean {
  return parts.some((part) => part != null && TEST_PATTERN.test(part));
}
