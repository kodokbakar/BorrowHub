import React, { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { inventoryApi, lendingApi } from "../api/endpoints";
import type {
  ApprovedRequestResult,
  Item,
  Loan,
  LoanRequest,
  ReturnCondition,
} from "../api/types";
import { AlertToast } from "../components/ui/AlertToast";
import {
  ConditionBadge,
  LoanStatusBadge,
  RequestStatusBadge,
} from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { Input } from "../components/ui/Input";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Modal } from "../components/ui/Modal";
import { Select } from "../components/ui/Select";
import { useAuth } from "../context/AuthContext";

export function AdminLendingView() {
  const { activeOrg } = useAuth();
  const orgId = activeOrg?.id;
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState<
    "pending" | "checkout" | "active" | "overdue"
  >("pending");

  const [feedback, setFeedback] = useState<{
    type: "success" | "error";
    message: string;
    requestId?: string;
  } | null>(null);

  // Fetch items to map names
  const { data: items = [] } = useQuery({
    queryKey: ["items", orgId],
    queryFn: () => (orgId ? inventoryApi.listItems(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  const itemMap = React.useMemo(() => {
    const map = new Map<string, Item>();
    items.forEach((i) => map.set(i.id, i));
    return map;
  }, [items]);

  // Query pending requests
  const {
    data: pendingRequests = [],
    isLoading: isLoadingPending,
    refetch: refetchPending,
  } = useQuery({
    queryKey: ["admin-pending-requests", orgId],
    queryFn: () =>
      orgId
        ? lendingApi.listLoanRequests(orgId, "pending")
        : Promise.resolve([]),
    enabled: !!orgId,
  });

  // Query all loans
  const {
    data: allLoans = [],
    isLoading: isLoadingLoans,
    refetch: refetchLoans,
  } = useQuery({
    queryKey: ["admin-loans", orgId],
    queryFn: () => (orgId ? lendingApi.listLoans(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  // Query overdue loans
  const {
    data: overdueLoans = [],
    isLoading: isLoadingOverdue,
    refetch: refetchOverdue,
  } = useQuery({
    queryKey: ["admin-overdue", orgId],
    queryFn: () =>
      orgId ? lendingApi.listOverdue(orgId) : Promise.resolve([]),
    enabled: !!orgId,
  });

  // Pending checkout loans
  const pendingCheckoutLoans = React.useMemo(() => {
    return allLoans.filter((l) => l.status === "pending_checkout");
  }, [allLoans]);

  // Currently borrowed loans
  const checkedOutLoans = React.useMemo(() => {
    return allLoans.filter(
      (l) => l.status === "checked_out" || l.status === "overdue",
    );
  }, [allLoans]);

  // ---------------- APPROVAL & REJECTION STATE ----------------
  const [approvingRequestId, setApprovingRequestId] = useState<string | null>(
    null,
  );
  const [approvalResult, setApprovalResult] =
    useState<ApprovedRequestResult | null>(null);

  const [rejectingRequest, setRejectingRequest] = useState<LoanRequest | null>(
    null,
  );
  const [rejectNotes, setRejectNotes] = useState("");
  const [isSubmittingReject, setIsSubmittingReject] = useState(false);

  // ---------------- CHECKOUT STATE ----------------
  const [checkoutModalLoan, setCheckoutModalLoan] = useState<Loan | null>(null);
  const [checkoutTokenInput, setCheckoutTokenInput] = useState("");
  const [isSubmittingCheckout, setIsSubmittingCheckout] = useState(false);

  // ---------------- RETURN STATE ----------------
  const [returnModalLoan, setReturnModalLoan] = useState<Loan | null>(null);
  const [returnCondition, setReturnCondition] =
    useState<ReturnCondition>("good");
  const [returnNotes, setReturnNotes] = useState("");
  const [isSubmittingReturn, setIsSubmittingReturn] = useState(false);

  if (!orgId) {
    return (
      <div className="py-12">
        <EmptyState
          title="Tidak ada organisasi aktif"
          description="Pilih organisasi terlebih dahulu untuk mengelola alur peminjaman."
        />
      </div>
    );
  }

  const handleApprove = async (requestId: string) => {
    setApprovingRequestId(requestId);
    setFeedback(null);
    try {
      const result = await lendingApi.approveLoanRequest(
        requestId,
        "Disetujui oleh admin operasional",
      );
      setApprovalResult(result);
      setFeedback({
        type: "success",
        message:
          "Permohonan disetujui! Unit fisik dialokasikan dan token pengambilan dibuat.",
      });
      await queryClient.invalidateQueries({ queryKey: ["loan-requests", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["loans", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["notifications", orgId] });
      void refetchPending();
      void refetchLoans();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Approve error fields:", err.fields);
        setFeedback({
          type: "error",
          message:
            err.message ||
            "Gagal menyetujui permohonan. Periksa ketersediaan unit fisik.",
          requestId: err.requestId,
        });
      } else {
        setFeedback({
          type: "error",
          message: "Terjadi kesalahan saat menyetujui permohonan.",
        });
      }
    } finally {
      setApprovingRequestId(null);
    }
  };

  const handleConfirmReject = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rejectingRequest) return;

    setIsSubmittingReject(true);
    setFeedback(null);
    try {
      await lendingApi.rejectLoanRequest(
        rejectingRequest.id,
        rejectNotes.trim() || undefined,
      );
      setFeedback({
        type: "success",
        message: "Permohonan berhasil ditolak.",
      });
      setRejectingRequest(null);
      setRejectNotes("");
      await queryClient.invalidateQueries({ queryKey: ["loan-requests", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["notifications", orgId] });
      void refetchPending();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Reject error fields:", err.fields);
        setFeedback({
          type: "error",
          message: err.message || "Gagal menolak permohonan peminjaman.",
          requestId: err.requestId,
        });
      } else {
        setFeedback({
          type: "error",
          message: "Gagal menolak permohonan peminjaman.",
        });
      }
    } finally {
      setIsSubmittingReject(false);
    }
  };

  const handleConfirmCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!checkoutModalLoan || !checkoutTokenInput.trim()) return;

    setIsSubmittingCheckout(true);
    setFeedback(null);
    try {
      await lendingApi.checkoutLoan(
        checkoutModalLoan.id,
        checkoutTokenInput.trim(),
      );
      setFeedback({
        type: "success",
        message: `Peralatan berhasil diserahkan (Check-out) untuk pinjaman #${checkoutModalLoan.id.slice(0, 8)}!`,
      });
      setCheckoutModalLoan(null);
      setCheckoutTokenInput("");
      await queryClient.invalidateQueries({ queryKey: ["loans", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["notifications", orgId] });
      void refetchLoans();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Checkout error fields:", err.fields);
        setFeedback({
          type: "error",
          message:
            err.message ||
            "Gagal check-out. Pastikan token verifikasi valid dan belum kadaluarsa.",
          requestId: err.requestId,
        });
      } else {
        setFeedback({
          type: "error",
          message: "Terjadi kesalahan saat proses check-out.",
        });
      }
    } finally {
      setIsSubmittingCheckout(false);
    }
  };

  const handleConfirmReturn = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!returnModalLoan) return;

    setIsSubmittingReturn(true);
    setFeedback(null);
    try {
      await lendingApi.returnLoan(
        returnModalLoan.id,
        returnCondition,
        returnNotes.trim() || undefined,
      );
      setFeedback({
        type: "success",
        message: `Pengembalian pinjaman #${returnModalLoan.id.slice(0, 8)} berhasil dicatat!`,
      });
      setReturnModalLoan(null);
      setReturnNotes("");
      await queryClient.invalidateQueries({ queryKey: ["loans", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      await queryClient.invalidateQueries({ queryKey: ["notifications", orgId] });
      void refetchLoans();
      void refetchOverdue();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) console.error("Return error fields:", err.fields);
        setFeedback({
          type: "error",
          message: err.message || "Gagal memproses pengembalian.",
          requestId: err.requestId,
        });
      } else {
        setFeedback({
          type: "error",
          message: "Terjadi kesalahan saat mencatat pengembalian.",
        });
      }
    } finally {
      setIsSubmittingReturn(false);
    }
  };
  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleString("id-ID", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Title */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          Alur Peminjaman Admin
        </h1>
        <p className="mt-1 text-sm text-slate-500">
          Tinjau permohonan pinjam, validasi token check-out fisik, inspeksi
          kondisi pengembalian, dan pantau keterlambatan.
        </p>
      </div>

      {feedback && (
        <AlertToast
          type={feedback.type}
          message={feedback.message}
          requestId={feedback.requestId}
          onClose={() => setFeedback(null)}
        />
      )}

      {/* Tabs */}
      <div className="flex border-b border-slate-200">
        <button
          type="button"
          onClick={() => setActiveTab("pending")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "pending"
              ? "border-slate-900 text-slate-900"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Menunggu Persetujuan ({pendingRequests.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("checkout")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "checkout"
              ? "border-slate-900 text-slate-900"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Check-Out / Penyerahan ({pendingCheckoutLoans.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("active")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "active"
              ? "border-slate-900 text-slate-900"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Sedang Dipinjam & Kembali ({checkedOutLoans.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("overdue")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "overdue"
              ? "border-rose-600 text-rose-600"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Terlambat ({overdueLoans.length})
        </button>
      </div>

      {/* PENDING APPROVAL TAB */}
      {activeTab === "pending" && (
        <div>
          {isLoadingPending ? (
            <LoadingSpinner />
          ) : pendingRequests.length === 0 ? (
            <EmptyState
              title="Semua permohonan telah ditinjau"
              description="Tidak ada permohonan peminjaman baru yang menunggu persetujuan."
            />
          ) : (
            <div className="space-y-4">
              {pendingRequests.map((req) => {
                const item = itemMap.get(req.item_id);
                const isApproving = approvingRequestId === req.id;
                return (
                  <div
                    key={req.id}
                    className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border border-slate-200 bg-white p-5 shadow-xs"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-slate-900">
                          {item?.name || `Item: ${req.item_id.slice(0, 8)}`}
                        </h3>
                        <RequestStatusBadge status={req.status} />
                      </div>
                      <p className="text-xs text-slate-500">
                        Peminjam ID:{" "}
                        <span className="font-mono">
                          {req.requester_user_id.slice(0, 8)}...
                        </span>{" "}
                        | Jumlah: {req.quantity} unit
                      </p>
                      <p className="text-xs text-slate-600">
                        Jadwal: {formatDate(req.start_time)} s/d{" "}
                        {formatDate(req.end_time)}
                      </p>
                      <p className="text-xs text-slate-700 bg-slate-50 p-2 rounded-md">
                        <strong>Tujuan:</strong> {req.purpose}
                      </p>
                      {req.notes && (
                        <p className="text-xs text-slate-500 italic">
                          Catatan: {req.notes}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => {
                          setRejectingRequest(req);
                          setRejectNotes("");
                        }}
                      >
                        Tolak
                      </Button>
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => handleApprove(req.id)}
                        isLoading={isApproving}
                      >
                        Setujui Permohonan
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* CHECK-OUT TAB */}
      {activeTab === "checkout" && (
        <div>
          {isLoadingLoans ? (
            <LoadingSpinner />
          ) : pendingCheckoutLoans.length === 0 ? (
            <EmptyState
              title="Tidak ada antrean check-out"
              description="Belum ada pinjaman yang disetujui dan menunggu serah terima unit."
            />
          ) : (
            <div className="space-y-4">
              {pendingCheckoutLoans.map((loan) => (
                <div
                  key={loan.id}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border border-slate-200 bg-white p-5 shadow-xs"
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-slate-900">
                        Pinjaman #{loan.id.slice(0, 8)}
                      </h3>
                      <LoanStatusBadge status={loan.status} />
                    </div>
                    <p className="text-xs text-slate-500">
                      Unit ID:{" "}
                      <span className="font-mono text-slate-700 font-semibold">
                        {loan.unit_id.slice(0, 8)}...
                      </span>{" "}
                      | Peminjam ID: {loan.borrower_user_id.slice(0, 8)}...
                    </p>
                    <p className="text-xs text-slate-600">
                      Rencana Kembali: {formatDate(loan.expected_return_time)}
                    </p>
                  </div>

                  <div>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => {
                        setCheckoutModalLoan(loan);
                        setCheckoutTokenInput("");
                      }}
                    >
                      Verifikasi & Serahkan Unit
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ACTIVE & RETURN TAB */}
      {activeTab === "active" && (
        <div>
          {isLoadingLoans ? (
            <LoadingSpinner />
          ) : checkedOutLoans.length === 0 ? (
            <EmptyState
              title="Tidak ada barang dipinjam"
              description="Tidak ada unit peralatan yang saat ini berada di tangan peminjam."
            />
          ) : (
            <div className="space-y-4">
              {checkedOutLoans.map((loan) => {
                const isOverdue = loan.status === "overdue";
                return (
                  <div
                    key={loan.id}
                    className={`flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border p-5 shadow-xs bg-white ${
                      isOverdue
                        ? "border-rose-300 ring-1 ring-rose-200"
                        : "border-slate-200"
                    }`}
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-slate-900">
                          Pinjaman #{loan.id.slice(0, 8)}
                        </h3>
                        <LoanStatusBadge status={loan.status} />
                      </div>
                      <p className="text-xs text-slate-500">
                        Unit ID:{" "}
                        <span className="font-mono text-slate-700">
                          {loan.unit_id.slice(0, 8)}...
                        </span>{" "}
                        | Peminjam ID: {loan.borrower_user_id.slice(0, 8)}...
                      </p>
                      <p className="text-xs text-slate-600">
                        Batas Pengembalian:{" "}
                        <strong
                          className={
                            isOverdue ? "text-rose-600" : "text-slate-800"
                          }
                        >
                          {formatDate(loan.expected_return_time)}
                        </strong>
                      </p>
                      {loan.checkout_time && (
                        <p className="text-[11px] text-slate-400">
                          Check-out pada: {formatDate(loan.checkout_time)}
                        </p>
                      )}
                    </div>

                    <div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setReturnModalLoan(loan);
                          setReturnCondition("good");
                          setReturnNotes("");
                        }}
                      >
                        Proses Pengembalian
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* OVERDUE TAB */}
      {activeTab === "overdue" && (
        <div>
          {isLoadingOverdue ? (
            <LoadingSpinner />
          ) : overdueLoans.length === 0 ? (
            <EmptyState
              title="Tidak ada keterlambatan"
              description="Semua peralatan dipinjam tepat waktu dan belum ada yang melewati batas pengembalian."
            />
          ) : (
            <div className="space-y-4">
              {overdueLoans.map((loan) => (
                <div
                  key={loan.id}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border border-rose-300 bg-rose-50/30 p-5 shadow-xs"
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-slate-900">
                        Pinjaman #{loan.id.slice(0, 8)}
                      </h3>
                      <LoanStatusBadge status="overdue" />
                    </div>
                    <p className="text-xs text-slate-500">
                      Unit ID:{" "}
                      <span className="font-mono text-slate-800 font-semibold">
                        {loan.unit_id.slice(0, 8)}...
                      </span>{" "}
                      | Peminjam ID: {loan.borrower_user_id.slice(0, 8)}...
                    </p>
                    <p className="text-xs text-rose-700 font-medium">
                      Jatuh Tempo: {formatDate(loan.expected_return_time)}
                    </p>
                  </div>

                  <div>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => {
                        setReturnModalLoan(loan);
                        setReturnCondition("good");
                        setReturnNotes("");
                      }}
                    >
                      Proses Pengembalian
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* MODAL: Approved Request Result (Displays Checkout Token) */}
      <Modal
        isOpen={!!approvalResult}
        onClose={() => setApprovalResult(null)}
        title="Permohonan Berhasil Disetujui"
        description="Berikan kode token sekali-pakai ini kepada peminjam saat mengambil barang."
      >
        <div className="space-y-4">
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-center">
            <span className="text-xs font-semibold text-emerald-800 uppercase tracking-wider">
              Token Pengambilan (Check-out)
            </span>
            <div className="mt-2 font-mono text-2xl font-bold tracking-wider text-slate-900 bg-white py-2 px-4 rounded-lg border border-emerald-300 select-all">
              {approvalResult?.checkout_token}
            </div>
            <p className="mt-2 text-xs text-emerald-700">
              Token ini hanya dapat digunakan satu kali dan akan divalidasi saat
              penyerahan unit fisik.
            </p>
          </div>

          <div className="text-xs text-slate-600 space-y-1">
            <p>
              <strong>Pinjaman ID:</strong> {approvalResult?.loan.id}
            </p>
            <p>
              <strong>Unit Fisik Terpilih:</strong>{" "}
              {approvalResult?.loan.unit_id}
            </p>
            <p>
              <strong>Batas Kembali:</strong>{" "}
              {approvalResult &&
                formatDate(approvalResult.loan.expected_return_time)}
            </p>
          </div>

          <div className="flex justify-end pt-2 border-t border-slate-100">
            <Button
              variant="primary"
              onClick={() => setApprovalResult(null)}
            >
              Selesai & Tutup
            </Button>
          </div>
        </div>
      </Modal>

      {/* MODAL: Reject Request */}
      <Modal
        isOpen={!!rejectingRequest}
        onClose={() => setRejectingRequest(null)}
        title="Tolak Permohonan Peminjaman"
      >
        <form onSubmit={handleConfirmReject} className="space-y-4">
          <Input
            label="Alasan Penolakan"
            placeholder="Contoh: Unit sedang dijadwalkan untuk perawatan mendesak"
            value={rejectNotes}
            onChange={(e) => setRejectNotes(e.target.value)}
            required
          />
          <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setRejectingRequest(null)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="danger"
              isLoading={isSubmittingReject}
            >
              Tolak Permohonan
            </Button>
          </div>
        </form>
      </Modal>

      {/* MODAL: Checkout / Serah Terima */}
      <Modal
        isOpen={!!checkoutModalLoan}
        onClose={() => setCheckoutModalLoan(null)}
        title={`Check-out Pinjaman #${checkoutModalLoan?.id.slice(0, 8)}`}
        description="Masukkan token yang ditunjukkan peminjam untuk memvalidasi serah terima."
      >
        <form onSubmit={handleConfirmCheckout} className="space-y-4">
          <Input
            label="Kode Token Sekali Pakai (Checkout Token)"
            placeholder="Masukkan token dari peminjam"
            value={checkoutTokenInput}
            onChange={(e) => setCheckoutTokenInput(e.target.value)}
            className="font-mono text-base"
            required
          />
          <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setCheckoutModalLoan(null)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              isLoading={isSubmittingCheckout}
            >
              Konfirmasi Serah Terima
            </Button>
          </div>
        </form>
      </Modal>

      {/* MODAL: Return / Pengembalian */}
      <Modal
        isOpen={!!returnModalLoan}
        onClose={() => setReturnModalLoan(null)}
        title={`Proses Pengembalian Pinjaman #${returnModalLoan?.id.slice(0, 8)}`}
        description="Inspeksi fisik unit peralatan dan catat kondisi pengembalian."
      >
        <form onSubmit={handleConfirmReturn} className="space-y-4">
          <Select
            label="Kondisi Fisik Saat Kembali"
            value={returnCondition}
            onChange={(e) =>
              setReturnCondition(e.target.value as ReturnCondition)
            }
            options={[
              { value: "new", label: "Baru (New) - Sangat Bersih" },
              { value: "good", label: "Baik (Good) - Berfungsi Normal" },
              { value: "fair", label: "Cukup (Fair) - Terdapat Tanda Pakai" },
              { value: "poor", label: "Kurang (Poor) - Perlu Pembersihan" },
              { value: "damaged", label: "Rusak (Damaged) - Memerlukan Perbaikan" },
            ]}
          />
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Catatan Pengembalian (Opsional)
            </label>
            <textarea
              rows={2}
              className="block w-full rounded-lg border border-slate-300 bg-white p-2.5 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-slate-900"
              placeholder="Catatan kebersihan, kelengkapan kabel, baut, dll"
              value={returnNotes}
              onChange={(e) => setReturnNotes(e.target.value)}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setReturnModalLoan(null)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              isLoading={isSubmittingReturn}
            >
              Catat Pengembalian Selesai
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
