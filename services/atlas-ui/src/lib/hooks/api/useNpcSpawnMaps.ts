import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { npcsService } from "@/services/api/npcs.service";
import { useTenant } from "@/context/tenant-context";
import type { NpcSpawnMap } from "@/types/models/npc";

export const npcSpawnMapsKeys = {
  all: ["data", "npcs", "maps"] as const,
  byNpc: (tenantId: string | undefined, npcId: number) =>
    ["data", "npcs", "maps", tenantId ?? "no-tenant", npcId] as const,
};

export function useNpcSpawnMaps(npcId: number): UseQueryResult<NpcSpawnMap[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: npcSpawnMapsKeys.byNpc(activeTenant?.id, npcId),
    queryFn: () => npcsService.getNpcSpawnMaps(npcId),
    enabled: !!activeTenant && npcId > 0,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
