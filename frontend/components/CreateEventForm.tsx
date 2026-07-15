"use client";

import { useState } from "react";

import { LoginModal } from "@/components/AuthButton";
import { Button } from "@/components/ui/Button";
import { Segmented } from "@/components/ui/Segmented";
import { Switch } from "@/components/ui/Switch";
import { VenuePicker } from "@/components/VenuePicker";
import { createEvent, getCategories, patchEvent, type CreateEventInput, uploadFile } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { cn } from "@/lib/cn";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";

export const eventFormSchema = z
  .object({
    title: z.string().min(1, "Укажите название"),
    description: z.string().optional(),
    categoryIds: z.array(z.string()).optional(),
    format: z.enum(["offline", "online"]),
    venueId: z.string().optional(),
    startsAt: z.string().min(1, "Укажите дату и время"),
    endsAt: z.string().optional(),
    isFree: z.boolean(),
    priceMin: z.coerce.number().int().min(0).optional(),
    status: z.enum(["draft", "published"]),
    signupMode: z.enum(["open", "application", "external"]),
    capacity: z.coerce.number().int().positive("Лимит мест должен быть больше нуля").optional(),
    curatorQuestion: z.string().optional(),
    externalRegistrationUrl: z.string().optional(),
  })
  .superRefine((v, ctx) => {
    if (v.signupMode === "application" && !v.curatorQuestion?.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["curatorQuestion"],
        message: "Для режима «по заявке» нужен вопрос кандидату",
      });
    }
    if (v.signupMode === "external") {
      const url = v.externalRegistrationUrl?.trim() ?? "";
      const ok = /^https?:\/\/.+/.test(url);
      if (!ok) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["externalRegistrationUrl"],
          message: "Укажите ссылку для внешней регистрации (http/https)",
        });
      }
    }
  });

export type FormValues = z.input<typeof eventFormSchema>;

/**
 * Converts an ISO instant to the `datetime-local` input value ("YYYY-MM-DDTHH:mm")
 * in the browser's local timezone — the inverse of `new Date(v.startsAt)` used on
 * submit below. No date library: the repo deliberately sticks to native Date +
 * Intl (see lib/calendar.ts), and this is a small enough conversion to inline.
 */
export function toDatetimeLocalValue(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const inputCls =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export interface CreateEventFormProps {
  /** Default "create". "edit" reuses this form to PATCH an existing event. */
  mode?: "create" | "edit";
  /** Required in edit mode: the event being edited. */
  eventId?: string;
  /** Seed values in edit mode, mapped from the fetched event (incl. the cover). */
  initial?: Partial<FormValues> & { coverFileId?: string; coverPreviewUrl?: string };
}

export function CreateEventForm({ mode = "create", eventId, initial }: CreateEventFormProps) {
  const router = useRouter();
  const { isAuthed, ready } = useAuth();

  // Once an event is published, the backend locks its signup mode (422 on
  // change) — lock the control client-side too so submits don't fail.
  const isPublishedEdit = mode === "edit" && initial?.status === "published";

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(eventFormSchema),
    defaultValues: {
      format: initial?.format ?? "offline",
      isFree: initial?.isFree ?? true,
      // Default to Черновик so an accidental "Сохранить" never publishes a
      // half-built event. Publishing is an explicit choice (status control, or
      // the "Опубликовать" action on /events/mine behind a confirm).
      status: initial?.status ?? "draft",
      categoryIds: initial?.categoryIds ?? [],
      venueId: initial?.venueId ?? "",
      signupMode: initial?.signupMode ?? "open",
      title: initial?.title,
      description: initial?.description,
      startsAt: initial?.startsAt,
      endsAt: initial?.endsAt,
      priceMin: initial?.priceMin,
      capacity: initial?.capacity,
      curatorQuestion: initial?.curatorQuestion,
      externalRegistrationUrl: initial?.externalRegistrationUrl,
    },
  });

  const isFree = useWatch({ control, name: "isFree" });
  const signupMode = useWatch({ control, name: "signupMode" });
  const startsAt = useWatch({ control, name: "startsAt" });
  const venueId = useWatch({ control, name: "venueId" });

  const [coverFileId, setCoverFileId] = useState<string | undefined>(initial?.coverFileId);
  const [coverPreviewUrl, setCoverPreviewUrl] = useState<string | undefined>(
    initial?.coverPreviewUrl,
  );
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

  const editMutation = useMutation({
    mutationFn: (patch: Partial<CreateEventInput>) => patchEvent(eventId as string, patch),
    onSuccess: (event) => router.push(`/events/${event.id}`),
  });

  // Non-blocking heads-up: changing the start time or venue on an already-
  // published event doesn't notify anyone automatically (no re-moderation,
  // no participant email) — the organizer has to do that themselves.
  const showChangeNotice =
    isPublishedEdit &&
    ((initial?.startsAt != null && startsAt !== initial.startsAt) ||
      (initial?.venueId != null && venueId !== initial.venueId));

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
      signup_mode: v.signupMode,
      capacity:
        v.signupMode !== "external" && v.capacity != null && String(v.capacity) !== ""
          ? Number(v.capacity)
          : undefined,
      curator_question:
        v.signupMode === "application" ? v.curatorQuestion?.trim() || undefined : undefined,
      external_registration_url:
        v.signupMode === "external" ? v.externalRegistrationUrl?.trim() || undefined : undefined,
    };

    if (mode === "edit") {
      // Once published, signup_mode is locked server-side (422 to change it) —
      // omit it from the patch entirely rather than resend the same value, so
      // we never trip that guard from a stale/disabled control.
      const patch: Partial<CreateEventInput> = { ...input };
      if (isPublishedEdit) {
        delete patch.signup_mode;
      }
      editMutation.mutate(patch);
      return;
    }

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
          <Link
            href={mode === "edit" && eventId ? `/events/${eventId}` : "/"}
            className="text-[17px] text-accent"
          >
            Отмена
          </Link>
          <span className="text-[17px] font-semibold">
            {mode === "edit" ? "Редактирование события" : "Новое событие"}
          </span>
          <Button
            type="submit"
            variant="plain"
            disabled={mode === "edit" ? editMutation.isPending : mutation.isPending}
          >
            {(mode === "edit" ? editMutation.isPending : mutation.isPending)
              ? "Сохранение…"
              : "Сохранить"}
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
          {showChangeNotice && (
            <p className="text-[13px] text-label-secondary">
              Участники уже записаны — предупредите их об изменении самостоятельно.
            </p>
          )}
        </Section>

        <Section title="Запись">
          <Field label="Как записываются">
            <Controller
              control={control}
              name="signupMode"
              render={({ field }) => (
                <Segmented
                  options={[
                    { value: "open", label: "Открытая" },
                    { value: "application", label: "По заявке" },
                    { value: "external", label: "Внешняя ссылка" },
                  ]}
                  value={field.value}
                  onChange={field.onChange}
                  disabled={isPublishedEdit}
                />
              )}
            />
            {isPublishedEdit && (
              <span className="mt-1.5 block text-[13px] text-label-secondary">
                Режим записи зафиксирован после публикации
              </span>
            )}
          </Field>

          {signupMode !== "external" && (
            <Field label="Лимит мест" error={errors.capacity?.message}>
              <input
                type="number"
                min={1}
                className={inputCls}
                placeholder="Оставьте пустым — без ограничения"
                {...register("capacity")}
              />
            </Field>
          )}

          {signupMode === "application" && (
            <Field label="Вопрос кандидату" error={errors.curatorQuestion?.message}>
              <textarea
                className={cn(inputCls, "min-h-[72px] resize-y")}
                placeholder="Покажется в форме заявки. Например: «Над чем работаете?»"
                {...register("curatorQuestion")}
              />
            </Field>
          )}

          {signupMode === "external" && (
            <Field label="Ссылка для регистрации" error={errors.externalRegistrationUrl?.message}>
              <input
                type="url"
                className={inputCls}
                placeholder="https://…"
                {...register("externalRegistrationUrl")}
              />
            </Field>
          )}
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
                    { value: "published", label: "Опубликовать" },
                  ]}
                  value={field.value}
                  onChange={field.onChange}
                />
              )}
            />
          </Field>
        </Section>

        {mode === "create" && mutation.isError && (
          <p className="mt-4 text-[15px] text-red-500">
            {mutation.error instanceof Error && mutation.error.message.includes("429")
              ? "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
              : "Не удалось сохранить событие. Проверьте, что бэкенд запущен."}
          </p>
        )}

        {mode === "edit" && editMutation.isError && (
          <p className="mt-4 text-[15px] text-red-500">
            {editMutation.error instanceof Error && editMutation.error.message.includes("409")
              ? /occupied|capacity/i.test(editMutation.error.message)
                ? "Нельзя уменьшить лимит мест ниже числа уже записавшихся"
                : "Это событие нельзя редактировать в текущем статусе"
              : "Не удалось сохранить изменения. Проверьте, что бэкенд запущен."}
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
