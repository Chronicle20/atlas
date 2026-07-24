import { useQueries } from "@tanstack/react-query";
import { itemStringsService } from "@/services/api/item-strings.service";
import { itemStringKeys } from "@/lib/hooks/api/useItemStrings";
import { useTenant } from "@/context/tenant-context";

/**
 * Batched per-id item-name resolution. Reuses useItemName's query keys so
 * individual lookups cache-share across the browser grid, pool rows, and
 * future visits. `undefined` = still loading or failed (caller degrades to
 * placeholder + numeric id).
 */
export function useItemNames(
  ids: number[],
): Record<number, string | undefined> {
  const { activeTenant } = useTenant();
  const results = useQueries({
    queries: ids.map((id) => ({
      queryKey: itemStringKeys.byId(String(id)),
      queryFn: async () => {
        const item = await itemStringsService.getItemString(String(id));
        return item.attributes.name;
      },
      enabled: !!activeTenant,
      staleTime: 10 * 60 * 1000,
      gcTime: 30 * 60 * 1000,
    })),
  });
  const names: Record<number, string | undefined> = {};
  ids.forEach((id, i) => {
    names[id] = results[i]?.data;
  });
  return names;
}
