import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import { Skeleton } from "@/components/ui/skeleton";
import { useSkillDefinition } from "@/lib/hooks/api/useSkillDefinition";
import type { Tenant } from "@/services/api/tenants.service";
import { SkillTooltipContent } from "./SkillTooltipContent";
import { cn } from "@/lib/utils";

interface Props {
  skillId: number;
  learnedLevel?: number | undefined;
  learnedMasterLevel?: number | undefined;
  tenant: Tenant;
}

export function SkillWidget({ skillId, learnedLevel, learnedMasterLevel, tenant }: Props) {
  const { data, isLoading } = useSkillDefinition(tenant, skillId);
  if (isLoading) return <Skeleton className="h-24 w-full" />;
  if (!data) return null;

  // The skill's true max level is effects.length (each effect entry = one
  // master-level row). atlas-skills's per-character `masterLevel` is a soft
  // cap that may be 0 for unlearned skills, so fall back to the definition.
  const masterLevel = learnedMasterLevel && learnedMasterLevel > 0
    ? learnedMasterLevel
    : data.effects.length;
  const learned = learnedLevel != null && learnedLevel > 0;
  const displayLevel = learnedLevel ?? 0;
  const levelLine = `${displayLevel} / ${masterLevel}`;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div
            tabIndex={0}
            className={cn(
              "flex flex-col items-center gap-1 rounded border p-2 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              !learned && "opacity-50",
            )}
          >
            <img src={data.iconUrl} alt={data.name} className="w-16 h-16 object-contain" />
            <span className="text-sm text-center">{data.name}</span>
            <span className="text-xs text-muted-foreground">{levelLine}</span>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <SkillTooltipContent definition={data} learnedLevel={learnedLevel} />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
