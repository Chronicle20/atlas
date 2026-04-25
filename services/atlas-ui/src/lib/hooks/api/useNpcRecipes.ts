import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { recipesService } from "@/services/api/recipes.service";
import { useTenant } from "@/context/tenant-context";
import type { Recipe } from "@/types/models/recipe";

export const npcRecipesKeys = {
  all: ["recipes", "byNpc"] as const,
  byNpc: (npcId: number, tenantId?: string) =>
    ["recipes", "byNpc", npcId, tenantId ?? "no-tenant"] as const,
};

export function useNpcRecipes(npcId: number): UseQueryResult<Recipe[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: npcRecipesKeys.byNpc(npcId, activeTenant?.id),
    queryFn: () => recipesService.getByNpc(npcId),
    enabled: !!npcId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
