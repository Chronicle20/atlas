import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { CharacterTemplate } from "@/types/models/template";
import type { MapleStoryCharacterData } from "@/types/models/maplestory";
import { useCharacterImage } from "@/lib/hooks/useCharacterImage";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";
import type { PreviewPicks } from "./editorState";
import {
  buildPreviewLoadout,
  EQUIP_SLOT_BY_POOL,
  type EquipmentPoolKey,
} from "./previewLoadout";

interface PreviewCardProps {
  template: CharacterTemplate;
  picks: PreviewPicks;
}

function WornIcon({ id }: { id: number }) {
  const { activeTenant } = useTenant();
  const name = useItemName(String(id));
  if (!activeTenant) return null;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <img
          data-testid="worn-icon"
          src={getAssetIconUrl(
            activeTenant.id,
            activeTenant.attributes.region,
            activeTenant.attributes.majorVersion,
            activeTenant.attributes.minorVersion,
            "item",
            id,
          )}
          alt={name.data ?? String(id)}
          width={28}
          height={28}
          loading="lazy"
          className="rounded border bg-muted/40 p-0.5 [image-rendering:pixelated]"
        />
      </TooltipTrigger>
      <TooltipContent>
        {name.data ?? "Unknown item"} · {id}
      </TooltipContent>
    </Tooltip>
  );
}

export function PreviewCard({ template, picks }: PreviewCardProps) {
  const { activeTenant } = useTenant();
  const loadout = buildPreviewLoadout(template, picks);

  const character: MapleStoryCharacterData = {
    id: "template-preview",
    name: "preview",
    level: 1,
    jobId: 0,
    hair: loadout.hair,
    face: loadout.face,
    skinColor: loadout.skin,
    gender: template.gender,
    equipment: loadout.equipment,
    tenant: activeTenant?.id ?? "",
    region: activeTenant?.attributes.region ?? "",
    majorVersion: activeTenant?.attributes.majorVersion ?? 0,
    minorVersion: activeTenant?.attributes.minorVersion ?? 0,
  };

  const image = useCharacterImage(
    character,
    { stance: "stand1", resize: 2 },
    { enabled: !!activeTenant },
  );

  const wornIds = (
    Object.keys(EQUIP_SLOT_BY_POOL) as EquipmentPoolKey[]
  ).flatMap((pool) => {
    const first = template[pool][0];
    return first !== undefined ? [first] : [];
  });

  return (
    <TooltipProvider>
      <div className="rounded-lg border bg-card p-3 lg:sticky lg:top-4">
        <p className="text-xs font-medium text-muted-foreground">
          Live preview
        </p>
        <div className="mx-auto mt-2 flex h-[200px] w-[154px] items-end justify-center rounded-md bg-gradient-to-b from-primary/5 to-primary/15">
          {image.isLoading && <Skeleton className="h-[160px] w-[120px]" />}
          {image.isError && (
            <div className="flex flex-col items-center gap-2 pb-6 text-center">
              <p className="text-xs text-muted-foreground">Preview failed</p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void image.refetch()}
              >
                Retry
              </Button>
            </div>
          )}
          {!image.isLoading && !image.isError && image.imageUrl && (
            <img
              src={image.imageUrl}
              alt="Live preview of the selected template"
              width={192}
              height={256}
              className="max-h-full w-auto [image-rendering:pixelated] drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]"
            />
          )}
        </div>
        {wornIds.length > 0 && (
          <div className="mt-2 flex justify-center gap-1">
            {wornIds.map((id) => (
              <WornIcon key={id} id={id} />
            ))}
          </div>
        )}
        <p className="mt-2 text-center text-xs text-muted-foreground">
          Composited from the highlighted picks and first-of-pool equipment.
        </p>
      </div>
    </TooltipProvider>
  );
}
