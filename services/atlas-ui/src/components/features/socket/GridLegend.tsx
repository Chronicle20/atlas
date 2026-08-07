import {
  STATE_LABEL,
  STATE_SWATCH_CLASS,
} from "@/components/features/socket/cell-state";
import type { DefinitionState } from "@/lib/socket/model";
import { cn } from "@/lib/utils";

export interface GridLegendProps {
  /** Definition rows currently rendered - gap rows are not counted. */
  rowCount: number;
  /** True when `withOpcodeGaps` is contributing blank rows. */
  showsOpcodeGaps: boolean;
}

const STATES: DefinitionState[] = ["defined", "unsupported", "undefined"];

/**
 * Reads the grid's cell states back to the user, in the same colours the
 * cells use (both sides render from `cell-state.ts`). Sits below the grid
 * inside the frame, where the prototype puts it.
 */
export function GridLegend({ rowCount, showsOpcodeGaps }: GridLegendProps) {
  return (
    <div className="text-muted-foreground bg-muted/30 flex flex-wrap items-center gap-x-4 gap-y-1 border-t px-3 py-2 text-xs">
      {STATES.map((state) => (
        <span key={state} className="inline-flex items-center gap-1.5">
          <i
            aria-hidden="true"
            className={cn("h-3 w-5 rounded-sm", STATE_SWATCH_CLASS[state])}
          />
          {STATE_LABEL[state]}
        </span>
      ))}
      <span className="inline-flex items-center gap-1.5">
        <span aria-hidden="true">⌀</span>
        supplies no options where a sibling does
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span aria-hidden="true">⚠</span>
        two entries share one opcode
      </span>
      {showsOpcodeGaps && (
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden="true" className="font-mono">
            0x—
          </span>
          opcode in the baseline&apos;s range that nothing defines — or, when
          named, one the baseline binds to a definition listed at its lower
          opcode
        </span>
      )}
      <span className="ml-auto">
        {`${rowCount} ${rowCount === 1 ? "definition" : "definitions"} · click a cell for detail`}
      </span>
    </div>
  );
}
