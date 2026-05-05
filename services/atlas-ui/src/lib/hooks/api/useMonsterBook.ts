import {
  useInfiniteQuery,
  useQuery,
  type UseInfiniteQueryResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  monsterBookService,
  type ListCardsOptions,
} from "@/services/api/monster-book.service";
import type {
  MonsterBookCard,
  MonsterBookCollection,
} from "@/types/monster-book";

const PAGE_SIZE = 100;

// Card-name resolution happens on every visible card, so we hold the
// underlying `consumable → monster` data warmer than the collection /
// card list (which we want fresh per character visit).
const CARD_NAME_STALE_TIME = 5 * 60 * 1000;
const CARD_NAME_GC_TIME = 30 * 60 * 1000;

const COLLECTION_STALE_TIME = 60 * 1000;
const COLLECTION_GC_TIME = 5 * 60 * 1000;

export const monsterBookKeys = {
  all: ["monster-book"] as const,
  collection: (tenantId: string | undefined, characterId: number) =>
    ["monster-book", "collection", tenantId, characterId] as const,
  cards: (
    tenantId: string | undefined,
    characterId: number,
    opts?: Omit<ListCardsOptions, "offset">,
  ) => ["monster-book", "cards", tenantId, characterId, opts ?? null] as const,
  cardConsumable: (tenantId: string | undefined, cardId: number) =>
    ["monster-book", "card-consumable", tenantId, cardId] as const,
  monsterName: (tenantId: string | undefined, monsterId: number) =>
    ["monster-book", "monster-name", tenantId, monsterId] as const,
} as const;

/**
 * Fetch the monster-book collection summary for a character.
 * Disabled until both a tenant is selected and a positive characterId
 * is supplied.
 */
export function useMonsterBookCollection(
  tenant: Tenant | null | undefined,
  characterId: number,
): UseQueryResult<MonsterBookCollection, Error> {
  return useQuery({
    queryKey: monsterBookKeys.collection(tenant?.id, characterId),
    queryFn: () => monsterBookService.getCollection(characterId),
    enabled:
      !!tenant?.id && Number.isFinite(characterId) && characterId > 0,
    staleTime: COLLECTION_STALE_TIME,
    gcTime: COLLECTION_GC_TIME,
  });
}

/**
 * Paginated card list for the widget. We accumulate page offsets
 * client-side; once a page returns fewer rows than the page size, the
 * server has nothing more to give.
 */
export function useMonsterBookCards(
  tenant: Tenant | null | undefined,
  characterId: number,
  opts?: Omit<ListCardsOptions, "offset" | "limit">,
): UseInfiniteQueryResult<{ pages: MonsterBookCard[][]; pageParams: number[] }, Error> {
  return useInfiniteQuery({
    queryKey: monsterBookKeys.cards(tenant?.id, characterId, opts),
    queryFn: ({ pageParam }: { pageParam: number }) =>
      monsterBookService.listCards(characterId, {
        ...opts,
        offset: pageParam,
        limit: PAGE_SIZE,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      if (!lastPage || lastPage.length < PAGE_SIZE) return undefined;
      return allPages.reduce((acc, page) => acc + page.length, 0);
    },
    enabled:
      !!tenant?.id && Number.isFinite(characterId) && characterId > 0,
    staleTime: COLLECTION_STALE_TIME,
    gcTime: COLLECTION_GC_TIME,
  });
}

export interface CardNameResult {
  /** Monster id linked to this card (via the consumable data row), if any. */
  monsterId: number | null;
  /** Display name; null while still loading or unresolved. */
  name: string | null;
  isLoading: boolean;
  isError: boolean;
}

/**
 * Resolve a monster-book card to its display name + monster id by
 * chasing `consumable.monsterId → monster.name` through atlas-data. The
 * monster query stays disabled until the consumable resolves, and both
 * queries are gated on tenant selection.
 */
export function useMonsterBookCardName(
  tenant: Tenant | null | undefined,
  cardId: number,
): CardNameResult {
  const consumableQuery = useQuery({
    queryKey: monsterBookKeys.cardConsumable(tenant?.id, cardId),
    queryFn: () => monsterBookService.getCardConsumableMonsterId(cardId),
    enabled: !!tenant?.id && cardId > 0,
    staleTime: CARD_NAME_STALE_TIME,
    gcTime: CARD_NAME_GC_TIME,
  });

  const monsterId = consumableQuery.data ?? null;

  const monsterQuery = useQuery({
    queryKey: monsterBookKeys.monsterName(tenant?.id, monsterId ?? 0),
    queryFn: () => monsterBookService.getMonsterName(monsterId as number),
    enabled: !!tenant?.id && monsterId !== null && monsterId > 0,
    staleTime: CARD_NAME_STALE_TIME,
    gcTime: CARD_NAME_GC_TIME,
  });

  return {
    monsterId,
    name: monsterQuery.data ?? null,
    isLoading: consumableQuery.isLoading || monsterQuery.isLoading,
    isError: consumableQuery.isError || monsterQuery.isError,
  };
}
