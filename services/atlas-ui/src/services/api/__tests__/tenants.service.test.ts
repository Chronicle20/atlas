import { beforeEach, describe, expect, it, vi } from "vitest";

const patch = vi.fn();
const put = vi.fn();
const getOne = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    patch: (...args: unknown[]) => patch(...args),
    put: (...args: unknown[]) => put(...args),
    getOne: (...args: unknown[]) => getOne(...args),
    get: vi.fn(),
    getList: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
    setTenant: vi.fn(),
  },
}));

import { tenantsService } from "@/services/api/tenants.service";
import type { TenantConfig } from "@/services/api/tenants.service";
import type { MapleLifeConfig } from "@/types/models/template";

const SEED_ML: MapleLifeConfig = {
  looks: [
    {
      gender: 0,
      faces: [20000],
      hairs: [30000],
      hairColors: [0],
      skinColors: [0],
    },
  ],
  classes: [
    {
      ordinal: 0,
      gender: 0,
      jobId: 0,
      level: 1,
      mapId: 100000000,
      stats: { str: 12, dex: 5, int: 4, luk: 4, hp: 50, mp: 5 },
      ap: 0,
      sp: "61,0,0,0,0,0,0,0,0,0",
      meso: 0,
      equipment: [],
      inventory: [],
    },
  ],
};

function seededConfig(): TenantConfig {
  return {
    id: "t1",
    attributes: {
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
      usesPin: false,
      characters: { templates: [], presets: [] },
      npcs: [],
      worlds: [],
      socket: {
        handlers: [],
        writers: [],
        unsupported: { handlers: [], writers: [] },
      },
      cashShop: { commodities: {} },
      mapleLife: SEED_ML,
    },
  };
}

describe("tenantsService.updateTenantConfiguration", () => {
  beforeEach(() => {
    patch.mockReset().mockResolvedValue(undefined);
    put.mockReset();
    getOne.mockReset();
  });

  it("preserves an undeclared mapleLife block across an unrelated save", async () => {
    const tenant = seededConfig();

    await tenantsService.updateTenantConfiguration(tenant, {
      characters: { templates: [], presets: [] },
    });

    const body = patch.mock.calls[0]![1] as {
      data: { attributes: TenantConfig["attributes"] };
    };
    expect(body.data.attributes.mapleLife).toEqual(SEED_ML);
  });

  it("a mapleLife-only save leaves every other section untouched", async () => {
    const tenant = seededConfig();

    await tenantsService.updateTenantConfiguration(tenant, {
      mapleLife: SEED_ML,
    });

    const body = patch.mock.calls[0]![1] as {
      data: { attributes: TenantConfig["attributes"] };
    };
    expect(body.data.attributes.characters).toEqual(
      tenant.attributes.characters,
    );
    expect(body.data.attributes.npcs).toEqual(tenant.attributes.npcs);
    expect(body.data.attributes.worlds).toEqual(tenant.attributes.worlds);
    expect(body.data.attributes.socket).toEqual(tenant.attributes.socket);
    expect(body.data.attributes.cashShop).toEqual(tenant.attributes.cashShop);
  });

  it("PATCHes the configuration path with the tenant id", async () => {
    const tenant = seededConfig();

    await tenantsService.updateTenantConfiguration(tenant, {
      mapleLife: SEED_ML,
    });

    expect(patch.mock.calls[0]![0]).toBe("/api/configurations/tenants/t1");
  });
});
