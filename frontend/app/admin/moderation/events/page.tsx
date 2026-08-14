import { AdminModeration } from "@/components/AdminModeration";

// Deliberately NOT wrapped in the ≥900px desktop-only gate: approve/reject is the one
// admin gesture that happens on the go, so the queue works below 900px too
// (QA 14.08, finding 6). The data tables (users/organizers) keep the gate.
export default function Page() {
  return <AdminModeration />;
}
