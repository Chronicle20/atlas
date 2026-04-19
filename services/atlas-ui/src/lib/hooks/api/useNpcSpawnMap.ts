import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { npcsService } from "@/services/api/npcs.service";
import { useTenant } from "@/context/tenant-context";
import type { NpcSpawnMap } from "@/types/models/npc";

export const npcSpawnMapKeys = {
  all: ["npcs", "spawn-map"] as const,
  byId: (npcId: number, tenantId?: string) =>
    ["npcs", "spawn-map", npcId, tenantId ?? ""] as const,
};

export function useNpcSpawnMap(npcId: number): UseQueryResult<NpcSpawnMap | null, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: npcSpawnMapKeys.byId(npcId, activeTenant?.id),
    queryFn: () => npcsService.getSpawnMap(npcId),
    enabled: !!npcId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
