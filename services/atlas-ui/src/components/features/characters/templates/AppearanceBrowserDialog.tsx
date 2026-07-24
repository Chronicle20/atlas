import { useMemo, useState } from "react";
import type { CharacterTemplate } from "@/types/models/template";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ErrorDisplay } from "@/components/common";
import {
  generateCharacterUrl,
  isFemaleCosmeticId,
} from "@/services/api/characterRender.service";
import { useTenant } from "@/context/tenant-context";
import { useFaceIds, useHairIds } from "@/lib/hooks/api/useCosmetics";
import { useItemNames } from "@/lib/hooks/api/useItemNames";
import type { AppearancePoolKey, PreviewPicks } from "./editorState";
import { buildVariantLoadout } from "./previewLoadout";
import { AppearanceThumb } from "./AppearanceThumb";

export const PAGE_SIZE = 24;

const HAIR_COLOR_DIGITS = [0, 1, 2, 3, 4, 5, 6, 7];
// No enumeration endpoint exists for skins; seed data uses 0-3. Offer 0-9
// with rendered previews and let the operator judge (PRD open question 1).
const SKIN_IDS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

const TITLES: Record<AppearancePoolKey, string> = {
  faces: "Browse faces",
  hairs: "Browse hairs",
  hairColors: "Add hair colors",
  skinColors: "Add skin tones",
};

const NOUN: Record<AppearancePoolKey, string> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hair color",
  skinColors: "skin tone",
};

interface AppearanceBrowserDialogProps {
  dimension: AppearancePoolKey;
  template: CharacterTemplate;
  picks: PreviewPicks;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (id: number) => void;
}

export function AppearanceBrowserDialog({
  dimension,
  template,
  picks,
  open,
  onOpenChange,
  onAdd,
}: AppearanceBrowserDialogProps) {
  const { activeTenant } = useTenant();
  const [showAll, setShowAll] = useState(false);
  const [page, setPage] = useState(0);

  const isEnumerated = dimension === "faces" || dimension === "hairs";
  const faces = useFaceIds();
  const hairs = useHairIds();
  const enumQuery = dimension === "faces" ? faces : hairs;

  const candidates = useMemo(() => {
    if (dimension === "hairColors") return HAIR_COLOR_DIGITS;
    if (dimension === "skinColors") return SKIN_IDS;
    const all = enumQuery.data ?? [];
    const wantFemale = template.gender === 1;
    if (!showAll) {
      return all.filter((id) => isFemaleCosmeticId(id) === wantFemale);
    }
    // Show-all: lead with the currently-hidden (opposite-gender) candidates
    // so toggling doesn't bury them behind however many pages of
    // already-visible same-gender ids happen to sort first.
    const opposite = all.filter((id) => isFemaleCosmeticId(id) !== wantFemale);
    const wanted = all.filter((id) => isFemaleCosmeticId(id) === wantFemale);
    return [...opposite, ...wanted];
  }, [dimension, enumQuery.data, showAll, template.gender]);

  const pageCount = Math.max(1, Math.ceil(candidates.length / PAGE_SIZE));
  const clampedPage = Math.min(page, pageCount - 1);
  const pageIds = candidates.slice(
    clampedPage * PAGE_SIZE,
    (clampedPage + 1) * PAGE_SIZE,
  );

  // Names only exist for faces/hairs (item-strings covers them by id; the
  // search index does NOT — enumerate + resolve per page, never search).
  const names = useItemNames(isEnumerated ? pageIds : []);

  const inPool = template[dimension];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{TITLES[dimension]}</DialogTitle>
        </DialogHeader>
        {isEnumerated && (
          <div className="flex items-center gap-2">
            <Switch
              id="show-all-genders"
              checked={showAll}
              onCheckedChange={(v) => {
                setShowAll(v);
                setPage(0);
              }}
              aria-label="Show all genders"
            />
            <Label htmlFor="show-all-genders" className="text-sm">
              Show all genders
            </Label>
          </div>
        )}
        {isEnumerated && enumQuery.isError ? (
          <ErrorDisplay
            error={`Failed to enumerate ${dimension}`}
            retry={() => void enumQuery.refetch()}
          />
        ) : (
          <>
            <div className="grid max-h-[420px] grid-cols-4 gap-2 overflow-y-auto sm:grid-cols-6">
              {activeTenant &&
                pageIds.map((id) => (
                  <div key={id} className="flex flex-col items-center gap-0.5">
                    <AppearanceThumb
                      url={generateCharacterUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        buildVariantLoadout(template, picks, dimension, id),
                        { stance: "stand1", resize: 2 },
                      )}
                      idLabel={id}
                      ariaLabel={`Add ${NOUN[dimension]} ${id}`}
                      marked={inPool.includes(id)}
                      onSelect={() => onAdd(id)}
                    />
                    {isEnumerated && (
                      <span className="max-w-[76px] truncate text-[10px] text-muted-foreground">
                        {names[id] ?? "…"}
                      </span>
                    )}
                  </div>
                ))}
              {isEnumerated && enumQuery.isLoading && (
                <p className="col-span-full text-sm text-muted-foreground">
                  Loading candidates…
                </p>
              )}
            </div>
            {pageCount > 1 && (
              <div className="flex items-center justify-between">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={clampedPage === 0}
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                >
                  Previous
                </Button>
                <span className="text-xs text-muted-foreground">
                  Page {clampedPage + 1} of {pageCount}
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={clampedPage >= pageCount - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
