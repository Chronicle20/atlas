import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import {
  skillsService,
  type SkillDefinition,
} from "@/services/api/skills.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

export interface SkillDefinitionWithIcon extends SkillDefinition {
  iconUrl: string;
}

export const skillDefinitionKeys = {
  all: ["skill-definition"] as const,
  detail: (tenantId: string | undefined, skillId: number) =>
    ["skill-definition", tenantId, skillId] as const,
};

/** Shared fetcher: skill definition + deterministic icon URL. Reused by the batch hook. */
export async function fetchSkillDefinitionWithIcon(
  tenant: Tenant,
  skillId: number,
): Promise<SkillDefinitionWithIcon> {
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
}

/**
 * Retry policy shared with the batch hook.
 *
 * A 404 here has two causes we cannot tell apart from a single response: a
 * genuinely-missing skill id, or a skill that is briefly unavailable while
 * atlas-data re-ingests a freshly published baseline (a preset row fetches
 * per-skill, so it surfaces every such blip individually — see
 * task-186). So a 404 gets a small bounded number of retries: enough for a
 * re-ingest blip to self-heal without a manual reload, few enough that a
 * permanently-invalid id does not hammer the backend. Non-404 errors keep the
 * original three-attempt budget.
 */
export const SKILL_DEFINITION_404_MAX_RETRIES = 2;

export function skillDefinitionRetry(
  failureCount: number,
  error: Error,
): boolean {
  const msg = error?.message?.toLowerCase() ?? "";
  if (msg.includes("404") || msg.includes("not found")) {
    return failureCount < SKILL_DEFINITION_404_MAX_RETRIES;
  }
  return failureCount < 3;
}

export function useSkillDefinition(
  tenant: Tenant | null | undefined,
  skillId: number,
): UseQueryResult<SkillDefinitionWithIcon, Error> {
  return useQuery({
    queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
    queryFn: () => {
      if (!tenant) throw new Error("Tenant is required");
      return fetchSkillDefinitionWithIcon(tenant, skillId);
    },
    enabled: !!tenant?.id && skillId > 0,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
    retry: skillDefinitionRetry,
  });
}
