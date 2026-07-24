import { useState } from "react";
import { Sparkles } from "lucide-react";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";

export function SkillIcon({
  def,
  name,
}: {
  def: SkillDefinitionWithIcon;
  name: string;
}) {
  const [failed, setFailed] = useState(false);
  if (failed) {
    return (
      <span
        data-testid={`skill-icon-fallback-${def.id}`}
        className="inline-flex h-8 w-8 items-center justify-center text-muted-foreground"
      >
        <Sparkles className="h-4 w-4" />
      </span>
    );
  }
  return (
    <img
      src={def.iconUrl}
      alt={name}
      width={32}
      height={32}
      loading="lazy"
      className="object-contain"
      onError={() => setFailed(true)}
    />
  );
}
