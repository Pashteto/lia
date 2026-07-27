import { cn } from "@/lib/cn";
import { useId } from "react";
import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";

const BOX =
  "w-full border bg-transparent px-[11px] py-[9px] text-[12.5px] text-ink placeholder:text-field-text swiss-focus disabled:bg-inactive disabled:text-muted-2";

function boxClass(error?: string) {
  return cn(BOX, error ? "border-signal" : "border-ink");
}

function Wrap({ id, label, error, children }: { id: string; label: string; error?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-[5px]">
      <label htmlFor={id} className={cn("cap", error && "text-signal")}>
        {label}
      </label>
      {children}
      {error ? <p className="text-[11px] text-signal">{error}</p> : null}
    </div>
  );
}

export function Input({ label, error, className, id, ...props }: { label: string; error?: string } & InputHTMLAttributes<HTMLInputElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <input id={inputId} className={cn(boxClass(error), className)} {...props} />
    </Wrap>
  );
}

export function Textarea({ label, error, className, id, ...props }: { label: string; error?: string } & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <textarea id={inputId} className={cn(boxClass(error), "min-h-[52px] resize-y", className)} {...props} />
    </Wrap>
  );
}

export function Select({ label, error, className, id, children, ...props }: { label: string; error?: string } & SelectHTMLAttributes<HTMLSelectElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Wrap id={inputId} label={label} error={error}>
      <select id={inputId} className={cn(boxClass(error), "appearance-none", className)} {...props}>
        {children}
      </select>
    </Wrap>
  );
}
