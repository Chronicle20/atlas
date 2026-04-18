import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import type { MerchantShop, MerchantListing, ListingSearchResult } from "@/types/models/merchant";

const BASE_PATH = "/api/merchants";

export const merchantsService = {
  async getAllShops(options?: QueryOptions): Promise<MerchantShop[]> {
    return api.getList<MerchantShop>(`${BASE_PATH}${buildQueryString(options)}`, options);
  },

  async getShopById(shopId: string): Promise<MerchantShop> {
    return api.getOne<MerchantShop>(`${BASE_PATH}/${shopId}`);
  },

  async getShopListings(shopId: string): Promise<MerchantListing[]> {
    return api.getList<MerchantListing>(`${BASE_PATH}/${shopId}/relationships/listings`);
  },

  async searchListings(itemId: number): Promise<ListingSearchResult[]> {
    return api.getList<ListingSearchResult>(`${BASE_PATH}/search/listings?itemId=${itemId}`);
  },
};
