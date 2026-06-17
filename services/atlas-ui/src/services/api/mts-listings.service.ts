import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";

/**
 * Read-only browser for atlas-mts marketplace listings.
 *
 * Backed by the atlas-mts public browse endpoint:
 *   GET /api/worlds/{worldId}/listings
 *     ?category=&subCategory=&saleType=&sellerName=&itemId=&page=&pageSize=
 *
 * The browse endpoint only ever surfaces active listings. The response is a
 * flat JSON:API list of `listings` resources (no total/lastPage metadata), so
 * pagination is driven client-side off the returned count.
 */

export interface MtsListingAttributes {
  worldId: number;
  sellerId: number;
  sellerName: string;
  saleType: string;
  state: string;
  templateId: number;
  quantity: number;
  strength: number;
  dexterity: number;
  intelligence: number;
  luck: number;
  hp: number;
  mp: number;
  weaponAttack: number;
  magicAttack: number;
  weaponDefense: number;
  magicDefense: number;
  accuracy: number;
  avoidability: number;
  hands: number;
  speed: number;
  jump: number;
  slots: number;
  level: number;
  itemLevel: number;
  itemExp: number;
  ringId: number;
  viciousCount: number;
  flags: number;
  listValue: number;
  buyNowPrice?: number;
  commissionRate: number;
  category: string;
  subCategory: string;
  endsAt?: string;
  currentBid: number;
  highBidderId: number;
  minIncrement: number;
  createdAt: string;
  updatedAt: string;
}

export interface MtsListing {
  id: string;
  attributes: MtsListingAttributes;
}

export interface BrowseListingsFilter {
  category?: string | undefined;
  subCategory?: string | undefined;
  /** Maps to the backend `saleType` query param (BUY_NOW / AUCTION). */
  saleType?: string | undefined;
  sellerName?: string | undefined;
  itemId?: number | undefined;
  page?: number | undefined;
  pageSize?: number | undefined;
}

/**
 * Build the flat query string the atlas-mts browse endpoint expects. The
 * endpoint reads bare query params (NOT JSON:API filter brackets), so only
 * provided, non-empty filters are emitted.
 */
export function buildBrowseListingsQuery(filter: BrowseListingsFilter): string {
  const params = new URLSearchParams();
  if (filter.category) params.set("category", filter.category);
  if (filter.subCategory) params.set("subCategory", filter.subCategory);
  if (filter.saleType) params.set("saleType", filter.saleType);
  if (filter.sellerName) params.set("sellerName", filter.sellerName);
  if (filter.itemId !== undefined && filter.itemId > 0) params.set("itemId", String(filter.itemId));
  if (filter.page !== undefined) params.set("page", String(filter.page));
  if (filter.pageSize !== undefined) params.set("pageSize", String(filter.pageSize));
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export const mtsListingsService = {
  /**
   * Browse active listings for a world. Returns the raw page of listings; the
   * caller derives "has next page" from the returned length vs the page size.
   */
  async browse(
    worldId: number,
    filter: BrowseListingsFilter,
    options?: ServiceOptions,
  ): Promise<MtsListing[]> {
    const query = buildBrowseListingsQuery(filter);
    return api.getList<MtsListing>(`/api/worlds/${worldId}/listings${query}`, options);
  },
};
