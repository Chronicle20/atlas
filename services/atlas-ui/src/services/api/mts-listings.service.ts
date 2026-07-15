import { apiClient } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { ApiPagedResponse } from "@/types/api/responses";

/**
 * Read-only browser for atlas-mts marketplace listings.
 *
 * Backed by the atlas-mts public browse endpoint:
 *   GET /api/worlds/{worldId}/listings
 *     ?category=&subCategory=&saleType=&sellerName=&itemId=&page[number]=&page[size]=
 *
 * The browse endpoint only ever surfaces active listings. The response is a
 * JSON:API list of `listings` resources with a pagination `meta` block
 * (`meta.total` = full match count, `meta.page.last` = last page), so the total
 * and last page are authoritative — never inferred from the returned length.
 *
 * The WIRE convention (task-117) is 1-based `page[number]` (page[number]=1 is
 * the first page). `BrowseListingsFilter.page` stays the pre-task-117
 * ZERO-BASED caller convention (page=0 is the first page) so MarketplacePage's
 * existing 1-based-display -> 0-based-filter conversion doesn't have to
 * change; `buildBrowseListingsQuery` does the +1 onto the wire's
 * `page[number]` internally.
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

/** A page of listings plus the authoritative total/last-page from `meta`. */
export interface MtsListingPage {
  listings: MtsListing[];
  /** Total number of listings matching the filter across all pages. */
  total: number;
  /** Last page number (1-based, for display), derived from meta.page.last. */
  lastPage: number;
}

export interface BrowseListingsFilter {
  category?: string | undefined;
  subCategory?: string | undefined;
  /** Maps to the backend `saleType` query param ("fixed" / "auction"). */
  saleType?: string | undefined;
  sellerName?: string | undefined;
  itemId?: number | undefined;
  page?: number | undefined;
  pageSize?: number | undefined;
}

/**
 * Build the query string the atlas-mts browse endpoint expects: bare filter
 * params plus the standard `page[number]`/`page[size]` paging params (task-117).
 * `filter.page` is this module's ZERO-BASED caller convention; it is rendered
 * onto the wire's ONE-BASED `page[number]` as `page + 1`.
 */
export function buildBrowseListingsQuery(filter: BrowseListingsFilter): string {
  const params = new URLSearchParams();
  if (filter.category) params.set("category", filter.category);
  if (filter.subCategory) params.set("subCategory", filter.subCategory);
  if (filter.saleType) params.set("saleType", filter.saleType);
  if (filter.sellerName) params.set("sellerName", filter.sellerName);
  if (filter.itemId !== undefined && filter.itemId > 0) params.set("itemId", String(filter.itemId));
  if (filter.page !== undefined) params.set("page[number]", String(filter.page + 1));
  if (filter.pageSize !== undefined) params.set("page[size]", String(filter.pageSize));
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export const mtsListingsService = {
  /**
   * Browse active listings for a world. Returns the page of listings together
   * with the authoritative `total` and `lastPage` from the response `meta`, so
   * pagination is exact rather than inferred from the page length.
   */
  async browse(
    worldId: number,
    filter: BrowseListingsFilter,
    options?: ServiceOptions,
  ): Promise<MtsListingPage> {
    const query = buildBrowseListingsQuery(filter);
    const resp = await apiClient.get<ApiPagedResponse<MtsListing>>(
      `/api/worlds/${worldId}/listings${query}`,
      options,
    );
    const total = resp.meta?.total ?? resp.data.length;
    const lastPage = resp.meta?.page?.last ?? 1;
    return { listings: resp.data, total, lastPage };
  },
};
