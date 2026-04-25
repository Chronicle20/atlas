import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { recipesService } from "@/services/api/recipes.service";
import { useTenant } from "@/context/tenant-context";
import type { Recipe } from "@/types/models/recipe";

export const itemRecipesKeys = {
  all: ["recipes", "byItem"] as const,
  byItem: (itemId: string, tenantId?: string) =>
    ["recipes", "byItem", itemId, tenantId ?? "no-tenant"] as const,
};

export function useItemRecipes(itemId: string): UseQueryResult<Recipe[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: itemRecipesKeys.byItem(itemId, activeTenant?.id),
    queryFn: () => recipesService.getByItem(itemId),
    enabled: !!itemId && !!activeTenant,
    staleTime: 10 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  });
}
