// services/atlas-ui/src/components/features/accounts/WorldCharactersSection.tsx
import { useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common";
import { useCharacterSlots } from "@/lib/hooks/api/useCharacterSlots";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";
import { cn } from "@/lib/utils";
import { getWorldIconUrl } from "@/lib/utils/asset-url";
import { FilledSlotTile } from "./FilledSlotTile";
import { EmptySlotTile } from "./EmptySlotTile";
import { tileFrameClasses } from "./tile-frame";

type CharacterTemplate =
  TenantConfigAttributes["characters"]["templates"][number];

interface WorldCharactersSectionProps {
  tenant: Tenant;
  account: Account;
  worldId: number;
  worldName: string;
  characters: Character[];
  charactersLoading: boolean;
  charactersError: Error | null;
  emptyTemplate?: CharacterTemplate;
  hasPresets: boolean;
  onAddClick: () => void;
}

// Placeholder tile count for a section's loading skeleton, used only until
// the world's real slot count has loaded. It carries no game-rule meaning
// (the authoritative default/cap live server-side in
// libs/atlas-constants/character) — it just keeps the skeleton grid from
// rendering zero tiles while `useCharacterSlots` is in flight.
const LOADING_SKELETON_TILE_COUNT = 4;

export function WorldCharactersSection({
  tenant,
  account,
  worldId,
  worldName,
  characters,
  charactersLoading,
  charactersError,
  emptyTemplate,
  hasPresets,
  onAddClick,
}: WorldCharactersSectionProps) {
  const slotsQuery = useCharacterSlots(tenant, account.id, worldId);
  const [iconLoadFailed, setIconLoadFailed] = useState(false);

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
          worldId,
        )
      : "";

  const loading = charactersLoading || slotsQuery.isLoading;
  const slots = slotsQuery.data?.attributes.slots ?? 0;
  const overCapacity = !loading && characters.length > slots;
  const emptyCount = loading ? 0 : Math.max(0, slots - characters.length);

  const renderBody = () => {
    if (charactersError) {
      return <ErrorDisplay error={charactersError.message} />;
    }
    if (slotsQuery.error) {
      return <ErrorDisplay error={slotsQuery.error.message} />;
    }
    if (loading) {
      return (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {Array.from({ length: LOADING_SKELETON_TILE_COUNT }).map((_, i) => (
            <Skeleton
              key={i}
              className={cn(tileFrameClasses, "animate-pulse")}
            />
          ))}
        </div>
      );
    }
    return (
      <>
        {overCapacity && (
          <p className="text-xs text-muted-foreground mb-2">
            Over capacity: this world has more characters than allocated slots.
          </p>
        )}
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {characters.map((c) => (
            <FilledSlotTile key={c.id} character={c} tenant={tenant} />
          ))}
          {Array.from({ length: emptyCount }).map((_, i) => (
            <EmptySlotTile
              key={`empty-${i}`}
              onClick={onAddClick}
              disabled={!hasPresets}
              {...(emptyTemplate && { template: emptyTemplate })}
              {...(tenant.attributes.region && {
                region: tenant.attributes.region,
              })}
              {...(tenant.attributes.majorVersion && {
                majorVersion: tenant.attributes.majorVersion,
              })}
            />
          ))}
        </div>
      </>
    );
  };

  return (
    <section>
      <h3 className="flex items-center gap-1 text-sm font-semibold mb-2">
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
        {worldName || `World ${worldId}`}
        {!loading && !slotsQuery.error && (
          <span className="ml-2 text-xs font-normal text-muted-foreground">
            {characters.length}/{slots} slots
          </span>
        )}
      </h3>
      {renderBody()}
    </section>
  );
}
