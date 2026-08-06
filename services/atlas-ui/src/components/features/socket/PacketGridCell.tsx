import { memo } from "react";
import { cn } from "@/lib/utils";
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
 * State is never colour-only: Unsupported renders the literal "n/a" and an
 * options omission renders a labelled glyph, so the grid is readable in
 * monochrome and to a screen reader.
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
  const values = cell.bindings
    .map((b) => b.opCodeValue)
    .filter((v): v is number => v !== null);
  const lowest = values.length > 0 ? Math.min(...values) : null;
  const extra = cell.bindings.length - 1;

  return (
    <td
      role="gridcell"
      aria-colindex={colIndex}
      className={cn(
        "border-b px-2 py-1 text-sm tabular-nums",
        isBaselineColumn && "bg-muted/40 border-x border-primary/40",
        cell.state === "defined" && "bg-primary/5",
        isScoped && "ring-2 ring-primary ring-inset",
      )}
    >
      <button
        type="button"
        // The accessible name carries the column label so a cell is
        // distinguishable by object without a mouse, and so FR-5.2's
        // cell-scoping is reachable from the keyboard.
        aria-label={`${definitionName} in ${object.label}`}
        className="flex w-full items-center gap-1 text-left"
        onClick={() => onSelect(object.key)}
      >
        {cell.state === "unsupported" && (
          <span className="text-muted-foreground italic">n/a</span>
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
          <span aria-label="duplicate opcode" title="Two entries share this opcode">
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
