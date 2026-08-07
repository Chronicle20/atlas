import { memo } from "react";
import { cn } from "@/lib/utils";
import { STATE_CELL_CLASS } from "@/components/features/socket/cell-state";
import { formatOpcode } from "@/lib/socket/opcode";
import type { Cell } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface PacketGridCellProps {
  cell: Cell;
  object: SocketObject;
  definitionName: string;
  isBaselineColumn: boolean;
  isScoped: boolean;
  colIndex: number;
  onSelect: (scopeKey: string) => void;
}

/**
 * One object's view of one Definition.
 *
 * State is never colour-only: Unsupported renders the literal "n/a",
 * Undefined renders a "·" placeholder, and an options omission renders a
 * labelled glyph - so the grid is readable in monochrome and to a screen
 * reader.
 *
 * EVERY cell is clickable, including Undefined and Unsupported ones. That
 * placeholder is not decoration: it gives the button a hit area, and an
 * empty cell is precisely where you go to define the definition here or
 * record it as audited-absent. A zero-height button (which is what an empty
 * Undefined cell used to render) made those two actions unreachable from the
 * grid.
 */
export const PacketGridCell = memo(function PacketGridCell({
  cell,
  object,
  definitionName,
  isBaselineColumn,
  isScoped,
  colIndex,
  onSelect,
}: PacketGridCellProps) {
  // FR-2.6's "lowest opcode" comparison is model semantics, not display
  // logic - it lives in buildRows (matrix.ts) as Cell.lowestOpCodeValue so
  // this component only renders what the model already decided.
  const lowest = cell.lowestOpCodeValue;
  const extra = cell.bindings.length - 1;

  return (
    <td
      role="gridcell"
      aria-colindex={colIndex}
      className={cn(
        "border-b p-0 text-sm tabular-nums",
        isBaselineColumn && "border-x border-primary/40",
        STATE_CELL_CLASS[cell.state],
        isScoped && "ring-2 ring-primary ring-inset",
      )}
    >
      <button
        type="button"
        // The accessible name carries the column label so a cell is
        // distinguishable by object without a mouse, and so FR-5.2's
        // cell-scoping is reachable from the keyboard.
        aria-label={`${definitionName} in ${object.label}`}
        className="flex h-7 w-full items-center gap-1 px-2 text-left"
        onClick={() => onSelect(object.key)}
      >
        {cell.state === "unsupported" && <span className="italic">n/a</span>}
        {cell.state === "undefined" && (
          <span className="text-muted-foreground/40" aria-hidden="true">
            ·
          </span>
        )}
        {cell.state === "defined" && lowest !== null && (
          <>
            <span>{formatOpcode(lowest)}</span>
            {extra > 0 && (
              <span className="text-muted-foreground text-xs">{`+${extra}`}</span>
            )}
          </>
        )}
        {cell.hasDuplicateOpcode && (
          <span
            aria-label="duplicate opcode"
            title="Two entries share this opcode"
          >
            ⚠
          </span>
        )}
        {cell.optionsMissing && (
          <span
            aria-label={`${object.label} supplies no options where a sibling does`}
            title="Supplies no options where a sibling does"
          >
            ⌀
          </span>
        )}
      </button>
    </td>
  );
});
