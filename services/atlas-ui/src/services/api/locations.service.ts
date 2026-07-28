import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { CharacterLocation, ChangeMapData } from "@/types/models/location";

const BASE_PATH = "/api/characters";

export const locationsService = {
  async getByCharacterId(
    characterId: string,
    options?: ServiceOptions,
  ): Promise<CharacterLocation> {
    return api.getOne<CharacterLocation>(
      `${BASE_PATH}/${characterId}/location`,
      options,
    );
  },

  async changeMap(
    characterId: string,
    data: ChangeMapData,
    options?: ServiceOptions,
  ): Promise<void> {
    const requestBody = {
      data: {
        type: "character-locations",
        id: characterId,
        attributes: data,
      },
    };
    return api.patch<void>(
      `${BASE_PATH}/${characterId}/location`,
      requestBody,
      options,
    );
  },
};
