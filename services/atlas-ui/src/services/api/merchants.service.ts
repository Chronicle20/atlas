import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import {
  fetchAll,
  fetchPaged,
  type PagedResult,
} from "@/services/api/pagination";
import type {
  MerchantShop,
  MerchantListing,
  ListingSearchResult,
} from "@/types/models/merchant";

const BASE_PATH = "/api/merchants";

export const merchantsService = {
  /**
   * Get every open merchant shop (matching `options`), draining all pages
   * (task-117).
   */
  async getAllShops(options?: QueryOptions): Promise<MerchantShop[]> {
    return fetchAll<MerchantShop>(
      `${BASE_PATH}${buildQueryString(options)}`,
      undefined,
      options,
    );
  },

  /**
   * Get a single page of open merchant shops (matching `options`). Used by
   * the Merchants list view's "Shops" tab (task-117), which pages
   * server-side.
   */
  async getShopsPage(
    page: { number: number; size: number },
    options?: QueryOptions,
  ): Promise<PagedResult<MerchantShop>> {
    return fetchPaged<MerchantShop>(
      `${BASE_PATH}${buildQueryString(options)}`,
      page,
      options,
    );
  },

  async getShopById(shopId: string): Promise<MerchantShop> {
    return api.getOne<MerchantShop>(`${BASE_PATH}/${shopId}`);
  },

  /**
   * Get every listing for one shop, draining all pages (task-117). Used by
   * the shop detail view, which needs the complete listing set (bounded in
   * practice by `shop.MaxListings`).
   */
  async getShopListings(shopId: string): Promise<MerchantListing[]> {
    return fetchAll<MerchantListing>(
      `${BASE_PATH}/${shopId}/relationships/listings`,
    );
  },

  /**
   * Get every listing matching an item id, draining all pages (task-117).
   * Used by the Merchants search-listings tab, which needs the complete
   * match set for a given item before it paginates the combined results.
   */
  async searchListings(itemId: number): Promise<ListingSearchResult[]> {
    return fetchAll<ListingSearchResult>(
      `${BASE_PATH}/search/listings?itemId=${itemId}`,
    );
  },
};
