import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { notificationApi } from "../api/endpoints";
import type { NotificationStatus } from "../api/types";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useAuth } from "../context/AuthContext";

export function NotificationsView() {
  const { activeOrg } = useAuth();
  const orgId = activeOrg?.id;

  const [filter, setFilter] = useState<"all" | "unread">("all");
  const [markingId, setMarkingId] = useState<string | null>(null);

  const statusParam: NotificationStatus | undefined =
    filter === "unread" ? "unread" : undefined;

  const {
    data: notifications = [],
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["notifications", orgId, filter],
    queryFn: () =>
      orgId
        ? notificationApi.listNotifications(orgId, statusParam)
        : Promise.resolve([]),
    enabled: !!orgId,
  });

  const handleMarkAsRead = async (notifId: string) => {
    if (!orgId) return;
    setMarkingId(notifId);
    try {
      await notificationApi.markAsRead(notifId, orgId);
      void refetch();
    } finally {
      setMarkingId(null);
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

  if (!orgId) {
    return (
      <div className="py-12">
        <EmptyState
          title="Tidak ada organisasi aktif"
          description="Pilih organisasi terlebih dahulu untuk melihat notifikasi."
        />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">
            Pusat Notifikasi
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Pemberitahuan persetujuan peminjaman, jatuh tempo, dan aktivitas
            peralatan untuk{" "}
            <span className="font-semibold text-slate-800">{activeOrg.name}</span>.
          </p>
        </div>

        {/* Filter buttons */}
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setFilter("all")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
              filter === "all"
                ? "bg-slate-900 text-white shadow-xs"
                : "bg-slate-100 text-slate-700 hover:bg-slate-200"
            }`}
          >
            Semua
          </button>
          <button
            type="button"
            onClick={() => setFilter("unread")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
              filter === "unread"
                ? "bg-slate-900 text-white shadow-xs"
                : "bg-slate-100 text-slate-700 hover:bg-slate-200"
            }`}
          >
            Belum Dibaca
          </button>
        </div>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : notifications.length === 0 ? (
        <EmptyState
          title="Tidak ada notifikasi"
          description={
            filter === "unread"
              ? "Semua notifikasi telah Anda baca."
              : "Belum ada riwayat notifikasi untuk organisasi ini."
          }
        />
      ) : (
        <div className="divide-y divide-slate-100 rounded-2xl border border-slate-200 bg-white shadow-xs overflow-hidden">
          {notifications.map((notif) => {
            const isUnread = notif.status === "unread";
            return (
              <div
                key={notif.id}
                className={`p-4 sm:p-5 flex flex-col sm:flex-row sm:items-start justify-between gap-3 transition-colors ${
                  isUnread ? "bg-sky-50/40" : "bg-white"
                }`}
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <h3
                      className={`text-sm ${
                        isUnread
                          ? "font-bold text-slate-900"
                          : "font-medium text-slate-700"
                      }`}
                    >
                      {notif.title}
                    </h3>
                    {isUnread ? (
                      <Badge variant="blue" size="sm">
                        Baru
                      </Badge>
                    ) : (
                      <Badge variant="slate" size="sm">
                        Dibaca
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-slate-600 leading-relaxed">
                    {notif.message}
                  </p>
                  <p className="text-[11px] text-slate-400">
                    {formatDate(notif.created_at)}
                  </p>
                </div>

                {isUnread && (
                  <div className="shrink-0 pt-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleMarkAsRead(notif.id)}
                      isLoading={markingId === notif.id}
                    >
                      Tandai Dibaca
                    </Button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
