import { beforeEach, describe, expect, it, vi } from "vitest";

const patch = vi.fn();
const put = vi.fn();
const getOne = vi.fn();
const post = vi.fn();

vi.mock("@/lib/api/client", () => ({
  api: {
    patch: (...args: unknown[]) => patch(...args),
    put: (...args: unknown[]) => put(...args),
    getOne: (...args: unknown[]) => getOne(...args),
    get: vi.fn(),
    getList: vi.fn(),
    post: (...args: unknown[]) => post(...args),
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
      faces: [20000, 20001, 20002],
      hairs: [30030, 30020, 30000],
      hairColors: [0, 7, 3, 2],
      skinColors: [0, 1, 2, 3],
    },
  ],
  classes: [
    {
      ordinal: 0,
      gender: 0,
      jobId: 100,
      level: 30,
      mapId: 102000000,
      stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
      ap: 123,
      sp: "61,0,0,0,0,0,0,0,0,0",
      spSkillId: 1000001,
      meso: 100000,
      equipment: [{ templateId: 1040021, useAverageStats: true }],
      inventory: [{ templateId: 2000002, quantity: 100 }],
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
    post.mockReset();
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

describe("reset", () => {
  beforeEach(() => {
    // The server wraps the reset response in the same JSON:API envelope as
    // tenant create (server.MarshalResponse[ViewRestModel] in resource.go),
    // not a bare resource. Resolve the mock with that shape so these tests
    // are diagnostic of the real wire contract.
    post.mockReset().mockResolvedValue({ data: seededConfig() });
  });

  it("posts with no body for a whole-document reset", async () => {
    await tenantsService.reset("t1");

    expect(post.mock.calls[0]![0]).toBe("/api/configurations/tenants/t1/reset");
    expect(post.mock.calls[0]![1]).toBeUndefined();
  });

  it("posts the sections envelope when scoped", async () => {
    await tenantsService.reset("t1", ["socket"]);

    expect(post.mock.calls[0]![0]).toBe("/api/configurations/tenants/t1/reset");
    expect(post.mock.calls[0]![1]).toEqual({
      data: { type: "tenants", attributes: { sections: ["socket"] } },
    });
  });

  it("does not skip tenant headers", async () => {
    await tenantsService.reset("t1");

    const options = post.mock.calls[0]![2] as
      { skipTenantHeaders?: boolean } | undefined;
    expect(options?.skipTenantHeaders).not.toBe(true);
  });
});

describe("updateTenantConfiguration computed-key hygiene", () => {
  beforeEach(() => {
    patch.mockReset().mockResolvedValue(undefined);
  });

  function driftedConfig(): TenantConfig {
    const config = seededConfig();
    return {
      ...config,
      attributes: {
        ...config.attributes,
        baselineTemplateId: "b",
        baselineRevision: "r",
        storedRevision: "s",
        templateDrift: true,
        sectionDrift: { socket: true },
        usesPin: false,
      },
    };
  }

  it("strips the five computed keys from the PATCH body", async () => {
    const tenant = driftedConfig();

    await tenantsService.updateTenantConfiguration(tenant, {});

    const body = patch.mock.calls[0]![1] as {
      data: { attributes: TenantConfig["attributes"] };
    };
    expect(body.data.attributes).not.toHaveProperty("baselineTemplateId");
    expect(body.data.attributes).not.toHaveProperty("baselineRevision");
    expect(body.data.attributes).not.toHaveProperty("storedRevision");
    expect(body.data.attributes).not.toHaveProperty("templateDrift");
    expect(body.data.attributes).not.toHaveProperty("sectionDrift");
    expect(body.data.attributes.usesPin).toBe(false);
    expect(body.data.attributes.region).toBe("GMS");
  });

  it("still applies the caller's updates", async () => {
    const tenant = driftedConfig();

    await tenantsService.updateTenantConfiguration(tenant, {
      usesPin: true,
    });

    const body = patch.mock.calls[0]![1] as {
      data: { attributes: TenantConfig["attributes"] };
    };
    expect(body.data.attributes.usesPin).toBe(true);
  });
});
