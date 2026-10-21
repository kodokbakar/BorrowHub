import React, { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { lendingApi } from "../api/endpoints";
import type { Item, ItemAvailability } from "../api/types";
import { AlertToast } from "../components/ui/AlertToast";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Modal } from "../components/ui/Modal";

interface BookingModalProps {
  item: Item | null;
  organizationId: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function BookingModal({
  item,
  organizationId,
  isOpen,
  onClose,
  onSuccess,
}: BookingModalProps) {
  // Default: start tomorrow 09:00, end tomorrow + 2 days 17:00
  const getDefaultDates = () => {
    const now = new Date();
    const start = new Date(now.getTime() + 24 * 60 * 60 * 1000);
    start.setHours(9, 0, 0, 0);
    const end = new Date(start.getTime() + 48 * 60 * 60 * 1000);
    end.setHours(17, 0, 0, 0);

    const toLocalISO = (d: Date) => {
      const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`);
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    };

    return {
      startStr: toLocalISO(start),
      endStr: toLocalISO(end),
    };
  };

  const defaults = getDefaultDates();
  const [startTime, setStartTime] = useState(defaults.startStr);
  const [endTime, setEndTime] = useState(defaults.endStr);
  const [purpose, setPurpose] = useState("");
  const [notes, setNotes] = useState("");

  const [availability, setAvailability] = useState<ItemAvailability | null>(null);
  const [isCheckingAvail, setIsCheckingAvail] = useState(false);
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorFeedback, setErrorFeedback] = useState<{
    message: string;
    requestId?: string;
  } | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  if (!item) return null;

  const handleCheckAvailability = async () => {
    if (!startTime || !endTime) {
      setErrorFeedback({ message: "Pilih waktu mulai dan selesai terlebih dahulu." });
      return;
    }
    setErrorFeedback(null);
    setIsCheckingAvail(true);
    try {
      const sISO = new Date(startTime).toISOString();
      const eISO = new Date(endTime).toISOString();
      const avail = await lendingApi.getAvailability(
        item.id,
        organizationId,
        sISO,
        eISO,
      );
      setAvailability(avail);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Availability check fields:", err.fields);
        const fieldDetails = err.fields
          ? Object.entries(err.fields)
              .map(([k, v]) => `${k}: ${v}`)
              .join(", ")
          : "";
        const baseMessage =
          err.status === 409 || err.code === "CONFLICT"
            ? err.message || "Konflik ketersediaan unit untuk jadwal yang dipilih."
            : err.message || "Gagal memeriksa ketersediaan.";
        setErrorFeedback({
          message: fieldDetails ? `${baseMessage} (${fieldDetails})` : baseMessage,
          requestId: err.requestId,
        });
      } else {
        setErrorFeedback({ message: "Gagal memeriksa ketersediaan." });
      }
    } finally {
      setIsCheckingAvail(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorFeedback(null);
    setSuccessMsg(null);

    if (!purpose.trim()) {
      setErrorFeedback({ message: "Tujuan peminjaman wajib diisi." });
      return;
    }
    const sDate = new Date(startTime);
    const eDate = new Date(endTime);
    if (sDate >= eDate) {
      setErrorFeedback({ message: "Waktu selesai harus setelah waktu mulai." });
      return;
    }
    setIsSubmitting(true);
    try {
      await lendingApi.createLoanRequest({
        organization_id: organizationId,
        item_id: item.id,
        quantity: 1,
        start_time: sDate.toISOString(),
        end_time: eDate.toISOString(),
        purpose: purpose.trim(),
        notes: notes.trim() || undefined,
      });

      setSuccessMsg("Pengajuan peminjaman berhasil dibuat!");
      await queryClient.invalidateQueries({
        queryKey: ["loan-requests", organizationId],
      });
      await queryClient.invalidateQueries({
        queryKey: ["items", organizationId],
      });
      await queryClient.invalidateQueries({
        queryKey: ["notifications", organizationId],
      });
      setTimeout(() => {
        onSuccess?.();
        onClose();
        setSuccessMsg(null);
      }, 1200);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Loan request error fields:", err.fields);
        const fieldDetails = err.fields
          ? Object.entries(err.fields)
              .map(([k, v]) => `${k}: ${v}`)
              .join(", ")
          : "";
        const baseMessage =
          err.status === 409 || err.code === "CONFLICT"
            ? err.message || "Jadwal bentrok atau unit tidak tersedia pada rentang waktu ini."
            : err.message || "Gagal mengajukan peminjaman.";
        setErrorFeedback({
          message: fieldDetails ? `${baseMessage} (${fieldDetails})` : baseMessage,
          requestId: err.requestId,
        });
      } else {
        setErrorFeedback({ message: "Terjadi kesalahan." });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Ajukan Peminjaman: ${item.name}`}
      description="Pilih jadwal peminjaman dan isi keperluan penggunaan peralatan."
      maxWidth="lg"
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
        {successMsg && (
          <AlertToast
            type="success"
            message={successMsg}
            onClose={() => setSuccessMsg(null)}
          />
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label="Waktu Mulai"
            type="datetime-local"
            value={startTime}
            onChange={(e) => {
              setStartTime(e.target.value);
              setAvailability(null);
            }}
            required
          />
          <Input
            label="Waktu Selesai"
            type="datetime-local"
            value={endTime}
            onChange={(e) => {
              setEndTime(e.target.value);
              setAvailability(null);
            }}
            required
          />
        </div>

        {/* Live availability check section */}
        <div className="rounded-xl border border-border bg-surface-subtle p-3.5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-fg">
              Pemeriksaan Ketersediaan Live
            </span>
            <Button
              type="button"
              variant="outline"
              size="md"
              className="min-h-[44px]"
              onClick={handleCheckAvailability}
              isLoading={isCheckingAvail}
            >
              Cek Unit
            </Button>
          </div>

          {availability && (
            <div
              className={`mt-3 flex items-center justify-between p-3 rounded-lg border text-xs ${
                availability.is_available
                  ? "bg-status-available-bg text-status-available-fg border-status-available-border"
                  : "bg-status-overdue-bg text-status-overdue-fg border-status-overdue-border"
              }`}
            >
              <span className="font-medium font-tabular">
                Tersedia: <strong className="font-bold">{availability.available_units}</strong> dari{" "}
                {availability.total_units} unit
              </span>
              {availability.is_available ? (
                <Badge variant="available">Unit Tersedia</Badge>
              ) : (
                <Badge variant="overdue">Tidak Tersedia pada Jadwal Ini</Badge>
              )}
            </div>
          )}
        </div>
        <Input
          label="Tujuan Peminjaman"
          placeholder="Contoh: Liputan dokumentasi workshop tahunan"
          value={purpose}
          onChange={(e) => setPurpose(e.target.value)}
          required
        />

        <div>
          <label className="block text-sm font-medium text-fg mb-1">
            Catatan Tambahan (Opsional)
          </label>
          <textarea
            rows={2}
            className="block w-full min-h-[44px] rounded-lg border border-border-strong bg-surface p-2.5 text-sm text-fg focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent placeholder:text-fg-subtle"
            placeholder="Kebutuhan aksesoris tambahan atau catatan lokasi"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>

        <div className="flex justify-end gap-2 pt-2 border-t border-border">
          <Button
            type="button"
            variant="ghost"
            size="md"
            className="min-h-[44px]"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Batal
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="md"
            className="min-h-[44px]"
            isLoading={isSubmitting}
            disabled={availability !== null && !availability.is_available}
          >
            Kirim Permohonan
          </Button>
        </div>
      </form>
    </Modal>
  );
}
