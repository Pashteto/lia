import { ComingSoon } from "@/components/ComingSoon";
import { MOCK_EVENTS } from "@/lib/mock-events";

// TODO: build from design/screens/event-detail.html
// (cover → large title → facts grid → sections → sticky glass "Записаться" bar).
export default async function EventDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const event = MOCK_EVENTS.find((e) => e.id === id);
  return (
    <ComingSoon
      kicker="Детали события"
      title={event?.title ?? "Событие"}
      note="Экран деталей и регистрации ещё не реализован в этом скаффолде. Смотри design/screens/event-detail.html."
    />
  );
}
