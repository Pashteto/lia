/** Index to select after acting on `current`. If last, clamp to newLast; if empty → -1. */
export function nextQueueIndex(
  current: number,
  lengthAfterRemoval: number,
): number {
  if (lengthAfterRemoval === 0) return -1;
  if (current >= lengthAfterRemoval) return lengthAfterRemoval - 1;
  return current;
}
