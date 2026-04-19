import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { useMobSkillData } from "@/lib/hooks/useMobSkillData";
import { getMobSkillCanonicalName } from "@/lib/data/mob-skill-names";
import {
  mobSkillsService,
  type MobSkillDetailAttributes,
} from "@/services/api/mob-skills.service";

interface MonsterSkillChipProps {
  skillId: number;
  level: number;
}

export function MonsterSkillChip({ skillId, level }: MonsterSkillChipProps) {
  const { activeTenant } = useTenant();
  const { name: apiName } = useMobSkillData(skillId);
  const canonical = getMobSkillCanonicalName(skillId);
  const displayName = canonical ?? (apiName || undefined);
  const label = displayName ? `${displayName} · L${level}` : `#${skillId} · L${level}`;

  const detailQuery = useQuery({
    queryKey: ["mob-skill-detail", activeTenant?.id ?? "no-tenant", skillId, level],
    queryFn: () => mobSkillsService.getMobSkillDetail(skillId, level),
    enabled: !!activeTenant,
    staleTime: 30 * 60 * 1000,
  });

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant="outline" className="px-2.5 py-1 text-xs font-medium cursor-help">
            {label}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <SkillTooltipBody
            skillId={skillId}
            level={level}
            detail={detailQuery.data}
            isLoading={detailQuery.isLoading}
          />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function SkillTooltipBody({
  skillId,
  level,
  detail,
  isLoading,
}: {
  skillId: number;
  level: number;
  detail: MobSkillDetailAttributes | undefined;
  isLoading: boolean;
}) {
  if (isLoading) {
    return <p className="text-xs text-muted-foreground">Loading effect…</p>;
  }
  if (!detail) {
    return (
      <p className="text-xs text-muted-foreground">
        #{skillId} · L{level}
      </p>
    );
  }
  const rows = summarizeEffect(detail);
  return (
    <div className="space-y-0.5 text-xs">
      <p className="font-mono text-muted-foreground">
        #{skillId} · L{level}
      </p>
      {rows.length > 0 ? (
        rows.map((r) => (
          <div key={r.label} className="flex gap-3">
            <span className="text-muted-foreground">{r.label}</span>
            <span className="ml-auto">{r.value}</span>
          </div>
        ))
      ) : (
        <p className="text-muted-foreground italic">No derived effect data.</p>
      )}
    </div>
  );
}

function summarizeEffect(a: MobSkillDetailAttributes): { label: string; value: string }[] {
  const rows: { label: string; value: string }[] = [];
  if (a.mp_con > 0) rows.push({ label: "MP cost", value: a.mp_con.toLocaleString() });
  if (a.duration > 0) rows.push({ label: "Duration", value: `${a.duration}s` });
  if (a.prop > 0 && a.prop !== 100) rows.push({ label: "Proc", value: `${a.prop}%` });
  if (a.hp > 0 && a.hp < 100) rows.push({ label: "HP trigger", value: `≤ ${a.hp}%` });
  if (a.interval > 0) rows.push({ label: "Interval", value: `${a.interval}s` });
  if (a.count > 0) rows.push({ label: "Count", value: a.count.toLocaleString() });
  if (a.limit > 0) rows.push({ label: "Limit", value: a.limit.toLocaleString() });
  if (a.summons && a.summons.length > 0) {
    rows.push({ label: "Summons", value: `${a.summons.length} mob${a.summons.length === 1 ? "" : "s"}` });
  }
  const aoeW = a.rb_x - a.lt_x;
  const aoeH = a.rb_y - a.lt_y;
  if (aoeW > 0 && aoeH > 0 && (a.lt_x !== 0 || a.lt_y !== 0 || a.rb_x !== 0 || a.rb_y !== 0)) {
    rows.push({ label: "AoE", value: `${aoeW}×${aoeH}` });
  }
  return rows;
}
