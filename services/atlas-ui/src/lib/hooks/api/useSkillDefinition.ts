import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { skillsService, type SkillDefinition } from "@/services/api/skills.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

export interface SkillDefinitionWithIcon extends SkillDefinition {
  iconUrl: string;
}

export const skillDefinitionKeys = {
  all: ["skill-definition"] as const,
  detail: (tenantId: string | undefined, skillId: number) =>
    ["skill-definition", tenantId, skillId] as const,
};

export function useSkillDefinition(
  tenant: Tenant | null | undefined,
  skillId: number
): UseQueryResult<SkillDefinitionWithIcon, Error> {
  return useQuery({
    queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
    queryFn: async () => {
      if (!tenant) throw new Error("Tenant is required");
      const def = await skillsService.getSkillById(skillId.toString());
      return {
        ...def,
        iconUrl: getAssetIconUrl(
          tenant.id,
          tenant.attributes.region,
          tenant.attributes.majorVersion,
          tenant.attributes.minorVersion,
          "skill",
          skillId,
        ),
      };
    },
    enabled: !!tenant?.id && skillId > 0,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
