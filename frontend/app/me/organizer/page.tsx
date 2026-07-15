"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ImageUpload } from "@/components/ui/ImageUpload";
import { useAuth } from "@/lib/auth-context";
import {
  getMyOrganizer,
  saveMyOrganizer,
  submitMyOrganizer,
  uploadFile,
  type Organizer,
  type VerificationStatus,
} from "@/lib/api";

const STATUS_LABEL: Record<VerificationStatus, string> = {
  draft: "Черновик",
  pending: "На проверке",
  verified: "Подтверждён",
  rejected: "Отклонён",
};

const STATUS_CHIP_CLASS: Record<VerificationStatus, string> = {
  draft: "bg-fill text-label-secondary",
  pending: "bg-accent/15 text-accent",
  verified: "bg-green-500/15 text-green-700 dark:text-green-400",
  rejected: "bg-red-500/15 text-red-600 dark:text-red-400",
};

const inputCls =
  "w-full rounded-control bg-fill px-3.5 py-2.5 text-[17px] text-label outline-none placeholder:text-label-secondary focus:ring-2 focus:ring-accent";

export default function MyOrganizerPage() {
  const { ready, isAuthed } = useAuth();
  const [org, setOrg] = useState<Organizer | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [website, setWebsite] = useState("");
  const [logoFileId, setLogoFileId] = useState<string | undefined>();
  const [logoPreviewUrl, setLogoPreviewUrl] = useState<string | undefined>();
  const [logoUploading, setLogoUploading] = useState(false);
  const [logoError, setLogoError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!ready || !isAuthed) return;
    getMyOrganizer()
      .then((o) => {
        if (o) {
          setOrg(o);
          setName(o.name);
          setDescription(o.description);
          setWebsite(o.website_url);
          setLogoPreviewUrl(o.logo_url);
        }
      })
      .catch((e) => setError(String(e)));
  }, [ready, isAuthed]);

  if (!ready) {
    return <div className="min-h-screen bg-bg-grouped" />;
  }

  if (!isAuthed) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16">
        <Link href="/" className="inline-flex items-center text-[17px] text-accent">
          ‹ События
        </Link>
        <div className="mt-8 text-center">
          <h1 className="text-[28px] font-bold tracking-[-0.022em]">Профиль организатора</h1>
          <p className="mt-3 text-label-secondary">
            Войдите, чтобы создать профиль организатора.
          </p>
        </div>
      </main>
    );
  }

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
      const { status } = await submitMyOrganizer();
      setOrg((prev) => (prev ? { ...prev, verification_status: status } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const onLogo = async (file: File) => {
    setLogoError(undefined);
    setLogoUploading(true);
    try {
      const { id, url } = await uploadFile(file);
      setLogoFileId(id);
      setLogoPreviewUrl(url);
    } catch (e) {
      setLogoError(e instanceof Error ? e.message : String(e));
    } finally {
      setLogoUploading(false);
    }
  };

  const canSubmit =
    org &&
    (org.verification_status === "draft" || org.verification_status === "rejected");

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 max-sm:pb-28">
      <Link href="/" className="mb-4 inline-flex items-center text-[17px] text-accent">
        ‹ События
      </Link>

      <header className="flex items-center justify-between">
        <h1 className="text-[28px] font-bold tracking-[-0.022em]">Профиль организатора</h1>
        {org && (
          <span
            className={`inline-block rounded-full px-3 py-1 text-[13px] font-medium ${STATUS_CHIP_CLASS[org.verification_status]}`}
          >
            {STATUS_LABEL[org.verification_status]}
          </span>
        )}
      </header>

      {org?.verification_status === "rejected" && org.latest_reason && (
        <div className="mt-4 rounded-card bg-red-500/10 px-4 py-3 text-[15px] text-red-600 dark:text-red-400">
          Причина отклонения: {org.latest_reason}
        </div>
      )}
      {org?.verification_status === "verified" && (
        <div className="mt-4 rounded-card bg-green-500/10 px-4 py-3 text-[15px] text-green-700 dark:text-green-400">
          Ваш профиль подтверждён. На ваших событиях отображается значок ✓.
        </div>
      )}

      <div className="mt-6 space-y-4">
        <label className="block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Название *</span>
          <input
            className={inputCls}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Название организатора"
          />
        </label>
        <label className="block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Описание</span>
          <textarea
            className={`${inputCls} min-h-[96px] resize-y`}
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Кратко о вас или вашей организации"
          />
        </label>
        <label className="block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Сайт</span>
          <input
            className={inputCls}
            value={website}
            onChange={(e) => setWebsite(e.target.value)}
            placeholder="https://"
          />
        </label>
        <label className="block">
          <span className="mb-1.5 block text-[13px] text-label-secondary">Логотип</span>
          <ImageUpload
            label="логотип"
            previewUrl={logoPreviewUrl}
            uploading={logoUploading}
            error={logoError}
            onFile={onLogo}
          />
        </label>
      </div>

      {error && <p className="mt-4 text-[15px] text-red-600">{error}</p>}

      <div className="mt-6 flex gap-3">
        <button
          onClick={save}
          disabled={busy || !name.trim()}
          className="rounded-capsule bg-accent px-5 py-2.5 text-[17px] font-semibold text-white disabled:opacity-50"
        >
          Сохранить
        </button>
        {canSubmit && (
          <button
            onClick={submit}
            disabled={busy}
            className="rounded-capsule border border-separator px-5 py-2.5 text-[17px] font-semibold text-accent disabled:opacity-50"
          >
            Отправить на проверку
          </button>
        )}
      </div>
    </main>
  );
}
