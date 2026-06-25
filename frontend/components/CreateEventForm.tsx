"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import { Segmented } from "@/components/ui/Segmented";
import { Switch } from "@/components/ui/Switch";
import { VenuePicker } from "@/components/VenuePicker";
import { createEvent, getCategories, type CreateEventInput, uploadFile } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
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
  const { isAuthed, ready } = useAuth();

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
      // Default to published so a created event is immediately visible in the
      // discovery feed (which lists status=published). Users can still pick
      // "Черновик" in the status control to keep it hidden.
      status: "published",
      categoryIds: [],
      venueId: "",
    },
  });

  const isFree = useWatch({ control, name: "isFree" });

  const [coverFileId, setCoverFileId] = useState<string | undefined>(undefined);
  const [coverPreviewUrl, setCoverPreviewUrl] = useState<string | undefined>(undefined);
  const [coverUploading, setCoverUploading] = useState(false);
  const [coverError, setCoverError] = useState<string | undefined>(undefined);

  const { data: categories = [] } = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
  });

  const mutation = useMutation({
    mutationFn: (input: CreateEventInput) => createEvent(input),
    onSuccess: (event) => router.push(`/events/${event.id}`),
  });

  const handleCoverChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setCoverError(undefined);
    setCoverUploading(true);
    try {
      const { id, url } = await uploadFile(file);
      setCoverFileId(id);
      setCoverPreviewUrl(url);
    } catch (err) {
      setCoverError(err instanceof Error ? err.message : "Ошибка загрузки");
    } finally {
      setCoverUploading(false);
    }
  };

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
      cover_file_id: coverFileId,
    };
    mutation.mutate(input);
  };

  // Gate: creating an event requires a signed-in user (backend returns 401
  // otherwise). Avoid flashing the form before the session is read.
  if (!ready) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }
  if (!isAuthed) {
    return <CreateEventGate />;
  }

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
        <div className="mb-6">
          <label className="block">
            <span className="mb-1.5 block text-[13px] text-label-secondary">Обложка</span>
            <div className="rounded-card bg-bg-secondary p-4">
              {coverPreviewUrl && (
                <div className="relative mb-3 aspect-[3/2] w-full overflow-hidden rounded-[10px] bg-fill">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={coverPreviewUrl}
                    alt="Предпросмотр обложки"
                    className="h-full w-full object-cover"
                  />
                </div>
              )}
              <input
                type="file"
                accept="image/*"
                disabled={coverUploading}
                onChange={handleCoverChange}
                className="block w-full text-[15px] text-label file:mr-3 file:rounded-full file:border-0 file:bg-fill file:px-4 file:py-1.5 file:text-[14px] file:font-medium file:text-label file:transition hover:file:bg-fill-secondary disabled:opacity-60"
              />
              {coverUploading && (
                <p className="mt-2 text-[13px] text-label-secondary">Загрузка…</p>
              )}
              {coverError && (
                <p className="mt-2 text-[13px] text-red-500">{coverError}</p>
              )}
            </div>
          </label>
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
            {mutation.error instanceof Error && mutation.error.message.includes("429")
              ? "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
              : "Не удалось сохранить событие. Проверьте, что бэкенд запущен."}
          </p>
        )}
      </div>
    </form>
  );
}

// Shown when an unauthenticated user reaches /events/new. Prompts for demo-login
// rather than rendering a form that would 401 on submit.
function CreateEventGate() {
  const [showLogin, setShowLogin] = useState(false);
  return (
    <div className="min-h-screen bg-bg-grouped">
      <header className="glass sticky top-0 z-10 border-b border-separator">
        <div className="mx-auto flex max-w-2xl items-center justify-between px-5 py-3">
          <Link href="/" className="text-[17px] text-accent">
            Отмена
          </Link>
          <span className="text-[17px] font-semibold">Новое событие</span>
          <span className="w-16" />
        </div>
      </header>
      <div className="mx-auto max-w-md px-5 pt-16 text-center">
        <div className="mb-4 text-[40px]" aria-hidden>
          🔐
        </div>
        <h1 className="mb-2 text-[22px] font-bold tracking-[-0.022em]">
          Войдите, чтобы создать событие
        </h1>
        <p className="mb-6 text-[15px] text-label-secondary">
          Создание событий доступно авторизованным пользователям. Демо-вход
          занимает пару секунд — нужен только email.
        </p>
        <Button onClick={() => setShowLogin(true)}>Войти</Button>
      </div>
      {showLogin && <LoginModal onClose={() => setShowLogin(false)} />}
    </div>
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
