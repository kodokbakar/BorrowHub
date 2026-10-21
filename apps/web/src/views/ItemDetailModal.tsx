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
          <div>
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
              Deskripsi Peralatan
            </span>
            <p className="mt-1 text-sm text-slate-700 whitespace-pre-line">
              {item.description || "Tidak ada deskripsi detail untuk peralatan ini."}
            </p>
          </div>

          <div className="flex items-center gap-3 py-2 border-y border-slate-100">
            <span className="text-xs text-slate-500">Status Peminjaman:</span>
            {item.is_lendable ? (
              <Badge variant="emerald">Dapat Dipinjam</Badge>
            ) : (
              <Badge variant="rose">Tidak Dapat Dipinjam</Badge>
            )}
          </div>

          {/* Units breakdown for admin */}
          {isAdmin && (
            <div className="mt-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                  Daftar Unit Fisik ({units.length})
                </span>
              </div>
              {isLoadingUnits ? (
                <LoadingSpinner size="sm" />
              ) : units.length === 0 ? (
                <p className="text-xs text-slate-400 italic">
                  Belum ada unit fisik terdaftar untuk item ini.
                </p>
              ) : (
                <div className="max-h-48 overflow-y-auto divide-y divide-slate-100 rounded-lg border border-slate-200">
                  {units.map((u) => (
                    <div
                      key={u.id}
                      className="flex items-center justify-between p-2.5 text-xs bg-white"
                    >
                      <div>
                        <span className="font-mono font-medium text-slate-800">
                          {u.barcode}
                        </span>
                        {u.serial_number && (
                          <span className="text-slate-400 ml-2">
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

          <div className="flex justify-end gap-2 pt-4 border-t border-slate-100">
            <Button variant="ghost" onClick={onClose}>
              Tutup
            </Button>
            {item.is_lendable && (
              <Button
                variant="primary"
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
