import { describe, expect, it } from "bun:test";
import React from "react";
import { renderToString } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, type AuthContextValue } from "../src/context/AuthContext";
import { CatalogView } from "../src/views/CatalogView";
import { ItemDetailModal } from "../src/views/ItemDetailModal";
import { BookingModal } from "../src/views/BookingModal";
import { MyLoansView, type MyLoansViewProps } from "../src/views/MyLoansView";
import type { Item, Category, LoanRequest, Loan, Unit } from "../src/api/types";

const createMockQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });

const mockMemberAuth: AuthContextValue = {
  user: {
    id: "user-mem-1",
    email: "member@borrowhub.local",
    display_name: "Budi Santoso",
    status: "active",
    terms_accepted_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  organizations: [
    {
      id: "org-lab-1",
      name: "Laboratorium Multimedia",
      role: "member",
      slug: "lab-multimedia",
      timezone: "Asia/Jakarta",
      visibility: "public",
      max_loan_duration_days: 7,
      status: "active",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  ],
  activeOrg: {
    id: "org-lab-1",
    name: "Laboratorium Multimedia",
    role: "member",
    slug: "lab-multimedia",
    timezone: "Asia/Jakarta",
    visibility: "public",
    max_loan_duration_days: 7,
    status: "active",
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  role: "member",
  isAdmin: false,
  isOwner: false,
  isAuthenticated: true,
  isLoading: false,
  login: async () => {},
  register: async () => {},
  logout: async () => {},
  switchOrg: () => {},
  refreshProfile: async () => {},
};

const mockSampleItem: Item = {
  id: "item-fx3-001",
  organization_id: "org-lab-1",
  category_id: "cat-cam-1",
  name: "Sony FX3 Cinema Camera",
  description: "Kamera bioskop full-frame untuk produksi video",
  is_lendable: true,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

const mockSampleCategory: Category = {
  id: "cat-cam-1",
  organization_id: "org-lab-1",
  name: "Kamera & Optik",
  slug: "kamera-optik",
  description: "Bodi kamera dan lensa video",
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

const mockSampleUnit: Unit = {
  id: "unit-fx3-a",
  organization_id: "org-lab-1",
  item_id: "item-fx3-001",
  barcode: "BC-FX3-001",
  serial_number: "SN-987654321",
  status: "available",
  condition: "good",
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe("Tahap 3 - Member Experience Views Smoke Tests", () => {
  describe("CatalogView Component", () => {
    it("renders empty state when active organization is missing", () => {
      const qc = createMockQueryClient();
      const authWithoutOrg: AuthContextValue = {
        ...mockMemberAuth,
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
            React.createElement(CatalogView, null),
          ),
        ),
      );

      expect(html).toContain("Tidak ada organisasi aktif");
      expect(html).toContain("bg-surface");
      expect(html).toContain("border-border-strong");
    });

    it("renders catalog header, 44px search input, and filter toolbar with semantic tokens", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement(CatalogView, null),
          ),
        ),
      );

      expect(html).toContain("Katalog Peralatan");
      expect(html).toContain("Laboratorium Multimedia");
      // Search input with min 44px touch target and semantic surface token
      expect(html).toContain('placeholder="Cari peralatan..."');
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("bg-surface");
      expect(html).toContain("text-fg");
      expect(html).toContain("border-border");
    });

    it("renders category filter pills and item cards with semantic tokens", () => {
      const qc = createMockQueryClient();
      qc.setQueryData(["categories", "org-lab-1"], [mockSampleCategory]);
      qc.setQueryData(["items", "org-lab-1"], [mockSampleItem]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement(CatalogView, null),
          ),
        ),
      );

      // Pill button min 44px
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("bg-accent text-accent-fg");
      expect(html).toContain("Kamera &amp; Optik");
      // Item card semantic tokens
      expect(html).toContain("bg-surface border border-border rounded-xl");
      expect(html).toContain("Sony FX3 Cinema Camera");
      expect(html).toContain("Ajukan Pinjam");
    });
  });

  describe("ItemDetailModal Component", () => {
    it("returns empty string when isOpen is false", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(ItemDetailModal, {
            item: mockSampleItem,
            category: mockSampleCategory,
            organizationId: "org-lab-1",
            isOpen: false,
            onClose: () => {},
          }),
        ),
      );

      expect(html).toBe("");
    });

    it("renders equipment details, status token, and 44px CTA buttons", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(ItemDetailModal, {
            item: mockSampleItem,
            category: mockSampleCategory,
            organizationId: "org-lab-1",
            isOpen: true,
            onClose: () => {},
          }),
        ),
      );

      expect(html).toContain('role="dialog"');
      expect(html).toContain("Sony FX3 Cinema Camera");
      expect(html).toContain("Kamera &amp; Optik");
      expect(html).toContain("Kamera bioskop full-frame untuk produksi video");
      // Status badge token
      expect(html).toContain("bg-status-available-bg");
      expect(html).toContain("text-status-available-fg");
      // Touch target verification on primary CTA
      expect(html).toContain("Ajukan Pinjam");
      expect(html).toContain("min-h-[44px]");
    });

    it("renders admin units breakdown with monospace tabular barcodes and serial numbers", () => {
      const qc = createMockQueryClient();
      qc.setQueryData(["units", mockSampleItem.id], [mockSampleUnit]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(ItemDetailModal, {
            item: mockSampleItem,
            category: mockSampleCategory,
            organizationId: "org-lab-1",
            isOpen: true,
            isAdmin: true,
            onClose: () => {},
          }),
        ),
      );

      expect(html).toContain("Daftar Unit Fisik");
      expect(html).toContain("BC-FX3-001");
      expect(html).toContain("SN:");
      expect(html).toContain("987654321");
      expect(html).toContain("font-mono font-tabular");
    });
  });

  describe("BookingModal Component", () => {
    it("returns empty string when isOpen is false", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(BookingModal, {
            item: mockSampleItem,
            organizationId: "org-lab-1",
            isOpen: false,
            onClose: () => {},
          }),
        ),
      );

      expect(html).toBe("");
    });

    it("renders datetime inputs, purpose field, and live availability container", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(BookingModal, {
            item: mockSampleItem,
            organizationId: "org-lab-1",
            isOpen: true,
            onClose: () => {},
          }),
        ),
      );

      expect(html).toContain('role="dialog"');
      expect(html).toContain("Ajukan Peminjaman: Sony FX3 Cinema Camera");
      expect(html).toContain("Waktu Mulai");
      expect(html).toContain("Waktu Selesai");
      expect(html).toContain("Tujuan Peminjaman");
      // Inputs min 44px touch target
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("bg-surface");
      // Availability check panel
      expect(html).toContain("Pemeriksaan Ketersediaan Live");
      expect(html).toContain("Cek Unit");
      // Form submit button
      expect(html).toContain("Kirim Permohonan");
    });
  });

  describe("MyLoansView Component", () => {
    it("renders accessible tablist navigation with 44px touch targets", () => {
      const qc = createMockQueryClient();
      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement(MyLoansView, null),
          ),
        ),
      );

      expect(html).toContain("Peminjaman Saya");
      expect(html).toContain('role="tablist"');
      expect(html).toContain('role="tab"');
      expect(html).toContain('aria-selected="true"');
      expect(html).toContain("min-h-[44px]");
      expect(html).toContain("Permohonan Peminjaman");
      expect(html).toContain("Peminjaman Aktif &amp; Selesai");
    });

    it("renders empty state when user has no requests or loans", () => {
      const qc = createMockQueryClient();
      qc.setQueryData(["loan-requests", "org-lab-1"], []);
      qc.setQueryData(["loans", "org-lab-1"], []);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement(MyLoansView, null),
          ),
        ),
      );

      expect(html).toContain("Belum ada permohonan");
      expect(html).toContain("bg-surface");
      expect(html).toContain("border-border-strong");
    });

    it("renders pending loan request with 44px cancel button and tabular dates", () => {
      const qc = createMockQueryClient();
      const pendingReq: LoanRequest = {
        id: "req-pending-1",
        organization_id: "org-lab-1",
        item_id: "item-fx3-001",
        requester_user_id: "user-mem-1",
        status: "pending",
        quantity: 1,
        start_time: "2026-10-25T09:00:00Z",
        end_time: "2026-10-27T17:00:00Z",
        purpose: "Liputan workshop",
        created_at: "2026-10-22T08:00:00Z",
        updated_at: "2026-10-22T08:00:00Z",
      };
      qc.setQueryData(["loan-requests", "org-lab-1"], [pendingReq]);
      qc.setQueryData(["items", "org-lab-1"], [mockSampleItem]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement(MyLoansView, null),
          ),
        ),
      );

      expect(html).toContain("Sony FX3 Cinema Camera");
      expect(html).toContain("font-tabular");
      expect(html).toContain("Batalkan");
      expect(html).toContain("min-h-[44px]");
    });

    it("renders overdue loan card with highlighted overdue styles and tabular fonts", () => {
      const qc = createMockQueryClient();
      const overdueLoan: Loan = {
        id: "loan-overdue-12345678",
        organization_id: "org-lab-1",
        loan_request_id: "req-1",
        reservation_id: "res-1",
        unit_id: "unit-fx3-a",
        borrower_user_id: "user-mem-1",
        status: "overdue",
        expected_return_time: "2026-10-20T10:00:00Z",
        checkout_time: "2026-10-18T09:00:00Z",
        created_at: "2026-10-18T09:00:00Z",
        updated_at: "2026-10-21T00:00:00Z",
      };
      qc.setQueryData(["loans", "org-lab-1"], [overdueLoan]);
      qc.setQueryData(["items", "org-lab-1"], [mockSampleItem]);

      const html = renderToString(
        React.createElement(
          QueryClientProvider,
          { client: qc },
          React.createElement(
            AuthContext.Provider,
            { value: mockMemberAuth },
            React.createElement<MyLoansViewProps>(MyLoansView, { initialTab: "loans" }),
          ),
        ),
      );

      expect(html).toContain("Pinjaman #");
      expect(html).toContain("loan-ove");
      expect(html).toContain("border-status-overdue-border");
      expect(html).toContain("bg-status-overdue-bg/20");
      expect(html).toContain("ring-status-overdue-border");
      expect(html).toContain("text-status-overdue-fg");
      expect(html).toContain("font-tabular");
      expect(html).toContain("Peralatan sudah melewati batas waktu kembali");
    });
  });
});
