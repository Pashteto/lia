import { permanentRedirect } from "next/navigation";

// Consolidated into U6 (/me). Kept as a redirect so old links and bookmarks work.
export default function MyPracticesPage() {
  permanentRedirect("/me");
}
