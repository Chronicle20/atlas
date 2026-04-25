import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { itemsService } from "@/services/api/items.service";
import type { EquipmentData } from "@/types/models/item";

export const equipmentDataKeys = {
  all: ["equipment-data"] as const,
  detail: (templateId: number) => ["equipment-data", templateId] as const,
};

interface UseEquipmentDataOptions {
  enabled?: boolean;
}

/**
 * Wraps `GET /api/data/equipment/{templateId}` (atlas-data) to expose the rich
 * equipment shape — req stats, req job bitmask, base stats, slots, etc. —
 * needed by the in-game-style asset tooltip. Templates are immutable per
 * tenant version, so cache aggressively (30 min stale, 24h gc).
 */
export function useEquipmentData(
  templateId: number,
  options?: UseEquipmentDataOptions,
): UseQueryResult<EquipmentData, Error> {
  const enabled = (options?.enabled ?? true) && templateId > 0;
  return useQuery({
    queryKey: equipmentDataKeys.detail(templateId),
    queryFn: () => itemsService.getEquipment(templateId.toString()),
    enabled,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
    retry: (failureCount, error) => {
      const msg = error?.message?.toLowerCase() ?? "";
      if (msg.includes("404") || msg.includes("not found")) return false;
      return failureCount < 3;
    },
  });
}
