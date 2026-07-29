import { AdminDesktopOnly } from "@/components/AdminDesktopOnly";
import { AdminUsers } from "@/components/AdminUsers";

export default function Page() {
  return (
    <AdminDesktopOnly>
      <AdminUsers />
    </AdminDesktopOnly>
  );
}
