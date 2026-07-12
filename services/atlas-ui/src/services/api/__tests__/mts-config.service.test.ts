import { describe, it, expect, vi, beforeEach } from "vitest";
import { mtsConfigService, type MtsConfig } from "@/services/api/mts-config.service";
import { mtsConfigSchema } from "@/lib/schemas/mts-config.schema";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), patch: vi.fn() },
}));

const TENANT_ID = "11111111-1111-1111-1111-111111111111";

const sampleAttributes = {
  listingFee: 5000,
  commissionRate: 0.1,
  maxActiveListings: 10,
  minLevel: 10,
  auctionMinHours: 24,
  auctionMaxHours: 168,
  priceFloor: 110,
  pageSize: 16,
  minBidIncrement: 1,
};

const sampleConfig: MtsConfig = {
  id: "mts-config-1",
  attributes: sampleAttributes,
};

describe("mtsConfigService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getConfig fetches the single mts-configs object under the tenant config path", async () => {
    (api.getOne as ReturnType<typeof vi.fn>).mockResolvedValue(sampleConfig);

    const res = await mtsConfigService.getConfig(TENANT_ID);

    expect(api.getOne).toHaveBeenCalledWith(
      `/api/tenants/${TENANT_ID}/configurations/mts-configs`,
      undefined,
    );
    expect(res.attributes.listingFee).toBe(5000);
    expect(res.attributes.commissionRate).toBe(0.1);
  });

  it("updateConfig PATCHes a JSON:API envelope to the by-id path", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    const res = await mtsConfigService.updateConfig(TENANT_ID, sampleConfig, {
      listingFee: 7000,
      pageSize: 20,
    });

    expect(api.patch).toHaveBeenCalledWith(
      `/api/tenants/${TENANT_ID}/configurations/mts-configs/mts-config-1`,
      {
        data: {
          id: "mts-config-1",
          type: "mts-configs",
          attributes: {
            ...sampleAttributes,
            listingFee: 7000,
            pageSize: 20,
          },
        },
      },
      undefined,
    );
    // Returns the merged config without re-fetching.
    expect(res.attributes.listingFee).toBe(7000);
    expect(res.attributes.pageSize).toBe(20);
    expect(res.attributes.commissionRate).toBe(0.1);
  });
});

describe("mtsConfigSchema", () => {
  it("accepts a valid config matching the backend defaults", () => {
    const parsed = mtsConfigSchema.safeParse(sampleAttributes);
    expect(parsed.success).toBe(true);
  });

  it("rejects a commissionRate outside 0..1", () => {
    const parsed = mtsConfigSchema.safeParse({ ...sampleAttributes, commissionRate: 1.5 });
    expect(parsed.success).toBe(false);
  });

  it("rejects when auctionMaxHours is below auctionMinHours", () => {
    const parsed = mtsConfigSchema.safeParse({
      ...sampleAttributes,
      auctionMinHours: 100,
      auctionMaxHours: 24,
    });
    expect(parsed.success).toBe(false);
  });

  it("rejects a non-integer listingFee", () => {
    const parsed = mtsConfigSchema.safeParse({ ...sampleAttributes, listingFee: 12.5 });
    expect(parsed.success).toBe(false);
  });
});
