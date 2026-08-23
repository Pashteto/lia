"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Chip } from "@/components/ui/Chip";
import { EventModule } from "@/components/ui/EventModule";
import { Input, Textarea } from "@/components/ui/Field";
import { ImageUpload } from "@/components/ui/ImageUpload";
import { AppHeader, ORG_NAV } from "@/components/ui/AppHeader";
import { AuthGate } from "@/components/ui/AuthGate";
import { Skeleton } from "@/components/ui/Skeleton";
import { Stepper } from "@/components/ui/Stepper";
import { joinLocal, splitLocal } from "@/components/ui/DateTimeField";
import { VenuePicker } from "@/components/VenuePicker";
import { VerifyEmailInterstitial } from "@/components/VerifyEmailInterstitial";
import { uploadErrorMessage, validateImageFile } from "@/lib/upload-errors";
import {
  createEvent,
  EMAIL_NOT_VERIFIED,
  fetchTrustedPlatforms,
  getCategories,
  getMyOrganizer,
  patchEvent,
  type CreateEventInput,
  uploadFile,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { categoryNumeral } from "@/lib/category-numerals";
import { cn } from "@/lib/cn";
import { formatModuleDate } from "@/lib/format";
import { moderationNote } from "@/lib/publish-copy";
import { hostOf, matchPlatform, type TrustedPlatform } from "@/lib/platforms";
import { priceLabel } from "@/lib/price-label";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { Controller, useForm, useWatch, type FieldErrors } from "react-hook-form";
import { z } from "zod";

export const eventFormSchema = z
  .object({
    title: z.string().min(1, "Укажите название"),
    description: z.string().optional(),
    categoryIds: z.array(z.string()).optional(),
    format: z.enum(["offline", "online"]),
    venueId: z.string().optional(),
    startsAt: z.string({ error: "Укажите дату и время" }).min(1, "Укажите дату и время"),
    endsAt: z.string().optional(),
    isFree: z.boolean(),
    priceMin: z.coerce.number().int().min(0).optional(),
    status: z.enum(["draft", "published"]),
    signupMode: z.enum(["open", "application", "external"]),
    capacity: z.preprocess(
      (v) => (v === "" || v == null ? undefined : v),
      z.coerce.number().int().positive("Лимит мест должен быть больше нуля").optional(),
    ),
    curatorQuestion: z.string().optional(),
    externalRegistrationUrl: z.string().optional(),
    capacityLimited: z.boolean().optional(),
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
      const ok = /^https:\/\/.+/.test(url);
      if (!ok) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["externalRegistrationUrl"],
          message: "Укажите ссылку для внешней регистрации (https)",
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

// «Запись», not «Билеты»: the platform is not about selling tickets — most
// events are free / by signup / by application (design review P2).
const STEPS = ["Основное", "Когда и где", "Запись", "Публикация"] as const;
const NEXT_LABELS = [
  "ДАЛЕЕ · КОГДА И ГДЕ",
  "ДАЛЕЕ · ЗАПИСЬ",
  "ДАЛЕЕ · ПУБЛИКАЦИЯ",
  "СОХРАНИТЬ",
] as const;

/** RHF field → wizard step (cover is outside the schema). */
const FIELD_STEP: Record<string, number> = {
  title: 0,
  categoryIds: 0,
  description: 0,
  format: 1,
  venueId: 1,
  startsAt: 1,
  endsAt: 1,
  isFree: 2,
  priceMin: 2,
  capacity: 2,
  signupMode: 2,
  curatorQuestion: 2,
  externalRegistrationUrl: 2,
  status: 3,
};

const STEP_FIELDS: Record<number, (keyof FormValues)[]> = {
  0: ["title", "categoryIds", "description"],
  1: ["format", "venueId", "startsAt", "endsAt"],
  2: [
    "isFree",
    "priceMin",
    "capacity",
    "signupMode",
    "curatorQuestion",
    "externalRegistrationUrl",
  ],
  3: ["status"],
};

const FIELD_ORDER = [
  "title",
  "categoryIds",
  "description",
  "format",
  "venueId",
  "startsAt",
  "endsAt",
  "isFree",
  "priceMin",
  "capacity",
  "signupMode",
  "curatorQuestion",
  "externalRegistrationUrl",
  "status",
] as const;

/** First wizard step that owns a present RHF / Zod error path. */
export function firstErrorStep(
  errors: Partial<Record<string, unknown>> | undefined,
): number {
  if (!errors) return 0;
  for (const key of FIELD_ORDER) {
    if (errors[key]) return FIELD_STEP[key] ?? 0;
  }
  return 0;
}

function firstErrorStepFromZodPaths(paths: string[]): number {
  for (const key of FIELD_ORDER) {
    if (paths.includes(key)) return FIELD_STEP[key] ?? 0;
  }
  return 0;
}

const DT_BOX =
  "border border-on-surface bg-transparent px-[11px] py-[9px] text-[12.5px] text-on-surface swiss-focus";

const mskClock = new Intl.DateTimeFormat("ru-RU", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
  timeZone: "Europe/Moscow",
});

export function valuesToInput(v: FormValues, coverFileId?: string): CreateEventInput {
  return {
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
    capacity_limited: v.signupMode === "external" ? (v.capacityLimited ?? false) : undefined,
  };
}

export interface CreateEventFormProps {
  /** Default "create". "edit" reuses this form to PATCH an existing event. */
  mode?: "create" | "edit";
  /** Required in edit mode: the event being edited. */
  eventId?: string;
  /** Seed values in edit mode, mapped from the fetched event (incl. the cover). */
  initial?: Partial<FormValues> & {
    coverFileId?: string;
    coverPreviewUrl?: string;
    venueName?: string;
  };
}

export function CreateEventForm({ mode = "create", eventId, initial }: CreateEventFormProps) {
  const router = useRouter();
  const { isAuthed, ready } = useAuth();

  // Once an event is published, the backend locks its signup mode (422 on
  // change) — lock the control client-side too so submits don't fail.
  const isPublishedEdit = mode === "edit" && initial?.status === "published";

  const [step, setStep] = useState(0);
  const [savedAt, setSavedAt] = useState<Date | null>(null);
  const [venueName, setVenueName] = useState(initial?.venueName ?? "");
  const [showCheckBanner, setShowCheckBanner] = useState(false);
  const autosaveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [platforms, setPlatforms] = useState<TrustedPlatform[]>([]);

  // A failed load just degrades to "no hint" — never blocks the form.
  useEffect(() => {
    fetchTrustedPlatforms().then(setPlatforms).catch(() => {});
  }, []);

  const {
    register,
    handleSubmit,
    control,
    getValues,
    reset,
    trigger,
    formState: { errors, isDirty },
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
      capacityLimited: initial?.capacityLimited ?? false,
    },
  });

  const isFree = useWatch({ control, name: "isFree" });
  const signupMode = useWatch({ control, name: "signupMode" });
  const startsAt = useWatch({ control, name: "startsAt" });
  const endsAt = useWatch({ control, name: "endsAt" });
  const venueId = useWatch({ control, name: "venueId" });
  const title = useWatch({ control, name: "title" });
  const categoryIds = useWatch({ control, name: "categoryIds" });
  const format = useWatch({ control, name: "format" });
  const priceMin = useWatch({ control, name: "priceMin" });
  const extUrl = useWatch({ control, name: "externalRegistrationUrl" });
  const matched = extUrl ? matchPlatform(extUrl.trim(), platforms) : null;

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

  // Honest moderation note: verified organizers publish instantly.
  const { data: myOrganizerData } = useQuery({
    queryKey: ["my-organizer"],
    queryFn: () => getMyOrganizer().catch(() => null),
  });
  const organizerVerified = myOrganizerData?.verification_status === "verified";

  const mutation = useMutation({
    mutationFn: (input: CreateEventInput) => createEvent(input),
    onSuccess: (event) => {
      if (event.moderationRequired) {
        router.push(`/events/${event.id}?moderation=1`);
        return;
      }
      router.push(`/events/${event.id}`);
    },
  });

  const editMutation = useMutation({
    mutationFn: (patch: Partial<CreateEventInput>) => patchEvent(eventId as string, patch),
    onSuccess: (event) => {
      if (event.moderationRequired) {
        router.push(`/events/${event.id}?moderation=1`);
        return;
      }
      router.push(`/events/${event.id}`);
    },
  });

  /** Create-mode ghost ЧЕРНОВИК — explicit first save, then edit route. */
  const draftMutation = useMutation({
    mutationFn: (input: CreateEventInput) => createEvent({ ...input, status: "draft" }),
    onSuccess: (event) => router.push(`/events/${event.id}/edit`),
  });

  /** Edit-mode silent autosave — must not navigate. */
  const autosaveMutation = useMutation({
    mutationFn: (patch: Partial<CreateEventInput>) => patchEvent(eventId as string, patch),
    onSuccess: () => {
      setSavedAt(new Date());
      reset(getValues());
    },
  });
  const autosaveMutate = autosaveMutation.mutate;

  // Non-blocking heads-up: changing the start time or venue on an already-
  // published event doesn't notify anyone automatically (no re-moderation,
  // no participant email) — the organizer has to do that themselves.
  const showChangeNotice =
    isPublishedEdit &&
    ((initial?.startsAt != null && startsAt !== initial.startsAt) ||
      (initial?.venueId != null && venueId !== initial.venueId));

  const flushAutosave = useCallback(
    (opts?: { force?: boolean; coverId?: string }) => {
      if (mode !== "edit" || !eventId) return;
      if (!opts?.force && !isDirty) return;
      if (autosaveTimer.current) clearTimeout(autosaveTimer.current);
      const coverId = opts?.coverId ?? coverFileId;
      autosaveTimer.current = setTimeout(() => {
        const v = getValues();
        const parsed = eventFormSchema.safeParse(v);
        if (!parsed.success) return;
        const input = valuesToInput(v, coverId);
        const patch: Partial<CreateEventInput> = { ...input };
        if (isPublishedEdit) delete patch.signup_mode;
        autosaveMutate(patch);
      }, 700);
    },
    [mode, eventId, isDirty, getValues, coverFileId, isPublishedEdit, autosaveMutate],
  );

  useEffect(() => {
    return () => {
      if (autosaveTimer.current) clearTimeout(autosaveTimer.current);
    };
  }, []);

  const handleCoverFile = async (file: File) => {
    const tooBig = validateImageFile(file);
    if (tooBig) {
      setCoverError(tooBig);
      return;
    }
    setCoverError(undefined);
    setCoverUploading(true);
    try {
      const { id, url } = await uploadFile(file);
      setCoverFileId(id);
      setCoverPreviewUrl(url);
      // Cover lives outside RHF dirty tracking — force edit-mode patch with new id.
      flushAutosave({ force: true, coverId: id });
    } catch (err) {
      setCoverError(uploadErrorMessage(err));
    } finally {
      setCoverUploading(false);
    }
  };

  const onSubmit = (v: FormValues) => {
    setShowCheckBanner(false);
    const input = valuesToInput(v, coverFileId);

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

  /** Jump to the first step that owns a validation error so messages aren't hidden. */
  const revealFieldErrors = (errs: FieldErrors<FormValues>) => {
    setStep(firstErrorStep(errs));
    setShowCheckBanner(true);
  };

  const goNext = async () => {
    const fields = STEP_FIELDS[step] ?? [];
    const ok = await trigger(fields);
    if (!ok) {
      setShowCheckBanner(true);
      return;
    }
    setShowCheckBanner(false);
    setStep((s) => Math.min(3, s + 1));
  };

  const saveDraftGhost = async () => {
    const v = { ...getValues(), status: "draft" as const };
    const parsed = eventFormSchema.safeParse(v);
    if (!parsed.success) {
      await trigger();
      const paths = parsed.error.issues.map((i) => String(i.path[0] ?? ""));
      setStep(firstErrorStepFromZodPaths(paths));
      setShowCheckBanner(true);
      return;
    }
    setShowCheckBanner(false);
    draftMutation.mutate(valuesToInput(v, coverFileId));
  };

  const firstCat = categories.find((c) => (categoryIds ?? []).includes(c.id));
  const previewDate =
    startsAt && String(startsAt).length > 0
      ? formatModuleDate(
          new Date(startsAt).toISOString(),
          endsAt && String(endsAt).length > 0 ? new Date(endsAt).toISOString() : undefined,
        )
      : "—";
  const previewVenue =
    format === "online" ? "Онлайн" : venueName.trim() || "—";
  const previewPrice = priceLabel(isFree ? 0 : Number(priceMin) || 0, isFree ? "free" : "from");
  const savedChip = savedAt ? `ЧЕРНОВИК СОХРАНЁН · ${mskClock.format(savedAt)}` : null;
  const stepCounter = `${String(step + 1).padStart(2, "0")}/04`;

  const pending =
    mode === "edit"
      ? editMutation.isPending
      : mutation.isPending || draftMutation.isPending;

  const headerActions = (
    <>
      <span className="font-mono text-[9px] font-bold tracking-[0.12em] sm:hidden">
        {stepCounter}
      </span>
      {savedChip ? <span className="cap max-sm:hidden">{savedChip}</span> : null}
    </>
  );

  // Gate: creating an event requires a signed-in user (backend returns 401
  // otherwise). Avoid flashing the form before the session is read.
  if (!ready) {
    return (
      <>
        <AppHeader nav={ORG_NAV} mobileCaption="НОВОЕ СОБЫТИЕ" />
        <Skeleton className="h-[48px] w-full border-x-0 border-t-0" />
        <Skeleton className="h-[320px] w-full border-x-0 border-t-0" />
      </>
    );
  }
  if (!isAuthed) {
    return (
      <>
        <AppHeader nav={ORG_NAV} mobileCaption="НОВОЕ СОБЫТИЕ" />
        <AuthGate
          title="Войдите, чтобы создать событие"
          reassurance="Создание событий доступно авторизованным пользователям. Лента и карта доступны без входа."
        />
      </>
    );
  }

  return (
    <>
      <AppHeader
        nav={ORG_NAV}
        mobileCaption={mode === "edit" ? "РЕДАКТИРОВАНИЕ" : "НОВОЕ СОБЫТИЕ"}
        actions={headerActions}
      />

      {/* Desktop stepper */}
      <div className="mx-auto hidden max-w-[1360px] sm:block">
        <Stepper
          steps={[...STEPS]}
          current={step}
          fillMode="inclusive"
          numeralPrefix="Шаг "
        />
      </div>

      {/* Mobile 4-segment progress (ink / inactive) */}
      <div className="flex border-b border-ink sm:hidden" aria-hidden>
        {STEPS.map((_, i) => (
          <div
            key={STEPS[i]}
            className={cn(
              "h-[5px] flex-1",
              i <= step ? "bg-ink" : "bg-inactive",
              i > 0 && "border-l border-ink",
            )}
          />
        ))}
      </div>

      <form
        onSubmit={handleSubmit(onSubmit, revealFieldErrors)}
        onBlurCapture={() => flushAutosave()}
        className="mx-auto max-w-[1360px] pb-[64px] max-sm:pb-[88px]"
      >
        <div className="grid grid-cols-1 sm:grid-cols-[1fr_240px]">
          {/* Left: step fields */}
          <div className="flex flex-col gap-[11px] border-b border-ink px-[20px] py-[16px] max-sm:px-[14px] max-sm:py-[13px] sm:border-r sm:border-b-0">
            {showCheckBanner ? (
              <p
                role="alert"
                className="border border-signal bg-signal-tint px-[11px] py-[8px] text-[12.5px] font-bold text-signal"
              >
                Проверьте поля
              </p>
            ) : null}
            {/* 01 Основное — keep mounted */}
            <div hidden={step !== 0} className="flex flex-col gap-[11px]">
              <Input
                label="Название"
                error={errors.title?.message}
                placeholder="Например, «Читаем Зебальда»"
                className="font-bold"
                {...register("title")}
              />
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Категория</span>
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
                      <div className="flex flex-wrap gap-[5px]">
                        {categories.length === 0 && (
                          <span className="text-[11px] text-text-dim">
                            Категории недоступны (бэкенд офлайн)
                          </span>
                        )}
                        {categories.map((c) => {
                          const on = selected.includes(c.id);
                          const numeral = categoryNumeral(c.slug, categories);
                          return (
                            <Chip
                              key={c.id}
                              variant={on ? "active" : "default"}
                              onClick={() => toggle(c.id)}
                            >
                              {numeral} {c.label}
                            </Chip>
                          );
                        })}
                      </div>
                    );
                  }}
                />
              </div>
              <Textarea
                label="Описание"
                placeholder="О чём встреча, чего ждать участникам"
                className="min-h-[52px] text-field-text"
                {...register("description")}
              />
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Обложка</span>
                <div className="min-h-[62px] border border-ink [&_.rounded-card]:rounded-none [&_.rounded-card]:bg-transparent [&_.rounded-card]:p-2 [&_.rounded-card]:shadow-none [&_button]:rounded-none">
                  <ImageUpload
                    label="обложку"
                    previewUrl={coverPreviewUrl}
                    uploading={coverUploading}
                    error={coverError}
                    onFile={handleCoverFile}
                  />
                </div>
              </div>
            </div>

            {/* 02 Когда и где */}
            <div hidden={step !== 1} className="flex flex-col gap-[11px]">
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Формат</span>
                <Controller
                  control={control}
                  name="format"
                  render={({ field }) => (
                    <div className="flex flex-wrap gap-[5px]">
                      {(
                        [
                          ["offline", "Очно"],
                          ["online", "Онлайн"],
                        ] as const
                      ).map(([value, label]) => (
                        <Chip
                          key={value}
                          variant={field.value === value ? "active" : "default"}
                          onClick={() => field.onChange(value)}
                        >
                          {label}
                        </Chip>
                      ))}
                    </div>
                  )}
                />
              </div>
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Место</span>
                <div className="border border-ink p-[1px] [&_input]:rounded-none [&_input]:border-0 [&_input]:bg-transparent [&_input]:px-[11px] [&_input]:py-[9px] [&_input]:text-[12.5px] [&_input]:shadow-none [&_input]:ring-0 [&_input]:outline-none [&_input]:swiss-focus">
                  <Controller
                    control={control}
                    name="venueId"
                    render={({ field }) => (
                      <VenuePicker
                        value={field.value ?? ""}
                        onChange={field.onChange}
                        onLabelChange={setVenueName}
                        initialLabel={initial?.venueName}
                      />
                    )}
                  />
                </div>
              </div>
              <Controller
                control={control}
                name="startsAt"
                render={({ field }) => (
                  <SwissDateTime
                    label="Начало"
                    error={errors.startsAt?.message}
                    value={field.value ?? ""}
                    onChange={field.onChange}
                  />
                )}
              />
              <Controller
                control={control}
                name="endsAt"
                render={({ field }) => (
                  <SwissDateTime
                    label="Окончание"
                    value={field.value ?? ""}
                    onChange={field.onChange}
                  />
                )}
              />
              {showChangeNotice && (
                <p className="text-[11px] text-text-dim">
                  Участники уже записаны — предупредите их об изменении самостоятельно.
                </p>
              )}
            </div>

            {/* 03 Запись */}
            <div hidden={step !== 2} className="flex flex-col gap-[11px]">
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Цена</span>
                <Controller
                  control={control}
                  name="isFree"
                  render={({ field }) => (
                    <div className="flex flex-wrap gap-[5px]">
                      <Chip
                        variant={field.value ? "active" : "default"}
                        onClick={() => field.onChange(true)}
                      >
                        Бесплатно
                      </Chip>
                      <Chip
                        variant={!field.value ? "active" : "default"}
                        onClick={() => field.onChange(false)}
                      >
                        Платно
                      </Chip>
                    </div>
                  )}
                />
              </div>
              <div hidden={!!isFree}>
                <Input
                  label="Цена от, ₽"
                  type="number"
                  min={0}
                  placeholder="2500"
                  {...register("priceMin")}
                />
                {Number(priceMin) > 0 && signupMode !== "external" && (
                  <span className="mt-[4px] block text-[11px] text-text-dim">
                    Посетители увидят „Оплата на месте“
                  </span>
                )}
              </div>

              <div className="flex flex-col gap-[5px]">
                <span className="cap">Как записываются</span>
                <Controller
                  control={control}
                  name="signupMode"
                  render={({ field }) => (
                    <div className="flex flex-wrap gap-[5px]">
                      {(
                        [
                          ["open", "Открытая"],
                          ["application", "По заявке"],
                          ["external", "Внешняя ссылка"],
                        ] as const
                      ).map(([value, label]) => (
                        <Chip
                          key={value}
                          variant={field.value === value ? "active" : "default"}
                          disabled={isPublishedEdit}
                          onClick={() => {
                            if (!isPublishedEdit) field.onChange(value);
                          }}
                        >
                          {label}
                        </Chip>
                      ))}
                    </div>
                  )}
                />
                {isPublishedEdit && (
                  <span className="text-[11px] text-text-dim">
                    Режим записи зафиксирован после публикации
                  </span>
                )}
              </div>

              <div hidden={signupMode === "external"}>
                <Input
                  label="Лимит мест"
                  type="number"
                  min={1}
                  error={errors.capacity?.message}
                  placeholder="Оставьте пустым — без ограничения"
                  {...register("capacity")}
                />
              </div>
              <div hidden={signupMode !== "application"}>
                <Textarea
                  label="Вопрос кандидату"
                  error={errors.curatorQuestion?.message}
                  placeholder="Покажется в форме заявки. Например: «Над чем работаете?»"
                  {...register("curatorQuestion")}
                />
              </div>
              <div hidden={signupMode !== "external"} className="flex flex-col gap-[8px]">
                <div>
                  <Input
                    label="Ссылка для регистрации"
                    type="url"
                    error={errors.externalRegistrationUrl?.message}
                    placeholder="https://…"
                    {...register("externalRegistrationUrl")}
                  />
                  {signupMode === "external" && extUrl?.trim() && (
                    matched ? (
                      <span className="mt-[4px] block text-[11px] text-emerald-700">
                        Платформа: {matched.displayName}
                      </span>
                    ) : hostOf(extUrl.trim()) ? (
                      <span className="mt-[4px] block text-[11px] text-amber-700">
                        Неизвестная платформа — событие уйдёт на проверку модератору
                      </span>
                    ) : null
                  )}
                </div>
                <label className="flex items-center gap-[7px] text-[12px]">
                  <input type="checkbox" {...register("capacityLimited")} />
                  Количество мест ограничено
                </label>
              </div>
            </div>

            {/* 04 Публикация */}
            <div hidden={step !== 3} className="flex flex-col gap-[11px]">
              <div className="flex flex-col gap-[5px]">
                <span className="cap">Статус</span>
                <Controller
                  control={control}
                  name="status"
                  render={({ field }) => (
                    <div className="flex flex-wrap gap-[5px]">
                      <Chip
                        variant={field.value === "draft" ? "active" : "default"}
                        onClick={() => field.onChange("draft")}
                      >
                        Черновик
                      </Chip>
                      <Chip
                        variant={field.value === "published" ? "active" : "default"}
                        onClick={() => field.onChange("published")}
                      >
                        Опубликовать
                      </Chip>
                    </div>
                  )}
                />
              </div>
            </div>

            {/* Errors */}
            {mode === "create" &&
              mutation.isError &&
              !(mutation.error instanceof Error && mutation.error.message === EMAIL_NOT_VERIFIED) && (
                <p className="text-[12.5px] text-signal">
                  {mutation.error instanceof Error && mutation.error.message.includes("429")
                    ? "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
                    : "Не удалось сохранить событие. Проверьте, что бэкенд запущен."}
                </p>
              )}
            {mode === "create" &&
              draftMutation.isError &&
              !(
                draftMutation.error instanceof Error &&
                draftMutation.error.message === EMAIL_NOT_VERIFIED
              ) && (
                <p className="text-[12.5px] text-signal">
                  Не удалось сохранить черновик. Проверьте, что бэкенд запущен.
                </p>
              )}
            {mode === "edit" &&
              editMutation.isError &&
              !(
                editMutation.error instanceof Error &&
                editMutation.error.message === EMAIL_NOT_VERIFIED
              ) && (
                <p className="text-[12.5px] text-signal">
                  {editMutation.error instanceof Error && editMutation.error.message.includes("409")
                    ? /occupied|capacity/i.test(editMutation.error.message)
                      ? "Нельзя уменьшить лимит мест ниже числа уже записавшихся"
                      : "Это событие нельзя редактировать в текущем статусе"
                    : "Не удалось сохранить изменения. Проверьте, что бэкенд запущен."}
                </p>
              )}

            {/* Mobile moderation note above CTAs — denser than desktop rail */}
            <div className="mt-auto border-t border-ink pt-[10px] sm:hidden">
              <p className="cap mb-[5px]">{moderationNote(organizerVerified)}</p>
            </div>

            {/* Actions */}
            <div className="mt-auto flex gap-[8px] max-sm:flex-col">
              {step < 3 ? (
                <Button
                  type="button"
                  className="min-h-[44px] flex-1"
                  onClick={() => void goNext()}
                >
                  <span className="sm:hidden">ДАЛЕЕ</span>
                  <span className="hidden sm:inline">{NEXT_LABELS[step]}</span>
                </Button>
              ) : (
                <Button type="submit" className="min-h-[44px] flex-1" disabled={pending}>
                  {pending ? "Сохранение…" : (
                    <>
                      <span className="sm:hidden">СОХРАНИТЬ</span>
                      <span className="hidden sm:inline">{NEXT_LABELS[3]}</span>
                    </>
                  )}
                </Button>
              )}
              {mode === "create" ? (
                <Button
                  type="button"
                  variant="ghost"
                  className="min-h-[44px] max-sm:hidden sm:flex-none sm:px-[16px]"
                  disabled={draftMutation.isPending}
                  onClick={() => void saveDraftGhost()}
                >
                  {draftMutation.isPending ? "Сохранение…" : "ЧЕРНОВИК"}
                </Button>
              ) : step > 0 ? (
                <Button
                  type="button"
                  variant="ghost"
                  className="min-h-[44px] max-sm:w-full sm:flex-none sm:px-[16px]"
                  onClick={() => {
                    setShowCheckBanner(false);
                    setStep((s) => Math.max(0, s - 1));
                  }}
                >
                  Назад
                </Button>
              ) : null}
            </div>
          </div>

          {/* Right rail — desktop preview + moderation note */}
          <aside className="hidden flex-col border-b border-ink px-[16px] py-[14px] sm:flex">
            <p className="cap mb-[8px]">Превью в ленте</p>
            <div className="border border-ink bg-white">
              <EventModule
                numeral={firstCat ? categoryNumeral(firstCat.slug, categories) : "—"}
                category={firstCat?.label ?? "—"}
                title={title?.trim() || "Без названия"}
                venue={previewVenue}
                date={previewDate}
                price={previewPrice}
                cover={coverPreviewUrl}
                href={mode === "edit" && eventId ? `/events/${eventId}` : "#"}
              />
            </div>
            <div className="mt-auto border-t border-ink pt-[10px]">
              <p className="cap mb-[5px]">После отправки</p>
              <p className="text-[10.5px] leading-[1.45] text-text-dim">
                {organizerVerified
                  ? "Событие появляется в ленте сразу после публикации. Модераторы могут снять его, если оно нарушает правила."
                  : "Событие уйдёт на модерацию — обычно до 24 часов. После одобрения оно появится в ленте."}
              </p>
            </div>
          </aside>
        </div>

        {((mode === "create" &&
          ((mutation.isError &&
            mutation.error instanceof Error &&
            mutation.error.message === EMAIL_NOT_VERIFIED) ||
            (draftMutation.isError &&
              draftMutation.error instanceof Error &&
              draftMutation.error.message === EMAIL_NOT_VERIFIED))) ||
          (mode === "edit" &&
            ((editMutation.isError &&
              editMutation.error instanceof Error &&
              editMutation.error.message === EMAIL_NOT_VERIFIED) ||
              (autosaveMutation.isError &&
                autosaveMutation.error instanceof Error &&
                autosaveMutation.error.message === EMAIL_NOT_VERIFIED)))) && (
          <VerifyEmailInterstitial
            onClose={() => {
              mutation.reset();
              editMutation.reset();
              draftMutation.reset();
              autosaveMutation.reset();
            }}
          />
        )}
      </form>
    </>
  );
}

function SwissDateTime({
  label,
  error,
  value,
  onChange,
}: {
  label: string;
  error?: string;
  value: string;
  onChange: (v: string) => void;
}) {
  const { date, time } = splitLocal(value);
  return (
    <div className="flex flex-col gap-[5px]">
      <span className={cn("cap", error && "text-signal")}>{label}</span>
      <div className="flex flex-wrap items-center gap-[8px]">
        <input
          type="date"
          className={DT_BOX}
          value={date}
          onChange={(e) => onChange(joinLocal(e.target.value, time))}
        />
        <input
          type="time"
          className={DT_BOX}
          value={time}
          onChange={(e) => onChange(joinLocal(date, e.target.value))}
        />
        <span className="cap">Мск</span>
      </div>
      {error ? <p className="text-[11px] text-signal">{error}</p> : null}
    </div>
  );
}
