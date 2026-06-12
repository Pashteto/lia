import { Button } from "@/components/ui/Button";
import { Kicker } from "@/components/ui/Kicker";
import Link from "next/link";

/** Placeholder for screens not yet built in the scaffold. */
export function ComingSoon({
  kicker,
  title,
  note,
}: {
  kicker: string;
  title: string;
  note: string;
}) {
  return (
    <main className="mx-auto flex min-h-[70vh] max-w-3xl flex-col items-start justify-center px-5">
      <Kicker>{kicker}</Kicker>
      <h1 className="mt-2 text-[34px] font-bold tracking-[-0.022em]">{title}</h1>
      <p className="mt-3 max-w-md text-[17px] text-label-secondary">{note}</p>
      <Link href="/" className="mt-6">
        <Button variant="tinted">К событиям</Button>
      </Link>
    </main>
  );
}
