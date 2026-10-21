import React, { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { inventoryApi, lendingApi } from "../api/endpoints";
import type { Item, Loan, LoanRequest } from "../api/types";
import { AlertToast } from "../components/ui/AlertToast";
import {
  ConditionBadge,
  LoanStatusBadge,
  RequestStatusBadge,
} from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useAuth } from "../context/AuthContext";

export function MyLoansView() {
  const { user, activeOrg } = useAuth();
  const orgId = activeOrg?.id;
  const [subTab, setSubTab] = useState<"requests" | "loans">("requests");
  const queryClient = useQueryClient();
  const [cancellingId, setCancellingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<{
    message: string;
    requestId?: string;
  } | null>(null);

  // Fetch all items to map item names
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

  // Fetch loan requests
  const {
    data: allRequests = [],
    isLoading: isLoadingRequests,
    refetch: refetchRequests,
  } = useQuery({
    queryKey: ["loan-requests", orgId],
    queryFn: () =>
      orgId ? lendingApi.listLoanRequests(orgId) : Promise.resolve([]),
    enabled: !!orgId,
  });

  // Filter requests belonging to this user
  const myRequests = React.useMemo(() => {
    if (!user) return [];
    return allRequests.filter((r) => r.requester_user_id === user.id);
  }, [allRequests, user]);

  // Fetch loans
  const {
    data: allLoans = [],
    isLoading: isLoadingLoans,
    refetch: refetchLoans,
  } = useQuery({
    queryKey: ["loans", orgId],
    queryFn: () => (orgId ? lendingApi.listLoans(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  // Filter loans belonging to this user
  const myLoans = React.useMemo(() => {
    if (!user) return [];
    return allLoans.filter((l) => l.borrower_user_id === user.id);
  }, [allLoans, user]);

  const handleCancelRequest = async (requestId: string) => {
    setActionError(null);
    setCancellingId(requestId);
    try {
      await lendingApi.cancelLoanRequest(requestId, "Dibatalkan oleh peminjam");
      if (orgId) {
        await queryClient.invalidateQueries({ queryKey: ["loan-requests", orgId] });
        await queryClient.invalidateQueries({ queryKey: ["notifications", orgId] });
      }
      await refetchRequests();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.fields) {
          console.error("Cancel loan request error fields:", err.fields);
        }
        const fieldDetails = err.fields
          ? Object.entries(err.fields)
              .map(([k, v]) => `${k}: ${v}`)
              .join(", ")
          : "";
        setActionError({
          message: err.message
            ? fieldDetails
              ? `${err.message} (${fieldDetails})`
              : err.message
            : "Gagal membatalkan permohonan peminjaman.",
          requestId: err.requestId,
        });
      } else {
        setActionError({
          message: "Gagal membatalkan permohonan peminjaman.",
        });
      }
    } finally {
      setCancellingId(null);
    }
  };

  const formatDate = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString("id-ID", {
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

  if (!orgId) {
    return (
      <div className="py-12">
        <EmptyState
          title="Tidak ada organisasi aktif"
          description="Pilih organisasi terlebih dahulu untuk melihat riwayat peminjaman."
        />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">
          Peminjaman Saya
        </h1>
        <p className="mt-1 text-sm text-slate-500">
          Pantau status permohonan pinjam dan barang yang sedang Anda gunakan di{" "}
          <span className="font-semibold text-slate-800">{activeOrg.name}</span>.
        </p>
      </div>

      {actionError && (
        <AlertToast
          type="error"
          message={actionError.message}
          requestId={actionError.requestId}
          onClose={() => setActionError(null)}
        />
      )}

      {/* Sub Tabs */}
      <div className="flex border-b border-slate-200">
        <button
          type="button"
          onClick={() => setSubTab("requests")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            subTab === "requests"
              ? "border-slate-900 text-slate-900"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Permohonan Peminjaman ({myRequests.length})
        </button>
        <button
          type="button"
          onClick={() => setSubTab("loans")}
          className={`py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            subTab === "loans"
              ? "border-slate-900 text-slate-900"
              : "border-transparent text-slate-500 hover:text-slate-700"
          }`}
        >
          Peminjaman Aktif & Selesai ({myLoans.length})
        </button>
      </div>

      {/* Sub Tab: Requests */}
      {subTab === "requests" && (
        <div>
          {isLoadingRequests ? (
            <LoadingSpinner />
          ) : myRequests.length === 0 ? (
            <EmptyState
              title="Belum ada permohonan"
              description="Anda belum memiliki riwayat pengajuan peminjaman peralatan."
            />
          ) : (
            <div className="space-y-4">
              {myRequests.map((req) => {
                const item = itemMap.get(req.item_id);
                return (
                  <div
                    key={req.id}
                    className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border border-slate-200 bg-white p-5 shadow-xs"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-slate-900">
                          {item?.name || `Item ID: ${req.item_id.slice(0, 8)}`}
                        </h3>
                        <RequestStatusBadge status={req.status} />
                      </div>
                      <p className="text-xs text-slate-500">
                        Jadwal: {formatDate(req.start_time)} s/d{" "}
                        {formatDate(req.end_time)}
                      </p>
                      <p className="text-xs text-slate-600">
                        <strong>Keperluan:</strong> {req.purpose}
                      </p>
                      {req.review_notes && (
                        <p className="text-xs text-amber-700 bg-amber-50 p-2 rounded-md mt-2">
                          <strong>Catatan Peninjau:</strong> {req.review_notes}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      {req.status === "pending" && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleCancelRequest(req.id)}
                          isLoading={cancellingId === req.id}
                        >
                          Batalkan
                        </Button>
                      )}
                      {req.status === "approved" && (
                        <span className="text-xs text-emerald-600 font-medium">
                          Telah disetujui, siap diambil di Admin
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Sub Tab: Loans */}
      {subTab === "loans" && (
        <div>
          {isLoadingLoans ? (
            <LoadingSpinner />
          ) : myLoans.length === 0 ? (
            <EmptyState
              title="Tidak ada data peminjaman"
              description="Anda belum memiliki rekaman peminjaman unit fisik peralatan."
            />
          ) : (
            <div className="space-y-4">
              {myLoans.map((loan) => {
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
                    <div className="space-y-1.5">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-slate-900">
                          Pinjaman #{loan.id.slice(0, 8)}
                        </span>
                        <LoanStatusBadge status={loan.status} />
                      </div>

                      <div className="text-xs text-slate-600 space-y-0.5">
                        <p>
                          <strong>Batas Pengembalian:</strong>{" "}
                          <span
                            className={
                              isOverdue
                                ? "font-bold text-rose-600"
                                : "text-slate-800"
                            }
                          >
                            {formatDate(loan.expected_return_time)}
                          </span>
                        </p>
                        {loan.checkout_time && (
                          <p className="text-slate-500">
                            Waktu Ambil: {formatDate(loan.checkout_time)}
                          </p>
                        )}
                        {loan.actual_return_time && (
                          <p className="text-slate-500">
                            Waktu Kembali: {formatDate(loan.actual_return_time)}
                          </p>
                        )}
                        {loan.return_condition && (
                          <div className="flex items-center gap-1.5 pt-1">
                            <span className="text-slate-500">
                              Kondisi Pengembalian:
                            </span>
                            <ConditionBadge condition={loan.return_condition} />
                          </div>
                        )}
                      </div>
                    </div>

                    <div>
                      {loan.status === "pending_checkout" && (
                        <div className="rounded-lg bg-amber-50 p-3 text-xs text-amber-800 max-w-xs">
                          Tunjukkan kode token yang diberikan admin untuk
                          mengambil unit fisik.
                        </div>
                      )}
                      {isOverdue && (
                        <div className="rounded-lg bg-rose-50 p-3 text-xs text-rose-800 font-medium">
                          Peralatan sudah melewati batas waktu kembali. Segera
                          serahkan ke pengelola.
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
