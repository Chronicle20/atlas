import type { CharacterPresetAttributes } from "@/types/models/template";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useSyncedNumberInput } from "./useSyncedNumberInput";

interface BaseStatsSectionProps {
  attrs: CharacterPresetAttributes;
  onSetStat: (
    stat: "str" | "dex" | "int" | "luk" | "hp" | "mp",
    value: number,
  ) => void;
}

const STATS = ["str", "dex", "int", "luk", "hp", "mp"] as const;

export function BaseStatsSection({ attrs, onSetStat }: BaseStatsSectionProps) {
  // Local echo so the DOM value reflects keystrokes as they land — the
  // canonical value only updates once the reducer round-trips onSetStat.
  // Re-synced whenever the underlying attr changes (e.g. switching presets).
  const [strInput, setStrInput] = useSyncedNumberInput(attrs.stats.str);
  const [dexInput, setDexInput] = useSyncedNumberInput(attrs.stats.dex);
  const [intInput, setIntInput] = useSyncedNumberInput(attrs.stats.int);
  const [lukInput, setLukInput] = useSyncedNumberInput(attrs.stats.luk);
  const [hpInput, setHpInput] = useSyncedNumberInput(attrs.stats.hp);
  const [mpInput, setMpInput] = useSyncedNumberInput(attrs.stats.mp);

  const draftByStat: Record<
    (typeof STATS)[number],
    [string, (v: string) => void]
  > = {
    str: [strInput, setStrInput],
    dex: [dexInput, setDexInput],
    int: [intInput, setIntInput],
    luk: [lukInput, setLukInput],
    hp: [hpInput, setHpInput],
    mp: [mpInput, setMpInput],
  };

  return (
    <section className="space-y-4">
      <h3 className="text-sm font-semibold">Base stats</h3>
      <div className="grid gap-3 sm:grid-cols-3">
        {STATS.map((stat) => {
          const [draft, setDraft] = draftByStat[stat];
          return (
            <div key={stat} className="space-y-1">
              <Label htmlFor={`preset-stat-${stat}`}>
                {stat.toUpperCase()}
              </Label>
              <Input
                id={`preset-stat-${stat}`}
                aria-label={stat.toUpperCase()}
                type="number"
                min={0}
                value={draft}
                onChange={(e) => {
                  setDraft(e.target.value);
                  onSetStat(stat, Number(e.target.value));
                }}
              />
            </div>
          );
        })}
      </div>
      <p className="text-xs text-muted-foreground">
        Written verbatim to the created character (not derived from level).
      </p>
    </section>
  );
}
