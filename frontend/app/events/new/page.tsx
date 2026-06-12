import { CreateEventForm } from "@/components/CreateEventForm";

// Create event — built from design/screens/create-event.html.
// Grouped inset form (Основное / Формат / Место и время / Участники / Публикация);
// submits to POST /api/v1/events and redirects to the new event's detail page.
export default function CreateEventPage() {
  return <CreateEventForm />;
}
