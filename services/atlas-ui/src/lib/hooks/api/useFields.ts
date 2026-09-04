import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import {
  fieldsService,
  type FieldData,
  type FieldFilters,
} from "@/services/api/fields.service";

// Runtime read model (live occupancy), not a definition — short-lived cache,
// no polling (FR-39). Deliberately disjoint from the "maps" definition key
// namespace (FR-41) and deliberately omits tenant from the key (D9):
// tenant-context.tsx clears the whole query client on tenant change.
export const fieldKeys = {
  all: ["fields"] as const,
  list: (f: FieldFilters) => ["fields", "list", f] as const,
};

const RUNTIME_STALE_TIME = 5 * 1000;
const RUNTIME_GC_TIME = 60 * 1000;

/**
 * The runtime cache profile shared by every fields query — exported so tests
 * can assert on the staleness/GC settings and the absence of polling (FR-39)
 * directly instead of reaching into React Query internals.
 */
export function fieldQueryOptions(filters: FieldFilters) {
  return {
    queryKey: fieldKeys.list(filters),
    queryFn: () => fieldsService.getFields(filters),
    staleTime: RUNTIME_STALE_TIME,
    gcTime: RUNTIME_GC_TIME,
  };
}

export function useFields(
  filters: FieldFilters,
): UseQueryResult<FieldData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    ...fieldQueryOptions(filters),
    enabled: !!activeTenant,
  });
}

// Fields for a single map, spanning every world and channel (FR-9).
export function useFieldsForMap(
  mapId: string,
): UseQueryResult<FieldData[], Error> {
  const filters: FieldFilters = { mapId: Number(mapId) };
  const { activeTenant } = useTenant();
  return useQuery({
    ...fieldQueryOptions(filters),
    enabled: !!mapId && !!activeTenant,
  });
}
