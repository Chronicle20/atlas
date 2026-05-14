import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type {
  MonsterBookCard,
  MonsterBookCardAttributes,
  MonsterBookCollection,
  MonsterBookCollectionAttributes,
} from "@/types/monster-book";

const BASE_PATH = "/api/characters";

interface CollectionResource {
  id: string;
  type: string;
  attributes: MonsterBookCollectionAttributes;
}

interface CardResource {
  id: string;
  type: string;
  attributes: MonsterBookCardAttributes;
}

// `/api/data/consumables/{id}` returns the full consumable resource with
// the linked monster id under `attributes.monsterId`. The shared
// `ConsumableData` model omits this field, so the widget reads it through
// a thin local shape without leaking back into items.service.
interface ConsumableMonsterResource {
  id: string;
  attributes: { monsterId?: number };
}

interface MonsterNameResource {
  id: string;
  attributes: { name: string };
}

export interface ListCardsOptions extends ServiceOptions {
  offset?: number;
  limit?: number;
  isSpecial?: boolean;
}

function buildCardQuery(opts?: ListCardsOptions): string {
  if (!opts) return "";
  const params = new URLSearchParams();
  if (opts.offset !== undefined) params.set("page[offset]", String(opts.offset));
  if (opts.limit !== undefined) params.set("page[limit]", String(opts.limit));
  if (opts.isSpecial !== undefined) params.set("filter[isSpecial]", String(opts.isSpecial));
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

function flattenCollection(resource: CollectionResource): MonsterBookCollection {
  return {
    characterId: parseInt(resource.id, 10),
    bookLevel: resource.attributes.bookLevel,
    normalCount: resource.attributes.normalCount,
    specialCount: resource.attributes.specialCount,
    totalUniqueCards: resource.attributes.totalUniqueCards,
    coverCardId: resource.attributes.coverCardId,
    expBonusPercent: resource.attributes.expBonusPercent,
  };
}

function flattenCard(resource: CardResource): MonsterBookCard {
  return {
    cardId: parseInt(resource.id, 10),
    level: resource.attributes.level,
    isSpecial: resource.attributes.isSpecial,
    firstAcquiredAt: resource.attributes.firstAcquiredAt,
  };
}

export const monsterBookService = {
  /**
   * Fetch the monster-book collection summary for a character.
   * GET /api/characters/{characterId}/monster-book
   */
  async getCollection(
    characterId: number,
    options?: ServiceOptions,
  ): Promise<MonsterBookCollection> {
    const resource = await api.getOne<CollectionResource>(
      `${BASE_PATH}/${characterId}/monster-book`,
      options,
    );
    return flattenCollection(resource);
  },

  /**
   * List the cards owned by a character. Supports pagination via
   * `page[offset]` / `page[limit]` and filtering via `filter[isSpecial]`.
   * GET /api/characters/{characterId}/monster-book/cards
   */
  async listCards(
    characterId: number,
    opts?: ListCardsOptions,
  ): Promise<MonsterBookCard[]> {
    const url = `${BASE_PATH}/${characterId}/monster-book/cards${buildCardQuery(opts)}`;
    const resources = await api.getList<CardResource>(url, opts);
    return resources.map(flattenCard);
  },

  /**
   * Resolve the monster id linked to a monster-book card. Cards are
   * consumable items in the 238xxxxx range; their `attributes.monsterId`
   * points at the mob whose sprite + name we render. Returns `null` when
   * the consumable has no linked monster (e.g. malformed data dump).
   */
  async getCardConsumableMonsterId(cardId: number): Promise<number | null> {
    const resource = await api.getOne<ConsumableMonsterResource>(
      `/api/data/consumables/${cardId}`,
    );
    const monsterId = resource.attributes.monsterId;
    return monsterId !== undefined && monsterId > 0 ? monsterId : null;
  },

  /**
   * Resolve the display name of a monster by id.
   */
  async getMonsterName(monsterId: number): Promise<string> {
    const resource = await api.getOne<MonsterNameResource>(
      `/api/data/monsters/${monsterId}`,
    );
    return resource.attributes.name;
  },
};

export type { MonsterBookCard, MonsterBookCollection } from "@/types/monster-book";
