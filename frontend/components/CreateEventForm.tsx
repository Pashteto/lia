"use client";

import { Button } from "@/components/ui/Button";
import { Segmented } from "@/components/ui/Segmented";
import { Switch } from "@/components/ui/Switch";
import { VenuePicker } from "@/components/VenuePicker";
import { createEvent, getCategories, type CreateEventInput } from "@/lib/api";
import { cn } from "@/lib/cn";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";

const schema = z.object({
  title: z.string().min(1, "Укажите название"),
  description: z.string().optional(),
  categoryIds: z.array(z.string()).optional(),
  format: z.enum(["offline", "online"]),
  venueId: z.string().optional(),
  startsAt: z.string().min(1, "Укажите дату и время"),
  endsAt: z.string().optional(),
  isFree: z.boolean(),
  priceMin: z.coerce.number().int().min(0).optional(),
  status: z.enum(["draft", "pending_review", "published"]),
});

type FormValues = z.input<typeof schema>;

const inputCls =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export function CreateEventForm() {
  const router = useRouter();

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      format: "offline",
      isFree: true,
      status: "draft",
      categoryIds: [],
      venueId: "",
    },
  });

  const isFree = useWatch({ control, name: "isFree" });

  const { data: categories = [] } = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
  });

  const mutation = useMutation({
    mutationFn: (input: CreateEventInput) => createEvent(input),
    onSuccess: (event) => router.push(`/events/${event.id}`),
  });

  const onSubmit = (v: FormValues) => {
    const input: CreateEventInput = {
      title: v.title,
      description: v.description || undefined,
      category_ids: v.categoryIds && v.categoryIds.length > 0 ? v.categoryIds : undefined,
      venue_id: v.venueId || undefined,
      format: v.format,
      status: v.status,
      price_type: v.isFree ? "free" : "from",
      price_min: v.isFree ? undefined : Number(v.priceMin) || 0,
      starts_at: new Date(v.startsAt).toISOString(),
      ends_at: v.endsAt ? new Date(v.endsAt).toISOString() : undefined,
    };
    mutation.mutate(input);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="min-h-screen bg-bg-grouped pb-16">
      {/* Glass nav with Cancel / Save */}
      <header className="glass sticky top-0 z-10 border-b border-separator">
        <div className="mx-auto flex max-w-2xl items-center justify-between px-5 py-3">
          <Link href="/" className="text-[17px] text-accent">
            Отмена
          </Link>
          <span className="text-[17px] font-semibold">Новое событие</span>
          <Button type="submit" variant="plain" disabled={mutation.isPending}>
            {mutation.isPending ? "Сохранение…" : "Сохранить"}
          </Button>
        </div>
      </header>

      <div className="mx-auto max-w-2xl px-5 pt-5">
        {/* Cover upload is not built yet — it needs image storage (S3/MinIO) and
            a cover field on the backend. Shown as a compact, intentional notice
            rather than a large empty drop zone that reads as broken. */}
        <div className="mb-6 flex items-center gap-3 rounded-card bg-bg-secondary px-4 py-3 text-[14px] text-label-secondary">
          <span aria-hidden className="text-[18px]">🖼️</span>
          <span>Загрузка обложки появится позже — пока событие сохраняется без изображения.</span>
        </div>

        <Section title="Основное">
          <Field label="Название" error={errors.title?.message}>
            <input
              className={inputCls}
              placeholder="Например, «Читаем Зебальда»"
              {...register("title")}
            />
          </Field>
          <Field label="Описание">
            <textarea
              className={cn(inputCls, "min-h-[96px] resize-y")}
              placeholder="О чём встреча, чего ждать участникам"
              {...register("description")}
            />
          </Field>
          <Field label="Категории">
            <Controller
              control={control}
              name="categoryIds"
              render={({ field }) => {
                const selected = field.value ?? [];
                const toggle = (id: string) =>
                  field.onChange(
                    selected.includes(id)
                      ? selected.filter((s) => s !== id)
                      : [...selected, id],
                  );
                return (
                  <div className="flex flex-wrap gap-2">
                    {categories.length === 0 && (
                      <span className="text-[13px] text-label-secondary">
                        Категории недоступны (бэкенд офлайн)
                      </span>
                    )}
                    {categories.map((c) => {
                      const on = selected.includes(c.id);
                      return (
                        <button
                          key={c.id}
                          type="button"
                          onClick={() => toggle(c.id)}
                          className={cn(
                            "rounded-full px-3 py-1.5 text-[15px] transition",
                            on ? "bg-accent text-white" : "bg-fill text-label",
                          )}
                        >
                          {c.label}
                        </button>
                      );
                    })}
                  </div>
                );
              }}
            />
          </Field>
        </Section>

        <Section title="Формат">
          <Field label="Формат">
            <Controller
              control={control}
              name="format"
              render={({ field }) => (
                <Segmented
                  options={[
                    { value: "offline", label: "Очно" },
                    { value: "online", label: "Онлайн" },
                  ]}
                  value={field.value}
                  onChange={field.onChange}
                />
              )}
            />
          </Field>
        </Section>

        <Section title="Место и время">
          <Field label="Место">
            <Controller
              control={control}
              name="venueId"
              render={({ field }) => (
                <VenuePicker value={field.value ?? ""} onChange={field.onChange} />
              )}
            />
          </Field>
          <Field label="Начало" error={errors.startsAt?.message}>
            <input type="datetime-local" className={inputCls} {...register("startsAt")} />
          </Field>
          <Field label="Окончание">
            <input type="datetime-local" className={inputCls} {...register("endsAt")} />
          </Field>
        </Section>

        <Section title="Участники и цена">
          <Controller
            control={control}
            name="isFree"
            render={({ field }) => (
              <div className="flex items-center justify-between">
                <span className="text-[17px]">Бесплатно</span>
                <Switch checked={field.value} onChange={field.onChange} />
              </div>
            )}
          />
          {!isFree && (
            <Field label="Цена от, ₽">
              <input
                type="number"
                min={0}
                className={inputCls}
                placeholder="2500"
                {...register("priceMin")}
              />
            </Field>
          )}
        </Section>

        <Section title="Публикация">
          <Field label="Статус">
            <Controller
              control={control}
              name="status"
              render={({ field }) => (
                <Segmented
                  options={[
                    { value: "draft", label: "Черновик" },
                    { value: "pending_review", label: "На модерацию" },
                    { value: "published", label: "Опубликовать" },
                  ]}
                  value={field.value}
                  onChange={field.onChange}
                />
              )}
            />
          </Field>
        </Section>

        {mutation.isError && (
          <p className="mt-4 text-[15px] text-red-500">
            Не удалось сохранить событие. Проверьте, что бэкенд запущен.
          </p>
        )}
      </div>
    </form>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mb-6">
      <h2 className="mb-2 px-1 text-[13px] font-semibold uppercase tracking-[0.03em] text-label-secondary">
        {title}
      </h2>
      <div className="space-y-4 rounded-card bg-bg-secondary p-4 shadow-card-subtle">
        {children}
      </div>
    </section>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[13px] text-label-secondary">{label}</span>
      {children}
      {error && <span className="mt-1 block text-[13px] text-red-500">{error}</span>}
    </label>
  );
}
