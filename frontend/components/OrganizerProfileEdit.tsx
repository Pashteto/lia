"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { AuthGate } from "@/components/ui/AuthGate";
import { Button } from "@/components/ui/Button";
import { Input, Textarea } from "@/components/ui/Field";
import { Skeleton } from "@/components/ui/Skeleton";
import { Stepper } from "@/components/ui/Stepper";
import {
  fetchMyEvents,
  getMyOrganizer,
  saveMyOrganizer,
  submitMyOrganizer,
  uploadFile,
  type Organizer,
  type VerificationStatus,
} from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { ORG_VERIFY_STEPS, orgVerificationStep } from "@/lib/org-verification";
import { padCount } from "@/lib/org-seats";
import { tileCount } from "@/lib/tile-count";
import { uploadErrorMessage, validateImageFile } from "@/lib/upload-errors";

function previewCaption(status: VerificationStatus | undefined): string {
  if (status === "verified") return "Проверенный организатор";
  if (status === "pending") return "На проверке";
  return "Организатор";
}

function LogoSlot({
  previewUrl,
  uploading,
  error,
  onFile,
}: {
  previewUrl?: string;
  uploading: boolean;
  error?: string;
  onFile: (file: File) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <div className="flex flex-col gap-[5px]">
      <span className="cap">Логотип</span>
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        disabled={uploading}
        aria-label={previewUrl ? "Заменить логотип" : "Загрузить логотип"}
        className="swiss-focus relative h-[62px] w-[62px] shrink-0 border border-ink bg-transparent disabled:opacity-60"
      >
        {previewUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={previewUrl} alt="" className="h-full w-full object-cover" />
        ) : (
          <span className="absolute inset-0 flex items-center justify-center bg-paper" aria-hidden>
            <span className="cap">Лого</span>
          </span>
        )}
        {uploading ? (
          <span className="absolute inset-0 flex items-center justify-center bg-paper/80 text-[9px] font-bold uppercase tracking-[0.07em]">
            …
          </span>
        ) : null}
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/heic,image/heif,.heic,.heif"
        className="sr-only"
        disabled={uploading}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) onFile(file);
          e.target.value = "";
        }}
      />
      {error ? <p className="text-[11px] text-signal">{error}</p> : null}
    </div>
  );
}

function EditSkeleton() {
  return (
    <main className="mx-auto max-w-[1360px]">
      <Skeleton className="h-[52px] w-full border-x-0 border-t-0" />
      <div className="grid md:grid-cols-[1fr_250px]">
        <Skeleton className="h-[320px] w-full border-x-0 border-t-0" />
        <Skeleton className="hidden h-[320px] w-full border-x-0 border-t-0 md:block" />
      </div>
    </main>
  );
}

export function OrganizerProfileEdit() {
  const { ready, isAuthed, email } = useAuth();
  const [org, setOrg] = useState<Organizer | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [website, setWebsite] = useState("");
  const [logoFileId, setLogoFileId] = useState<string | undefined>();
  const [logoPreviewUrl, setLogoPreviewUrl] = useState<string | undefined>();
  const [logoUploading, setLogoUploading] = useState(false);
  const [logoError, setLogoError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const events = useQuery({
    queryKey: ["my-events"],
    queryFn: fetchMyEvents,
    enabled: ready && isAuthed,
  });

  useEffect(() => {
    if (!ready || !isAuthed) return;
    let cancelled = false;
    getMyOrganizer()
      .then((o) => {
        if (cancelled) return;
        if (o) {
          setOrg(o);
          setName(o.name);
          setDescription(o.description);
          setWebsite(o.website_url);
          setLogoPreviewUrl(o.logo_url);
        }
        setLoaded(true);
      })
      .catch((e) => {
        if (!cancelled) {
          setError(String(e));
          setLoaded(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [ready, isAuthed]);

  if (!ready) return <EditSkeleton />;

  if (!isAuthed) {
    return (
      <AuthGate
        title="Войдите, чтобы создать профиль организатора"
        reassurance="Лента и карта доступны без входа."
      />
    );
  }

  if (!loaded) return <EditSkeleton />;

  const status: VerificationStatus = org?.verification_status ?? "draft";
  const step = orgVerificationStep(status);
  const canSubmit = status === "draft" || status === "rejected";

  const publishedCount = (events.data ?? []).filter((e) => e.status === "published").length;
  const eventsValue = events.isLoading
    ? "—"
    : publishedCount < 100
      ? padCount(publishedCount)
      : String(publishedCount);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const saved = await saveMyOrganizer({
        name,
        description,
        website_url: website,
        logo_file_id: logoFileId,
      });
      setOrg(saved);
      setLogoPreviewUrl(saved.logo_url ?? logoPreviewUrl);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      const { status: next } = await submitMyOrganizer();
      setOrg((prev) => (prev ? { ...prev, verification_status: next } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const onLogo = async (file: File) => {
    const tooBig = validateImageFile(file);
    if (tooBig) {
      setLogoError(tooBig);
      return;
    }
    setLogoError(undefined);
    setLogoUploading(true);
    try {
      const { id, url } = await uploadFile(file);
      setLogoFileId(id);
      setLogoPreviewUrl(url);
    } catch (e) {
      setLogoError(uploadErrorMessage(e));
    } finally {
      setLogoUploading(false);
    }
  };

  const previewName = name.trim() || "Организатор";

  return (
    <main className="mx-auto max-w-[1360px] pb-[64px]">
      <Stepper
        steps={[...ORG_VERIFY_STEPS]}
        current={step}
        fillMode="exclusive"
        className="w-full"
      />

      {status === "rejected" && org?.latest_reason ? (
        <p className="border-b border-on-surface px-[20px] py-[10px] text-[11.5px] text-signal max-sm:px-[14px]">
          Причина отклонения: {org.latest_reason}
        </p>
      ) : null}

      <div className="grid md:grid-cols-[1fr_250px]">
        {/* Form */}
        <div className="flex flex-col gap-[11px] border-b border-on-surface px-[20px] py-[16px] max-sm:px-[14px] md:border-r md:border-b-0">
          <Input
            label="Название"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Название организатора"
            className="font-bold"
          />
          <Textarea
            label="Описание"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Кратко о вас или вашей организации"
            className="min-h-[46px] text-[11.5px] leading-[1.4] text-text-dim"
            rows={2}
          />

          <div className="flex gap-[10px]">
            <LogoSlot
              previewUrl={logoPreviewUrl}
              uploading={logoUploading}
              error={logoError}
              onFile={onLogo}
            />
            <div className="flex min-w-0 flex-1 flex-col gap-[6px]">
              <span className="cap">Контакты</span>
              {email ? (
                <div
                  className="border border-ink px-[11px] py-[9px] text-[11.5px] text-text-dim"
                  title="Email аккаунта"
                >
                  {email}
                </div>
              ) : null}
              <input
                aria-label="Сайт"
                value={website}
                onChange={(e) => setWebsite(e.target.value)}
                placeholder="https://"
                className="w-full border border-ink bg-transparent px-[11px] py-[9px] text-[11.5px] text-text-dim placeholder:text-field-text swiss-focus"
              />
            </div>
          </div>

          {error ? <p className="text-[11.5px] text-signal">{error}</p> : null}

          <div className="mt-auto flex flex-col gap-[8px] pt-[8px]">
            <Button
              type="button"
              onClick={save}
              disabled={busy || !name.trim()}
              className="w-full min-h-[44px]"
            >
              СОХРАНИТЬ
            </Button>
            {canSubmit ? (
              <Button
                type="button"
                variant="ghost"
                onClick={submit}
                disabled={busy || !org}
                className="w-full min-h-[44px]"
              >
                ОТПРАВИТЬ НА ПРОВЕРКУ
              </Button>
            ) : null}
          </div>
        </div>

        {/* Public preview */}
        <aside className="flex flex-col px-[16px] py-[14px] max-sm:border-b max-sm:border-on-surface">
          <span className="cap mb-[8px]">Публичный профиль</span>
          <div className="flex flex-1 flex-col border border-ink bg-white px-[14px] py-[13px]">
            <div className="mb-[10px] flex items-center gap-[9px]">
              {logoPreviewUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={logoPreviewUrl}
                  alt=""
                  className="h-[34px] w-[34px] shrink-0 object-cover"
                />
              ) : (
                <span className="h-[34px] w-[34px] shrink-0 bg-ink" aria-hidden />
              )}
              <span className="min-w-0">
                <span className="block text-[14px] font-black tracking-[-0.02em]">
                  {previewName}
                  {status === "verified" ? " ✓" : ""}
                </span>
                <span className="cap">{previewCaption(status)}</span>
              </span>
            </div>
            {description.trim() ? (
              <p className="mb-[10px] line-clamp-3 text-[10.5px] leading-[1.45] text-text-dim">
                {description.trim()}
              </p>
            ) : (
              <p className="mb-[10px] text-[10.5px] leading-[1.45] text-muted-2">Нет описания</p>
            )}
            <div className="mt-auto flex gap-[14px] border-t border-rule-inner pt-[8px]">
              <span>
                <span className="cap block">Событий</span>
                <span className="font-mono text-[13px] font-bold">{eventsValue}</span>
              </span>
              <span>
                <span className="cap block">Подписчиков</span>
                <span className="font-mono text-[13px] font-bold">
                  {tileCount(org?.followers_count)}
                </span>
              </span>
            </div>
          </div>
        </aside>
      </div>
    </main>
  );
}
