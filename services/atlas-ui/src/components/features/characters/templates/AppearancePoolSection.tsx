import { useState, type ReactNode } from "react";
import { Plus, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CharacterLoadout } from "@/services/api/characterRender.service";
import { generateCharacterUrl } from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import type { AppearancePoolKey } from "./editorState";
import { AppearanceThumb, THUMB_SIZE } from "./AppearanceThumb";

interface AppearancePoolSectionProps {
  dimension: AppearancePoolKey;
  title: string;
  pool: number[];
  selectedIndex: number;
  variantLoadout: (
    dimension: AppearancePoolKey,
    id: number,
  ) => CharacterLoadout;
  onPick: (index: number) => void;
  onRemoveEntry: (entryIndex: number) => void;
  /** Editor supplies the AppearanceBrowserDialog here (open state owned locally). */
  renderAddDialog: (
    open: boolean,
    onOpenChange: (open: boolean) => void,
  ) => ReactNode;
  /** Extra copy under the header — FR-6.5 value domain, FR-6.6 allow-list note. */
  description?: ReactNode;
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
  pool,
  selectedIndex,
  variantLoadout,
  onPick,
  onRemoveEntry,
  renderAddDialog,
  description,
}: AppearancePoolSectionProps) {
  const { activeTenant } = useTenant();
  const [addOpen, setAddOpen] = useState(false);

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
      {description && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
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
                variantLoadout(dimension, id),
                { stance: "stand1", resize: 2 },
              )}
              idLabel={id}
              ariaLabel={`Preview ${NOUN[dimension]} ${id}`}
              selected={selectedIndex === idx}
              onSelect={() => onPick(idx)}
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
