import { Copy, Send } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import type { MapleStoryCharacterData } from "@/types/models/maplestory";
import { useCharacterImage } from "@/lib/hooks/useCharacterImage";
import { useTenant } from "@/context/tenant-context";
import { jobLabel } from "./presetJobs";
import { buildPresetLoadout } from "./presetLoadout";
import type { WorkingPreset } from "./presetEditorState";

interface PresetCardProps {
  preset: WorkingPreset;
  dirty: boolean;
  onOpen: () => void;
  onDuplicate: () => void;
  /** Hover quick-action; shown only when the caller can offer apply capability. */
  onApply?: () => void;
}

export function PresetCard({ preset, dirty, onOpen, onDuplicate, onApply }: PresetCardProps) {
  const { activeTenant } = useTenant();
  const { attributes: attrs } = preset;
  const loadout = buildPresetLoadout(attrs);

  const character: MapleStoryCharacterData = {
    id: preset.key,
    name: "preview",
    level: attrs.level,
    jobId: attrs.jobId,
    hair: loadout.hair,
    face: loadout.face,
    skinColor: loadout.skin,
    gender: attrs.gender,
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

  return (
    <div className="group relative flex flex-col overflow-hidden rounded-xl border bg-card transition hover:border-primary hover:shadow-lg">
      <div className="absolute right-2 top-2 z-10 flex gap-1 opacity-0 pointer-events-none transition-opacity group-hover:opacity-100 group-hover:pointer-events-auto group-focus-within:opacity-100 group-focus-within:pointer-events-auto">
        <button
          type="button"
          aria-label="Duplicate"
          className="flex h-7 w-7 items-center justify-center rounded-md border bg-background/90 text-muted-foreground backdrop-blur hover:border-primary hover:text-foreground"
          onClick={(e) => {
            e.stopPropagation();
            onDuplicate();
          }}
        >
          <Copy className="h-3.5 w-3.5" />
        </button>
        {onApply && (
          <button
            type="button"
            aria-label="Apply to account"
            className="flex h-7 w-7 items-center justify-center rounded-md border bg-background/90 text-muted-foreground backdrop-blur hover:border-primary hover:text-foreground"
            onClick={(e) => {
              e.stopPropagation();
              onApply();
            }}
          >
            <Send className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
      <button
        type="button"
        aria-label={`Open preset ${attrs.name}`}
        onClick={onOpen}
        className="flex flex-1 flex-col text-left"
      >
        <span className="relative flex h-[118px] items-center justify-center overflow-hidden bg-gradient-to-b from-primary/5 to-primary/15">
          {dirty && (
            <span
              data-testid="dirty-dot"
              title="Unsaved changes"
              className="absolute left-2 top-2 h-1.5 w-1.5 rounded-full bg-primary"
            >
              <span className="sr-only">Unsaved changes</span>
            </span>
          )}
          {image.isLoading && <Skeleton className="h-[100px] w-[80px]" />}
          {!image.isLoading && !image.isError && image.imageUrl && (
            <img
              src={image.imageUrl}
              alt=""
              className="mt-2 h-[104px] w-auto object-contain [image-rendering:pixelated]"
            />
          )}
        </span>
        <span className="flex flex-1 flex-col gap-1.5 p-3">
          <span className="text-[13.5px] font-bold">{attrs.name}</span>
          <span className="flex flex-wrap items-center gap-1.5">
            <Badge variant="secondary">{jobLabel(attrs.jobId)}</Badge>
            <span className="text-[11.5px] font-semibold text-muted-foreground">
              Lv {attrs.level}
              {attrs.gm > 0 ? ` · GM ${attrs.gm}` : ""}
            </span>
          </span>
          {attrs.description && (
            <span className="line-clamp-2 min-h-[2.8em] text-[11.5px] leading-snug text-muted-foreground">
              {attrs.description}
            </span>
          )}
          {attrs.tags.length > 0 && (
            <span className="mt-auto flex flex-wrap gap-1">
              {attrs.tags.map((tag) => (
                <Badge key={tag} variant="outline" className="font-normal">
                  {tag}
                </Badge>
              ))}
            </span>
          )}
        </span>
      </button>
    </div>
  );
}
