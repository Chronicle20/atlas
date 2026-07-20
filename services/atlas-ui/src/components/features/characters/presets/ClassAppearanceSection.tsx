import { useState } from "react";
import type { CharacterPresetAttributes } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useTenant } from "@/context/tenant-context";
import { generateCharacterUrl } from "@/services/api/characterRender.service";
import { PRESET_JOBS, jobLabel } from "./presetJobs";
import {
  buildPresetVariantLoadout,
  type PresetAppearanceDimension,
} from "./presetLoadout";
import { AppearanceBrowserDialog } from "../templates/AppearanceBrowserDialog";
import { AppearanceThumb } from "../templates/AppearanceThumb";
import { useSyncedNumberInput } from "./useSyncedNumberInput";

type AppearanceField = "face" | "hair" | "hairColor" | "skinColor";

interface ClassAppearanceSectionProps {
  attrs: CharacterPresetAttributes;
  onSetField: (
    path: "jobId" | "gender" | AppearanceField,
    value: number,
  ) => void;
}

// GMS-shaped face/hair id ranges; not an enumeration — just a small on-hand
// set for quick reselection. Exhaustive browsing lives behind the "+" dialog.
const FACE_STARTER_IDS = [20000, 20001, 21000, 21001];
const HAIR_STARTER_IDS = [30000, 30010, 30020, 30030];
const HAIR_COLOR_IDS = [0, 1, 2, 3, 4, 5, 6, 7];
const SKIN_IDS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

const FIELD_BY_DIMENSION: Record<PresetAppearanceDimension, AppearanceField> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hairColor",
  skinColors: "skinColor",
};

const NOUN: Record<PresetAppearanceDimension, string> = {
  faces: "face",
  hairs: "hair",
  hairColors: "hair color",
  skinColors: "skin tone",
};

const ROW_TITLE: Record<PresetAppearanceDimension, string> = {
  faces: "Face",
  hairs: "Hair",
  hairColors: "Hair color",
  skinColors: "Skin tone",
};

/** Current value first (deduped), followed by the curated starter ids. */
function onHandIds(current: number, starters: number[]): number[] {
  return [current, ...starters.filter((id) => id !== current)];
}

export function ClassAppearanceSection({
  attrs,
  onSetField,
}: ClassAppearanceSectionProps) {
  const { activeTenant } = useTenant();
  const [browserDimension, setBrowserDimension] =
    useState<PresetAppearanceDimension | null>(null);

  // Local echo so the DOM value reflects keystrokes as they land — the
  // canonical value only updates once the reducer round-trips onSetField.
  const [jobIdInput, setJobIdInput] = useSyncedNumberInput(attrs.jobId);

  const renderThumbRow = (
    dimension: PresetAppearanceDimension,
    ids: number[],
    withBrowser: boolean,
  ) => {
    const field = FIELD_BY_DIMENSION[dimension];
    return (
      <div key={dimension} className="space-y-1">
        <Label>{ROW_TITLE[dimension]}</Label>
        <div className="flex flex-wrap items-center gap-2">
          {activeTenant &&
            ids.map((id) => (
              <AppearanceThumb
                key={id}
                url={generateCharacterUrl(
                  activeTenant.id,
                  activeTenant.attributes.region,
                  activeTenant.attributes.majorVersion,
                  activeTenant.attributes.minorVersion,
                  buildPresetVariantLoadout(attrs, dimension, id),
                  { stance: "stand1", resize: 2 },
                )}
                idLabel={id}
                ariaLabel={`${NOUN[dimension]} ${id}`}
                selected={attrs[field] === id}
                onSelect={() => onSetField(field, id)}
              />
            ))}
          {withBrowser && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={`Browse ${NOUN[dimension]}s`}
              onClick={() => setBrowserDimension(dimension)}
            >
              +
            </Button>
          )}
        </div>
      </div>
    );
  };

  return (
    <section className="space-y-4">
      <h3 className="text-sm font-semibold">Class &amp; appearance</h3>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="space-y-1">
          <Label htmlFor="preset-job">Class</Label>
          <Select
            value={String(attrs.jobId)}
            onValueChange={(v) => onSetField("jobId", Number(v))}
          >
            <SelectTrigger id="preset-job" aria-label="Class">
              <SelectValue>{jobLabel(attrs.jobId)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {PRESET_JOBS.map((j) => (
                <SelectItem key={j.id} value={String(j.id)}>
                  {j.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label htmlFor="preset-job-advanced">Advanced job id</Label>
          <Input
            id="preset-job-advanced"
            aria-label="Advanced job id"
            type="number"
            value={jobIdInput}
            onChange={(e) => {
              setJobIdInput(e.target.value);
              onSetField("jobId", Number(e.target.value));
            }}
          />
        </div>
        <div className="space-y-1">
          <Label>Gender</Label>
          <div className="flex gap-2">
            <Button
              type="button"
              variant={attrs.gender === 0 ? "default" : "outline"}
              aria-pressed={attrs.gender === 0}
              onClick={() => onSetField("gender", 0)}
            >
              Male
            </Button>
            <Button
              type="button"
              variant={attrs.gender === 1 ? "default" : "outline"}
              aria-pressed={attrs.gender === 1}
              onClick={() => onSetField("gender", 1)}
            >
              Female
            </Button>
          </div>
        </div>
      </div>

      <div className="space-y-3">
        {renderThumbRow("faces", onHandIds(attrs.face, FACE_STARTER_IDS), true)}
        {renderThumbRow("hairs", onHandIds(attrs.hair, HAIR_STARTER_IDS), true)}
        {renderThumbRow("hairColors", HAIR_COLOR_IDS, false)}
        {renderThumbRow("skinColors", SKIN_IDS, false)}
      </div>

      {browserDimension && (
        <AppearanceBrowserDialog
          dimension={browserDimension}
          gender={attrs.gender}
          variantLoadout={(dim, id) =>
            buildPresetVariantLoadout(attrs, dim, id)
          }
          open
          onOpenChange={(open) => {
            if (!open) setBrowserDimension(null);
          }}
          onSelect={(id) =>
            onSetField(FIELD_BY_DIMENSION[browserDimension], id)
          }
          selectMode="replace"
          selectedId={attrs[FIELD_BY_DIMENSION[browserDimension]]}
        />
      )}
    </section>
  );
}
