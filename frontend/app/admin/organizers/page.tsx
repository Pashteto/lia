import { AdminDesktopOnly } from "@/components/AdminDesktopOnly";
import { AdminOrganizers } from "@/components/AdminOrganizers";

export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ filter?: string }>;
}) {
  const sp = await searchParams;
  return (
    <AdminDesktopOnly>
      <AdminOrganizers initialFilter={sp.filter} />
    </AdminDesktopOnly>
  );
}
