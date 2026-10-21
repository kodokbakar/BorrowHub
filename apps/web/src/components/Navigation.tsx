import React from "react";
import { useAuth } from "../context/AuthContext";

interface NavigationProps {
  currentTab: string;
  onTabChange: (tab: string) => void;
}

export function Navigation({ currentTab, onTabChange }: NavigationProps) {
  const { isAdmin } = useAuth();

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
    <nav className="border-b border-slate-200 bg-white">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex gap-2 sm:gap-6 overflow-x-auto py-2 -mb-px">
          {tabs.map((tab) => {
            const isActive = currentTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={`whitespace-nowrap px-3 py-2 text-sm font-medium rounded-lg transition-colors cursor-pointer ${
                  isActive
                    ? "bg-slate-900 text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
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
