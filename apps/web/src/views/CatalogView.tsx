import React, { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { inventoryApi } from "../api/endpoints";
import type { Category, Item } from "../api/types";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { CreateOrgModal } from "../components/CreateOrgModal";
import { EmptyState } from "../components/ui/EmptyState";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useAuth } from "../context/AuthContext";
import { ItemDetailModal } from "./ItemDetailModal";
export function CatalogView() {
  const { activeOrg, organizations, isAdmin } = useAuth();
  const [selectedCategoryId, setSelectedCategoryId] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [selectedItem, setSelectedItem] = useState<Item | null>(null);
  const [isCreateOrgOpen, setIsCreateOrgOpen] = useState(false);
  const orgId = activeOrg?.id;

  // Fetch categories
  const { data: categories = [], isLoading: isLoadingCategories } = useQuery({
    queryKey: ["categories", orgId],
    queryFn: () => (orgId ? inventoryApi.listCategories(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  // Fetch items
  const {
    data: items = [],
    isLoading: isLoadingItems,
    refetch: refetchItems,
  } = useQuery({
    queryKey: ["items", orgId],
    queryFn: () => (orgId ? inventoryApi.listItems(orgId) : Promise.resolve([])),
    enabled: !!orgId,
  });

  const categoryMap = useMemo(() => {
    const map = new Map<string, Category>();
    categories.forEach((c) => map.set(c.id, c));
    return map;
  }, [categories]);

  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      const matchCategory =
        selectedCategoryId === "all" || item.category_id === selectedCategoryId;
      const matchSearch =
        searchQuery.trim() === "" ||
        item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (item.description &&
          item.description.toLowerCase().includes(searchQuery.toLowerCase()));
      return matchCategory && matchSearch;
    });
  }, [items, selectedCategoryId, searchQuery]);

  if (!orgId) {
    return (
      <div className="py-12">
        <EmptyState
          title="Tidak ada organisasi aktif"
          description={
            organizations.length === 0
              ? "Anda belum terdaftar dalam organisasi apapun. Buat organisasi baru untuk mulai mengelola atau meminjam peralatan."
              : "Silakan pilih organisasi terlebih dahulu untuk melihat katalog peralatan."
          }
          actionLabel="+ Buat Organisasi Baru"
          onAction={() => setIsCreateOrgOpen(true)}
        />
        <CreateOrgModal
          isOpen={isCreateOrgOpen}
          onClose={() => setIsCreateOrgOpen(false)}
        />
      </div>
    );
  }
  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Title & Filter Bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">
            Katalog Peralatan
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Jelajahi dan ajukan peminjaman peralatan operasional organisasi{" "}
            <span className="font-semibold text-slate-800">{activeOrg.name}</span>.
          </p>
        </div>

        {/* Search input */}
        <div className="w-full md:w-72">
          <div className="relative">
            <span className="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none text-slate-400">
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </span>
            <input
              type="text"
              placeholder="Cari peralatan..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="block w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-slate-300 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-900"
            />
          </div>
        </div>
      </div>

      {/* Category Pills Filter */}
      {!isLoadingCategories && categories.length > 0 && (
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1">
          <button
            type="button"
            onClick={() => setSelectedCategoryId("all")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-full transition-colors cursor-pointer ${
              selectedCategoryId === "all"
                ? "bg-slate-900 text-white"
                : "bg-slate-100 text-slate-700 hover:bg-slate-200"
            }`}
          >
            Semua ({items.length})
          </button>
          {categories.map((cat) => {
            const count = items.filter((i) => i.category_id === cat.id).length;
            const isSelected = selectedCategoryId === cat.id;
            return (
              <button
                key={cat.id}
                type="button"
                onClick={() => setSelectedCategoryId(cat.id)}
                className={`whitespace-nowrap px-3 py-1.5 text-xs font-semibold rounded-full transition-colors cursor-pointer ${
                  isSelected
                    ? "bg-slate-900 text-white"
                    : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                }`}
              >
                {cat.name} ({count})
              </button>
            );
          })}
        </div>
      )}

      {/* Items Grid */}
      {isLoadingItems ? (
        <LoadingSpinner />
      ) : filteredItems.length === 0 ? (
        <EmptyState
          title="Tidak ada peralatan ditemukan"
          description={
            searchQuery || selectedCategoryId !== "all"
              ? "Coba ubah kata kunci pencarian atau pilih kategori lain."
              : "Belum ada item peralatan yang terdaftar di katalog."
          }
          actionLabel={searchQuery ? "Bersihkan Pencarian" : undefined}
          onAction={() => {
            setSearchQuery("");
            setSelectedCategoryId("all");
          }}
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredItems.map((item) => {
            const category = categoryMap.get(item.category_id);
            return (
              <div
                key={item.id}
                className="group flex flex-col justify-between rounded-2xl border border-slate-200 bg-white p-5 shadow-xs hover:border-slate-300 hover:shadow-md transition-all"
              >
                <div>
                  <div className="flex items-center justify-between gap-2">
                    {category && (
                      <span className="text-xs font-medium text-slate-500 bg-slate-100 px-2 py-0.5 rounded-md">
                        {category.name}
                      </span>
                    )}
                    {item.is_lendable ? (
                      <Badge variant="emerald" size="sm">
                        Tersedia
                      </Badge>
                    ) : (
                      <Badge variant="rose" size="sm">
                        Tidak Tersedia
                      </Badge>
                    )}
                  </div>

                  <h3 className="mt-3 text-base font-semibold text-slate-900 group-hover:text-slate-800">
                    {item.name}
                  </h3>

                  <p className="mt-1.5 text-xs text-slate-600 line-clamp-2 leading-relaxed">
                    {item.description || "Tidak ada deskripsi detail."}
                  </p>
                </div>

                <div className="mt-5 flex items-center justify-between pt-3 border-t border-slate-100">
                  <span className="text-[11px] text-slate-400">
                    ID: {item.id.slice(0, 8)}...
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setSelectedItem(item)}
                  >
                    {item.is_lendable ? "Ajukan Pinjam" : "Lihat Detail"}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Item Detail / Booking Modal */}
      {selectedItem && (
        <ItemDetailModal
          item={selectedItem}
          category={categoryMap.get(selectedItem.category_id)}
          organizationId={orgId}
          isOpen={!!selectedItem}
          onClose={() => {
            setSelectedItem(null);
            void refetchItems();
          }}
          isAdmin={isAdmin}
        />
      )}
    </div>
  );
}
