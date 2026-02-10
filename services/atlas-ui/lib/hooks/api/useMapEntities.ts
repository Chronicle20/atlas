import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useTenant } from '@/context/tenant-context';
import {
  mapEntitiesService,
  type MapPortalData,
  type MapNpcData,
  type MapReactorData,
  type MapMonsterData,
} from '@/services/api/map-entities.service';

export const mapEntityKeys = {
  portals: (mapId: string) => ['maps', mapId, 'portals'] as const,
  npcs: (mapId: string) => ['maps', mapId, 'npcs'] as const,
  reactors: (mapId: string) => ['maps', mapId, 'reactors'] as const,
  monsters: (mapId: string) => ['maps', mapId, 'monsters'] as const,
};

export function useMapPortals(mapId: string): UseQueryResult<MapPortalData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapEntityKeys.portals(mapId),
    queryFn: () => mapEntitiesService.getPortals(mapId, activeTenant!, { useCache: false }),
    enabled: !!mapId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useMapNpcs(mapId: string): UseQueryResult<MapNpcData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapEntityKeys.npcs(mapId),
    queryFn: () => mapEntitiesService.getNpcs(mapId, activeTenant!, { useCache: false }),
    enabled: !!mapId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useMapReactors(mapId: string): UseQueryResult<MapReactorData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapEntityKeys.reactors(mapId),
    queryFn: () => mapEntitiesService.getReactors(mapId, activeTenant!, { useCache: false }),
    enabled: !!mapId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useMapMonsters(mapId: string): UseQueryResult<MapMonsterData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: mapEntityKeys.monsters(mapId),
    queryFn: () => mapEntitiesService.getMonsters(mapId, activeTenant!, { useCache: false }),
    enabled: !!mapId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
