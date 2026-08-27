import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type {
  CharacterSlots,
  CharacterSlotsAttributes,
} from "@/types/models/character-slots";

const BASE_PATH = "/api/accounts";

function transformCharacterSlots(data: CharacterSlots): CharacterSlots {
  return {
    ...data,
    attributes: {
      ...data.attributes,
      worldId: Number(data.attributes.worldId),
      slots: Number(data.attributes.slots),
    },
  };
}

export const characterSlotsService = {
  /**
   * Get the character-slot count for one (account, world) pair.
   * `GET accounts/{accountId}/worlds/{worldId}/character-slots`.
   */
  async getCharacterSlots(
    accountId: string,
    worldId: number,
    options?: ServiceOptions,
  ): Promise<CharacterSlots> {
    const slots = await api.getOne<CharacterSlots>(
      `${BASE_PATH}/${accountId}/worlds/${worldId}/character-slots`,
      options,
    );
    return transformCharacterSlots(slots);
  },
};

export type { CharacterSlots, CharacterSlotsAttributes };
