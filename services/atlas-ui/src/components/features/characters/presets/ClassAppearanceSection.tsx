import { useMemo, useState } from "react";
import type { CharacterPresetAttributes } from "@/types/models/template";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { useTenant } from "@/context/tenant-context";
import {
  generateCharacterUrl,
  isFemaleCosmeticId,
} from "@/services/api/characterRender.service";
import { useFaceIds, useHairIds } from "@/lib/hooks/api/useCosmetics";
import {
  buildPresetVariantLoadout,
  type PresetAppearanceDimension,
} from "./presetLoadout";
import { AppearanceBrowserDialog } from "../templates/AppearanceBrowserDialog";
import { collapseHairBases } from "../templates/hairBases";
import { AppearanceThumb } from "../templates/AppearanceThumb";
import { JobCombobox } from "./JobCombobox";

type AppearanceField = "face" | "hair" | "hairColor" | "skinColor";

interface ClassAppearanceSectionProps {
  attrs: CharacterPresetAttributes;
  onSetField: (
    path: "jobId" | "gender" | AppearanceField,
    value: number,
  ) => void;
}

const HAIR_COLOR_IDS = [0, 1, 2, 3, 4, 5, 6, 7];
const SKIN_IDS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];
const STARTER_COUNT = 4;

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

/** On-hand row entry: `value` is stored/compared, `renderId` is drawn. */
interface RowTile {
  value: number;
  renderId: number;
}

/** Row order is FIXED — selecting never reshuffles. The current value is
 * appended at the end only when it isn't already an on-hand candidate. */
function withCurrent(tiles: RowTile[], current: RowTile): RowTile[] {
  return tiles.some((t) => t.value === current.value)
    ? tiles
    : [...tiles, current];
}

export function ClassAppearanceSection({
  attrs,
  onSetField,
}: ClassAppearanceSectionProps) {
  const { activeTenant } = useTenant();
  const [browserDimension, setBrowserDimension] =
    useState<PresetAppearanceDimension | null>(null);

  const faces = useFaceIds();
  const hairs = useHairIds();
  const wantFemale = attrs.gender === 1;

  const faceStarters = useMemo<RowTile[]>(
    () =>
      (faces.data ?? [])
        .filter((id) => isFemaleCosmeticId(id) === wantFemale)
        .slice(0, STARTER_COUNT)
        .map((id) => ({ value: id, renderId: id })),
    [faces.data, wantFemale],
  );

  const hairBases = useMemo(
    () => collapseHairBases(hairs.data ?? []),
    [hairs.data],
  );
  const hairStarters = useMemo<RowTile[]>(
    () =>
      hairBases
        .filter((t) => isFemaleCosmeticId(t.value) === wantFemale)
        .slice(0, STARTER_COUNT),
    [hairBases, wantFemale],
  );

  const hairIdSet = useMemo(() => new Set(hairs.data ?? []), [hairs.data]);

  // Colors the SELECTED hair actually exists in (enumeration lists every
  // base+digit variant). Unconstrained until the enumeration loads.
  const validColorDigits = useMemo(() => {
    if (hairIdSet.size === 0) return HAIR_COLOR_IDS;
    const valid = HAIR_COLOR_IDS.filter((d) => hairIdSet.has(attrs.hair + d));
    return valid.length > 0 ? valid : HAIR_COLOR_IDS;
  }, [hairIdSet, attrs.hair]);

  /** Selecting a hair keeps the current color when the new hair has it,
   * otherwise snaps to the new hair's lowest existing color digit — a
   * base+color combination that doesn't exist renders no hair at all. */
  const selectHair = (base: number) => {
    onSetField("hair", base);
    if (hairIdSet.size > 0 && !hairIdSet.has(base + attrs.hairColor)) {
      const [lowest] = HAIR_COLOR_IDS.filter((d) => hairIdSet.has(base + d));
      if (lowest !== undefined) onSetField("hairColor", lowest);
    }
  };

  const currentHairTile: RowTile = {
    value: attrs.hair,
    renderId:
      hairBases.find((t) => t.value === attrs.hair)?.renderId ?? attrs.hair,
  };

  const renderThumbRow = (
    dimension: PresetAppearanceDimension,
    tiles: RowTile[],
    withBrowser: boolean,
    onPick: (value: number) => void,
  ) => {
    const field = FIELD_BY_DIMENSION[dimension];
    return (
      <div key={dimension} className="space-y-1">
        <Label>{ROW_TITLE[dimension]}</Label>
        <div className="flex flex-wrap items-center gap-2">
          {activeTenant &&
            tiles.map((t) => (
              <AppearanceThumb
                key={t.value}
                url={generateCharacterUrl(
                  activeTenant.id,
                  activeTenant.attributes.region,
                  activeTenant.attributes.majorVersion,
                  activeTenant.attributes.minorVersion,
                  buildPresetVariantLoadout(attrs, dimension, t.renderId),
                  { stance: "stand1", resize: 2 },
                )}
                idLabel={t.value}
                ariaLabel={`${NOUN[dimension]} ${t.value}`}
                selected={attrs[field] === t.value}
                onSelect={() => onPick(t.value)}
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

  const asTiles = (ids: number[]): RowTile[] =>
    ids.map((id) => ({ value: id, renderId: id }));

  return (
    <section className="space-y-4">
      <h3 className="text-sm font-semibold">Class &amp; appearance</h3>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="space-y-1">
          <Label>Class</Label>
          <JobCombobox
            value={attrs.jobId}
            onChange={(id) => onSetField("jobId", id)}
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
        {renderThumbRow(
          "faces",
          withCurrent(faceStarters, {
            value: attrs.face,
            renderId: attrs.face,
          }),
          true,
          (v) => onSetField("face", v),
        )}
        {renderThumbRow(
          "hairs",
          withCurrent(hairStarters, currentHairTile),
          true,
          selectHair,
        )}
        {renderThumbRow("hairColors", asTiles(validColorDigits), false, (v) =>
          onSetField("hairColor", v),
        )}
        {renderThumbRow("skinColors", asTiles(SKIN_IDS), false, (v) =>
          onSetField("skinColor", v),
        )}
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
            browserDimension === "hairs"
              ? selectHair(id)
              : onSetField(FIELD_BY_DIMENSION[browserDimension], id)
          }
          selectMode="replace"
          selectedId={attrs[FIELD_BY_DIMENSION[browserDimension]]}
        />
      )}
    </section>
  );
}
