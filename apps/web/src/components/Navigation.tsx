import React from "react";
import { useAuth } from "../context/AuthContext";

export interface NavigationProps {
  currentTab: string;
  onTabChange: (tab: string) => void;
  isAdmin?: boolean;
}

export function Navigation({
  currentTab,
  onTabChange,
  isAdmin: propIsAdmin,
}: NavigationProps) {
  let contextIsAdmin = false;
  try {
    const auth = useAuth();
    contextIsAdmin = auth.isAdmin;
  } catch {
    // Allows rendering outside AuthProvider in unit tests or isolated previews
  }
  const isAdmin = propIsAdmin ?? contextIsAdmin;
  const tabs = [
    { id: "catalog", label: "Katalog Peralatan" },
    { id: "my-loans", label: "Peminjaman Saya" },
    { id: "notifications", label: "Notifikasi" },
    ...(isAdmin
      ? [
          { id: "admin-inventory", label: "Admin Inventaris" },
          { id: "admin-lending", label: "Admin Peminjaman & Persetujuan" },
        ]
      : []),
  ];

  return (
    <nav className="border-b border-border bg-surface text-fg" aria-label="Navigasi Utama">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div
          role="tablist"
          aria-orientation="horizontal"
          className="flex gap-2 sm:gap-4 overflow-x-auto py-2 -mb-px"
        >
          {tabs.map((tab) => {
            const isActive = currentTab === tab.id;
            return (
              <button
                key={tab.id}
                id={`tab-${tab.id}`}
                role="tab"
                aria-selected={isActive}
                aria-controls={`panel-${tab.id}`}
                tabIndex={0}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={`min-h-[44px] inline-flex items-center whitespace-nowrap px-3.5 py-2 text-sm font-medium rounded-lg transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent ${
                  isActive
                    ? "bg-accent text-accent-fg font-semibold shadow-xs"
                    : "text-fg-muted hover:text-fg hover:bg-surface-hover"
                }`}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>
    </nav>
  );
}
