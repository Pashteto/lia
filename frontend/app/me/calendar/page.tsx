import { CalendarView } from "@/components/CalendarView";
import { pageTitle } from "@/lib/brand";

export const metadata = { title: pageTitle("Календарь") };

// U4 · Календарь. CalendarView owns AppHeader because the mobile header caption
// («ИЮЛЬ ’26») tracks the client-side month state.
export default function CalendarPage() {
  return <CalendarView />;
}
