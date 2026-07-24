import { useState, type ReactNode } from "react";
import { Plus, TriangleAlert } from "lucide-react";
import type { CharacterTemplate } from "@/types/models/template";
import { Button } from "@/components/ui/button";
import { generateCharacterUrl } from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import {
  PICK_KEY_BY_POOL,
  type AppearancePoolKey,
  type PreviewPicks,
} from "./editorState";
import { buildVariantLoadout } from "./previewLoadout";
import { AppearanceThumb, THUMB_SIZE } from "./AppearanceThumb";

interface AppearancePoolSectionProps {
  dimension: AppearancePoolKey;
  title: string;
  template: CharacterTemplate;
  picks: PreviewPicks;
  onPick: (pick: keyof PreviewPicks, idx: number) => void;
  onRemoveEntry: (entryIndex: number) => void;
  /** Editor supplies the AppearanceBrowserDialog here (open state owned locally). */
  renderAddDialog: (
    open: boolean,
    onOpenChange: (open: boolean) => void,
  ) => ReactNode;
}

// Singular noun for aria labels, e.g. "Preview face 20000".
const NOUN: Record<AppearancePoolKey, string> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hair color",
  skinColors: "skin tone",
};

export function AppearancePoolSection({
  dimension,
  title,
  template,
  picks,
  onPick,
  onRemoveEntry,
  renderAddDialog,
}: AppearancePoolSectionProps) {
  const { activeTenant } = useTenant();
  const [addOpen, setAddOpen] = useState(false);
  const pickKey = PICK_KEY_BY_POOL[dimension]!;
  const pool = template[dimension];

  return (
    <section className="space-y-2">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className="text-xs text-muted-foreground">
          {pool.length} options · player picks one
        </span>
        {pool.length === 0 && (
          <span className="flex items-center gap-1 text-xs text-warning-foreground">
            <TriangleAlert className="size-3" />
            character creation will fail while this pool is empty
          </span>
        )}
      </div>
      <div className="flex flex-wrap items-start gap-2">
        {activeTenant &&
          pool.map((id, idx) => (
            <AppearanceThumb
              key={`${id}-${idx}`}
              url={generateCharacterUrl(
                activeTenant.id,
                activeTenant.attributes.region,
                activeTenant.attributes.majorVersion,
                activeTenant.attributes.minorVersion,
                buildVariantLoadout(template, picks, dimension, id),
                { stance: "stand1", resize: 2 },
              )}
              idLabel={id}
              ariaLabel={`Preview ${NOUN[dimension]} ${id}`}
              selected={picks[pickKey] === idx}
              onSelect={() => onPick(pickKey, idx)}
              onRemove={() => onRemoveEntry(idx)}
              removeAriaLabel={`Remove ${NOUN[dimension]} ${id}`}
            />
          ))}
        <Button
          type="button"
          variant="outline"
          className="flex-col gap-1 text-xs"
          style={{ width: THUMB_SIZE, height: THUMB_SIZE }}
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-4" /> Add
        </Button>
      </div>
      {renderAddDialog(addOpen, setAddOpen)}
    </section>
  );
}
