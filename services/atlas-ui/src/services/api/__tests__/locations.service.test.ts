import { describe, it, expect, vi, beforeEach } from "vitest";
import { locationsService } from "@/services/api/locations.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), patch: vi.fn() },
}));

describe("locationsService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getByCharacterId hits the atlas-maps location endpoint", async () => {
    (api.getOne as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "7",
      type: "character-locations",
      attributes: { worldId: 0, channelId: 1, mapId: 100000000, instance: "" },
    });
    const res = await locationsService.getByCharacterId("7");
    expect(api.getOne).toHaveBeenCalledWith("/api/characters/7/location", undefined);
    expect(res.attributes.mapId).toBe(100000000);
  });

  it("changeMap PATCHes a character-locations JSON:API envelope", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);
    await locationsService.changeMap("7", { mapId: 104000000 });
    expect(api.patch).toHaveBeenCalledWith(
      "/api/characters/7/location",
      { data: { type: "character-locations", id: "7", attributes: { mapId: 104000000 } } },
      undefined,
    );
  });
});
