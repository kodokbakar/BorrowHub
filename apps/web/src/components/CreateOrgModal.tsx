import React, { useState } from "react";
import { ApiError } from "../api/client";
import { identityApi } from "../api/endpoints";
import { AlertToast } from "./ui/AlertToast";
import { Button } from "./ui/Button";
import { Input } from "./ui/Input";
import { Modal } from "./ui/Modal";
import { useAuth } from "../context/AuthContext";

export interface CreateOrgModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (orgId: string) => void;
}

export function CreateOrgModal({
  isOpen,
  onClose,
  onSuccess,
}: CreateOrgModalProps) {
  const { switchOrg, refreshProfile } = useAuth();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugModified, setSlugModified] = useState(false);
  const [maxLoanDays, setMaxLoanDays] = useState(14);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorFeedback, setErrorFeedback] = useState<{
    message: string;
    requestId?: string;
  } | null>(null);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setName(val);
    if (!slugModified) {
      const generatedSlug = val
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "");
      setSlug(generatedSlug);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorFeedback(null);

    const trimmedName = name.trim();
    const trimmedSlug = slug.trim();

    if (!trimmedName || !trimmedSlug) {
      setErrorFeedback({ message: "Nama dan slug organisasi wajib diisi." });
      return;
    }

    setIsSubmitting(true);
    try {
      const newOrg = await identityApi.createOrganization({
        name: trimmedName,
        slug: trimmedSlug,
        max_loan_duration_days: Number(maxLoanDays) || 14,
      });

      switchOrg(newOrg.id);
      await refreshProfile();

      setName("");
      setSlug("");
      setSlugModified(false);
      onSuccess?.(newOrg.id);
      onClose();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) {
          console.error("Create org validation error fields:", err.fields);
        }
        setErrorFeedback({
          message: err.message || "Gagal membuat organisasi.",
          requestId: err.requestId,
        });
      } else {
        setErrorFeedback({ message: "Terjadi kesalahan saat membuat organisasi." });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Buat Organisasi Baru"
      description="Buat workspace organisasi baru untuk mengelola inventaris dan peminjaman peralatan."
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {errorFeedback && (
          <AlertToast
            type="error"
            message={errorFeedback.message}
            requestId={errorFeedback.requestId}
            onClose={() => setErrorFeedback(null)}
          />
        )}

        <Input
          label="Nama Organisasi"
          value={name}
          onChange={handleNameChange}
          placeholder="Contoh: Lab Robotika, Departemen IT"
          required
        />

        <Input
          label="Slug Organisasi (URL Friendly)"
          value={slug}
          onChange={(e) => {
            setSlugModified(true);
            setSlug(e.target.value);
          }}
          placeholder="contoh: lab-robotika"
          helperText="Digunakan untuk identifikasi unik organisasi Anda."
          required
        />

        <Input
          label="Maksimal Durasi Peminjaman (Hari)"
          type="number"
          min={1}
          max={90}
          value={maxLoanDays}
          onChange={(e) => setMaxLoanDays(parseInt(e.target.value, 10) || 14)}
          helperText="Batas maksimal peminjaman standar untuk anggota."
        />

        <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Batal
          </Button>
          <Button type="submit" variant="primary" isLoading={isSubmitting}>
            Buat Organisasi
          </Button>
        </div>
      </form>
    </Modal>
  );
}
