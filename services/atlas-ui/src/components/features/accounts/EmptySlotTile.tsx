// services/atlas-ui/src/components/features/accounts/EmptySlotTile.tsx
import { useMemo } from "react";
import { CharacterRenderer } from "@/components/features/characters/CharacterRenderer";
import type { Asset } from "@/services/api/inventory.service";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";
import { cn } from "@/lib/utils";
import { synthesizeEquippedAssetsFromTemplateIds } from "@/lib/utils/maplestory";
import { tileFrameClasses } from "./tile-frame";

type CharacterTemplate =
  TenantConfigAttributes["characters"]["templates"][number];

interface EmptySlotTileProps {
  onClick: () => void;
  disabled?: boolean;
  template?: CharacterTemplate;
  region?: string;
  majorVersion?: number;
}

function synthesizeCharacter(template: CharacterTemplate): Character {
  const hairBase = template.hairs[0] ?? 30000;
  const hairColor = template.hairColors[0] ?? 0;
  const face = template.faces[0] ?? 20000;
  const skinColor = template.skinColors[0] ?? 0;
  return {
    id: "empty",
    attributes: {
      accountId: 0,
      worldId: 0,
      name: "",
      level: 1,
      experience: 0,
      gachaponExperience: 0,
      strength: 0,
      dexterity: 0,
      intelligence: 0,
      luck: 0,
      hp: 0,
      maxHp: 0,
      mp: 0,
      maxMp: 0,
      meso: 0,
      hpMpUsed: 0,
      jobId: 0,
      skinColor,
      gender: template.gender,
      fame: 0,
      hair: hairBase + hairColor,
      face,
      ap: 0,
      sp: "",
      mapId: template.mapId,
      spawnPoint: 0,
      gm: 0,
      x: 0,
      y: 0,
      stance: 0,
    },
  };
}

function synthesizeStarterEquipment(template: CharacterTemplate): Asset[] {
  // Pick the first available item from each starter equipment array so the
  // silhouette wears a top, bottom, shoes, and a weapon if the template
  // configures them. Misc "items" act as a catch-all (gloves, capes, etc.).
  const candidates: number[] = [];
  for (const list of [
    template.tops,
    template.bottoms,
    template.shoes,
    template.weapons,
    template.items,
  ]) {
    if (list && list.length > 0 && typeof list[0] === "number") {
      candidates.push(list[0] as number);
    }
  }
  return synthesizeEquippedAssetsFromTemplateIds(candidates);
}

export function EmptySlotTile({
  onClick,
  disabled,
  template,
  region,
  majorVersion,
}: EmptySlotTileProps) {
  const character = useMemo(
    () => (template ? synthesizeCharacter(template) : null),
    [template],
  );
  const starterEquipment = useMemo(
    () => (template ? synthesizeStarterEquipment(template) : []),
    [template],
  );

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label="Add character to slot"
      className={cn(
        "group flex flex-col items-center gap-2 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        disabled ? "cursor-not-allowed" : "cursor-pointer",
      )}
    >
      <div
        className={cn(
          tileFrameClasses,
          "flex items-center justify-center bg-muted/40 transition-colors group-hover:bg-accent/50 group-disabled:opacity-50",
        )}
      >
        {character ? (
          <div className="grayscale opacity-40 transition-opacity group-hover:opacity-60">
            <CharacterRenderer
              character={character}
              inventory={starterEquipment}
              size="medium"
              lazy
              {...(region && { region })}
              {...(majorVersion && { majorVersion })}
            />
          </div>
        ) : (
          <span className="text-3xl text-muted-foreground" aria-hidden>
            +
          </span>
        )}
      </div>
      <span className="text-sm text-muted-foreground">+ Add character</span>
    </button>
  );
}
