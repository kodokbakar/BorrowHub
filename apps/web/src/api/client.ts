import type { ApiErrorDetail, ApiErrorEnvelope, ApiSuccessEnvelope } from "./types";

export type ServiceName = "identity" | "inventory" | "lending" | "notification";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields?: Record<string, string>;
  readonly requestId?: string;

  constructor(
    status: number,
    code: string,
    message: string,
    fields?: Record<string, string>,
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.requestId = requestId;
  }
}

const TOKEN_KEY = "borrowhub_access_token";
const REFRESH_TOKEN_KEY = "borrowhub_refresh_token";

let memoryAccessToken: string | null = null;
let memoryRefreshToken: string | null = null;
let unauthorizedHandler: (() => void) | null = null;
let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

export const authStorage = {
  getAccessToken(): string | null {
    if (memoryAccessToken) return memoryAccessToken;
    if (typeof window !== "undefined") {
      try {
        return localStorage.getItem(TOKEN_KEY);
      } catch {
        return null;
      }
    }
    return null;
  },

  setAccessToken(token: string | null): void {
    memoryAccessToken = token;
    if (typeof window !== "undefined") {
      try {
        if (token) {
          localStorage.setItem(TOKEN_KEY, token);
        } else {
          localStorage.removeItem(TOKEN_KEY);
        }
      } catch {
        // ignore storage errors
      }
    }
  },

  getRefreshToken(): string | null {
    if (memoryRefreshToken) return memoryRefreshToken;
    if (typeof window !== "undefined") {
      try {
        return localStorage.getItem(REFRESH_TOKEN_KEY);
      } catch {
        return null;
      }
    }
    return null;
  },

  setRefreshToken(token: string | null): void {
    memoryRefreshToken = token;
    if (typeof window !== "undefined") {
      try {
        if (token) {
          localStorage.setItem(REFRESH_TOKEN_KEY, token);
        } else {
          localStorage.removeItem(REFRESH_TOKEN_KEY);
        }
      } catch {
        // ignore storage errors
      }
    }
  },

  clear(): void {
    memoryAccessToken = null;
    memoryRefreshToken = null;
    if (typeof window !== "undefined") {
      try {
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
      } catch {
        // ignore
      }
    }
  },

  onUnauthorized(handler: (() => void) | null): void {
    unauthorizedHandler = handler;
  },
};

export function getServiceBaseUrl(service: ServiceName): string {
  if (typeof window !== "undefined") {
    const isDev =
      import.meta.env?.DEV ||
      window.location.hostname === "localhost" ||
      window.location.hostname === "127.0.0.1" ||
      window.location.hostname.endsWith(".localhost");

    if (isDev) {
      return `/api/${service}`;
    }

    const host = window.location.host;
    const protocol = window.location.protocol;
    const parts = host.split(".");
    if (parts.length >= 2) {
      return `${protocol}//${service}.${parts.slice(-2).join(".")}`;
    }
    return `${protocol}//${service}.${host}`;
  }

  return `/api/${service}`;
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  params?: Record<string, string | number | boolean | undefined | null>;
  organizationId?: string;
  skipAuth?: boolean;
}

export async function request<T>(
  service: ServiceName,
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const baseUrl = getServiceBaseUrl(service);
  const cleanPath = path.startsWith("/") ? path : `/${path}`;

  let urlStr = `${baseUrl}${cleanPath}`;

  if (options.params) {
    const searchParams = new URLSearchParams();
    for (const [key, value] of Object.entries(options.params)) {
      if (value !== undefined && value !== null && value !== "") {
        searchParams.append(key, String(value));
      }
    }
    const query = searchParams.toString();
    if (query) {
      urlStr += (urlStr.includes("?") ? "&" : "?") + query;
    }
  }

  const headers = new Headers(options.headers);

  if (!options.skipAuth) {
    const token = authStorage.getAccessToken();
    if (token && !headers.has("Authorization")) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  if (options.organizationId && !headers.has("X-Organization-ID")) {
    headers.set("X-Organization-ID", options.organizationId);
  }

  let body: BodyInit | undefined;
  if (options.body !== undefined) {
    if (!headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    body = JSON.stringify(options.body);
  }
  let response: Response;
  try {
    response = await fetch(urlStr, {
      method: options.method || "GET",
      ...options,
      headers,
      body,
    });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : "Network request failed";
    throw new ApiError(0, "NETWORK_ERROR", msg);
  }

  // Handle 401 Unauthorized with token refresh if possible
  if (response.status === 401 && !options.skipAuth && !cleanPath.includes("/auth/")) {
    const newToken = await tryRefreshToken();
    if (newToken) {
      // Retry request with new token
      headers.set("Authorization", `Bearer ${newToken}`);
      response = await fetch(urlStr, {
        ...options,
        headers,
        body,
      });
    } else {
      authStorage.clear();
      unauthorizedHandler?.();
      throw new ApiError(401, "UNAUTHORIZED", "Authentication expired. Please log in again.");
    }
  }

  // Handle error responses
  if (!response.ok) {
    let errorDetail: ApiErrorDetail | null = null;
    try {
      const errJson = (await response.json()) as ApiErrorEnvelope;
      if (errJson && errJson.error) {
        errorDetail = errJson.error;
      }
    } catch {
      // non-JSON response body
    }

    if (errorDetail) {
      throw new ApiError(
        response.status,
        errorDetail.code || "UNKNOWN_ERROR",
        errorDetail.message || `Request failed with status ${response.status}`,
        errorDetail.fields,
        errorDetail.request_id || response.headers.get("X-Request-ID") || undefined,
      );
    }

    throw new ApiError(
      response.status,
      `HTTP_${response.status}`,
      `Request failed with status ${response.status}: ${response.statusText}`,
      undefined,
      response.headers.get("X-Request-ID") || undefined,
    );
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as unknown as T;
  }

  // Parse JSON and unwrap { data } envelope if present
  try {
    const json = await response.json();
    if (json && typeof json === "object" && "data" in json) {
      return (json as ApiSuccessEnvelope<T>).data;
    }
    return json as T;
  } catch {
    return undefined as unknown as T;
  }
}

async function tryRefreshToken(): Promise<string | null> {
  const refreshToken = authStorage.getRefreshToken();
  if (!refreshToken) {
    return null;
  }

  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const baseUrl = getServiceBaseUrl("identity");
      const res = await fetch(`${baseUrl}/api/v1/auth/refresh`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!res.ok) {
        authStorage.clear();
        return null;
      }

      const json = await res.json();
      const tokens = (json && "data" in json ? json.data : json) as {
        access_token: string;
        refresh_token: string;
      };

      if (tokens && tokens.access_token) {
        authStorage.setAccessToken(tokens.access_token);
        if (tokens.refresh_token) {
          authStorage.setRefreshToken(tokens.refresh_token);
        }
        return tokens.access_token;
      }
      return null;
    } catch {
      return null;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export const api = {
  get<T>(
    service: ServiceName,
    path: string,
    params?: Record<string, string | number | boolean | undefined | null>,
    options?: RequestOptions,
  ): Promise<T> {
    return request<T>(service, path, { ...options, method: "GET", params });
  },

  post<T>(
    service: ServiceName,
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    return request<T>(service, path, { ...options, method: "POST", body });
  },

  patch<T>(
    service: ServiceName,
    path: string,
    body?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    return request<T>(service, path, { ...options, method: "PATCH", body });
  },

  delete<T>(
    service: ServiceName,
    path: string,
    options?: RequestOptions,
  ): Promise<T> {
    return request<T>(service, path, { ...options, method: "DELETE" });
  },
};
