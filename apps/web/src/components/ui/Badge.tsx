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

interface BadgeProps {
  children?: React.ReactNode;
  variant?:
    | "slate"
    | "blue"
    | "green"
    | "emerald"
    | "amber"
    | "orange"
    | "rose"
    | "purple";
  size?: "sm" | "md";
  className?: string;
}

export function Badge({
  children,
  variant = "slate",
  size = "md",
  className = "",
}: BadgeProps) {
  const variants = {
    slate: "bg-slate-100 text-slate-700 border-slate-200",
    blue: "bg-sky-50 text-sky-700 border-sky-200",
    green: "bg-green-50 text-green-700 border-green-200",
    emerald: "bg-emerald-50 text-emerald-700 border-emerald-200",
    amber: "bg-amber-50 text-amber-700 border-amber-200",
    orange: "bg-orange-50 text-orange-700 border-orange-200",
    rose: "bg-rose-50 text-rose-700 border-rose-200",
    purple: "bg-purple-50 text-purple-700 border-purple-200",
  };

  const sizes = {
    sm: "px-2 py-0.5 text-xs font-medium",
    md: "px-2.5 py-0.5 text-xs font-semibold",
  };

  return (
    <span
      className={`inline-flex items-center rounded-full border ${variants[variant]} ${sizes[size]} ${className}`}
    >
      {children}
    </span>
  );
}

export function UnitStatusBadge({ status }: { status: UnitStatus }) {
  switch (status) {
    case "available":
      return <Badge variant="emerald">Tersedia</Badge>;
    case "reserved":
      return <Badge variant="amber">Dipesan</Badge>;
    case "borrowed":
      return <Badge variant="blue">Dipinjam</Badge>;
    case "maintenance":
      return <Badge variant="orange">Pemeliharaan</Badge>;
    case "retired":
      return <Badge variant="slate">Dipensiunkan</Badge>;
    default:
      return <Badge variant="slate">{status}</Badge>;
  }
}

export function RequestStatusBadge({ status }: { status: RequestStatus }) {
  switch (status) {
    case "pending":
      return <Badge variant="amber">Menunggu Persetujuan</Badge>;
    case "approved":
      return <Badge variant="emerald">Disetujui</Badge>;
    case "rejected":
      return <Badge variant="rose">Ditolak</Badge>;
    case "cancelled":
      return <Badge variant="slate">Dibatalkan</Badge>;
    default:
      return <Badge variant="slate">{status}</Badge>;
  }
}

export function LoanStatusBadge({ status }: { status: LoanStatus }) {
  switch (status) {
    case "pending_checkout":
      return <Badge variant="amber">Menunggu Ambil</Badge>;
    case "checked_out":
      return <Badge variant="blue">Sedang Dipinjam</Badge>;
    case "returned":
      return <Badge variant="slate">Dikembalikan</Badge>;
    case "overdue":
      return <Badge variant="rose">Terlambat</Badge>;
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
      return <Badge variant="emerald">Baru</Badge>;
    case "good":
      return <Badge variant="green">Baik</Badge>;
    case "fair":
      return <Badge variant="amber">Cukup</Badge>;
    case "poor":
      return <Badge variant="orange">Kurang</Badge>;
    case "damaged":
      return <Badge variant="rose">Rusak</Badge>;
    default:
      return <Badge variant="slate">{condition}</Badge>;
  }
}

export function RoleBadge({ role }: { role: Role }) {
  switch (role) {
    case "owner":
      return <Badge variant="purple">Owner</Badge>;
    case "admin":
      return <Badge variant="blue">Admin</Badge>;
    case "member":
      return <Badge variant="slate">Member</Badge>;
    default:
      return <Badge variant="slate">{role}</Badge>;
  }
}
