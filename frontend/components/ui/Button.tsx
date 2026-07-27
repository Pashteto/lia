import { cn } from "@/lib/cn";
import type { ButtonHTMLAttributes } from "react";

type Variant = "primary" | "ghost" | "inverted" | "destructive" | "dark-ghost";
type Size = "md" | "sm";

const VARIANTS: Record<Variant, string> = {
  // Ink fill, white text; hover deepens to black.
  primary: "bg-ink text-white hover:bg-black",
  // Transparent, 1px ink border, ink text; hover inverts.
  ghost: "border border-ink text-ink hover-invert",
  // Admin primary on ink surface: paper fill, ink text.
  inverted: "bg-paper text-ink hover:opacity-90",
  // Red fill — destructive only (ОТКЛОНИТЬ etc.).
  destructive: "bg-signal text-white hover:opacity-90",
  // Admin tertiary on ink surface.
  "dark-ghost": "border border-muted-2 text-muted-2 hover:opacity-90",
};

const SIZES: Record<Size, string> = {
  md: "px-[11px] py-[11px] text-[11px]",
  sm: "px-[4px] py-[7px] text-[9px]",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

/** Swiss Grid CTA: uppercase 700 / 0.07em, zero radius, no motion. */
export function Button({ variant = "primary", size = "md", className, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap text-center font-bold uppercase tracking-[0.07em] transition-colors duration-[120ms] ease-linear select-none swiss-focus disabled:bg-inactive disabled:text-muted-2 disabled:border-0",
        VARIANTS[variant],
        SIZES[size],
        className,
      )}
      {...props}
    />
  );
}
