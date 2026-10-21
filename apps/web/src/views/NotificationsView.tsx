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
          <h1 className="text-2xl font-bold tracking-tight text-fg">
            Pusat Notifikasi
          </h1>
          <p className="mt-1 text-sm text-fg-muted">
            Pemberitahuan persetujuan peminjaman, jatuh tempo, dan aktivitas
            peralatan untuk{" "}
            <span className="font-semibold text-fg">{activeOrg.name}</span>.
          </p>
        </div>

        {/* Filter buttons */}
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setFilter("all")}
            className={`min-h-[44px] px-4 py-2 text-xs font-semibold rounded-lg transition-colors cursor-pointer inline-flex items-center justify-center ${
              filter === "all"
                ? "bg-accent text-accent-fg shadow-xs"
                : "bg-surface text-fg-muted hover:text-fg border border-border hover:bg-surface-hover"
            }`}
          >
            Semua
          </button>
          <button
            type="button"
            onClick={() => setFilter("unread")}
            className={`min-h-[44px] px-4 py-2 text-xs font-semibold rounded-lg transition-colors cursor-pointer inline-flex items-center justify-center ${
              filter === "unread"
                ? "bg-accent text-accent-fg shadow-xs"
                : "bg-surface text-fg-muted hover:text-fg border border-border hover:bg-surface-hover"
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
        <div className="divide-y divide-border-subtle rounded-2xl border border-border bg-surface shadow-xs overflow-hidden">
          {notifications.map((notif) => {
            const isUnread = notif.status === "unread";
            return (
              <div
                key={notif.id}
                className={`p-4 sm:p-5 flex flex-col sm:flex-row sm:items-start justify-between gap-3 transition-colors ${
                  isUnread ? "bg-status-borrowed-bg/30" : "bg-surface"
                }`}
              >
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2">
                    {isUnread && (
                      <span
                        className="w-2 h-2 rounded-full bg-status-overdue shrink-0"
                        aria-hidden="true"
                      />
                    )}
                    <h3
                      className={`text-sm ${
                        isUnread
                          ? "font-bold text-fg"
                          : "font-medium text-fg-muted"
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
                  <p className="text-xs text-fg-muted leading-relaxed">
                    {notif.message}
                  </p>
                  <p className="text-[11px] text-fg-subtle font-tabular">
                    {formatDate(notif.created_at)}
                  </p>
                </div>

                {isUnread && (
                  <div className="shrink-0 pt-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="min-h-[44px]"
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
