import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import type { Character } from "@/types/models/character";
import type { Tenant } from "@/services/api/tenants.service";
import { jobTreePath } from "@/lib/utils/job-tree";
import { useCharacterSkills } from "@/lib/hooks/api/useCharacterSkills";
import { useJobSkills } from "@/lib/hooks/api/useJobSkills";
import { SkillWidget } from "./SkillWidget";

interface Props {
  character: Character;
  tenant: Tenant;
}

export function SkillsSection({ character, tenant }: Props) {
  const path = jobTreePath(character.attributes.jobId);
  const { data: characterSkills } = useCharacterSkills(tenant, character.id);

  if (path.length === 0) {
    return <p className="text-sm text-muted-foreground">No skill book available for this job.</p>;
  }

  const learnedById = new Map<number, { level: number; masterLevel: number }>();
  for (const cs of characterSkills ?? []) {
    learnedById.set(parseInt(cs.id, 10), { level: cs.level, masterLevel: cs.masterLevel });
  }

  const defaultJob = String(character.attributes.jobId);

  return (
    <Tabs defaultValue={defaultJob}>
      <TabsList className="flex flex-wrap">
        {path.map((j) => (
          <TabsTrigger key={j.id} value={String(j.id)}>{j.name}</TabsTrigger>
        ))}
      </TabsList>
      {path.map((j) => (
        <TabsContent key={j.id} value={String(j.id)}>
          <JobSkillGrid jobId={j.id} tenant={tenant} learnedById={learnedById} />
        </TabsContent>
      ))}
    </Tabs>
  );
}

function JobSkillGrid({
  jobId, tenant, learnedById,
}: {
  jobId: number;
  tenant: Tenant;
  learnedById: Map<number, { level: number; masterLevel: number }>;
}) {
  const { data: skillIds, isLoading, isError } = useJobSkills(tenant, jobId);
  if (isLoading) {
    return (
      <div className="grid grid-cols-3 sm:grid-cols-5 lg:grid-cols-7 gap-2">
        {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-24" />)}
      </div>
    );
  }
  if (isError || !skillIds || skillIds.length === 0) {
    return <p className="text-sm text-muted-foreground">No skills in this book</p>;
  }
  return (
    <div className="grid grid-cols-3 sm:grid-cols-5 lg:grid-cols-7 gap-2">
      {skillIds.map((id) => {
        const learned = learnedById.get(id);
        return (
          <SkillWidget
            key={id}
            skillId={id}
            learnedLevel={learned?.level}
            learnedMasterLevel={learned?.masterLevel}
            tenant={tenant}
          />
        );
      })}
    </div>
  );
}
