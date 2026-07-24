import { useEffect, useRef, useState } from "react";
import { Sparkles, X } from "lucide-react";
import type { CharacterPresetSkillEntry } from "@/types/models/template";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSkillData } from "@/lib/hooks/useSkillData";
import { SkillSearchCombobox } from "./SkillSearchCombobox";

interface SkillsSectionProps {
  skills: CharacterPresetSkillEntry[];
  onAdd: (skillId: number) => void;
  onRemove: (index: number) => void;
  onSetLevel: (index: number, value: number) => void;
}

/**
 * Uncontrolled-feeling level input: keeps a local text draft so mid-edit
 * keystrokes (e.g. clearing before typing a new value) aren't clobbered by
 * the parent's clamped-to-1 prop echo. Resyncs from `value` whenever it
 * changes out from under the input (preset switch, external clamp) — but
 * NEVER while the field is focused, otherwise a reducer round-trip
 * triggered by clearing the field (clamped to 1) would echo back mid-edit
 * and prepend a stale "1" in front of the next digit typed. On blur the
 * draft is force-resynced to the current committed value to normalize
 * formatting (e.g. leading zeros, stale text from an aborted edit).
 *
 * Mirrors InventorySection's QuantityInput — same shape of bug, same fix.
 */
function LevelInput({
  value,
  onChange,
}: {
  value: number;
  onChange: (value: number) => void;
}) {
  const [draft, setDraft] = useState(String(value));
  const isFocusedRef = useRef(false);

  useEffect(() => {
    if (!isFocusedRef.current) {
      setDraft(String(value));
    }
  }, [value]);

  return (
    <Input
      type="number"
      min={1}
      aria-label="Level"
      className="w-20"
      value={draft}
      onFocus={() => {
        isFocusedRef.current = true;
      }}
      onBlur={() => {
        isFocusedRef.current = false;
        setDraft(String(value));
      }}
      onChange={(e) => {
        setDraft(e.target.value);
        const parsed = Number(e.target.value);
        if (!Number.isNaN(parsed)) {
          onChange(Math.max(1, parsed));
        }
      }}
    />
  );
}

function SkillRow({
  skillId,
  level,
  onSetLevel,
  onRemove,
}: {
  skillId: number;
  level: number;
  onSetLevel: (value: number) => void;
  onRemove: () => void;
}) {
  const skill = useSkillData(skillId);
  const [iconFailed, setIconFailed] = useState(false);

  return (
    <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
      {skill.iconUrl && !iconFailed ? (
        <img
          src={skill.iconUrl}
          alt=""
          width={28}
          height={28}
          loading="lazy"
          onError={() => setIconFailed(true)}
          className="[image-rendering:pixelated]"
        />
      ) : (
        <Sparkles className="size-7 p-1 text-muted-foreground" />
      )}
      <span className="flex-1 truncate text-sm">
        {skill.name ?? "Unknown skill"}
      </span>
      <span className="font-mono text-xs text-muted-foreground">{skillId}</span>
      <LevelInput value={level} onChange={onSetLevel} />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Remove skill ${skillId}`}
        onClick={onRemove}
      >
        <X className="size-4" />
      </Button>
    </div>
  );
}

export function SkillsSection({
  skills,
  onAdd,
  onRemove,
  onSetLevel,
}: SkillsSectionProps) {
  const existingIds = skills.map((e) => e.skillId);

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Granted skills</h3>
        <SkillSearchCombobox existingIds={existingIds} onAdd={onAdd} />
      </div>

      <div className="space-y-1">
        {skills.length === 0 && (
          <p className="text-sm text-muted-foreground">
            This preset grants no skills.
          </p>
        )}
        {skills.map((e, i) => (
          <SkillRow
            key={`${e.skillId}-${i}`}
            skillId={e.skillId}
            level={e.level}
            onSetLevel={(v) => onSetLevel(i, v)}
            onRemove={() => onRemove(i)}
          />
        ))}
      </div>
    </section>
  );
}
