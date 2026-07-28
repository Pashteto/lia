import { permanentRedirect } from "next/navigation";

// Consolidated into U6 (/me), «Заявки» tab.
export default function MyApplicationsPage() {
  permanentRedirect("/me?tab=applications");
}
