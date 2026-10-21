import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { notificationApi } from "../api/endpoints";
import { useAuth } from "../context/AuthContext";
import { CreateOrgModal } from "./CreateOrgModal";
import { RoleBadge } from "./ui/Badge";
import { Button } from "./ui/Button";

interface HeaderProps {
  currentTab: string;
  onTabChange: (tab: string) => void;
}

export function Header({ currentTab, onTabChange }: HeaderProps) {
  const { user, organizations, activeOrg, role, logout, switchOrg } = useAuth();
  const [isCreateOrgOpen, setIsCreateOrgOpen] = useState(false);
  // Query unread count
  const { data: unreadNotifications = [] } = useQuery({
    queryKey: ["notifications", activeOrg?.id, "unread"],
    queryFn: () =>
      activeOrg
        ? notificationApi.listNotifications(activeOrg.id, "unread")
        : Promise.resolve([]),
    enabled: !!activeOrg?.id,
    refetchInterval: 30000,
  });

  const unreadCount = unreadNotifications.length;

  return (
    <>
      <header className="sticky top-0 z-40 border-b border-border bg-surface/95 backdrop-blur-md text-fg">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8 h-16">
        {/* Brand */}
        <div className="flex items-center gap-4">
          <div
            onClick={() => onTabChange("catalog")}
            className="flex items-center gap-2.5 cursor-pointer"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-accent text-accent-fg font-bold shadow-xs">
              B
            </div>
            <span className="font-bold text-lg text-fg tracking-tight">
              BorrowHub
            </span>
          </div>

          {/* Org Switcher & Creation */}
          {organizations.length > 0 ? (
            <div className="hidden sm:flex items-center gap-2 pl-4 border-l border-border">
              <label htmlFor="org-select" className="sr-only">
                Pilih Organisasi
              </label>
              <select
                id="org-select"
                value={activeOrg?.id || ""}
                onChange={(e) => {
                  if (e.target.value === "__create__") {
                    setIsCreateOrgOpen(true);
                  } else {
                    switchOrg(e.target.value);
                  }
                }}
                className="min-h-[44px] text-xs font-semibold rounded-lg border border-border-strong bg-surface text-fg py-1.5 pl-2.5 pr-8 hover:bg-surface-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
              >
                {organizations.map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
                <option value="__create__">+ Buat Organisasi Baru...</option>
              </select>
              {role && <RoleBadge role={role} />}
            </div>
          ) : (
            <div className="hidden sm:flex items-center gap-2 pl-4 border-l border-border">
              <Button
                size="sm"
                variant="primary"
                onClick={() => setIsCreateOrgOpen(true)}
              >
                + Buat Organisasi
              </Button>
            </div>
          )}
        </div>

        {/* Right actions: Notification bell & User / Logout */}
        <div className="flex items-center gap-3">
          {/* Notification Bell */}
          <button
            type="button"
            onClick={() => onTabChange("notifications")}
            className={`relative inline-flex items-center justify-center min-h-[44px] min-w-[44px] rounded-lg text-fg-muted hover:bg-surface-hover hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent cursor-pointer ${
              currentTab === "notifications" ? "bg-surface-hover text-fg" : ""
            }`}
            aria-label="Notifikasi"
          >
            <svg
              className="h-5 w-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
              />
            </svg>
            {unreadCount > 0 && (
              <span className="absolute top-1.5 right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-status-overdue px-1 text-[10px] font-bold text-white shadow-xs">
                {unreadCount > 9 ? "9+" : unreadCount}
              </span>
            )}
          </button>

          {/* User info */}
          <div className="flex items-center gap-3 pl-3 border-l border-border">
            <div className="hidden md:block text-right">
              <p className="text-xs font-semibold text-fg leading-tight">
                {user?.display_name || user?.email}
              </p>
              <p className="text-[11px] text-fg-muted truncate max-w-[140px]">
                {user?.email}
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void logout()}
              title="Keluar"
            >
              Keluar
            </Button>
          </div>
        </div>
      </div>
      </header>
      <CreateOrgModal
        isOpen={isCreateOrgOpen}
        onClose={() => setIsCreateOrgOpen(false)}
      />
    </>
  );
}
