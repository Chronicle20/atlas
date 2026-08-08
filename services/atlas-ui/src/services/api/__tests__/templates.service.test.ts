import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api } from "@/lib/api/client";
import { templatesService } from "@/services/api/templates.service";
import type { Template, TemplateAttributes } from "@/types/models/template";

describe("templatesService.reseed", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    api.setTenant(null);
  });

  it("POSTs to the reseed sub-resource with no body", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({}),
    });

    await templatesService.reseed("abc-123");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    // templatesService goes through the ApiClient singleton (api.post),
    // which prefixes window.location.origin onto the path - unlike the
    // sibling services under this directory that call the global fetch
    // directly with a bare relative path. Assert on the suffix so the test
    // is robust to that prefix rather than hardcoding jsdom's default origin.
    expect(url.endsWith("/api/configurations/templates/abc-123/reseed")).toBe(
      true,
    );
    expect(init.method).toBe("POST");
    expect(init.body).toBeUndefined();
  });

  it("rejects when the server returns an error status", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({
        errors: [
          {
            status: "409",
            title: "no shipped template",
            detail: "nothing to reset to",
          },
        ],
      }),
    });

    await expect(templatesService.reseed("abc-123")).rejects.toBeDefined();
  });

  it("omits tenant headers even when a tenant is active", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({}),
    });

    // Templates are global (NFR-5): the reseed endpoint is deliberately not
    // tenant-scoped. Setting a tenant here reproduces what TenantProvider
    // does globally on the shared ApiClient singleton in the running app -
    // without this, createHeaders (lib/api/client.ts) has nothing to attach
    // and the assertion below would pass whether or not skipTenantHeaders
    // was wired through, making the test vacuous.
    api.setTenant({
      id: "tenant-1",
      attributes: {
        name: "gms",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
      },
    });

    await templatesService.reseed("abc-123");

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(headers.has("TENANT_ID")).toBe(false);
    expect(headers.has("REGION")).toBe(false);
    expect(headers.has("MAJOR_VERSION")).toBe(false);
    expect(headers.has("MINOR_VERSION")).toBe(false);
  });
});

describe("templatesService.cloneTemplate", () => {
  it("strips computed keys so a clone never POSTs stale drift metadata", () => {
    const source: Template = {
      id: "src-1",
      attributes: {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        shippedRevision: "abc123",
        storedRevision: "def456",
        seedDrift: true,
        usesPin: false,
        characters: { templates: [], presets: [] },
        npcs: [],
        socket: { handlers: [], writers: [] },
        worlds: [],
      } as TemplateAttributes,
    };

    const cloned = templatesService.cloneTemplate(source);

    expect(cloned).not.toHaveProperty("shippedRevision");
    expect(cloned).not.toHaveProperty("storedRevision");
    expect(cloned).not.toHaveProperty("seedDrift");
    expect(cloned.region).toBe("");
    expect(cloned.majorVersion).toBe(0);
    expect(cloned.minorVersion).toBe(0);
  });
});
