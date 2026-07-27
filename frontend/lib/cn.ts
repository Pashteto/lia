import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Classname joiner with Tailwind conflict resolution (last one wins). */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
