// services/atlas-ui/src/components/features/accounts/FilledSlotTile.tsx
import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { CharacterRenderer } from "@/components/features/characters/CharacterRenderer";
import { useInventory } from "@/lib/hooks/api/useInventory";
import type { Asset } from "@/services/api/inventory.service";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";
import type { Tenant } from "@/types/models/tenant";
import { cn } from "@/lib/utils";
import { getWorldIconUrl } from "@/lib/utils/asset-url";
import { tileFrameClasses } from "./tile-frame";

interface FilledSlotTileProps {
  character: Character;
  tenant: Tenant;
  worlds: TenantConfigAttributes["worlds"];
}

export function FilledSlotTile({ character, tenant, worlds }: FilledSlotTileProps) {
  const inventoryQuery = useInventory(tenant, character.id);
  const [iconLoadFailed, setIconLoadFailed] = useState(false);

  const equippedAssets = useMemo<Asset[]>(() => {
    return (
      inventoryQuery.data?.included?.filter(
        (item): item is Asset =>
          item.type === "assets" && "slot" in item.attributes && item.attributes.slot < 0,
      ) ?? []
    );
  }, [inventoryQuery.data]);

  const worldName = worlds[character.attributes.worldId]?.name ?? "";
  const worldIconUrl =
    !iconLoadFailed &&
    tenant.attributes.region &&
    typeof tenant.attributes.majorVersion === "number" &&
    typeof tenant.attributes.minorVersion === "number"
      ? getWorldIconUrl(
          tenant.id,
          tenant.attributes.region,
          tenant.attributes.majorVersion,
          tenant.attributes.minorVersion,
          character.attributes.worldId,
        )
      : "";

  return (
    <Link
      to={`/characters/${character.id}`}
      aria-label={character.attributes.name}
      className="flex flex-col items-center gap-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md"
    >
      <div
        className={cn(
          tileFrameClasses,
          "flex items-end justify-center hover:bg-accent/50",
        )}
      >
        <CharacterRenderer
          character={character}
          inventory={equippedAssets}
          size="medium"
          lazy
          frameMode="platform"
          {...(tenant.attributes.region && { region: tenant.attributes.region })}
          {...(tenant.attributes.majorVersion && {
            majorVersion: tenant.attributes.majorVersion,
          })}
        />
      </div>
      <div className="flex flex-col items-center text-center">
        <span className="text-sm font-medium leading-tight">
          {character.attributes.name}
        </span>
        {worldName && (
          <span className="flex items-center gap-1 text-xs text-muted-foreground leading-tight">
            {worldIconUrl && (
              <img
                src={worldIconUrl}
                width={16}
                height={16}
                alt=""
                loading="lazy"
                onError={() => setIconLoadFailed(true)}
              />
            )}
            {worldName}
          </span>
        )}
      </div>
    </Link>
  );
}
