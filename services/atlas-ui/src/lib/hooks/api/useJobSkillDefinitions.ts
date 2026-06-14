import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import {
  fetchSkillDefinitionWithIcon,
  skillDefinitionKeys,
  skillDefinitionRetry,
  type SkillDefinitionWithIcon,
} from "@/lib/hooks/api/useSkillDefinition";

export interface UseJobSkillDefinitionsResult {
  /** Resolved definitions, sorted ascending by skill id. */
  definitions: SkillDefinitionWithIcon[];
  isLoading: boolean;
  isError: boolean;
}

export function useJobSkillDefinitions(
  tenant: Tenant | null | undefined,
  skillIds: number[],
): UseJobSkillDefinitionsResult {
  const results = useQueries({
    queries: skillIds.map((skillId) => ({
      queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
      queryFn: () => {
        if (!tenant) throw new Error("Tenant is required");
        return fetchSkillDefinitionWithIcon(tenant, skillId);
      },
      enabled: !!tenant?.id && skillId > 0,
      staleTime: 30 * 60 * 1000,
      gcTime: 24 * 60 * 60 * 1000,
      retry: skillDefinitionRetry,
    })),
  });

  return useMemo(() => {
    const definitions = results
      .map((r) => r.data)
      .filter((d): d is SkillDefinitionWithIcon => d != null)
      .sort((a, b) => a.id - b.id);
    return {
      definitions,
      isLoading: results.some((r) => r.isLoading),
      isError: results.length > 0 && results.every((r) => r.isError),
    };
  }, [results]);
}
