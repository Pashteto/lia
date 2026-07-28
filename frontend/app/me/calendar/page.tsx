import { CalendarView } from "@/components/CalendarView";

export const metadata = { title: "Календарь — PRESENCE" };

// U4 · Календарь. CalendarView owns AppHeader because the mobile header caption
// («ИЮЛЬ ’26») tracks the client-side month state.
export default function CalendarPage() {
  return <CalendarView />;
}
