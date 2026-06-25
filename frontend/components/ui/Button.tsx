import { cn } from "@/lib/cn";
import type { ButtonHTMLAttributes } from "react";

type Variant = "filled" | "tinted" | "plain";

const VARIANTS: Record<Variant, string> = {
  // Filled: accent fill, white label — primary actions.
  filled: "bg-accent text-white hover:opacity-90 active:opacity-80",
  // Tinted: accent-tinted fill, accent label — secondary actions.
  tinted: "bg-accent/12 text-accent hover:bg-accent/20",
  // Plain: accent text only — tertiary / nav actions.
  plain: "text-accent hover:opacity-70 px-0",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

/** Apple-style button (filled / tinted / plain), ~12px continuous radius. */
export function Button({
  variant = "filled",
  className,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center gap-1.5 rounded-control px-4 py-2.5 text-[15px] font-semibold transition select-none disabled:opacity-40 active:scale-[0.97] motion-reduce:transform-none motion-reduce:transition-none",
        VARIANTS[variant],
        className,
      )}
      {...props}
    />
  );
}
