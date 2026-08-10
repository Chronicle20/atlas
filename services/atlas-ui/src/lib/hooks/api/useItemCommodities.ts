import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { commoditiesService } from "@/services/api/commodities.service";
import { useTenant } from "@/context/tenant-context";
import type { ItemCashShopCommodity } from "@/types/models/npc";

export const itemCommoditiesKeys = {
  all: ["items", "commodities"] as const,
  byItem: (itemId: string, tenantId?: string) =>
    ["items", itemId, "commodities", tenantId ?? ""] as const,
  bySerial: (serialNumber: string, tenantId?: string) =>
    ["commodities", "serial", serialNumber, tenantId ?? ""] as const,
};

export function useItemCommodities(
  itemId: string,
): UseQueryResult<ItemCashShopCommodity[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemCommoditiesKeys.byItem(itemId, activeTenant?.id),
    queryFn: () => commoditiesService.getByItem(itemId),
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}

/**
 * The reverse lookup: one commodity by its SERIAL NUMBER. Used to show which
 * item an already-chosen coupon serial actually grants. A serial that names
 * no commodity 404s — the caller renders that as "unknown serial" rather than
 * an error, since the operator may still be typing.
 */
export function useCommodityBySerial(
  serialNumber: string,
): UseQueryResult<ItemCashShopCommodity, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemCommoditiesKeys.bySerial(serialNumber, activeTenant?.id),
    queryFn: () => commoditiesService.getBySerialNumber(serialNumber),
    enabled: !!serialNumber && !!activeTenant,
    retry: false,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
