import React from "react";
import type {
  LoanStatus,
  RequestStatus,
  ReturnCondition,
  Role,
  UnitCondition,
  UnitStatus,
} from "../../api/types";

export type BadgeType =
  | "unit-status"
  | "request-status"
  | "loan-status"
  | "condition"
  | "role"
  | "custom";

export type BadgeVariant =
  | "available"
  | "reserved"
  | "borrowed"
  | "overdue"
  | "maintenance"
  | "slate"
  | "blue"
  | "green"
  | "emerald"
  | "amber"
  | "orange"
  | "rose"
  | "purple";

export interface BadgeProps {
  children?: React.ReactNode;
  variant?: BadgeVariant;
  size?: "sm" | "md";
  className?: string;
  dot?: boolean;
}

const variants: Record<BadgeVariant, { badge: string; dot: string }> = {
  available: {
    badge:
      "bg-status-available-bg text-status-available-fg border-status-available-border",
    dot: "bg-status-available",
  },
  reserved: {
    badge:
      "bg-status-reserved-bg text-status-reserved-fg border-status-reserved-border",
    dot: "bg-status-reserved",
  },
  borrowed: {
    badge:
      "bg-status-borrowed-bg text-status-borrowed-fg border-status-borrowed-border",
    dot: "bg-status-borrowed",
  },
  overdue: {
    badge:
      "bg-status-overdue-bg text-status-overdue-fg border-status-overdue-border",
    dot: "bg-status-overdue",
  },
  maintenance: {
    badge:
      "bg-status-maintenance-bg text-status-maintenance-fg border-status-maintenance-border",
    dot: "bg-status-maintenance",
  },
  emerald: {
    badge:
      "bg-status-available-bg text-status-available-fg border-status-available-border",
    dot: "bg-status-available",
  },
  green: {
    badge:
      "bg-status-available-bg text-status-available-fg border-status-available-border",
    dot: "bg-status-available",
  },
  amber: {
    badge:
      "bg-status-reserved-bg text-status-reserved-fg border-status-reserved-border",
    dot: "bg-status-reserved",
  },
  blue: {
    badge:
      "bg-status-borrowed-bg text-status-borrowed-fg border-status-borrowed-border",
    dot: "bg-status-borrowed",
  },
  rose: {
    badge:
      "bg-status-overdue-bg text-status-overdue-fg border-status-overdue-border",
    dot: "bg-status-overdue",
  },
  orange: {
    badge:
      "bg-status-maintenance-bg text-status-maintenance-fg border-status-maintenance-border",
    dot: "bg-status-maintenance",
  },
  slate: {
    badge: "bg-surface-hover text-fg-muted border-border",
    dot: "bg-fg-subtle",
  },
  purple: {
    badge: "bg-accent-subtle text-accent border-border",
    dot: "bg-accent",
  },
};

export function Badge({
  children,
  variant = "slate",
  size = "md",
  className = "",
  dot = true,
}: BadgeProps) {
  const currentVariant = variants[variant] || variants.slate;
  const sizes = {
    sm: "px-2 py-0.5 text-xs font-medium gap-1.5",
    md: "px-2.5 py-0.5 text-xs font-semibold gap-1.5",
  };

  return (
    <span
      className={`inline-flex items-center rounded-full border ${currentVariant.badge} ${sizes[size]} ${className}`}
    >
      {dot && (
        <span
          className={`rounded-full h-1.5 w-1.5 shrink-0 ${currentVariant.dot}`}
          aria-hidden="true"
        />
      )}
      {children}
    </span>
  );
}

export function UnitStatusBadge({ status }: { status: UnitStatus }) {
  switch (status) {
    case "available":
      return <Badge variant="available">Tersedia</Badge>;
    case "reserved":
      return <Badge variant="reserved">Dipesan</Badge>;
    case "borrowed":
      return <Badge variant="borrowed">Dipinjam</Badge>;
    case "maintenance":
      return <Badge variant="maintenance">Pemeliharaan</Badge>;
    case "retired":
      return <Badge variant="slate">Dipensiunkan</Badge>;
    default:
      return <Badge variant="slate">{status}</Badge>;
  }
}

export function RequestStatusBadge({ status }: { status: RequestStatus }) {
  switch (status) {
    case "pending":
      return <Badge variant="reserved">Menunggu Persetujuan</Badge>;
    case "approved":
      return <Badge variant="available">Disetujui</Badge>;
    case "rejected":
      return <Badge variant="overdue">Ditolak</Badge>;
    case "cancelled":
      return <Badge variant="slate">Dibatalkan</Badge>;
    default:
      return <Badge variant="slate">{status}</Badge>;
  }
}

export function LoanStatusBadge({ status }: { status: LoanStatus }) {
  switch (status) {
    case "pending_checkout":
      return <Badge variant="reserved">Menunggu Ambil</Badge>;
    case "checked_out":
      return <Badge variant="borrowed">Sedang Dipinjam</Badge>;
    case "returned":
      return <Badge variant="slate">Dikembalikan</Badge>;
    case "overdue":
      return <Badge variant="overdue">Terlambat</Badge>;
    case "cancelled":
      return <Badge variant="slate">Dibatalkan</Badge>;
    default:
      return <Badge variant="slate">{status}</Badge>;
  }
}

export function ConditionBadge({
  condition,
}: {
  condition: UnitCondition | ReturnCondition;
}) {
  switch (condition) {
    case "new":
      return <Badge variant="available">Baru</Badge>;
    case "good":
      return <Badge variant="available">Baik</Badge>;
    case "fair":
      return <Badge variant="reserved">Cukup</Badge>;
    case "poor":
      return <Badge variant="maintenance">Kurang</Badge>;
    case "damaged":
      return <Badge variant="overdue">Rusak</Badge>;
    default:
      return <Badge variant="slate">{condition}</Badge>;
  }
}

export function RoleBadge({ role }: { role: Role }) {
  switch (role) {
    case "owner":
      return <Badge variant="purple">Owner</Badge>;
    case "admin":
      return <Badge variant="borrowed">Admin</Badge>;
    case "member":
      return <Badge variant="slate">Member</Badge>;
    default:
      return <Badge variant="slate">{role}</Badge>;
  }
}
