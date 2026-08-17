import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  pendingChangesService,
  type PendingChange,
} from "@/services/api/pending-changes.service";
import type { Tenant } from "@/types/models/tenant";

export const pendingChangeKeys = {
  all: ["pending-changes"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    [...pendingChangeKeys.all, tenantId ?? "no-tenant", characterId] as const,
};

export function usePendingChanges(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<PendingChange[], Error> {
  return useQuery({
    queryKey: pendingChangeKeys.detail(tenant?.id, characterId),
    queryFn: () => pendingChangesService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

interface CancelVars {
  tenant: Tenant | null | undefined;
  characterId: string;
  id: string;
}

export function useCancelPendingChange(): UseMutationResult<
  void,
  Error,
  CancelVars
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: CancelVars) =>
      pendingChangesService.cancel(v.characterId, v.id),
    onSuccess: (_data, v) =>
      qc.invalidateQueries({
        queryKey: pendingChangeKeys.detail(v.tenant?.id, v.characterId),
      }),
  });
}
