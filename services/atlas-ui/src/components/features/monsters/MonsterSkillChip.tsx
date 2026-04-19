import { Badge } from "@/components/ui/badge";
import { useMobSkillData } from "@/lib/hooks/useMobSkillData";

interface MonsterSkillChipProps {
  skillId: number;
  level: number;
}

export function MonsterSkillChip({ skillId, level }: MonsterSkillChipProps) {
  const { name } = useMobSkillData(skillId);
  const label = name ? `${name} · L${level}` : `#${skillId} · L${level}`;
  return (
    <Badge variant="outline" className="px-2.5 py-1 text-xs font-medium">
      {label}
    </Badge>
  );
}
