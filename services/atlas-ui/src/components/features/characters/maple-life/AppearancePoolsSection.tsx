import { AppearanceBrowserDialog } from "../templates/AppearanceBrowserDialog";
import { AppearancePoolSection } from "../templates/AppearancePoolSection";
import { buildMapleLifeVariantLoadout } from "./mapleLifeLoadout";
import type {
  LookDimension,
  MapleLifeClassDraft,
  MapleLifeLookDraft,
  PreviewPicks,
} from "./mapleLifeEditorState";

interface AppearancePoolsSectionProps {
  look: MapleLifeLookDraft;
  draft: MapleLifeClassDraft;
  picks: PreviewPicks;
  onPick: (pick: keyof PreviewPicks, index: number) => void;
  onAddEntry: (dimension: LookDimension, id: number) => void;
  onRemoveEntry: (dimension: LookDimension, entryIndex: number) => void;
  /** Blocking messages keyed "faces" | "hairs" | "hairColors" | "skinColors". */
  errors?: Record<string, string[]>;
}

const POOLS: {
  dimension: LookDimension;
  title: string;
  pick: keyof PreviewPicks;
  domain: string;
}[] = [
  {
    dimension: "faces",
    title: "Faces",
    pick: "faceIdx",
    domain: "Full item ids, e.g. 20000 (male) / 21000 (female).",
  },
  {
    dimension: "hairs",
    title: "Hairs",
    pick: "hairIdx",
    domain: "Normalised style ids — (v/10)*10, e.g. 30030.",
  },
  {
    dimension: "hairColors",
    title: "Hair colours",
    pick: "hairColorIdx",
    domain: "Bare digits 0..9, not full item ids.",
  },
  {
    dimension: "skinColors",
    title: "Skin tones",
    pick: "skinIdx",
    domain: "Bare byte ordinals.",
  },
];

const ALLOW_LIST_NOTE =
  "This is an allow-list: the client sources its own carousel from WZ and the server only checks membership, so a list that diverges from the client's options produces player-visible ErrLookInvalid rejections.";

export function AppearancePoolsSection({
  look,
  draft,
  picks,
  onPick,
  onAddEntry,
  onRemoveEntry,
  errors,
}: AppearancePoolsSectionProps) {
  return (
    <div className="space-y-6">
      {POOLS.map(({ dimension, title, pick, domain }) => (
        <div key={dimension} className="space-y-1">
          <AppearancePoolSection
            dimension={dimension}
            title={title}
            pool={look[dimension]}
            selectedIndex={picks[pick]}
            variantLoadout={(_dim, id) =>
              buildMapleLifeVariantLoadout(draft, look, picks, dimension, id)
            }
            onPick={(idx) => onPick(pick, idx)}
            onRemoveEntry={(idx) => onRemoveEntry(dimension, idx)}
            description={
              <>
                {domain} {ALLOW_LIST_NOTE}
              </>
            }
            renderAddDialog={(open, onOpenChange) => (
              <AppearanceBrowserDialog
                dimension={dimension}
                gender={look.gender}
                variantLoadout={(_dim, id) =>
                  buildMapleLifeVariantLoadout(
                    draft,
                    look,
                    picks,
                    dimension,
                    id,
                  )
                }
                open={open}
                onOpenChange={onOpenChange}
                onSelect={(id) => onAddEntry(dimension, id)}
                selectMode="add"
                markedIds={look[dimension]}
              />
            )}
          />
          {errors?.[dimension]?.map((message) => (
            <p key={message} className="text-xs text-destructive">
              {message}
            </p>
          ))}
        </div>
      ))}
    </div>
  );
}
