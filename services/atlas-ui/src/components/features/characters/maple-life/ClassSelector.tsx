import { useRef, type KeyboardEvent } from "react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { draftIndex, type MapleLifeClassDraft } from "./mapleLifeEditorState";

interface ClassSelectorProps {
  ordinal: number;
  gender: number;
  /** Ten drafts, ordinal-major. Used for the job label and the absent marker. */
  drafts: MapleLifeClassDraft[];
  /** Job display names by id, or null while unknown (isPending/isError). */
  jobNameById: Map<number, string> | null;
  onSelect: (ordinal: number, gender: number) => void;
}

const GENDER_LABELS = ["Male", "Female"] as const;
const ORDINALS = [0, 1, 2, 3, 4] as const;

/**
 * Roving tabindex for a `role="tablist"` group of buttons, copied from
 * TemplateSelector.tsx (lines 25-57) so both segmented controls on this page
 * share one keyboard model instead of two hand-rolled ones.
 */
function useRovingTabs(count: number, moveTo: (index: number) => void) {
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const move = (index: number) => {
    moveTo(index);
    tabRefs.current[index]?.focus();
  };

  const handleKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number,
  ) => {
    if (count === 0) return;
    let nextIndex: number;
    switch (event.key) {
      case "ArrowRight":
        nextIndex = (index + 1) % count;
        break;
      case "ArrowLeft":
        nextIndex = (index - 1 + count) % count;
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = count - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    move(nextIndex);
  };

  return { tabRefs, handleKeyDown, move };
}

const tabClassName = (selected: boolean) =>
  cn(
    "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
    selected
      ? "bg-background font-medium shadow-sm"
      : "text-muted-foreground hover:text-foreground",
  );

/**
 * The fixed 5x2 (ordinal x gender) selection surface for a Maple Life
 * config's class rows (FR-4.1/4.2/4.4). No add/remove — the ten slots always
 * exist; a slot is either loaded or "not configured".
 */
export function ClassSelector({
  ordinal,
  gender,
  drafts,
  jobNameById,
  onSelect,
}: ClassSelectorProps) {
  const genderRoving = useRovingTabs(GENDER_LABELS.length, (index) =>
    onSelect(ordinal, index),
  );
  const ordinalRoving = useRovingTabs(ORDINALS.length, (index) =>
    onSelect(index, gender),
  );

  return (
    <div className="space-y-2">
      <div
        role="tablist"
        aria-label="Gender"
        className="flex w-fit items-center gap-1 rounded-lg bg-muted p-1"
      >
        {GENDER_LABELS.map((label, index) => (
          <button
            key={label}
            ref={(el) => {
              genderRoving.tabRefs.current[index] = el;
            }}
            type="button"
            role="tab"
            aria-selected={index === gender}
            tabIndex={index === gender ? 0 : -1}
            onClick={() => onSelect(ordinal, index)}
            onKeyDown={(event) => genderRoving.handleKeyDown(event, index)}
            className={tabClassName(index === gender)}
          >
            {label}
          </button>
        ))}
      </div>
      <div
        role="tablist"
        aria-label="Class ordinal"
        className="flex flex-wrap items-center gap-1 rounded-lg bg-muted p-1"
      >
        {ORDINALS.map((ord) => {
          const draft = drafts[draftIndex(ord, gender)];
          const jobId = draft?.jobId ?? 0;
          const jobLabel = jobNameById?.get(jobId) ?? String(jobId);
          const badgeText = ord < 2 ? "derived" : "unconfirmed";
          return (
            <button
              key={ord}
              ref={(el) => {
                ordinalRoving.tabRefs.current[ord] = el;
              }}
              type="button"
              role="tab"
              aria-selected={ord === ordinal}
              tabIndex={ord === ordinal ? 0 : -1}
              onClick={() => onSelect(ord, gender)}
              onKeyDown={(event) => ordinalRoving.handleKeyDown(event, ord)}
              className={tabClassName(ord === ordinal)}
            >
              <span>
                {ord} · {jobLabel}
              </span>
              {draft?.present === false && (
                <span className="text-xs text-muted-foreground">
                  not configured
                </span>
              )}
              <Badge variant="secondary">{badgeText}</Badge>
            </button>
          );
        })}
      </div>
    </div>
  );
}
