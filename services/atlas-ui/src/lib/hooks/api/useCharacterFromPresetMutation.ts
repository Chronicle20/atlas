import { useMutation, useQueryClient, type UseMutationResult } from "@tanstack/react-query";
import { factoryService, type CreateFromPresetPayload, type CreateFromPresetResponse } from "@/services/api/factory.service";
import { characterKeys } from "@/lib/hooks/api/useCharacters";
import type { Tenant } from "@/types/models/tenant";

export function useCreateCharacterFromPreset(
  tenant: Tenant,
): UseMutationResult<CreateFromPresetResponse, Error, CreateFromPresetPayload> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateFromPresetPayload) =>
      factoryService.createFromPreset(tenant, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: characterKeys.lists() });
    },
  });
}
