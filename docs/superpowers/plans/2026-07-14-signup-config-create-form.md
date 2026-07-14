# Signup Configuration in Create-Event Form (R1) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose signup mode (open / application / external), seat capacity, curator question, and external registration URL in the create-event form so organizers can build application/external/capped events (e.g. Tarsky's «Тур в Геленджик», 10 people by selection) without a raw API call.

**Architecture:** The backend already accepts and maps every field (`EventFromAPIInput` in `internal/http/formatter/event.go:150-216`, validation in `internal/models/event.go:83-119`). This is therefore **frontend-first**: add form fields + Zod validation + payload plumbing, and confirm the API model/type carry the fields. One optional backend touch: make the create-path 422 messages Russian.

**Tech Stack:** Next.js 15 App Router, TypeScript, react-hook-form, Zod, Tailwind. Backend Go (go-swagger models, go-pg). Spec: `docs/superpowers/specs/2026-07-14-signup-config-create-form-design.md`.

## Global Constraints

- All user-facing copy in **Russian**.
- No DB migration — schema stays **018**. Columns `signup_mode`, `capacity`, `curator_question`, `external_registration_url` already exist (migration 000012).
- `signup_mode` must default to **`open`** on the create path (never `''`) — the `events_signup_mode_check` + `use_zero` gotcha causes a 503 otherwise. The formatter already defaults it; do not regress.
- `external` mode has no local RSVP rows: hide capacity + curator question when external is selected.
- Backend rebuilds (if the optional task is done) need `make generate-api` first (regenerates the gitignored swagger model). Do NOT use `make generate-all`.

---

### Task 1: Extend the frontend create payload type

**Files:**
- Modify: `frontend/lib/api.ts:266-278` (`CreateEventInput` interface)

**Interfaces:**
- Produces: `CreateEventInput` gains optional `signup_mode?: "open" | "application" | "external"`, `capacity?: number`, `curator_question?: string`, `external_registration_url?: string`. Consumed by Task 3 (`onSubmit`) and by the backend `EventInput` (fields already read in the formatter).

- [ ] **Step 1: Add the fields to the interface**

In `frontend/lib/api.ts`, extend `CreateEventInput`:

```ts
export interface CreateEventInput {
  title: string;
  description?: string;
  category_ids?: string[];
  venue_id?: string;
  status?: EventStatus;
  format?: EventFormat;
  price_type?: PriceType;
  price_min?: number;
  starts_at: string; // ISO 8601
  ends_at?: string;
  cover_file_id?: string;
  // Signup configuration (backend maps these in EventFromAPIInput).
  signup_mode?: "open" | "application" | "external";
  capacity?: number;
  curator_question?: string;
  external_registration_url?: string;
}
```

- [ ] **Step 2: Type-check**

Run: `cd frontend && pnpm tsc --noEmit`
Expected: PASS (no usages yet; the interface just widened).

- [ ] **Step 3: Commit**

```bash
git add frontend/lib/api.ts
git commit -m "feat(r1): add signup config fields to CreateEventInput"
```

---

### Task 2: Zod schema — conditional signup validation

**Files:**
- Modify: `frontend/components/CreateEventForm.tsx:20-33` (the `schema` + `FormValues`)
- Test: `frontend/components/__tests__/create-event-schema.test.ts` (new)

**Interfaces:**
- Produces: the exported `schema` now includes `signupMode: "open"|"application"|"external"`, `capacity?: number`, `curatorQuestion?: string`, `externalRegistrationUrl?: string`, with a `superRefine` enforcing the conditional rules. Task 3 renders controls bound to these names.

- [ ] **Step 1: Export the schema and write failing tests**

First make `schema` importable — in `CreateEventForm.tsx` change `const schema =` to `export const eventFormSchema =` (and update the local `zodResolver(eventFormSchema)` + `z.input<typeof eventFormSchema>` references).

Create `frontend/components/__tests__/create-event-schema.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { eventFormSchema } from "@/components/CreateEventForm";

const base = {
  title: "Тур",
  format: "offline" as const,
  startsAt: "2026-08-01T10:00",
  isFree: true,
  status: "published" as const,
  signupMode: "open" as const,
};

describe("eventFormSchema signup rules", () => {
  it("open with no capacity passes", () => {
    expect(eventFormSchema.safeParse(base).success).toBe(true);
  });

  it("application requires a curator question", () => {
    const r = eventFormSchema.safeParse({ ...base, signupMode: "application" });
    expect(r.success).toBe(false);
  });

  it("application with a question passes", () => {
    const r = eventFormSchema.safeParse({
      ...base, signupMode: "application", curatorQuestion: "Над чем работаете?",
    });
    expect(r.success).toBe(true);
  });

  it("external requires a valid url", () => {
    const bad = eventFormSchema.safeParse({ ...base, signupMode: "external" });
    expect(bad.success).toBe(false);
    const ok = eventFormSchema.safeParse({
      ...base, signupMode: "external", externalRegistrationUrl: "https://t.me/x",
    });
    expect(ok.success).toBe(true);
  });

  it("capacity must be a positive integer when set", () => {
    expect(eventFormSchema.safeParse({ ...base, capacity: 0 }).success).toBe(false);
    expect(eventFormSchema.safeParse({ ...base, capacity: 10 }).success).toBe(true);
  });
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd frontend && pnpm vitest run components/__tests__/create-event-schema.test.ts`
Expected: FAIL — either import error (schema not exported) or assertions fail (no conditional rules yet). (If vitest is not configured, add it: `pnpm add -D vitest` and a `test` script `vitest`. Confirm with `pnpm vitest --version`.)

- [ ] **Step 3: Implement the schema**

Replace the schema block in `CreateEventForm.tsx`:

```ts
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

type FormValues = z.input<typeof eventFormSchema>;
```

Add `signupMode: "open"` to the form's `defaultValues`.

- [ ] **Step 4: Run to verify it passes**

Run: `cd frontend && pnpm vitest run components/__tests__/create-event-schema.test.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/components/CreateEventForm.tsx frontend/components/__tests__/create-event-schema.test.ts
git commit -m "feat(r1): conditional zod validation for signup mode"
```

---

### Task 3: Render the «Запись» section + forward the payload

**Files:**
- Modify: `frontend/components/CreateEventForm.tsx` (add a `Section` after «Место и время», extend `onSubmit`)

**Interfaces:**
- Consumes: `eventFormSchema` (Task 2), `CreateEventInput` (Task 1).
- Produces: form UI + a payload that carries the signup fields.

- [ ] **Step 1: Watch the mode and add the section**

Add near the existing `isFree` watch:

```tsx
const signupMode = useWatch({ control, name: "signupMode" });
```

Insert this section immediately after the «Место и время» `Section` (after line ~259):

```tsx
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
        />
      )}
    />
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
```

- [ ] **Step 2: Forward the fields in `onSubmit`**

Extend the `input` object in `onSubmit`:

```ts
const input: CreateEventInput = {
  // …existing fields…
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
```

- [ ] **Step 3: Build + lint**

Run: `cd frontend && pnpm lint && pnpm build`
Expected: PASS.

- [ ] **Step 4: Manual smoke (local backend up)**

Create an application event, capacity 10, with a curator question, entirely from the form. Then `GET /api/v1/events/{id}` and confirm `signup_mode":"application"`, `"capacity":10`, `"curator_question":"…"`. Repeat for `external` (URL required) and `open` (no capacity → payload omits `capacity`).

- [ ] **Step 5: Commit**

```bash
git add frontend/components/CreateEventForm.tsx
git commit -m "feat(r1): signup mode/capacity/curator/external fields in create form"
```

---

### Task 4 (optional, backend): Russian 422 messages for signup validation

**Files:**
- Modify: `backend/internal/models/event.go:100-116` (message strings only)
- Test: `backend/internal/models/event_test.go` (add cases)

**Interfaces:**
- Produces: `Event.Validate()` returns Russian messages for the three signup rules. No signature change.

- [ ] **Step 1: Add failing test cases**

In `backend/internal/models/event_test.go`, add:

```go
func TestEventValidate_SignupMessages(t *testing.T) {
	e := &models.Event{Title: "T", StartsAt: time.Now(), Status: models.EventPublished, SignupMode: "application"}
	if err := e.Validate(); err == nil || !strings.Contains(err.Error(), "вопрос") {
		t.Fatalf("want curator-question message, got %v", err)
	}
	cap0 := 0
	e2 := &models.Event{Title: "T", StartsAt: time.Now(), Status: models.EventPublished, SignupMode: "open", Capacity: &cap0}
	if err := e2.Validate(); err == nil || !strings.Contains(err.Error(), "больше нуля") {
		t.Fatalf("want capacity message, got %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/models/ -run TestEventValidate_SignupMessages -v`
Expected: FAIL (current messages are English).

- [ ] **Step 3: Update the messages**

In `Event.Validate()`:

```go
case "application":
    if e.CuratorQuestion == "" {
        return newValidationError("curator_question", "нужен вопрос кандидату для режима «по заявке»")
    }
case "external":
    if e.ExternalRegistrationURL == "" {
        return newValidationError("external_registration_url", "нужна ссылка для внешней регистрации")
    }
// …
if e.Capacity != nil && *e.Capacity <= 0 {
    return newValidationError("capacity", "лимит мест должен быть больше нуля")
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && go test ./internal/models/ -run TestEventValidate_SignupMessages -v`
Expected: PASS.

- [ ] **Step 5: Full backend gate + commit**

```bash
cd backend && go build ./... && go vet ./... && golangci-lint run
git add backend/internal/models/event.go backend/internal/models/event_test.go
git commit -m "feat(r1): russian validation messages for signup config"
```

---

## Self-Review

- **Spec coverage:** signup-mode control (Task 3) ✓; capacity for open/application (Task 3) ✓; curator question required for application (Tasks 2/3) ✓; external URL required for external (Tasks 2/3) ✓; capacity>0 (Task 2/4) ✓; Russian messages (Task 4) ✓; hide capacity/curator for external (Task 3) ✓; no migration ✓; `open` default preserved (Task 2 defaultValues + formatter unchanged) ✓; backwards-compatible open payload (Task 3 omits `capacity` when empty) ✓.
- **Placeholder scan:** none — all steps carry real code/commands.
- **Type consistency:** `signupMode` enum values match backend `open|application|external`; `CreateEventInput` field names are snake_case matching the swagger `EventInput` fields the formatter reads (`in.SignupMode`, `in.Capacity`, `in.CuratorQuestion`, `in.ExternalRegistrationURL`).

## Deploy

Frontend-only unless Task 4 is included. No migration (schema 018). Standard build-on-Mac→`save|ssh|load`. If Task 4 is included, run `make generate-api` before the backend build.
