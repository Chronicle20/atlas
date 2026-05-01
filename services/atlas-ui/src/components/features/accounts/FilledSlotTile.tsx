// services/atlas-ui/src/components/features/accounts/FilledSlotTile.tsx
import { Link } from "react-router-dom";
import { Globe } from "lucide-react";
import { CharacterRenderer } from "@/components/features/characters/CharacterRenderer";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";
import { tileFrameClasses } from "./tile-frame";

interface FilledSlotTileProps {
  character: Character;
  worlds: TenantConfigAttributes["worlds"];
}

export function FilledSlotTile({ character, worlds }: FilledSlotTileProps) {
  const flag = worlds[character.attributes.worldId]?.flag || "";

  return (
    <Link
      to={`/characters/${character.id}`}
      aria-label={character.attributes.name}
      className={`${tileFrameClasses} flex flex-col items-center justify-center gap-2 hover:bg-accent/50 focus-visible:ring-2 focus-visible:ring-ring`}
    >
      <CharacterRenderer character={character} size="medium" lazy />
      <div className="flex items-center justify-center gap-1.5">
        {flag ? (
          <img
            src={flag}
            width={18}
            height={18}
            alt=""
            loading="lazy"
            className="rounded-sm"
          />
        ) : (
          <Globe className="h-4 w-4 text-muted-foreground" aria-hidden />
        )}
        <span className="text-sm font-medium">{character.attributes.name}</span>
      </div>
    </Link>
  );
}
