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
};

export type { MonsterBookCard, MonsterBookCollection } from "@/types/monster-book";
