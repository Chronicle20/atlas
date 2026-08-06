import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.fn();
const getList = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => get(...args),
    getList: (...args: unknown[]) => getList(...args),
    getOne: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    setTenant: vi.fn(),
  },
}));

import { templatesService } from "@/services/api/templates.service";
import { tenantsService } from "@/services/api/tenants.service";

describe("templatesService.getSocketMatrix", () => {
  beforeEach(() => {
    get.mockReset();
  });

  it("requests the region/majorVersion/minorVersion/socket sparse fieldset", async () => {
    get.mockResolvedValue({
      data: [
        {
          id: "t1",
          attributes: {
            region: "GMS",
            majorVersion: 83,
            minorVersion: 1,
            socket: { handlers: [], writers: [] },
          },
        },
      ],
      meta: null,
    });

    await templatesService.getSocketMatrix();

    const requestedUrl = decodeURIComponent(get.mock.calls[0]![0] as string);
    expect(requestedUrl).toContain("/api/configurations/templates");
    expect(requestedUrl).toContain(
      "fields[templates]=region,majorVersion,minorVersion,socket",
    );
  });

  // Task 8 fixed sortTenantConfig dropping socket.unsupported on read; this
  // sparse reader skips sortTemplate entirely (per design: the normalizer
  // preserves stored order and the grid sorts its own rows), so unsupported
  // passes through completely untouched rather than needing its own fix.
  it("carries socket.unsupported through unchanged", async () => {
    get.mockResolvedValue({
      data: [
        {
          id: "t1",
          attributes: {
            region: "GMS",
            majorVersion: 83,
            minorVersion: 1,
            socket: {
              handlers: [],
              writers: [],
              unsupported: {
                handlers: ["LegacyHandler"],
                writers: ["LegacyWriter"],
              },
            },
          },
        },
      ],
      meta: null,
    });

    const result = await templatesService.getSocketMatrix();

    expect(result[0]!.attributes.socket.unsupported).toEqual({
      handlers: ["LegacyHandler"],
      writers: ["LegacyWriter"],
    });
  });
});

describe("tenantsService.getSocketMatrix", () => {
  beforeEach(() => {
    getList.mockReset();
  });

  it("requests the region/majorVersion/minorVersion/socket sparse fieldset", async () => {
    getList.mockResolvedValue([
      {
        id: "tn1",
        attributes: {
          region: "GMS",
          majorVersion: 95,
          minorVersion: 1,
          socket: { handlers: [], writers: [] },
        },
      },
    ]);

    await tenantsService.getSocketMatrix();

    const requestedUrl = getList.mock.calls[0]![0] as string;
    expect(requestedUrl).toContain("/api/configurations/tenants");
    expect(requestedUrl).toContain(
      "fields[tenants]=region,majorVersion,minorVersion,socket",
    );
  });

  it("carries socket.unsupported through unchanged", async () => {
    getList.mockResolvedValue([
      {
        id: "tn1",
        attributes: {
          region: "GMS",
          majorVersion: 95,
          minorVersion: 1,
          socket: {
            handlers: [],
            writers: [],
            unsupported: { handlers: ["OldHandler"], writers: [] },
          },
        },
      },
    ]);

    const result = await tenantsService.getSocketMatrix();

    expect(result[0]!.attributes.socket.unsupported).toEqual({
      handlers: ["OldHandler"],
      writers: [],
    });
  });
});
