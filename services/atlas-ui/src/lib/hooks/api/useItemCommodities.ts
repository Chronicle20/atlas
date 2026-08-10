import {
  useQueries,
  useQuery,
  type UseQueryResult,
} from "@tanstack/react-query";
import { commoditiesService } from "@/services/api/commodities.service";
import { itemStringsService } from "@/services/api/item-strings.service";
import { useTenant } from "@/context/tenant-context";
import type { ItemCashShopCommodity } from "@/types/models/npc";

export const itemCommoditiesKeys = {
  all: ["items", "commodities"] as const,
  byItem: (itemId: string, tenantId?: string) =>
    ["items", itemId, "commodities", tenantId ?? ""] as const,
  catalog: (tenantId?: string) =>
    ["commodities", "catalog", tenantId ?? ""] as const,
  serialName: (serialNumber: string, tenantId?: string) =>
    ["commodities", "serial-name", serialNumber, tenantId ?? ""] as const,
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
 * The whole cash-shop catalog, indexed both ways.
 *
 * A serial-number picker has to answer set questions — "does this item have a
 * serial at all?", "which serials?", "what does serial N grant?" — and the one
 * drained catalog answers all three in memory. `enabled` is left to the caller
 * so the drain only happens when a picker actually opens.
 */
export interface CommodityCatalog {
  /** Commodities for an item id, in serial order. Absent = no cash-shop entry. */
  byItemId: Map<number, ItemCashShopCommodity[]>;
  /** Commodity by serial number (its JSON:API id, as a string). */
  bySerial: Map<string, ItemCashShopCommodity>;
}

function indexCatalog(rows: ItemCashShopCommodity[]): CommodityCatalog {
  const byItemId = new Map<number, ItemCashShopCommodity[]>();
  const bySerial = new Map<string, ItemCashShopCommodity>();
  for (const row of rows) {
    bySerial.set(row.id, row);
    const existing = byItemId.get(row.itemId);
    if (existing) existing.push(row);
    else byItemId.set(row.itemId, [row]);
  }
  for (const list of byItemId.values()) {
    list.sort((a, b) => Number(a.id) - Number(b.id));
  }
  return { byItemId, bySerial };
}

export function useCommodityCatalog(
  enabled: boolean,
): UseQueryResult<CommodityCatalog, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemCommoditiesKeys.catalog(activeTenant?.id),
    queryFn: async () => indexCatalog(await commoditiesService.drainAll()),
    enabled: enabled && !!activeTenant,
    // The catalog is WZ-derived and only changes on a re-ingest.
    staleTime: 30 * 60 * 1000,
    gcTime: 60 * 60 * 1000,
  });
}

/**
 * Batched serial number → item NAME, for rendering rewards as "Zeta Nova Hat"
 * instead of a bare serial. Two hops per serial — commodity by id, then the
 * item's string — held in one query so the pair is cached as the one fact the
 * caller actually wants.
 *
 * Deliberately not `useCommodityCatalog`: a coupon list shows a handful of
 * distinct serials, and a by-id read is cheap (atlas-data storage GetById),
 * whereas the catalog drain is ~36 requests. The picker drains because it
 * needs the whole SET; a label only needs its own row.
 *
 * `undefined` = still loading, or the serial names no commodity (a stale
 * reward) — callers fall back to showing the serial.
 */
export function useCashItemNames(
  serialNumbers: number[],
): Record<number, string | undefined> {
  const { activeTenant } = useTenant();
  const unique = Array.from(new Set(serialNumbers.filter((sn) => sn > 0)));
  const results = useQueries({
    queries: unique.map((serialNumber) => ({
      queryKey: itemCommoditiesKeys.serialName(
        String(serialNumber),
        activeTenant?.id,
      ),
      queryFn: async () => {
        const commodity =
          await commoditiesService.getBySerialNumber(serialNumber);
        const item = await itemStringsService.getItemString(
          String(commodity.itemId),
        );
        return item.attributes.name;
      },
      enabled: !!activeTenant,
      retry: false,
      staleTime: 30 * 60 * 1000,
      gcTime: 60 * 60 * 1000,
    })),
  });

  const names: Record<number, string | undefined> = {};
  unique.forEach((serialNumber, i) => {
    names[serialNumber] = results[i]?.data;
  });
  return names;
}
