import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import {
  worldsService,
  type WorldData,
  type ChannelData,
} from "@/services/api/worlds.service";

// Topology, not per-second runtime state — same definition cache profile as
// useMapEntities.ts.
export const worldKeys = {
  all: ["worlds"] as const,
  list: () => ["worlds", "list"] as const,
  channels: (worldId: number) => ["worlds", worldId, "channels"] as const,
};

export function useWorlds(): UseQueryResult<WorldData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: worldKeys.list(),
    queryFn: () => worldsService.getWorlds(),
    enabled: !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useChannels(
  worldId: number,
): UseQueryResult<ChannelData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: worldKeys.channels(worldId),
    queryFn: () => worldsService.getChannels(worldId),
    enabled: !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
