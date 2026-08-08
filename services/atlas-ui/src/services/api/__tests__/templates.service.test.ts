import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { templatesService } from "@/services/api/templates.service";

describe("templatesService.reseed", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
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
});
