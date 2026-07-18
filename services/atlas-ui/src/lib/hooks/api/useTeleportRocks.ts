import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  teleportRocksService,
  type TeleportRockListType,
  type TeleportRockLists,
} from "@/services/api/teleport-rocks.service";
import { useTenant } from "@/context/tenant-context";
import type { Tenant } from "@/types/models/tenant";

export const teleportRockKeys = {
  all: ["teleport-rocks"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    [...teleportRockKeys.all, tenantId, characterId] as const,
};

export function useTeleportRockMaps(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<TeleportRockLists, Error> {
  return useQuery({
    queryKey: teleportRockKeys.detail(tenant?.id, characterId),
    queryFn: () => teleportRocksService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

interface AddVars {
  characterId: string;
  list: TeleportRockListType;
  mapId: number;
}

type RemoveVars = AddVars;

export function useAddTeleportRockMap(): UseMutationResult<
  TeleportRockLists,
  Error,
  AddVars
> {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  return useMutation({
    mutationFn: (v: AddVars) =>
      teleportRocksService.addMap(v.characterId, v.list, v.mapId),
    onSuccess: (data, v) =>
      qc.setQueryData(
        teleportRockKeys.detail(activeTenant?.id, v.characterId),
        data,
      ),
  });
}

export function useRemoveTeleportRockMap(): UseMutationResult<
  TeleportRockLists,
  Error,
  RemoveVars
> {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  return useMutation({
    mutationFn: (v: RemoveVars) =>
      teleportRocksService.removeMap(v.characterId, v.list, v.mapId),
    onSuccess: (data, v) =>
      qc.setQueryData(
        teleportRockKeys.detail(activeTenant?.id, v.characterId),
        data,
      ),
  });
}
