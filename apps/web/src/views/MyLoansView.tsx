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

export interface MyLoansViewProps {
  initialTab?: "requests" | "loans";
}

export function MyLoansView({
  initialTab = "requests",
}: MyLoansViewProps = {}) {
  const { user, activeOrg } = useAuth();
  const orgId = activeOrg?.id;
  const [subTab, setSubTab] = useState<"requests" | "loans">(initialTab);
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
        <h1 className="text-2xl font-bold tracking-tight text-fg">
          Peminjaman Saya
        </h1>
        <p className="mt-1 text-sm text-fg-muted">
          Pantau status permohonan pinjam dan barang yang sedang Anda gunakan di{" "}
          <span className="font-semibold text-fg">{activeOrg.name}</span>.
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
      <div className="flex border-b border-border gap-2" role="tablist">
        <button
          id="tab-requests"
          type="button"
          role="tab"
          aria-selected={subTab === "requests"}
          aria-controls="panel-requests"
          onClick={() => setSubTab("requests")}
          className={`min-h-[44px] inline-flex items-center px-4 py-2 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            subTab === "requests"
              ? "border-accent text-accent font-semibold"
              : "border-transparent text-fg-muted hover:text-fg hover:border-border-strong"
          }`}
        >
          Permohonan Peminjaman ({myRequests.length})
        </button>
        <button
          id="tab-loans"
          type="button"
          role="tab"
          aria-selected={subTab === "loans"}
          aria-controls="panel-loans"
          onClick={() => setSubTab("loans")}
          className={`min-h-[44px] inline-flex items-center px-4 py-2 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            subTab === "loans"
              ? "border-accent text-accent font-semibold"
              : "border-transparent text-fg-muted hover:text-fg hover:border-border-strong"
          }`}
        >
          Peminjaman Aktif & Selesai ({myLoans.length})
        </button>
      </div>

      {/* Sub Tab: Requests */}
      {subTab === "requests" && (
        <div role="tabpanel" id="panel-requests" aria-labelledby="tab-requests">
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
                    className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border border-border bg-surface p-5 shadow-xs"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-fg">
                          {item?.name || `Item ID: ${req.item_id.slice(0, 8)}`}
                        </h3>
                        <RequestStatusBadge status={req.status} />
                      </div>
                      <p className="text-xs text-fg-muted">
                        Jadwal: <span className="font-tabular">{formatDate(req.start_time)}</span> s/d{" "}
                        <span className="font-tabular">{formatDate(req.end_time)}</span>
                      </p>
                      <p className="text-xs text-fg-muted">
                        <strong className="text-fg">Keperluan:</strong> {req.purpose}
                      </p>
                      {req.review_notes && (
                        <p className="text-xs text-status-reserved-fg bg-status-reserved-bg border border-status-reserved-border p-2.5 rounded-md mt-2">
                          <strong>Catatan Peninjau:</strong> {req.review_notes}
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      {req.status === "pending" && (
                        <Button
                          variant="outline"
                          size="md"
                          className="min-h-[44px]"
                          onClick={() => handleCancelRequest(req.id)}
                          isLoading={cancellingId === req.id}
                        >
                          Batalkan
                        </Button>
                      )}
                      {req.status === "approved" && (
                        <span className="text-xs text-status-available-fg font-medium">
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
        <div role="tabpanel" id="panel-loans" aria-labelledby="tab-loans">
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
                    className={`flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl border p-5 shadow-xs transition-colors ${
                      isOverdue
                        ? "border-status-overdue-border bg-status-overdue-bg/20 ring-1 ring-status-overdue-border text-status-overdue-fg font-bold"
                        : "border-border bg-surface"
                    }`}
                  >
                    <div className="space-y-1.5">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-fg font-mono font-tabular">
                          Pinjaman #{loan.id.slice(0, 8)}
                        </span>
                        <LoanStatusBadge status={loan.status} />
                      </div>

                      <div className="text-xs text-fg-muted space-y-0.5">
                        <p>
                          <strong className="text-fg">Batas Pengembalian:</strong>{" "}
                          <span
                            className={
                              isOverdue
                                ? "font-bold text-status-overdue-fg font-tabular"
                                : "text-fg font-tabular"
                            }
                          >
                            {formatDate(loan.expected_return_time)}
                          </span>
                        </p>
                        {loan.checkout_time && (
                          <p className="text-fg-subtle">
                            Waktu Ambil: <span className="font-tabular">{formatDate(loan.checkout_time)}</span>
                          </p>
                        )}
                        {loan.actual_return_time && (
                          <p className="text-fg-subtle">
                            Waktu Kembali: <span className="font-tabular">{formatDate(loan.actual_return_time)}</span>
                          </p>
                        )}
                        {loan.return_condition && (
                          <div className="flex items-center gap-1.5 pt-1">
                            <span className="text-fg-subtle">
                              Kondisi Pengembalian:
                            </span>
                            <ConditionBadge condition={loan.return_condition} />
                          </div>
                        )}
                      </div>
                    </div>

                    <div>
                      {loan.status === "pending_checkout" && (
                        <div className="rounded-lg bg-status-reserved-bg border border-status-reserved-border p-3 text-xs text-status-reserved-fg max-w-xs">
                          Tunjukkan kode token yang diberikan admin untuk
                          mengambil unit fisik.
                        </div>
                      )}
                      {isOverdue && (
                        <div className="rounded-lg bg-status-overdue-bg border border-status-overdue-border p-3 text-xs text-status-overdue-fg font-medium">
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
