import { notFound } from "next/navigation";
import { AppHeader, USER_NAV } from "@/components/ui/AppHeader";
import { Chip } from "@/components/ui/Chip";
import { Button } from "@/components/ui/Button";
import { Cell, CellStrip } from "@/components/ui/Cell";
import { EventModule } from "@/components/ui/EventModule";
import { StatusChip } from "@/components/ui/StatusChip";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Skeleton } from "@/components/ui/Skeleton";
import { Stepper } from "@/components/ui/Stepper";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input, Textarea } from "@/components/ui/Field";
import { priceLabel } from "@/lib/price-label";
import { categoryNumeral } from "@/lib/category-numerals";

const CATS = [
  { slug: "festival" }, { slug: "mediation" }, { slug: "lecture" },
  { slug: "cinema" }, { slug: "performance" }, { slug: "concert" },
];

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="border-b border-on-surface p-[20px]">
      <p className="kick mb-[14px]">{title}</p>
      <div className="flex flex-wrap items-start gap-[10px]">{children}</div>
    </section>
  );
}

function Gallery() {
  return (
    <>
      <Section title="Chips">
        <Chip>Все · 6</Chip>
        <Chip variant="active">Сегодня</Chip>
        <Chip variant="signal">Модерация · 2</Chip>
        <StatusChip status="Опубликовано" />
        <StatusChip status="Черновик" />
        <StatusChip status="На модерации" />
      </Section>
      <Section title="Buttons">
        <Button>Записаться</Button>
        <Button variant="ghost">Черновик</Button>
        <Button variant="destructive">Отклонить</Button>
        <Button variant="inverted">Одобрить</Button>
        <Button variant="dark-ghost">На доработку</Button>
        <Button size="sm">Принять</Button>
        <Button disabled>Недоступно</Button>
      </Section>
      <Section title="Cells">
        <CellStrip cols={4} className="w-full">
          <Cell caption="Когда" value="12 июля, сб" />
          <Cell caption="Начало" value="16:00" mono />
          <Cell caption="Места" value="12 / 40" mono />
          <Cell caption="На модерации" value="02" mono invert />
        </CellStrip>
      </Section>
      <Section title="EventModule">
        <div className="grid w-full grid-cols-3 border-y border-on-surface [&>*+*]:border-l [&>*+*]:border-rule-inner max-sm:grid-cols-1">
          <EventModule
            numeral={categoryNumeral("festival", CATS)}
            category="Фестивали" title="Летний фестиваль медиаискусства"
            venue="Музей «Гараж»" date="12.07 · 16:00" price={priceLabel(0)} href="/"
          />
          <EventModule
            numeral={categoryNumeral("lecture", CATS)}
            category="Лекции" title="Разговор о новой вещественности"
            venue="ГМИИ им. Пушкина" date="15.07 · 19:00" price={priceLabel(80000)} href="/"
            matchReason="тихое, вечером, вдвоём"
          />
          <EventModule
            numeral={categoryNumeral("mediation", CATS)}
            category="Медиации" title="Медиация по выставке «Свет»"
            venue="Винзавод" date="18.07 · 12:00" price={priceLabel(150000)} href="/"
          />
        </div>
      </Section>
      <Section title="Progress / Skeleton">
        <div className="w-[200px]"><ProgressBar value={28} max={40} /></div>
        <div className="w-[200px]"><ProgressBar value={28} max={40} thin /></div>
        <Skeleton className="h-[40px] w-[200px]" />
      </Section>
      <Section title="Stepper">
        <Stepper className="w-full" steps={["Основное", "Когда и где", "Билеты", "Публикация"]} current={1} />
      </Section>
      <Section title="Fields">
        <div className="grid w-full max-w-[400px] gap-[11px]">
          <Input label="Название" placeholder="Название события" />
          <Input label="Почта" error="Неверный формат почты" defaultValue="user@" />
          <Textarea label="Описание" placeholder="Коротко о событии" />
        </div>
      </Section>
      <Section title="Empty state (U8)">
        <EmptyState
          title="Записей пока нет"
          text="Когда вы запишетесь на событие, оно появится здесь."
          actions={<Button>Найти событие</Button>}
        />
      </Section>
    </>
  );
}

/** Dev-only primitive gallery: paper surface + ink surface side by side. */
export default function DesignPreviewPage() {
  if (process.env.NODE_ENV === "production") notFound();
  return (
    <main className="mx-auto max-w-[1360px]">
      <AppHeader nav={USER_NAV} />
      <div className="grid grid-cols-2 max-lg:grid-cols-1">
        <div className="bg-surface text-on-surface">
          <Gallery />
        </div>
        <div data-surface="ink" className="bg-surface text-on-surface">
          <Gallery />
        </div>
      </div>
    </main>
  );
}
