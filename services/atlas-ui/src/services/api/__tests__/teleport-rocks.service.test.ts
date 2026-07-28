import { describe, it, expect, vi, beforeEach } from "vitest";
import { teleportRocksService } from "@/services/api/teleport-rocks.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

const resource = {
  id: "42",
  type: "teleport-rock-maps",
  attributes: {
    regular: [100000000],
    vip: [],
    regularCapacity: 5,
    vipCapacity: 10,
  },
};

describe("teleportRocksService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getByCharacterId flattens the resource", async () => {
    (api.getOne as ReturnType<typeof vi.fn>).mockResolvedValue(resource);
    const r = await teleportRocksService.getByCharacterId("42");
    expect(api.getOne).toHaveBeenCalledWith(
      "/api/characters/42/teleport-rock-maps",
    );
    expect(r).toEqual({
      regular: [100000000],
      vip: [],
      regularCapacity: 5,
      vipCapacity: 10,
    });
  });

  it("addMap posts the JSON:API envelope and unwraps the response", async () => {
    (api.post as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: resource,
    });
    const r = await teleportRocksService.addMap("42", "regular", 100000000);
    expect(api.post).toHaveBeenCalledWith(
      "/api/characters/42/teleport-rock-maps",
      {
        data: {
          type: "teleport-rock-maps",
          attributes: { list: "regular", mapId: 100000000 },
        },
      },
    );
    expect(r).toEqual({
      regular: [100000000],
      vip: [],
      regularCapacity: 5,
      vipCapacity: 10,
    });
  });

  it("removeMap deletes the nested path and unwraps the response", async () => {
    (api.delete as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: resource,
    });
    const r = await teleportRocksService.removeMap("42", "vip", 200000000);
    expect(api.delete).toHaveBeenCalledWith(
      "/api/characters/42/teleport-rock-maps/vip/200000000",
    );
    expect(r).toEqual({
      regular: [100000000],
      vip: [],
      regularCapacity: 5,
      vipCapacity: 10,
    });
  });
});
