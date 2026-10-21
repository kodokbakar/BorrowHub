import { describe, expect, it } from "bun:test";
import React from "react";
import { renderToString } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, type AuthContextValue } from "../src/context/AuthContext";
import { AdminInventoryView } from "../src/views/AdminInventoryView";
import { AdminLendingView } from "../src/views/AdminLendingView";
import { NotificationsView } from "../src/views/NotificationsView";
import type { Item, Category, LoanRequest, Loan, Unit, Notification } from "../src/api/types";

const createMockQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });

const mockAdminAuth: AuthContextValue = {
  user: {
    id: "user-admin-1",
    email: "admin@borrowhub.local",
    display_name: "Super Admin",
    status: "active",
    terms_accepted_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  organizations: [
    {
      id: "org-lab-admin",
      name: "Laboratorium Multimedia & Robotika",
      role: "admin",
      slug: "lab-multimedia-robotika",
      timezone: "Asia/Jakarta",
      visibility: "public",
      max_loan_duration_days: 14,
      status: "active",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  ],
  activeOrg: {
    id: "org-lab-admin",
    name: "Laboratorium Multimedia & Robotika",
    role: "admin",
    slug: "lab-multimedia-robotika",
    timezone: "Asia/Jakarta",
    visibility: "public",
    max_loan_duration_days: 14,
    status: "active",
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  role: "admin",
  isAdmin: true,
  isOwner: false,
  isAuthenticated: true,
  isLoading: false,
  login: async () => {},
  register: async () => {},
  logout: async () => {},
  switchOrg: () => {},
  refreshProfile: async () => {},
};

describe("Tahap 4 - Admin Workspace Views Smoke Tests", () => {
  describe("AdminInventoryView Component", () => {
    it("renders empty state when active organization is missing", () => {
      const qc = createMockQueryClient();
      const authWithoutOrg: AuthContextValue = {
        ...mockAdminAuth,
        activeOrg: null,
        organizations: [],
      };

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: authWithoutOrg },
            React.createElement(AdminInventoryView, null),
          ),
        ),
      );

      expect(html).toContain("Tidak ada organisasi aktif");
      expect(html).toContain("Pilih organisasi terlebih dahulu");
    });

    it("renders subtabs with role=tablist and semantic tokens", () => {
      const qc = createMockQueryClient();

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockAdminAuth },
            React.createElement(AdminInventoryView, null),
          ),
        ),
      );

      expect(html).toContain("Admin Inventaris &amp; Peralatan");
      expect(html).toContain("Laboratorium Multimedia &amp; Robotika");
      expect(html).toContain('role="tablist"');
      expect(html).toContain('role="tab"');
      expect(html).toContain('id="tab-items"');
      expect(html).toContain('id="tab-categories"');
      expect(html).toContain('aria-selected="true"');
      expect(html).toContain('role="tabpanel"');
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("border-border");
      expect(html).toContain("text-fg");
    });

    it("includes initial item query pre-populated in cache", () => {
      const qc = createMockQueryClient();
      const mockItem: Item = {
        id: "item-12345678-abcd",
        organization_id: "org-lab-admin",
        category_id: "cat-optics-1",
        name: "Lensa Sony FE 24-70mm GM II",
        description: "Lensa zoom standar profesional G Master",
        is_lendable: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      const mockCategory: Category = {
        id: "cat-optics-1",
        organization_id: "org-lab-admin",
        name: "Optik & Lensa",
        slug: "optik-lensa",
        description: "Lensa kamera profesional",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      qc.setQueryData(["items", "org-lab-admin"], [mockItem]);
      qc.setQueryData(["categories", "org-lab-admin"], [mockCategory]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockAdminAuth },
            React.createElement(AdminInventoryView, null),
          ),
        ),
      );

      expect(html).toContain("Lensa Sony FE 24-70mm GM II");
      expect(html).toContain("Optik &amp; Lensa");
      expect(html).toContain("Dapat Dipinjam");
      expect(html).toContain("font-mono font-tabular");
    });
  });

  describe("AdminLendingView Component", () => {
    it("renders empty state when active organization is missing", () => {
      const qc = createMockQueryClient();
      const authWithoutOrg: AuthContextValue = {
        ...mockAdminAuth,
        activeOrg: null,
      };

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: authWithoutOrg },
            React.createElement(AdminLendingView, null),
          ),
        ),
      );

      expect(html).toContain("Tidak ada organisasi aktif");
      expect(html).toContain("Pilih organisasi terlebih dahulu");
    });

    it("renders 4 subtabs with role=tablist, role=tab, role=tabpanel, and 44px min-height", () => {
      const qc = createMockQueryClient();

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockAdminAuth },
            React.createElement(AdminLendingView, null),
          ),
        ),
      );

      expect(html).toContain("Alur Peminjaman Admin");
      expect(html).toContain('role="tablist"');
      expect(html).toContain('id="tab-pending"');
      expect(html).toContain('id="tab-checkout"');
      expect(html).toContain('id="tab-active"');
      expect(html).toContain('id="tab-overdue"');
      expect(html).toContain('role="tabpanel"');
      expect(html).toContain('id="panel-pending"');
      expect(html).toContain("Menunggu Persetujuan");
      expect(html).toContain("Meja Check-out");
      expect(html).toContain("Pinjaman Aktif &amp; Selesai");
      expect(html).toContain("Terlambat");
      expect(html).toContain("min-h-[44px]");
    });

    it("renders pending loan request cards with CTAs and tabular numbers", () => {
      const qc = createMockQueryClient();
      const mockReq: LoanRequest = {
        id: "req-lend-001",
        organization_id: "org-lab-admin",
        item_id: "item-fx3-001",
        requester_user_id: "user-student-42",
        status: "pending",
        quantity: 2,
        start_time: "2026-11-01T08:00:00.000Z",
        end_time: "2026-11-03T17:00:00.000Z",
        purpose: "Produksi film dokumenter kampus",
        notes: "Perlu baterai cadangan",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      const mockItem: Item = {
        id: "item-fx3-001",
        organization_id: "org-lab-admin",
        category_id: "cat-cam-1",
        name: "Cinema Camera FX3",
        description: "Full frame camera",
        is_lendable: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      qc.setQueryData(["admin-pending-requests", "org-lab-admin"], [mockReq]);
      qc.setQueryData(["items", "org-lab-admin"], [mockItem]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockAdminAuth },
            React.createElement(AdminLendingView, null),
          ),
        ),
      );

      expect(html).toContain("Cinema Camera FX3");
      expect(html).toContain("user-stu");
      expect(html).toContain("Produksi film dokumenter kampus");
      expect(html).toContain("Perlu baterai cadangan");
      expect(html).toContain("Setujui Permohonan");
      expect(html).toContain("Tolak");
      expect(html).toContain("font-tabular");
      expect(html).toContain("bg-surface");
    });
  });

  describe("NotificationsView Component", () => {
    it("renders empty state when active organization is missing", () => {
      const qc = createMockQueryClient();
      const authWithoutOrg: AuthContextValue = {
        ...mockAdminAuth,
        activeOrg: null,
      };

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: authWithoutOrg },
            React.createElement(NotificationsView, null),
          ),
        ),
      );

      expect(html).toContain("Tidak ada organisasi aktif");
    });

    it("renders notification center with 44px filter pills and semantic styling", () => {
      const qc = createMockQueryClient();
      const mockNotifs: Notification[] = [
        {
          id: "notif-001",
          organization_id: "org-lab-admin",
          recipient_user_id: "user-admin-1",
          event_id: "evt-001",
          event_type: "loan_request.created",
          title: "Permohonan Peminjaman Baru",
          message: "Mahasiswa mengajukan peminjaman Sony FX3.",
          data: { requestId: "req-lend-001" },
          status: "unread",
          created_at: "2026-10-22T09:30:00.000Z",
          updated_at: "2026-10-22T09:30:00.000Z",
        },
        {
          id: "notif-002",
          organization_id: "org-lab-admin",
          recipient_user_id: "user-admin-1",
          event_id: "evt-002",
          event_type: "loan.returned",
          title: "Unit Dikembalikan",
          message: "Tripod Manfrotto telah dikembalikan dalam kondisi baik.",
          data: {},
          status: "read",
          created_at: "2026-10-21T14:15:00.000Z",
          updated_at: "2026-10-21T14:15:00.000Z",
        },
      ];
      qc.setQueryData(["notifications", "org-lab-admin", "all"], mockNotifs);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockAdminAuth },
            React.createElement(NotificationsView, null),
          ),
        ),
      );

      expect(html).toContain("Pusat Notifikasi");
      expect(html).toContain("Semua");
      expect(html).toContain("Belum Dibaca");
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("Permohonan Peminjaman Baru");
      expect(html).toContain("Mahasiswa mengajukan peminjaman Sony FX3.");
      expect(html).toContain("bg-status-overdue"); // Unread dot indicator
      expect(html).toContain("Tandai Dibaca");
      expect(html).toContain("font-tabular");
      expect(html).toContain("bg-surface");
    });
  });
});
