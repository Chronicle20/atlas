import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { commoditiesService } from "@/services/api/commodities.service";
import { useTenant } from "@/context/tenant-context";
import type { ItemCashShopCommodity } from "@/types/models/npc";

export const itemCommoditiesKeys = {
  all: ["items", "commodities"] as const,
  byItem: (itemId: string, tenantId?: string) =>
    ["items", itemId, "commodities", tenantId ?? ""] as const,
};

export function useItemCommodities(itemId: string): UseQueryResult<ItemCashShopCommodity[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemCommoditiesKeys.byItem(itemId, activeTenant?.id),
    queryFn: () => commoditiesService.getByItem(itemId),
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
