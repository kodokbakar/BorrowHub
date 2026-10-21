import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { inventoryApi } from "../api/endpoints";
import type { Category, Item } from "../api/types";
import { Badge, ConditionBadge, UnitStatusBadge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Modal } from "../components/ui/Modal";
import { BookingModal } from "./BookingModal";

interface ItemDetailModalProps {
  item: Item | null;
  category?: Category | null;
  organizationId: string;
  isOpen: boolean;
  onClose: () => void;
  isAdmin?: boolean;
}

export function ItemDetailModal({
  item,
  category,
  organizationId,
  isOpen,
  onClose,
  isAdmin,
}: ItemDetailModalProps) {
  const [isBookingOpen, setIsBookingOpen] = useState(false);

  // If admin, we can fetch physical units
  const { data: units = [], isLoading: isLoadingUnits } = useQuery({
    queryKey: ["units", item?.id],
    queryFn: () => (item ? inventoryApi.listUnits(item.id) : Promise.resolve([])),
    enabled: isOpen && !!item && !!isAdmin,
  });

  if (!item) return null;

  return (
    <>
      <Modal
        isOpen={isOpen && !isBookingOpen}
        onClose={onClose}
        title={item.name}
        description={category ? `Kategori: ${category.name}` : undefined}
        maxWidth="lg"
      >
        <div className="space-y-4">
          <div className="rounded-lg border border-border bg-surface-subtle p-3.5 space-y-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-fg-subtle">
              Deskripsi Peralatan
            </span>
            <p className="text-sm text-fg whitespace-pre-line leading-relaxed">
              {item.description || "Tidak ada deskripsi detail untuk peralatan ini."}
            </p>
          </div>

          <div className="flex items-center justify-between py-2.5 px-3 rounded-lg border border-border bg-surface">
            <span className="text-xs font-medium text-fg-muted">Status Peminjaman:</span>
            {item.is_lendable ? (
              <Badge variant="available">Dapat Dipinjam</Badge>
            ) : (
              <Badge variant="rose">Tidak Dapat Dipinjam</Badge>
            )}
          </div>

          {/* Units breakdown for admin */}
          {isAdmin && (
            <div className="mt-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-semibold uppercase tracking-wider text-fg-subtle">
                  Daftar Unit Fisik ({units.length})
                </span>
              </div>
              {isLoadingUnits ? (
                <LoadingSpinner size="sm" />
              ) : units.length === 0 ? (
                <p className="text-xs text-fg-subtle italic">
                  Belum ada unit fisik terdaftar untuk item ini.
                </p>
              ) : (
                <div className="max-h-48 overflow-y-auto divide-y divide-border rounded-lg border border-border">
                  {units.map((u) => (
                    <div
                      key={u.id}
                      className="flex items-center justify-between p-2.5 text-xs bg-surface"
                    >
                      <div>
                        <span className="font-mono font-tabular font-medium text-fg">
                          {u.barcode}
                        </span>
                        {u.serial_number && (
                          <span className="text-fg-subtle ml-2 font-mono font-tabular">
                            (SN: {u.serial_number})
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-1.5">
                        <UnitStatusBadge status={u.status} />
                        <ConditionBadge condition={u.condition} />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="flex justify-end gap-2 pt-4 border-t border-border">
            <Button variant="ghost" size="md" className="min-h-[44px]" onClick={onClose}>
              Tutup
            </Button>
            {item.is_lendable && (
              <Button
                variant="primary"
                size="md"
                className="min-h-[44px]"
                onClick={() => {
                  setIsBookingOpen(true);
                }}
              >
                Ajukan Pinjam
              </Button>
            )}
          </div>
        </div>
      </Modal>

      {/* Booking Form Modal */}
      <BookingModal
        item={item}
        organizationId={organizationId}
        isOpen={isBookingOpen}
        onClose={() => {
          setIsBookingOpen(false);
          onClose();
        }}
        onSuccess={() => {
          setIsBookingOpen(false);
          onClose();
        }}
      />
    </>
  );
}
