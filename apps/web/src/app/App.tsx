import React, { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Header } from "../components/Header";
import { Navigation } from "../components/Navigation";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { AuthProvider, useAuth } from "../context/AuthContext";
import { AdminInventoryView } from "../views/AdminInventoryView";
import { AdminLendingView } from "../views/AdminLendingView";
import { AuthView } from "../views/AuthView";
import { CatalogView } from "../views/CatalogView";
import { MyLoansView } from "../views/MyLoansView";
import { NotificationsView } from "../views/NotificationsView";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 1000 * 10,
    },
  },
});

function AppContent() {
  const { isAuthenticated, isLoading, isAdmin } = useAuth();
  const [currentTab, setCurrentTab] = useState<string>("catalog");

  useEffect(() => {
    if (!isAdmin && currentTab.startsWith("admin-")) {
      setCurrentTab("catalog");
    }
  }, [isAdmin, currentTab]);
  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface-subtle">
        <div className="text-center space-y-3">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-accent text-accent-fg font-bold text-xl shadow-md">
            B
          </div>
          <LoadingSpinner size="md" />
          <p className="text-xs text-fg-subtle">Memuat sesi BorrowHub...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <AuthView />;
  }

  return (
    <div className="min-h-screen bg-surface-subtle flex flex-col text-fg">
      <Header currentTab={currentTab} onTabChange={setCurrentTab} />
      <Navigation currentTab={currentTab} onTabChange={setCurrentTab} />

      <main className="flex-1 pb-16">
        {currentTab === "catalog" && <CatalogView />}
        {currentTab === "my-loans" && <MyLoansView />}
        {currentTab === "notifications" && <NotificationsView />}
        {currentTab === "admin-inventory" && isAdmin && <AdminInventoryView />}
        {currentTab === "admin-lending" && isAdmin && <AdminLendingView />}
      </main>
    </div>
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </QueryClientProvider>
  );
}
