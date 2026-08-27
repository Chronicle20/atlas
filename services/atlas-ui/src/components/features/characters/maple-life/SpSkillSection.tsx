import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { MapleLifeClassDraft } from "./mapleLifeEditorState";
import { KNOWN_SP_SKILL_IDS, SP_SKILL_LABELS } from "./mapleLifeWarnings";

interface SpSkillSectionProps {
  draft: MapleLifeClassDraft;
  onSetSpSkillId: (value: number | undefined) => void;
  errors?: Record<string, string[]>;
  warnings?: string[];
}

const NONE_VALUE = "none";
const DISABLED_REASON_ID = "maple-life-sp-skill-disabled-reason";
const DISABLED_REASON =
  "The client skips step 4 for class ordinal >= 2, so a value set here can never take effect and a player submitting sp != 0 is rejected outright.";

function FieldErrors({ messages }: { messages: string[] | undefined }) {
  if (!messages || messages.length === 0) return null;
  return (
    <div className="space-y-0.5">
      {messages.map((message) => (
        <p key={message} className="text-xs text-destructive">
          {message}
        </p>
      ))}
    </div>
  );
}

/**
 * The single SP skill offer for the selected class row (FR-8). The client
 * only reads this at step 4, which it skips for ordinal >= 2 — the control
 * is disabled there but a loaded non-zero value stays visible and its
 * errors still render, since the page never hides what it can't act on.
 */
export function SpSkillSection({
  draft,
  onSetSpSkillId,
  errors,
  warnings,
}: SpSkillSectionProps) {
  const disabled = draft.ordinal >= 2;
  const spSkillId = draft.spSkillId;
  const isKnown =
    spSkillId !== undefined &&
    (KNOWN_SP_SKILL_IDS as readonly number[]).includes(spSkillId);
  const selectValue = spSkillId === undefined ? NONE_VALUE : String(spSkillId);

  return (
    <section className="space-y-2">
      <Label htmlFor="maple-life-sp-skill">SP skill</Label>
      <Select
        value={selectValue}
        onValueChange={(v) =>
          onSetSpSkillId(v === NONE_VALUE ? undefined : Number(v))
        }
        disabled={disabled}
      >
        <SelectTrigger
          id="maple-life-sp-skill"
          aria-label="SP skill"
          aria-describedby={disabled ? DISABLED_REASON_ID : undefined}
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NONE_VALUE}>None</SelectItem>
          {KNOWN_SP_SKILL_IDS.map((id) => (
            <SelectItem key={id} value={String(id)}>
              {SP_SKILL_LABELS[id]}
            </SelectItem>
          ))}
          {spSkillId !== undefined && !isKnown && (
            <SelectItem value={String(spSkillId)}>
              {`Unknown skill ${spSkillId}`}
            </SelectItem>
          )}
        </SelectContent>
      </Select>
      {disabled && (
        <p id={DISABLED_REASON_ID} className="text-xs text-muted-foreground">
          {DISABLED_REASON}
        </p>
      )}
      {isKnown && spSkillId !== undefined && (
        <p className="text-xs text-muted-foreground">
          {`Grants its level-5 prerequisite automatically. Effective player cap: ${Math.min(
            10,
            (draft.spBooks[0] ?? 0) - 5,
          )}.`}
        </p>
      )}
      <FieldErrors messages={errors?.["spSkillId"]} />
      {warnings && warnings.length > 0 && (
        <div className="space-y-0.5">
          {warnings.map((message) => (
            <p key={message} className="text-xs text-warning-foreground">
              {message}
            </p>
          ))}
        </div>
      )}
    </section>
  );
}
