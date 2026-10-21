import { describe, expect, it } from "bun:test";
import React from "react";
import { renderToString } from "react-dom/server";
import { AlertToast } from "../src/components/ui/AlertToast";
import {
  Badge,
  ConditionBadge,
  LoanStatusBadge,
  RequestStatusBadge,
  RoleBadge,
  UnitStatusBadge,
} from "../src/components/ui/Badge";
import { Button } from "../src/components/ui/Button";
import { EmptyState } from "../src/components/ui/EmptyState";
import { Input } from "../src/components/ui/Input";
import { Modal } from "../src/components/ui/Modal";
import { Navigation } from "../src/components/Navigation";
import { Select } from "../src/components/ui/Select";

describe("Base UI Primitives & Navigation Shell Smoke Tests", () => {
  describe("Button Component", () => {
    it("renders variants with appropriate semantic tokens", () => {
      const primary = renderToString(
        React.createElement(Button, { variant: "primary" }, "Aksi Utama"),
      );
      expect(primary).toContain("bg-accent");
      expect(primary).toContain("text-accent-fg");
      expect(primary).toContain("focus-visible:ring-accent");

      const secondary = renderToString(
        React.createElement(Button, { variant: "secondary" }, "Batal"),
      );
      expect(secondary).toContain("bg-surface-hover");
      expect(secondary).toContain("text-fg");
      expect(secondary).toContain("border-border");

      const danger = renderToString(
        React.createElement(Button, { variant: "danger" }, "Hapus"),
      );
      expect(danger).toContain("bg-status-overdue");
      expect(danger).toContain("text-white");

      const outline = renderToString(
        React.createElement(Button, { variant: "outline" }, "Detail"),
      );
      expect(outline).toContain("border-border-strong");
      expect(outline).toContain("bg-surface");
      expect(outline).toContain("text-fg");

      const ghost = renderToString(
        React.createElement(Button, { variant: "ghost" }, "Kembali"),
      );
      expect(ghost).toContain("text-fg-muted");
    });

    it("enforces touch target sizes: 44px min-h on md/lg and 36px on sm", () => {
      const sm = renderToString(
        React.createElement(Button, { size: "sm" }, "Kecil"),
      );
      expect(sm).toContain("min-h-[36px]");

      const md = renderToString(
        React.createElement(Button, { size: "md" }, "Sedang"),
      );
      expect(md).toContain("min-h-[44px]");

      const lg = renderToString(
        React.createElement(Button, { size: "lg" }, "Besar"),
      );
      expect(lg).toContain("min-h-[44px]");
    });

    it("renders loading spinner and disabled state", () => {
      const loading = renderToString(
        React.createElement(Button, { isLoading: true }, "Simpan"),
      );
      expect(loading).toContain("animate-spin");
      expect(loading).toContain("disabled");

      const disabled = renderToString(
        React.createElement(Button, { disabled: true }, "Nonaktif"),
      );
      expect(disabled).toContain("disabled");
    });
  });

  describe("Badge Component", () => {
    it("renders status indicator dot with aria-hidden", () => {
      const badge = renderToString(
        React.createElement(Badge, { variant: "available" }, "Tersedia"),
      );
      expect(badge).toContain("rounded-full h-1.5 w-1.5 shrink-0");
      expect(badge).toContain('aria-hidden="true"');
      expect(badge).toContain("bg-status-available");
      expect(badge).toContain("bg-status-available-bg");
      expect(badge).toContain("text-status-available-fg");
    });

    it("supports all semantic Tahap 1 status variants", () => {
      const reserved = renderToString(
        React.createElement(Badge, { variant: "reserved" }, "Dipesan"),
      );
      expect(reserved).toContain("bg-status-reserved-bg");
      expect(reserved).toContain("bg-status-reserved");

      const borrowed = renderToString(
        React.createElement(Badge, { variant: "borrowed" }, "Dipinjam"),
      );
      expect(borrowed).toContain("bg-status-borrowed-bg");
      expect(borrowed).toContain("bg-status-borrowed");

      const overdue = renderToString(
        React.createElement(Badge, { variant: "overdue" }, "Terlambat"),
      );
      expect(overdue).toContain("bg-status-overdue-bg");
      expect(overdue).toContain("bg-status-overdue");

      const maintenance = renderToString(
        React.createElement(Badge, { variant: "maintenance" }, "Perawatan"),
      );
      expect(maintenance).toContain("bg-status-maintenance-bg");
      expect(maintenance).toContain("bg-status-maintenance");
    });

    it("renders specialized status badges correctly", () => {
      const unitAvail = renderToString(
        React.createElement(UnitStatusBadge, { status: "available" }),
      );
      expect(unitAvail).toContain("Tersedia");
      expect(unitAvail).toContain("bg-status-available");

      const unitMaint = renderToString(
        React.createElement(UnitStatusBadge, { status: "maintenance" }),
      );
      expect(unitMaint).toContain("Pemeliharaan");
      expect(unitMaint).toContain("bg-status-maintenance");

      const reqPending = renderToString(
        React.createElement(RequestStatusBadge, { status: "pending" }),
      );
      expect(reqPending).toContain("Menunggu Persetujuan");

      const loanOverdue = renderToString(
        React.createElement(LoanStatusBadge, { status: "overdue" }),
      );
      expect(loanOverdue).toContain("Terlambat");
      expect(loanOverdue).toContain("bg-status-overdue");

      const condGood = renderToString(
        React.createElement(ConditionBadge, { condition: "good" }),
      );
      expect(condGood).toContain("Baik");

      const condDamaged = renderToString(
        React.createElement(ConditionBadge, { condition: "damaged" }),
      );
      expect(condDamaged).toContain("Rusak");

      const roleAdmin = renderToString(
        React.createElement(RoleBadge, { role: "admin" }),
      );
      expect(roleAdmin).toContain("Admin");
    });
  });

  describe("Input & Select Components", () => {
    it("renders Input with 44px touch target, semantic styling, and ARIA attributes", () => {
      const normalInput = renderToString(
        React.createElement(Input, {
          label: "Email Pengguna",
          placeholder: "user@example.com",
        }),
      );
      expect(normalInput).toContain("min-h-[44px]");
      expect(normalInput).toContain("bg-surface");
      expect(normalInput).toContain("text-fg");
      expect(normalInput).toContain('aria-invalid="false"');

      const errorInput = renderToString(
        React.createElement(Input, {
          label: "Nama Lengkap",
          error: "Wajib diisi",
        }),
      );
      expect(errorInput).toContain('aria-invalid="true"');
      expect(errorInput).toContain("aria-describedby=");
      expect(errorInput).toContain("Wajib diisi");
      expect(errorInput).toContain("border-status-overdue");

      const helperInput = renderToString(
        React.createElement(Input, {
          label: "Kode Pos",
          helperText: "5 digit angka",
        }),
      );
      expect(helperInput).toContain("aria-describedby=");
      expect(helperInput).toContain("5 digit angka");
    });

    it("renders Select with 44px touch target, options, and error state", () => {
      const selectHtml = renderToString(
        React.createElement(Select, {
          label: "Pilih Lokasi",
          placeholder: "-- Pilih Gudang --",
          options: [
            { value: "gudang-a", label: "Gudang A" },
            { value: "gudang-b", label: "Gudang B" },
          ],
          error: "Pilih salah satu lokasi",
        }),
      );
      expect(selectHtml).toContain("min-h-[44px]");
      expect(selectHtml).toContain("bg-surface");
      expect(selectHtml).toContain("text-fg");
      expect(selectHtml).toContain('aria-invalid="true"');
      expect(selectHtml).toContain("Gudang A");
      expect(selectHtml).toContain("Pilih salah satu lokasi");
    });
  });

  describe("Modal Component", () => {
    it("returns null when isOpen is false", () => {
      const closed = renderToString(
        React.createElement(
          Modal,
          { isOpen: false, onClose: () => {}, title: "Ubah Profil" },
          React.createElement("p", null, "Konten"),
        ),
      );
      expect(closed).toBe("");
    });

    it("renders accessible modal with 44px close button and semantic tokens", () => {
      const openModal = renderToString(
        React.createElement(
          Modal,
          {
            isOpen: true,
            onClose: () => {},
            title: "Konfirmasi Peminjaman",
            description: "Harap periksa tanggal pengembalian",
          },
          React.createElement("div", null, "Detail Peminjaman"),
        ),
      );

      expect(openModal).toContain('role="dialog"');
      expect(openModal).toContain('aria-modal="true"');
      expect(openModal).toContain('aria-labelledby="modal-title"');
      expect(openModal).toContain('aria-describedby="modal-description"');
      expect(openModal).toContain("bg-surface");
      expect(openModal).toContain("border-border");
      expect(openModal).toContain("min-h-[44px]");
      expect(openModal).toContain("min-w-[44px]");
      expect(openModal).toContain("Konfirmasi Peminjaman");
      expect(openModal).toContain("Harap periksa tanggal pengembalian");
    });
  });

  describe("Navigation Component", () => {
    it("renders tablist with accessible tab roles and active pill state", () => {
      const navHtml = renderToString(
        React.createElement(Navigation, {
          currentTab: "catalog",
          onTabChange: () => {},
          isAdmin: true,
        }),
      );

      expect(navHtml).toContain('role="tablist"');
      expect(navHtml).toContain('aria-orientation="horizontal"');
      expect(navHtml).toContain('role="tab"');
      expect(navHtml).toContain("min-h-[44px]");
      expect(navHtml).toContain('aria-selected="true"');
      expect(navHtml).toContain("bg-accent text-accent-fg");
      expect(navHtml).toContain("Katalog Peralatan");
      expect(navHtml).toContain("Admin Inventaris");
    });

    it("renders inactive tabs with aria-selected false and muted style", () => {
      const navHtml = renderToString(
        React.createElement(Navigation, {
          currentTab: "my-loans",
          onTabChange: () => {},
          isAdmin: false,
        }),
      );

      expect(navHtml).toContain('aria-selected="false"');
      expect(navHtml).toContain("text-fg-muted");
      expect(navHtml).not.toContain("Admin Inventaris");
    });
  });

  describe("AlertToast Component", () => {
    it("renders semantic tokens for alert types and min 44px close button", () => {
      const successHtml = renderToString(
        React.createElement(AlertToast, {
          type: "success",
          message: "Data berhasil disimpan",
          onClose: () => {},
        }),
      );
      expect(successHtml).toContain("bg-status-available-bg");
      expect(successHtml).toContain("text-status-available-fg");
      expect(successHtml).toContain("min-h-[44px]");
      expect(successHtml).toContain("min-w-[44px]");

      const errorHtml = renderToString(
        React.createElement(AlertToast, {
          type: "error",
          message: "Terjadi kesalahan",
          requestId: "req-err-456",
        }),
      );
      expect(errorHtml).toContain("bg-status-overdue-bg");
      expect(errorHtml).toContain("Request ID: <!-- -->req-err-456");
    });
  });

  describe("EmptyState Component", () => {
    it("renders with semantic surface and border tokens", () => {
      const emptyHtml = renderToString(
        React.createElement(EmptyState, {
          title: "Tidak Ada Data",
          description: "Belum ada unit yang terdaftar pada sistem",
          actionLabel: "Tambah Unit",
          onAction: () => {},
        }),
      );

      expect(emptyHtml).toContain("bg-surface");
      expect(emptyHtml).toContain("border-border-strong");
      expect(emptyHtml).toContain("text-fg");
      expect(emptyHtml).toContain("text-fg-muted");
      expect(emptyHtml).toContain("Tidak Ada Data");
      expect(emptyHtml).toContain("Tambah Unit");
    });
  });
});
