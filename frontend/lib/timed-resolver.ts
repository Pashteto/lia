export interface TimedResolver {
  resolve(action: () => void): void;
  cancel(): void;
}

export function createTimedResolver(
  onTimeout: () => void,
  delayMs: number,
): TimedResolver {
  let resolved = false;
  const timer = setTimeout(() => {
    if (resolved) return;
    resolved = true;
    onTimeout();
  }, delayMs);

  return {
    resolve(action) {
      if (resolved) return;
      resolved = true;
      clearTimeout(timer);
      action();
    },
    cancel() {
      resolved = true;
      clearTimeout(timer);
    },
  };
}
