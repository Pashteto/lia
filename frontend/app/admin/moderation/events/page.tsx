import { AdminDesktopOnly } from "@/components/AdminDesktopOnly";
import { AdminModeration } from "@/components/AdminModeration";

export default function Page() {
  return (
    <AdminDesktopOnly>
      <AdminModeration />
    </AdminDesktopOnly>
  );
}
