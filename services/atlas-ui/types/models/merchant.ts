export interface MerchantShopAttributes {
  characterId: number;
  shopType: number;
  state: number;
  title: string;
  mapId: number;
  x: number;
  y: number;
  permitItemId: number;
  closeReason: number;
  mesoBalance: number;
  listingCount: number;
  visitors?: number[];
}

export interface MerchantShop {
  id: string;
  attributes: MerchantShopAttributes;
}

export interface MerchantListingAttributes {
  shopId: string;
  itemId: number;
  itemType: number;
  quantity: number;
  bundleSize: number;
  bundlesRemaining: number;
  pricePerBundle: number;
  itemSnapshot: unknown;
  displayOrder: number;
}

export interface MerchantListing {
  id: string;
  attributes: MerchantListingAttributes;
}

export interface ListingSearchResultAttributes {
  shopId: string;
  shopTitle: string;
  mapId: number;
  itemId: number;
  itemType: number;
  quantity: number;
  bundleSize: number;
  bundlesRemaining: number;
  pricePerBundle: number;
}

export interface ListingSearchResult {
  id: string;
  attributes: ListingSearchResultAttributes;
}

export function getShopTypeName(shopType: number): string {
  switch (shopType) {
    case 1: return "Character Shop";
    case 2: return "Hired Merchant";
    default: return "Unknown";
  }
}

export function getShopStateName(state: number): string {
  switch (state) {
    case 1: return "Draft";
    case 2: return "Open";
    case 3: return "Maintenance";
    case 4: return "Closed";
    default: return "Unknown";
  }
}

export function getShopTypeBadgeVariant(shopType: number): string {
  switch (shopType) {
    case 1: return "text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30";
    case 2: return "text-purple-600 bg-purple-100 dark:text-purple-400 dark:bg-purple-900/30";
    default: return "";
  }
}

export function getShopStateBadgeVariant(state: number): string {
  switch (state) {
    case 2: return "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/30";
    case 3: return "text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/30";
    case 4: return "text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/30";
    default: return "";
  }
}
