import { generateCharacterUrl } from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import { buildMapleLifeLoadout, combinationCount } from "./mapleLifeLoadout";
import type {
  MapleLifeClassDraft,
  MapleLifeLookDraft,
  PreviewPicks,
} from "./mapleLifeEditorState";

interface MapleLifePreviewCardProps {
  draft: MapleLifeClassDraft;
  look: MapleLifeLookDraft;
  picks: PreviewPicks;
}

export function MapleLifePreviewCard({
  draft,
  look,
  picks,
}: MapleLifePreviewCardProps) {
  const { activeTenant } = useTenant();

  return (
    <div className="sticky top-4 space-y-2 rounded-lg border bg-card p-3">
      <p className="text-xs font-medium text-muted-foreground">Live preview</p>
      <div className="mx-auto flex h-[200px] w-[154px] items-end justify-center rounded-md bg-gradient-to-b from-primary/5 to-primary/15">
        {activeTenant && (
          <img
            src={generateCharacterUrl(
              activeTenant.id,
              activeTenant.attributes.region,
              activeTenant.attributes.majorVersion,
              activeTenant.attributes.minorVersion,
              buildMapleLifeLoadout(draft, look, picks),
              { stance: "stand1", resize: 2 },
            )}
            alt="Live preview of the selected look"
            width={192}
            height={256}
            loading="lazy"
            className="max-h-full w-auto [image-rendering:pixelated] drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]"
          />
        )}
      </div>
      <p className="text-center text-xs text-muted-foreground">
        {combinationCount(look)} combinations offered
      </p>
      <p className="text-center text-xs text-muted-foreground">
        {`${look.faces.length} faces × ${look.hairs.length} hairs × ${look.hairColors.length} hair colours × ${look.skinColors.length} skin tones`}
      </p>
    </div>
  );
}
