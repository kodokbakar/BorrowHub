import { beforeEach, describe, expect, it } from "bun:test";
import {
  ApiError,
  authStorage,
  getServiceBaseUrl,
  request,
} from "../src/api/client";
import { identityApi } from "../src/api/endpoints";
import { AlertToast } from "../src/components/ui/AlertToast";
import { renderToString } from "react-dom/server";
describe("API Client & Auth Storage Smoke Tests", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    authStorage.clear();
    globalThis.fetch = originalFetch;
  });

  it("resolves service base URLs correctly", () => {
    expect(getServiceBaseUrl("identity")).toBe("/api/identity");
    expect(getServiceBaseUrl("inventory")).toBe("/api/inventory");
    expect(getServiceBaseUrl("lending")).toBe("/api/lending");
    expect(getServiceBaseUrl("notification")).toBe("/api/notification");
  });

  it("stores and clears authentication tokens in memory and storage", () => {
    expect(authStorage.getAccessToken()).toBeNull();
    expect(authStorage.getRefreshToken()).toBeNull();

    authStorage.setAccessToken("test-access-token");
    authStorage.setRefreshToken("test-refresh-token");

    expect(authStorage.getAccessToken()).toBe("test-access-token");
    expect(authStorage.getRefreshToken()).toBe("test-refresh-token");

    authStorage.clear();
    expect(authStorage.getAccessToken()).toBeNull();
    expect(authStorage.getRefreshToken()).toBeNull();
  });

  it("unwraps standard success envelopes containing data and meta", async () => {
    const mockPayload = { id: "item-123", name: "Sony FX3", is_lendable: true };
    const mockMeta = { request_id: "req-xyz-123" };

    globalThis.fetch = (async (url: string | URL | Request, init?: RequestInit) => {
      expect(String(url)).toBe("/api/inventory/api/v1/items/item-123");
      expect(init?.method).toBe("GET");
      return new Response(
        JSON.stringify({
          data: mockPayload,
          meta: mockMeta,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      );
    }) as unknown as typeof fetch;

    const result = await request<typeof mockPayload>(
      "inventory",
      "/api/v1/items/item-123",
    );
    expect(result).toEqual(mockPayload);
  });

  it("parses structured error envelopes and throws ApiError with details", async () => {
    const errorEnvelope = {
      error: {
        code: "VALIDATION_FAILED",
        message: "Waktu selesai harus setelah waktu mulai.",
        fields: { end_time: "must be after start_time" },
        request_id: "req-err-456",
      },
    };

    globalThis.fetch = (async () => {
      return new Response(JSON.stringify(errorEnvelope), {
        status: 400,
        headers: {
          "Content-Type": "application/json",
          "X-Request-ID": "req-err-456",
        },
      });
    }) as unknown as typeof fetch;

    let didThrow = false;
    try {
      await request("lending", "/api/v1/loan-requests", {
        method: "POST",
        body: { quantity: 1 },
      });
    } catch (err: unknown) {
      didThrow = true;
      expect(err instanceof ApiError).toBe(true);
      const apiErr = err as ApiError;
      expect(apiErr.status).toBe(400);
      expect(apiErr.code).toBe("VALIDATION_FAILED");
      expect(apiErr.message).toBe("Waktu selesai harus setelah waktu mulai.");
      expect(apiErr.fields).toEqual({ end_time: "must be after start_time" });
      expect(apiErr.requestId).toBe("req-err-456");
    }
    expect(didThrow).toBe(true);
  });

  it("attaches Authorization header when token is stored", async () => {
    authStorage.setAccessToken("my-secret-bearer-token");

    let capturedAuthHeader: string | null = null;
    let capturedOrgHeader: string | null = null;

    globalThis.fetch = (async (_url: string | URL | Request, init?: RequestInit) => {
      const headers = new Headers(init?.headers);
      capturedAuthHeader = headers.get("Authorization");
      capturedOrgHeader = headers.get("X-Organization-ID");
      return new Response(JSON.stringify({ data: { status: "ok" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }) as unknown as typeof fetch;

    await request("identity", "/api/v1/me", {
      organizationId: "org-uuid-999",
    });

    expect<string | null>(capturedAuthHeader).toBe("Bearer my-secret-bearer-token");
    expect<string | null>(capturedOrgHeader).toBe("org-uuid-999");
  });

  it("handles non-JSON server errors gracefully", async () => {
    globalThis.fetch = (async () => {
      return new Response("Bad Gateway", {
        status: 502,
        statusText: "Bad Gateway",
        headers: { "Content-Type": "text/plain" },
      });
    }) as unknown as typeof fetch;

    let didThrow = false;
    try {
      await request("identity", "/api/v1/me");
    } catch (err: unknown) {
      didThrow = true;
      expect(err instanceof ApiError).toBe(true);
      const apiErr = err as ApiError;
      expect(apiErr.status).toBe(502);
      expect(apiErr.code).toBe("HTTP_502");
    }
    expect(didThrow).toBe(true);
  });

  it("ensures dead endpoint listOrganizations is removed from identityApi", () => {
    expect((identityApi as unknown as Record<string, unknown>).listOrganizations).toBeUndefined();
  });

  it("calls identityApi.createOrganization with expected defaults and body", async () => {
    let capturedBody: string | null = null;
    globalThis.fetch = (async (_url: string | URL | Request, init?: RequestInit) => {
      capturedBody = init?.body as string;
      return new Response(
        JSON.stringify({
          data: {
            id: "org-new-123",
            name: "Lab AI",
            slug: "lab-ai",
            timezone: "UTC",
            visibility: "public",
            max_loan_duration_days: 14,
            created_at: new Date().toISOString(),
          },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      );
    }) as unknown as typeof fetch;

    const result = await identityApi.createOrganization({
      name: "Lab AI",
      slug: "lab-ai",
    });

    expect(result.id).toBe("org-new-123");
    expect(result.slug).toBe("lab-ai");
    const parsedBody = JSON.parse(capturedBody || "{}");
    expect(parsedBody.name).toBe("Lab AI");
    expect(parsedBody.slug).toBe("lab-ai");
    expect(parsedBody.timezone).toBe("UTC");
  });

  it("renders AlertToast with Request ID when provided", () => {
    const htmlWithErrorAndReq = renderToString(
      AlertToast({
        type: "error",
        message: "Operasi gagal",
        requestId: "req-tracking-777",
      }),
    );
    expect(htmlWithErrorAndReq).toContain("Operasi gagal");
    expect(htmlWithErrorAndReq).toContain("Request ID: <!-- -->req-tracking-777");

    const htmlWithoutReq = renderToString(
      AlertToast({
        type: "info",
        message: "Informasi umum",
      }),
    );
    expect(htmlWithoutReq).toContain("Informasi umum");
    expect(htmlWithoutReq).not.toContain("Request ID:");
  });
});
