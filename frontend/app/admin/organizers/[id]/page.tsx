import { AdminDesktopOnly } from "@/components/AdminDesktopOnly";
import { AdminOrganizerDetail } from "@/components/AdminOrganizerDetail";

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <AdminDesktopOnly>
      <AdminOrganizerDetail id={id} />
    </AdminDesktopOnly>
  );
}
