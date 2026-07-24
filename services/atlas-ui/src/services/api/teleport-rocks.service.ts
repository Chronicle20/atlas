import { api } from "@/lib/api/client";

const BASE_PATH = "/api/characters";

export type TeleportRockListType = "regular" | "vip";

export interface TeleportRockLists {
  regular: number[];
  vip: number[];
  regularCapacity: number;
  vipCapacity: number;
}

interface TeleportRockResource {
  id: string;
  type: "teleport-rock-maps";
  attributes: TeleportRockLists;
}

/**
 * `api.getOne` unwraps the JSON:API envelope (`.data`) before returning, but
 * `api.post` / `api.delete` pass the raw response straight through — so a
 * write call resolves to `{ data: TeleportRockResource }` while a read call
 * resolves to `TeleportRockResource` directly. Normalize both shapes here.
 */
function unwrap(
  r: TeleportRockResource | { data: TeleportRockResource },
): TeleportRockResource {
  return "data" in r ? r.data : r;
}

function flatten(r: TeleportRockResource): TeleportRockLists {
  return {
    regular: r.attributes.regular ?? [],
    vip: r.attributes.vip ?? [],
    regularCapacity: r.attributes.regularCapacity,
    vipCapacity: r.attributes.vipCapacity,
  };
}

export const teleportRocksService = {
  async getByCharacterId(characterId: string): Promise<TeleportRockLists> {
    const r = await api.getOne<TeleportRockResource>(
      `${BASE_PATH}/${characterId}/teleport-rock-maps`,
    );
    return flatten(r);
  },

  async addMap(
    characterId: string,
    list: TeleportRockListType,
    mapId: number,
  ): Promise<TeleportRockLists> {
    const r = await api.post<{ data: TeleportRockResource }>(
      `${BASE_PATH}/${characterId}/teleport-rock-maps`,
      { data: { type: "teleport-rock-maps", attributes: { list, mapId } } },
    );
    return flatten(unwrap(r));
  },

  async removeMap(
    characterId: string,
    list: TeleportRockListType,
    mapId: number,
  ): Promise<TeleportRockLists> {
    const r = await api.delete<{ data: TeleportRockResource }>(
      `${BASE_PATH}/${characterId}/teleport-rock-maps/${list}/${mapId}`,
    );
    return flatten(unwrap(r));
  },
};
