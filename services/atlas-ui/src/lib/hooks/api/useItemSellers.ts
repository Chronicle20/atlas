import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { npcShopCommoditiesService } from "@/services/api/npc-shop-commodities.service";
import { useTenant } from "@/context/tenant-context";
import type { ItemSellerCommodity } from "@/types/models/npc";

export const itemSellersKeys = {
  all: ["items", "sellers"] as const,
  byItem: (itemId: string, tenantId?: string) =>
    ["items", itemId, "sellers", tenantId ?? ""] as const,
};

export function useItemSellers(itemId: string): UseQueryResult<ItemSellerCommodity[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemSellersKeys.byItem(itemId, activeTenant?.id),
    queryFn: () => npcShopCommoditiesService.getByItem(itemId),
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
