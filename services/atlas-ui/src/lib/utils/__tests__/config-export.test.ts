import { describe, expect, it } from "vitest";

import {
  configExportFilename,
  toConfigExportPayload,
} from "@/lib/utils/config-export";

// Built in the on-the-wire key order that atlas-configurations emits
// (templates/rest.go and tenants/rest.go declare byte-identical field sets in
// this order), which is also the key order of the checked-in seed files.
function fixture(overrides: Record<string, unknown> = {}) {
  return {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    socket: {
      handlers: [
        { opCode: "0x0B8", validator: "v2", handler: "h2" },
        { opCode: "0x01", validator: "v1", handler: "h1" },
      ],
      writers: [
        { opCode: "0x1A", writer: "w2" },
        { opCode: "0x0F", writer: "w1" },
      ],
    },
    characters: { templates: [], presets: [] },
    npcs: [{ npcId: 9000000, impl: "shop" }],
    worlds: [{ name: "Scania" }],
    cashShop: { commodities: {} },
    ...overrides,
  };
}

describe("toConfigExportPayload", () => {
  it("emits no JSON:API envelope keys", () => {
    const out = toConfigExportPayload(fixture());

    expect(out).not.toHaveProperty("id");
    expect(out).not.toHaveProperty("type");
    expect(out).not.toHaveProperty("data");
  });

  it("preserves the seed-file key order", () => {
    const out = toConfigExportPayload(fixture());

    expect(Object.keys(out)).toEqual([
      "region",
      "majorVersion",
      "minorVersion",
      "usesPin",
      "socket",
      "characters",
      "npcs",
      "worlds",
      "cashShop",
    ]);
  });

  it("normalises null collections to empty arrays in place", () => {
    const out = toConfigExportPayload(fixture({ npcs: null, worlds: null }));

    expect(out.npcs).toEqual([]);
    expect(out.worlds).toEqual([]);
    expect(Object.keys(out).indexOf("npcs")).toBe(6);
    expect(Object.keys(out).indexOf("worlds")).toBe(7);
  });

  it("passes present collections through by value", () => {
    const out = toConfigExportPayload(fixture());

    expect(out.npcs).toEqual([{ npcId: 9000000, impl: "shop" }]);
    expect(out.worlds).toEqual([{ name: "Scania" }]);
  });

  it("sorts handlers and writers ascending by numeric opCode", () => {
    const out = toConfigExportPayload(fixture());

    expect(out.socket?.handlers?.map((h) => h.opCode)).toEqual([
      "0x01",
      "0x0B8",
    ]);
    expect(out.socket?.writers?.map((w) => w.opCode)).toEqual(["0x0F", "0x1A"]);
  });

  it("keeps both entries when two opCodes compare numerically equal", () => {
    const out = toConfigExportPayload(
      fixture({
        socket: {
          handlers: [
            { opCode: "0xB8", handler: "padded-off" },
            { opCode: "0x0B8", handler: "padded-on" },
          ],
          writers: [],
        },
      }),
    );

    expect(out.socket?.handlers).toHaveLength(2);
  });

  it("does not mutate the input", () => {
    const input = fixture();
    const before = JSON.stringify(input);

    toConfigExportPayload(input);

    expect(JSON.stringify(input)).toBe(before);
  });

  it("returns an absent socket untouched", () => {
    const out = toConfigExportPayload(fixture({ socket: null }));

    expect(out.socket).toBeNull();
  });
});

describe("toConfigExportPayload computed attributes", () => {
  it("strips the server-computed drift keys", () => {
    const out = toConfigExportPayload(
      fixture({
        shippedRevision: "aa".repeat(32),
        storedRevision: "bb".repeat(32),
        seedDrift: true,
      }) as never,
    ) as Record<string, unknown>;

    expect(out).not.toHaveProperty("shippedRevision");
    expect(out).not.toHaveProperty("storedRevision");
    expect(out).not.toHaveProperty("seedDrift");
  });

  it("leaves the configured document intact", () => {
    const out = toConfigExportPayload(
      fixture({ seedDrift: true }) as never,
    ) as Record<string, unknown>;

    expect(out.region).toBe("GMS");
    expect(out.majorVersion).toBe(83);
    expect(out.minorVersion).toBe(1);
    expect(out).toHaveProperty("socket");
    expect(out).toHaveProperty("characters");
  });

  it("strips the tenant computed keys", () => {
    const out = toConfigExportPayload(
      fixture({
        baselineTemplateId: "b",
        baselineRevision: "r",
        storedRevision: "s",
        templateDrift: true,
        sectionDrift: { socket: true },
        usesPin: true,
      }) as never,
    ) as Record<string, unknown>;

    expect(out).not.toHaveProperty("baselineTemplateId");
    expect(out).not.toHaveProperty("baselineRevision");
    expect(out).not.toHaveProperty("storedRevision");
    expect(out).not.toHaveProperty("templateDrift");
    expect(out).not.toHaveProperty("sectionDrift");
    expect(out.usesPin).toBe(true);
    expect(out).toHaveProperty("socket");
    expect(out).toHaveProperty("worlds");
  });

  it("still strips the template computed keys", () => {
    const out = toConfigExportPayload(
      fixture({
        shippedRevision: "aa".repeat(32),
        storedRevision: "bb".repeat(32),
        seedDrift: true,
      }) as never,
    ) as Record<string, unknown>;

    expect(out).not.toHaveProperty("shippedRevision");
    expect(out).not.toHaveProperty("storedRevision");
    expect(out).not.toHaveProperty("seedDrift");
  });
});

describe("configExportFilename", () => {
  const meta = {
    id: "8b1d4c4e-0000-4000-8000-000000000000",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  };

  it("names a template export after the seed convention", () => {
    expect(configExportFilename("template", meta)).toBe(
      "template_gms_83_1.json",
    );
  });

  it("prefixes a tenant export with tenant_", () => {
    expect(configExportFilename("tenant", meta)).toBe("tenant_gms_83_1.json");
  });

  it("sanitises characters outside [a-z0-9] in the region", () => {
    expect(
      configExportFilename("template", { ...meta, region: "GMS-Beta" }),
    ).toBe("template_gms_beta_83_1.json");
  });

  it("falls back to the id when the region is missing or empty", () => {
    expect(
      configExportFilename("template", { ...meta, region: undefined }),
    ).toBe("template_8b1d4c4e_0000_4000_8000_000000000000.json");
    expect(configExportFilename("tenant", { ...meta, region: "   " })).toBe(
      "tenant_8b1d4c4e_0000_4000_8000_000000000000.json",
    );
  });

  it("falls back to the id when a version is not a finite number", () => {
    expect(
      configExportFilename("template", { ...meta, majorVersion: undefined }),
    ).toBe("template_8b1d4c4e_0000_4000_8000_000000000000.json");
    expect(
      configExportFilename("tenant", { ...meta, minorVersion: Number.NaN }),
    ).toBe("tenant_8b1d4c4e_0000_4000_8000_000000000000.json");
  });
});
