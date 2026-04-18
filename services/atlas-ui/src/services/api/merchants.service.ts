import { BaseService, type QueryOptions } from './base.service';
import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';
import type { MerchantShop, MerchantListing, ListingSearchResult } from '@/types/models/merchant';

class MerchantsService extends BaseService {
  protected basePath = '/api/merchants';

  async getAllShops(tenant: Tenant, options?: QueryOptions): Promise<MerchantShop[]> {
    api.setTenant(tenant);
    return this.getAll<MerchantShop>(options);
  }

  async getShopById(shopId: string, tenant: Tenant): Promise<MerchantShop> {
    api.setTenant(tenant);
    return this.getById<MerchantShop>(shopId);
  }

  async getShopListings(shopId: string, tenant: Tenant): Promise<MerchantListing[]> {
    api.setTenant(tenant);
    return api.getList<MerchantListing>(`${this.basePath}/${shopId}/relationships/listings`);
  }

  async searchListings(itemId: number, tenant: Tenant): Promise<ListingSearchResult[]> {
    api.setTenant(tenant);
    return api.getList<ListingSearchResult>(`${this.basePath}/search/listings?itemId=${itemId}`);
  }
}

export const merchantsService = new MerchantsService();
