import { useEffect, useState } from "react";
import type { CharacterPresetAttributes } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MapPicker } from "../templates/MapPicker";

interface SpawnProgressionSectionProps {
  attrs: CharacterPresetAttributes;
  onSetField: (path: "mapId" | "level" | "gm" | "meso", value: number) => void;
}

export function SpawnProgressionSection({
  attrs,
  onSetField,
}: SpawnProgressionSectionProps) {
  // Local echo so the DOM value reflects keystrokes as they land — the
  // canonical value only updates once the reducer round-trips onSetField.
  // Re-synced whenever the underlying attr changes (e.g. switching presets).
  const [levelInput, setLevelInput] = useState(String(attrs.level));
  useEffect(() => setLevelInput(String(attrs.level)), [attrs.level]);

  const [gmInput, setGmInput] = useState(String(attrs.gm));
  useEffect(() => setGmInput(String(attrs.gm)), [attrs.gm]);

  const [mesoInput, setMesoInput] = useState(String(attrs.meso));
  useEffect(() => setMesoInput(String(attrs.meso)), [attrs.meso]);

  return (
    <section className="space-y-4">
      <h3 className="text-sm font-semibold">Spawn &amp; progression</h3>
      <div className="space-y-1">
        <Label>Spawn map</Label>
        <MapPicker value={attrs.mapId} onChange={(id) => onSetField("mapId", id)} />
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="space-y-1">
          <Label htmlFor="preset-level">Level</Label>
          <Input
            id="preset-level"
            aria-label="Level"
            type="number"
            min={1}
            max={250}
            value={levelInput}
            onChange={(e) => {
              setLevelInput(e.target.value);
              onSetField("level", Number(e.target.value));
            }}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="preset-gm">GM level</Label>
          <Input
            id="preset-gm"
            aria-label="GM level"
            type="number"
            min={0}
            value={gmInput}
            onChange={(e) => {
              setGmInput(e.target.value);
              onSetField("gm", Number(e.target.value));
            }}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="preset-meso">Meso</Label>
          <Input
            id="preset-meso"
            aria-label="Meso"
            type="number"
            min={0}
            value={mesoInput}
            onChange={(e) => {
              setMesoInput(e.target.value);
              onSetField("meso", Number(e.target.value));
            }}
          />
        </div>
      </div>
    </section>
  );
}
