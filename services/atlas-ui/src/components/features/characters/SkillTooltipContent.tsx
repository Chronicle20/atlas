import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { formatStatup } from "@/lib/utils/skill-effect-format";

interface Props {
  definition: SkillDefinitionWithIcon;
  learnedLevel?: number | undefined;
}

export function SkillTooltipContent({ definition, learnedLevel }: Props) {
  const idx = Math.max(0, (learnedLevel ?? 1) - 1);
  const effect = definition.effects?.[idx];

  return (
    <div className="space-y-1 max-w-sm text-xs">
      <div className="flex items-center gap-2">
        <img src={definition.iconUrl} alt={definition.name} className="w-6 h-6" />
        <span className="font-semibold">{definition.name}</span>
        <span className="text-muted-foreground">#{definition.id}</span>
      </div>
      {definition.description && (
        <p data-testid="skill-description">{definition.description}</p>
      )}
      {definition.element && <div><strong>Element:</strong> {definition.element}</div>}
      {definition.animationTime > 0 && <div><strong>Animation:</strong> {definition.animationTime}ms</div>}
      {effect?.cooldown != null && effect.cooldown > 0 && (
        <div><strong>Cooldown:</strong> {effect.cooldown}s</div>
      )}
      {effect?.MPConsume != null && effect.MPConsume > 0 && (
        <div><strong>MP Cost:</strong> {effect.MPConsume}</div>
      )}
      {effect?.HPConsume != null && effect.HPConsume > 0 && (
        <div><strong>HP Cost:</strong> {effect.HPConsume}</div>
      )}
      {effect?.statups && effect.statups.length > 0 && (
        <ul className="pl-3 list-disc">
          {effect.statups.map((s, i) => (
            <li key={i}>{formatStatup(s, effect.duration ?? 0)}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
