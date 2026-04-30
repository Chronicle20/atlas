import { describe, it, expect, vi, beforeEach } from "vitest";
import { factoryService } from "../factory.service";
import type { Tenant } from "@/types/models/tenant";

// Mock fetch for createFromPreset (uses fetch directly, like seed.service)
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

const mockTenant: Tenant = {
  id: "tenant-123",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

describe("factoryService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("createFromPreset", () => {
    it("returns transactionId unwrapped from the JSON:API envelope", async () => {
      const envelope = {
        data: {
          type: "create-character-response",
          id: "txn-abc",
          attributes: { transactionId: "txn-abc" },
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => envelope,
      });

      const result = await factoryService.createFromPreset(mockTenant, {
        presetId: "preset-1",
        accountId: 42,
        worldId: 0,
        name: "MyChar",
      });

      expect(result).toEqual({ transactionId: "txn-abc" });
    });

    it("sends a JSON:API-encoded body to /api/factory/characters/from-preset", async () => {
      const envelope = {
        data: {
          type: "create-character-response",
          id: "txn-xyz",
          attributes: { transactionId: "txn-xyz" },
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => envelope,
      });

      await factoryService.createFromPreset(mockTenant, {
        presetId: "preset-2",
        accountId: 7,
        worldId: 1,
        name: "AnotherChar",
      });

      const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(url).toBe("/api/factory/characters/from-preset");
      expect(init.method).toBe("POST");
      expect(JSON.parse(init.body as string)).toEqual({
        data: {
          type: "preset-create",
          attributes: {
            presetId: "preset-2",
            accountId: 7,
            worldId: 1,
            name: "AnotherChar",
          },
        },
      });
    });

    it("injects the four tenant headers", async () => {
      const envelope = {
        data: {
          type: "create-character-response",
          id: "txn-1",
          attributes: { transactionId: "txn-1" },
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => envelope,
      });

      await factoryService.createFromPreset(mockTenant, {
        presetId: "p",
        accountId: 1,
        worldId: 0,
        name: "N",
      });

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit];
      const headers = init.headers as Headers;
      expect(headers.get("TENANT_ID")).toBe("tenant-123");
      expect(headers.get("REGION")).toBe("GMS");
      expect(headers.get("MAJOR_VERSION")).toBe("83");
      expect(headers.get("MINOR_VERSION")).toBe("1");
    });

    it("throws an error with status when the response is not ok", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: async () => ({ error: "preset not found" }),
      });

      await expect(
        factoryService.createFromPreset(mockTenant, {
          presetId: "bad-preset",
          accountId: 1,
          worldId: 0,
          name: "X",
        }),
      ).rejects.toMatchObject({ message: "preset not found", status: 404 });
    });

    it("throws a status-based error when the error body is not JSON", async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 502,
        json: async () => { throw new SyntaxError("not json"); },
      });

      await expect(
        factoryService.createFromPreset(mockTenant, {
          presetId: "p",
          accountId: 1,
          worldId: 0,
          name: "N",
        }),
      ).rejects.toMatchObject({ status: 502 });
    });
  });
});
