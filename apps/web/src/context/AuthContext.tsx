import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { authStorage } from "../api/client";
import { identityApi } from "../api/endpoints";
import type { Role, User, UserOrganization } from "../api/types";

interface AuthContextValue {
  user: User | null;
  organizations: UserOrganization[];
  activeOrg: UserOrganization | null;
  role: Role | null;
  isAdmin: boolean;
  isOwner: boolean;
  isAuthenticated: boolean;
  isLoading: boolean;
  login(email: string, password: string): Promise<void>;
  register(
    email: string,
    password: string,
    displayName: string,
    termsAccepted: boolean,
  ): Promise<void>;
  logout(): Promise<void>;
  switchOrg(orgId: string): void;
  refreshProfile(): Promise<void>;
}

const ACTIVE_ORG_KEY = "borrowhub_active_org_id";

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [organizations, setOrganizations] = useState<UserOrganization[]>([]);
  const [activeOrgId, setActiveOrgId] = useState<string | null>(() => {
    if (typeof window !== "undefined") {
      try {
        return localStorage.getItem(ACTIVE_ORG_KEY);
      } catch {
        return null;
      }
    }
    return null;
  });
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const activeOrg = useMemo<UserOrganization | null>(() => {
    if (!organizations.length) return null;
    if (activeOrgId) {
      const match = organizations.find((o) => o.id === activeOrgId);
      if (match) return match;
    }
    return organizations[0] ?? null;
  }, [organizations, activeOrgId]);

  const role = activeOrg?.role ?? null;
  const isAdmin = role === "owner" || role === "admin";
  const isOwner = role === "owner";
  const isAuthenticated = !!user;

  const handleSelectOrg = useCallback((orgId: string) => {
    setActiveOrgId(orgId);
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(ACTIVE_ORG_KEY, orgId);
      } catch {
        // ignore
      }
    }
  }, []);

  const refreshProfile = useCallback(async () => {
    try {
      const me = await identityApi.getMe();
      setUser(me.user);
      setOrganizations(me.organizations || []);

      if (me.organizations && me.organizations.length > 0) {
        const storedOrgId = localStorage.getItem(ACTIVE_ORG_KEY);
        const valid = me.organizations.some((o) => o.id === storedOrgId);
        if (!valid || !storedOrgId) {
          const firstId = me.organizations[0]?.id;
          if (firstId) {
            handleSelectOrg(firstId);
          }
        }
      }
    } catch {
      setUser(null);
      setOrganizations([]);
      authStorage.clear();
    }
  }, [handleSelectOrg]);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await identityApi.login(email, password);
      authStorage.setAccessToken(res.tokens.access_token);
      authStorage.setRefreshToken(res.tokens.refresh_token);
      await refreshProfile();
    },
    [refreshProfile],
  );

  const register = useCallback(
    async (
      email: string,
      password: string,
      displayName: string,
      termsAccepted: boolean,
    ) => {
      const res = await identityApi.register(
        email,
        password,
        displayName,
        termsAccepted,
      );
      authStorage.setAccessToken(res.tokens.access_token);
      authStorage.setRefreshToken(res.tokens.refresh_token);
      await refreshProfile();
    },
    [refreshProfile],
  );

  const logout = useCallback(async () => {
    try {
      const refreshToken = authStorage.getRefreshToken() || undefined;
      await identityApi.logout(refreshToken);
    } catch {
      // ignore logout network errors
    } finally {
      authStorage.clear();
      setUser(null);
      setOrganizations([]);
      setActiveOrgId(null);
      if (typeof window !== "undefined") {
        try {
          localStorage.removeItem(ACTIVE_ORG_KEY);
        } catch {
          // ignore
        }
      }
    }
  }, []);

  useEffect(() => {
    authStorage.onUnauthorized(() => {
      setUser(null);
      setOrganizations([]);
      setActiveOrgId(null);
    });

    const init = async () => {
      const token = authStorage.getAccessToken() || authStorage.getRefreshToken();
      if (token) {
        await refreshProfile();
      }
      setIsLoading(false);
    };

    void init();
  }, [refreshProfile]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      organizations,
      activeOrg,
      role,
      isAdmin,
      isOwner,
      isAuthenticated,
      isLoading,
      login,
      register,
      logout,
      switchOrg: handleSelectOrg,
      refreshProfile,
    }),
    [
      user,
      organizations,
      activeOrg,
      role,
      isAdmin,
      isOwner,
      isAuthenticated,
      isLoading,
      login,
      register,
      logout,
      handleSelectOrg,
      refreshProfile,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
