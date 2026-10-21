import React, { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { inventoryApi } from "../api/endpoints";
import type {
  Category,
  ConditionLog,
  Item,
  Unit,
  UnitCondition,
  UnitStatus,
} from "../api/types";
import { AlertToast } from "../components/ui/AlertToast";
import {
  Badge,
  ConditionBadge,
  UnitStatusBadge,
} from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { Input } from "../components/ui/Input";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Modal } from "../components/ui/Modal";
import { Select } from "../components/ui/Select";
import { useAuth } from "../context/AuthContext";

export function AdminInventoryView() {
  const { activeOrg } = useAuth();
  const orgId = activeOrg?.id;

  const [activeTab, setActiveTab] = useState<"items" | "categories">("items");
  const queryClient = useQueryClient();
  const [feedback, setFeedback] = useState<{
    type: "success" | "error";
    message: string;
    requestId?: string;
  } | null>(null);

  const handleApiError = (err: unknown, fallback: string) => {
    if (err instanceof ApiError) {
      if (err.fields) {
        console.error("Inventory API validation fields:", err.fields);
      }
      const fieldDetails = err.fields
        ? Object.entries(err.fields)
            .map(([k, v]) => `${k}: ${v}`)
            .join(", ")
        : "";
      setFeedback({
        type: "error",
        message: err.message
          ? fieldDetails
            ? `${err.message} (${fieldDetails})`
            : err.message
          : fallback,
        requestId: err.requestId,
      });
    } else {
      setFeedback({ type: "error", message: fallback });
    }
  };

  // Categories query
  const {
    data: categories = [],
    isLoading: isLoadingCats,
    refetch: refetchCats,
  } = useQuery({
    queryKey: ["categories", orgId],
    queryFn: () => (orgId ? inventoryApi.listCategories(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  // Items query
  const {
    data: items = [],
    isLoading: isLoadingItems,
    refetch: refetchItems,
  } = useQuery({
    queryKey: ["items", orgId],
    queryFn: () => (orgId ? inventoryApi.listItems(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  // ---------------- CATEGORY STATE ----------------
  const [isCategoryModalOpen, setIsCategoryModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [catName, setCatName] = useState("");
  const [catSlug, setCatSlug] = useState("");
  const [catDesc, setCatDesc] = useState("");
  const [isSubmittingCat, setIsSubmittingCat] = useState(false);

  // ---------------- ITEM STATE ----------------
  const [isItemModalOpen, setIsItemModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<Item | null>(null);
  const [itemName, setItemName] = useState("");
  const [itemCatId, setItemCatId] = useState("");
  const [itemDesc, setItemDesc] = useState("");
  const [itemIsLendable, setItemIsLendable] = useState(true);
  const [isSubmittingItem, setIsSubmittingItem] = useState(false);

  // ---------------- UNIT MANAGEMENT STATE ----------------
  const [selectedItemForUnits, setSelectedItemForUnits] = useState<Item | null>(
    null,
  );
  const [isAddUnitModalOpen, setIsAddUnitModalOpen] = useState(false);
  const [unitBarcode, setUnitBarcode] = useState("");
  const [unitSerial, setUnitSerial] = useState("");
  const [unitCondition, setUnitCondition] = useState<UnitCondition>("good");
  const [unitNotes, setUnitNotes] = useState("");
  const [isSubmittingUnit, setIsSubmittingUnit] = useState(false);

  // Unit condition modal
  const [editingUnitCondition, setEditingUnitCondition] = useState<Unit | null>(
    null,
  );
  const [newCondition, setNewCondition] = useState<UnitCondition>("good");
  const [conditionReason, setConditionReason] = useState("");
  const [isSubmittingCondition, setIsSubmittingCondition] = useState(false);

  // Unit condition logs modal
  const [viewingLogsUnit, setViewingLogsUnit] = useState<Unit | null>(null);

  // Units query for selected item
  const {
    data: units = [],
    isLoading: isLoadingUnits,
    refetch: refetchUnits,
  } = useQuery({
    queryKey: ["admin-units", selectedItemForUnits?.id],
    queryFn: () =>
      selectedItemForUnits
        ? inventoryApi.listUnits(selectedItemForUnits.id)
        : Promise.resolve([]),
    enabled: !!selectedItemForUnits,
  });

  // Condition logs query
  const { data: conditionLogs = [], isLoading: isLoadingLogs } = useQuery({
    queryKey: ["condition-logs", viewingLogsUnit?.id],
    queryFn: () =>
      viewingLogsUnit
        ? inventoryApi.getConditionLogs(viewingLogsUnit.id)
        : Promise.resolve([]),
    enabled: !!viewingLogsUnit,
  });

  if (!orgId) {
    return (
      <div className="py-12">
        <EmptyState
          title="Tidak ada organisasi aktif"
          description="Pilih organisasi terlebih dahulu untuk mengelola inventaris."
        />
      </div>
    );
  }

  // ---------------- HANDLERS ----------------
  const handleOpenCreateCategory = () => {
    setEditingCategory(null);
    setCatName("");
    setCatSlug("");
    setCatDesc("");
    setIsCategoryModalOpen(true);
  };

  const handleOpenEditCategory = (cat: Category) => {
    setEditingCategory(cat);
    setCatName(cat.name);
    setCatSlug(cat.slug);
    setCatDesc(cat.description || "");
    setIsCategoryModalOpen(true);
  };

  const handleSaveCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!catName.trim() || !catSlug.trim()) return;

    setIsSubmittingCat(true);
    try {
      if (editingCategory) {
        await inventoryApi.updateCategory(editingCategory.id, {
          name: catName.trim(),
          slug: catSlug.trim(),
          description: catDesc.trim() || undefined,
        });
        setFeedback({ type: "success", message: "Kategori berhasil diperbarui." });
      } else {
        await inventoryApi.createCategory(orgId, {
          name: catName.trim(),
          slug: catSlug.trim(),
          description: catDesc.trim() || undefined,
        });
        setFeedback({ type: "success", message: "Kategori berhasil ditambahkan." });
      }
      setIsCategoryModalOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["categories", orgId] });
      void refetchCats();
    } catch (err: unknown) {
      handleApiError(err, "Gagal menyimpan kategori.");
    } finally {
      setIsSubmittingCat(false);
    }
  };

  const handleDeleteCategory = async (categoryId: string) => {
    if (!confirm("Yakin ingin menghapus kategori ini?")) return;
    try {
      await inventoryApi.deleteCategory(categoryId);
      setFeedback({ type: "success", message: "Kategori berhasil dihapus." });
      await queryClient.invalidateQueries({ queryKey: ["categories", orgId] });
      void refetchCats();
    } catch (err: unknown) {
      handleApiError(err, "Gagal menghapus kategori. Pastikan tidak ada item yang terkait.");
    }
  };

  const handleOpenCreateItem = () => {
    setEditingItem(null);
    setItemName("");
    setItemCatId(categories[0]?.id || "");
    setItemDesc("");
    setItemIsLendable(true);
    setIsItemModalOpen(true);
  };

  const handleOpenEditItem = (item: Item) => {
    setEditingItem(item);
    setItemName(item.name);
    setItemCatId(item.category_id);
    setItemDesc(item.description || "");
    setItemIsLendable(item.is_lendable);
    setIsItemModalOpen(true);
  };

  const handleSaveItem = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!itemName.trim() || !itemCatId) return;

    setIsSubmittingItem(true);
    try {
      if (editingItem) {
        await inventoryApi.updateItem(editingItem.id, {
          name: itemName.trim(),
          category_id: itemCatId,
          description: itemDesc.trim() || undefined,
          is_lendable: itemIsLendable,
        });
        setFeedback({ type: "success", message: "Item berhasil diperbarui." });
      } else {
        await inventoryApi.createItem(orgId, {
          name: itemName.trim(),
          category_id: itemCatId,
          description: itemDesc.trim() || undefined,
          is_lendable: itemIsLendable,
        });
        setFeedback({ type: "success", message: "Item berhasil dibuat." });
      }
      setIsItemModalOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      void refetchItems();
    } catch (err: unknown) {
      handleApiError(err, "Gagal menyimpan item.");
    } finally {
      setIsSubmittingItem(false);
    }
  };

  const handleDeleteItem = async (itemId: string) => {
    if (!confirm("Yakin ingin menghapus peralatan ini?")) return;
    try {
      await inventoryApi.deleteItem(itemId);
      setFeedback({ type: "success", message: "Item berhasil dihapus." });
      if (selectedItemForUnits?.id === itemId) {
        setSelectedItemForUnits(null);
      }
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      void refetchItems();
    } catch (err: unknown) {
      handleApiError(err, "Gagal menghapus item.");
    }
  };

  const handleAddUnit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedItemForUnits || !unitBarcode.trim()) return;

    setIsSubmittingUnit(true);
    try {
      await inventoryApi.createUnit(selectedItemForUnits.id, {
        barcode: unitBarcode.trim(),
        serial_number: unitSerial.trim() || undefined,
        condition: unitCondition,
        notes: unitNotes.trim() || undefined,
      });
      setFeedback({ type: "success", message: "Unit fisik berhasil ditambahkan." });
      setIsAddUnitModalOpen(false);
      setUnitBarcode("");
      setUnitSerial("");
      setUnitNotes("");
      await queryClient.invalidateQueries({ queryKey: ["admin-units", selectedItemForUnits.id] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      void refetchUnits();
    } catch (err: unknown) {
      handleApiError(err, "Gagal menambahkan unit fisik.");
    } finally {
      setIsSubmittingUnit(false);
    }
  };

  const handleToggleMaintenance = async (unit: Unit) => {
    const nextStatus: UnitStatus =
      unit.status === "maintenance" ? "available" : "maintenance";
    const reason =
      nextStatus === "maintenance"
        ? "Masuk pemeliharaan berkala"
        : "Selesai pemeliharaan, siap digunakan";

    try {
      await inventoryApi.updateUnit(unit.id, {
        status: nextStatus,
        reason,
      });
      setFeedback({
        type: "success",
        message: `Status unit berhasil diubah ke ${nextStatus}.`,
      });
      await queryClient.invalidateQueries({ queryKey: ["admin-units", unit.id] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      void refetchUnits();
    } catch (err: unknown) {
      handleApiError(err, "Gagal memperbarui status pemeliharaan unit.");
    }
  };

  const handleSaveCondition = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingUnitCondition) return;

    setIsSubmittingCondition(true);
    try {
      await inventoryApi.updateUnit(editingUnitCondition.id, {
        condition: newCondition,
        reason: conditionReason.trim() || undefined,
      });
      setFeedback({ type: "success", message: "Kondisi unit berhasil diperbarui." });
      setEditingUnitCondition(null);
      await queryClient.invalidateQueries({ queryKey: ["admin-units", editingUnitCondition.id] });
      await queryClient.invalidateQueries({ queryKey: ["items", orgId] });
      void refetchUnits();
    } catch (err: unknown) {
      handleApiError(err, "Gagal memperbarui kondisi unit.");
    } finally {
      setIsSubmittingCondition(false);
    }
  };

  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-fg">
            Admin Inventaris & Peralatan
          </h1>
          <p className="mt-1 text-sm text-fg-muted">
            Kelola kategori, master peralatan, dan unit fisik barcode untuk{" "}
            <span className="font-semibold text-fg">{activeOrg.name}</span>.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {activeTab === "categories" ? (
            <Button
              variant="primary"
              className="min-h-[44px]"
              onClick={handleOpenCreateCategory}
            >
              + Kategori Baru
            </Button>
          ) : (
            <Button
              variant="primary"
              className="min-h-[44px]"
              onClick={handleOpenCreateItem}
              disabled={categories.length === 0}
            >
              + Item Peralatan
            </Button>
          )}
        </div>
      </div>

      {feedback && (
        <AlertToast
          type={feedback.type}
          message={feedback.message}
          requestId={feedback.requestId}
          onClose={() => setFeedback(null)}
        />
      )}

      {/* Subtabs with accessibility */}
      <div
        className="flex border-b border-border"
        role="tablist"
        aria-label="Manajemen Inventaris"
      >
        <button
          type="button"
          role="tab"
          id="tab-items"
          aria-controls="panel-items"
          aria-selected={activeTab === "items"}
          onClick={() => setActiveTab("items")}
          className={`min-h-[44px] inline-flex items-center py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "items"
              ? "border-accent text-accent font-bold"
              : "border-transparent text-fg-muted hover:text-fg"
          }`}
        >
          Master Peralatan & Unit ({items.length})
        </button>
        <button
          type="button"
          role="tab"
          id="tab-categories"
          aria-controls="panel-categories"
          aria-selected={activeTab === "categories"}
          onClick={() => setActiveTab("categories")}
          className={`min-h-[44px] inline-flex items-center py-3 px-4 text-sm font-semibold border-b-2 -mb-px transition-colors cursor-pointer ${
            activeTab === "categories"
              ? "border-accent text-accent font-bold"
              : "border-transparent text-fg-muted hover:text-fg"
          }`}
        >
          Kategori ({categories.length})
        </button>
      </div>

      {/* CATEGORIES TAB PANEL */}
      {activeTab === "categories" && (
        <div
          id="panel-categories"
          role="tabpanel"
          aria-labelledby="tab-categories"
        >
          {isLoadingCats ? (
            <LoadingSpinner />
          ) : categories.length === 0 ? (
            <EmptyState
              title="Belum ada kategori"
              description="Tambahkan kategori pertama untuk mengelompokkan peralatan."
              actionLabel="+ Tambah Kategori"
              onAction={handleOpenCreateCategory}
            />
          ) : (
            <div className="overflow-hidden rounded-xl border border-border bg-surface shadow-xs">
              <table className="min-w-full divide-y divide-border text-left text-sm">
                <thead className="bg-surface-subtle text-xs font-semibold text-fg-muted">
                  <tr>
                    <th className="py-3 px-4">Nama Kategori</th>
                    <th className="py-3 px-4">Slug</th>
                    <th className="py-3 px-4">Deskripsi</th>
                    <th className="py-3 px-4 text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-subtle">
                  {categories.map((cat) => (
                    <tr
                      key={cat.id}
                      className="hover:bg-surface-hover/70 transition-colors"
                    >
                      <td className="py-3 px-4 font-medium text-fg">
                        {cat.name}
                      </td>
                      <td className="py-3 px-4 font-mono font-tabular text-xs text-fg-muted">
                        {cat.slug}
                      </td>
                      <td className="py-3 px-4 text-fg-muted text-xs max-w-xs truncate">
                        {cat.description || "-"}
                      </td>
                      <td className="py-3 px-4 text-right space-x-2">
                        <Button
                          variant="ghost"
                          className="min-h-[44px]"
                          onClick={() => handleOpenEditCategory(cat)}
                        >
                          Ubah
                        </Button>
                        <Button
                          variant="ghost"
                          className="min-h-[44px] text-status-overdue hover:text-status-overdue hover:bg-status-overdue-bg"
                          onClick={() => handleDeleteCategory(cat.id)}
                        >
                          Hapus
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ITEMS & UNITS TAB PANEL */}
      {activeTab === "items" && (
        <div
          id="panel-items"
          role="tabpanel"
          aria-labelledby="tab-items"
          className="grid grid-cols-1 lg:grid-cols-12 gap-6"
        >
          {/* Items list */}
          <div
            className={
              selectedItemForUnits ? "lg:col-span-5" : "lg:col-span-12"
            }
          >
            {isLoadingItems ? (
              <LoadingSpinner />
            ) : items.length === 0 ? (
              <EmptyState
                title="Belum ada item peralatan"
                description="Tambahkan item peralatan untuk mulai mencatat inventaris."
                actionLabel="+ Item Peralatan"
                onAction={handleOpenCreateItem}
              />
            ) : (
              <div className="space-y-3">
                {items.map((item) => {
                  const isSelected = selectedItemForUnits?.id === item.id;
                  const cat = categories.find((c) => c.id === item.category_id);
                  return (
                    <div
                      key={item.id}
                      onClick={() => setSelectedItemForUnits(item)}
                      className={`p-4 rounded-xl border transition-all cursor-pointer bg-surface ${
                        isSelected
                          ? "border-accent ring-2 ring-accent/10 shadow-sm"
                          : "border-border hover:border-border-strong shadow-xs"
                      }`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <div className="flex items-center gap-2">
                            <h3 className="font-semibold text-fg text-sm">
                              {item.name}
                            </h3>
                            {cat && (
                              <span className="text-[10px] font-medium bg-surface-subtle text-fg-muted px-1.5 py-0.5 rounded border border-border-subtle">
                                {cat.name}
                              </span>
                            )}
                          </div>
                          <p className="mt-1 text-xs text-fg-muted line-clamp-1">
                            {item.description || "Tidak ada deskripsi"}
                          </p>
                        </div>
                        <div className="flex items-center gap-1 shrink-0">
                          {item.is_lendable ? (
                            <Badge variant="available" size="sm">
                              Dapat Dipinjam
                            </Badge>
                          ) : (
                            <Badge variant="overdue" size="sm">
                              Non-Pinjam
                            </Badge>
                          )}
                        </div>
                      </div>

                      <div className="mt-3 flex items-center justify-between text-xs pt-2 border-t border-border-subtle">
                        <span className="text-fg-subtle font-mono font-tabular text-[11px]">
                          ID: {item.id.slice(0, 8)}
                        </span>
                        <div
                          className="flex items-center gap-1.5"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <Button
                            variant="ghost"
                            className="min-h-[44px]"
                            onClick={() => handleOpenEditItem(item)}
                          >
                            Ubah
                          </Button>
                          <Button
                            variant="ghost"
                            className="min-h-[44px] text-status-overdue hover:text-status-overdue hover:bg-status-overdue-bg"
                            onClick={() => handleDeleteItem(item.id)}
                          >
                            Hapus
                          </Button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Physical units panel */}
          {selectedItemForUnits && (
            <div className="lg:col-span-7 bg-surface rounded-2xl border border-border p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-border-subtle">
                <div>
                  <h2 className="font-semibold text-fg">
                    Unit Fisik: {selectedItemForUnits.name}
                  </h2>
                  <p className="text-xs text-fg-muted font-tabular">
                    Total: {units.length} unit fisik terdaftar
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="primary"
                    className="min-h-[44px]"
                    onClick={() => {
                      setUnitBarcode("");
                      setUnitSerial("");
                      setUnitCondition("good");
                      setUnitNotes("");
                      setIsAddUnitModalOpen(true);
                    }}
                  >
                    + Unit Fisik
                  </Button>
                  <Button
                    variant="ghost"
                    className="min-h-[44px]"
                    onClick={() => setSelectedItemForUnits(null)}
                    title="Tutup Panel"
                  >
                    ✕
                  </Button>
                </div>
              </div>

              {isLoadingUnits ? (
                <LoadingSpinner />
              ) : units.length === 0 ? (
                <EmptyState
                  title="Belum ada unit fisik"
                  description="Tambahkan barcode unit fisik pertama untuk item ini agar bisa dipinjamkan."
                  actionLabel="+ Tambah Unit Fisik"
                  onAction={() => setIsAddUnitModalOpen(true)}
                />
              ) : (
                <div className="divide-y divide-border-subtle max-h-[520px] overflow-y-auto">
                  {units.map((unit) => (
                    <div
                      key={unit.id}
                      className="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono font-tabular font-bold text-fg text-sm">
                            {unit.barcode}
                          </span>
                          <UnitStatusBadge status={unit.status} />
                          <ConditionBadge condition={unit.condition} />
                        </div>
                        {unit.serial_number && (
                          <p className="text-xs text-fg-muted font-mono font-tabular">
                            Serial: {unit.serial_number}
                          </p>
                        )}
                        {unit.notes && (
                          <p className="text-xs text-fg-subtle italic">
                            Catatan: {unit.notes}
                          </p>
                        )}
                      </div>

                      <div className="flex items-center gap-1.5 shrink-0">
                        <Button
                          variant="outline"
                          className={`min-h-[44px] ${
                            unit.status === "maintenance"
                              ? "text-status-available border-status-available-border hover:bg-status-available-bg"
                              : "text-status-reserved border-status-reserved-border hover:bg-status-reserved-bg"
                          }`}
                          onClick={() => handleToggleMaintenance(unit)}
                        >
                          {unit.status === "maintenance"
                            ? "Selesai Rawat"
                            : "Set Perawatan"}
                        </Button>
                        <Button
                          variant="ghost"
                          className="min-h-[44px]"
                          onClick={() => {
                            setEditingUnitCondition(unit);
                            setNewCondition(unit.condition);
                            setConditionReason("");
                          }}
                        >
                          Kondisi
                        </Button>
                        <Button
                          variant="ghost"
                          className="min-h-[44px]"
                          onClick={() => setViewingLogsUnit(unit)}
                        >
                          Riwayat
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Category Modal */}
      <Modal
        isOpen={isCategoryModalOpen}
        onClose={() => setIsCategoryModalOpen(false)}
        title={editingCategory ? "Ubah Kategori" : "Tambah Kategori Baru"}
      >
        <form onSubmit={handleSaveCategory} className="space-y-4">
          <Input
            label="Nama Kategori"
            placeholder="Contoh: Kamera & Lensa"
            value={catName}
            onChange={(e) => {
              setCatName(e.target.value);
              if (!editingCategory) {
                setCatSlug(
                  e.target.value
                    .toLowerCase()
                    .replace(/[^a-z0-9]+/g, "-")
                    .replace(/^-|-$/g, ""),
                );
              }
            }}
            required
          />
          <Input
            label="Slug URL"
            placeholder="kamera-lensa"
            value={catSlug}
            onChange={(e) => setCatSlug(e.target.value)}
            className="font-mono font-tabular"
            required
          />
          <div>
            <label className="block text-sm font-medium text-fg mb-1">
              Deskripsi (Opsional)
            </label>
            <textarea
              rows={2}
              className="block w-full rounded-lg border border-border-strong bg-surface p-2.5 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:ring-2 focus:ring-accent focus:border-accent"
              placeholder="Keterangan kategori peralatan"
              value={catDesc}
              onChange={(e) => setCatDesc(e.target.value)}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2 border-t border-border-subtle">
            <Button
              type="button"
              variant="ghost"
              className="min-h-[44px]"
              onClick={() => setIsCategoryModalOpen(false)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              className="min-h-[44px]"
              isLoading={isSubmittingCat}
            >
              Simpan
            </Button>
          </div>
        </form>
      </Modal>

      {/* Item Modal */}
      <Modal
        isOpen={isItemModalOpen}
        onClose={() => setIsItemModalOpen(false)}
        title={editingItem ? "Ubah Peralatan" : "Tambah Peralatan Baru"}
      >
        <form onSubmit={handleSaveItem} className="space-y-4">
          <Input
            label="Nama Peralatan"
            placeholder="Contoh: Sony A7 IV Body"
            value={itemName}
            onChange={(e) => setItemName(e.target.value)}
            required
          />
          <Select
            label="Kategori"
            value={itemCatId}
            onChange={(e) => setItemCatId(e.target.value)}
            options={categories.map((c) => ({ value: c.id, label: c.name }))}
            required
          />
          <div>
            <label className="block text-sm font-medium text-fg mb-1">
              Deskripsi (Opsional)
            </label>
            <textarea
              rows={3}
              className="block w-full rounded-lg border border-border-strong bg-surface p-2.5 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:ring-2 focus:ring-accent focus:border-accent"
              placeholder="Spesifikasi atau informasi penting peralatan"
              value={itemDesc}
              onChange={(e) => setItemDesc(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-2 min-h-[44px]">
            <input
              type="checkbox"
              id="is-lendable"
              checked={itemIsLendable}
              onChange={(e) => setItemIsLendable(e.target.checked)}
              className="h-5 w-5 rounded border-border-strong text-accent focus:ring-accent cursor-pointer"
            />
            <label
              htmlFor="is-lendable"
              className="text-sm font-medium text-fg cursor-pointer select-none"
            >
              Dapat diajukan pinjam oleh anggota (Lendable)
            </label>
          </div>
          <div className="flex justify-end gap-2 pt-2 border-t border-border-subtle">
            <Button
              type="button"
              variant="ghost"
              className="min-h-[44px]"
              onClick={() => setIsItemModalOpen(false)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              className="min-h-[44px]"
              isLoading={isSubmittingItem}
            >
              Simpan
            </Button>
          </div>
        </form>
      </Modal>

      {/* Add Unit Modal */}
      <Modal
        isOpen={isAddUnitModalOpen}
        onClose={() => setIsAddUnitModalOpen(false)}
        title={`Tambah Unit Fisik: ${selectedItemForUnits?.name}`}
      >
        <form onSubmit={handleAddUnit} className="space-y-4">
          <Input
            label="Kode Barcode / Aset"
            placeholder="Contoh: CAM-001"
            value={unitBarcode}
            onChange={(e) => setUnitBarcode(e.target.value)}
            className="font-mono font-tabular"
            required
          />
          <Input
            label="Nomor Seri / SN (Opsional)"
            placeholder="SN-12345678"
            value={unitSerial}
            onChange={(e) => setUnitSerial(e.target.value)}
            className="font-mono font-tabular"
          />
          <Select
            label="Kondisi Awal"
            value={unitCondition}
            onChange={(e) => setUnitCondition(e.target.value as UnitCondition)}
            options={[
              { value: "new", label: "Baru (New)" },
              { value: "good", label: "Baik (Good)" },
              { value: "fair", label: "Cukup (Fair)" },
              { value: "poor", label: "Kurang (Poor)" },
              { value: "damaged", label: "Rusak (Damaged)" },
            ]}
          />
          <Input
            label="Catatan Unit"
            placeholder="Kelengkapan box, tas, dll"
            value={unitNotes}
            onChange={(e) => setUnitNotes(e.target.value)}
          />
          <div className="flex justify-end gap-2 pt-2 border-t border-border-subtle">
            <Button
              type="button"
              variant="ghost"
              className="min-h-[44px]"
              onClick={() => setIsAddUnitModalOpen(false)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              className="min-h-[44px]"
              isLoading={isSubmittingUnit}
            >
              Tambahkan Unit
            </Button>
          </div>
        </form>
      </Modal>

      {/* Edit Condition Modal */}
      <Modal
        isOpen={!!editingUnitCondition}
        onClose={() => setEditingUnitCondition(null)}
        title={`Ubah Kondisi Unit: ${editingUnitCondition?.barcode}`}
      >
        <form onSubmit={handleSaveCondition} className="space-y-4">
          <Select
            label="Kondisi Unit Sekarang"
            value={newCondition}
            onChange={(e) => setNewCondition(e.target.value as UnitCondition)}
            options={[
              { value: "new", label: "Baru (New)" },
              { value: "good", label: "Baik (Good)" },
              { value: "fair", label: "Cukup (Fair)" },
              { value: "poor", label: "Kurang (Poor)" },
              { value: "damaged", label: "Rusak (Damaged)" },
            ]}
          />
          <Input
            label="Alasan Perubahan Kondisi"
            placeholder="Contoh: Tergores saat pemakaian di lapangan"
            value={conditionReason}
            onChange={(e) => setConditionReason(e.target.value)}
          />
          <div className="flex justify-end gap-2 pt-2 border-t border-border-subtle">
            <Button
              type="button"
              variant="ghost"
              className="min-h-[44px]"
              onClick={() => setEditingUnitCondition(null)}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="primary"
              className="min-h-[44px]"
              isLoading={isSubmittingCondition}
            >
              Perbarui Kondisi
            </Button>
          </div>
        </form>
      </Modal>

      {/* Condition Logs Modal */}
      <Modal
        isOpen={!!viewingLogsUnit}
        onClose={() => setViewingLogsUnit(null)}
        title={`Riwayat Kondisi Unit: ${viewingLogsUnit?.barcode}`}
        maxWidth="lg"
      >
        <div className="space-y-3">
          {isLoadingLogs ? (
            <LoadingSpinner size="sm" />
          ) : conditionLogs.length === 0 ? (
            <p className="text-xs text-fg-muted italic py-4 text-center">
              Belum ada riwayat perubahan kondisi yang tercatat.
            </p>
          ) : (
            <div className="max-h-72 overflow-y-auto divide-y divide-border-subtle">
              {conditionLogs.map((log) => (
                <div key={log.id} className="py-2.5 text-xs">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-1.5">
                      {log.previous_condition && (
                        <>
                          <ConditionBadge condition={log.previous_condition} />
                          <span className="text-fg-subtle">→</span>
                        </>
                      )}
                      <ConditionBadge condition={log.new_condition} />
                    </div>
                    <span className="text-fg-subtle font-mono font-tabular">
                      {new Date(log.created_at).toLocaleString("id-ID")}
                    </span>
                  </div>
                  {log.reason && (
                    <p className="mt-1 text-fg bg-surface-subtle p-2 rounded border border-border-subtle">
                      Alasan: {log.reason}
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
          <div className="flex justify-end pt-3 border-t border-border-subtle">
            <Button
              variant="ghost"
              className="min-h-[44px]"
              onClick={() => setViewingLogsUnit(null)}
            >
              Tutup
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
