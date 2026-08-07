import { memo } from "react";

import { formatOpcode } from "@/lib/socket/opcode";
import { cn } from "@/lib/utils";

export interface PacketGridGapRowProps {
  opCodeValue: number;
  /** Columns to span after the frozen name column (fname + one per object). */
  columnCount: number;
  baselineColumnIndex: number;
  rowIndex: number;
}

/**
 * An opcode inside the baseline's range that no visualized object binds.
 *
 * Nothing to click - there is no definition here to scope a drawer to. The
 * row exists so a contiguous opcode table reads as contiguous, and its holes
 * read as the slots where a definition has not been written yet.
 */
export const PacketGridGapRow = memo(function PacketGridGapRow({
  opCodeValue,
  columnCount,
  baselineColumnIndex,
  rowIndex,
}: PacketGridGapRowProps) {
  return (
    <tr
      role="row"
      aria-rowindex={rowIndex}
      className="bg-muted/20"
      data-testid="opcode-gap-row"
    >
      <th
        scope="row"
        role="rowheader"
        aria-colindex={1}
        className="bg-background text-muted-foreground/70 sticky left-0 z-10 border-b border-r px-2 py-1 text-left text-xs font-normal italic"
      >
        {`${formatOpcode(opCodeValue)} — no definition`}
      </th>
      {Array.from({ length: columnCount }, (_, i) => (
        <td
          key={i}
          role="gridcell"
          aria-colindex={i + 2}
          className={cn(
            "border-b px-2 py-1",
            i === baselineColumnIndex && "border-x border-primary/40",
          )}
        >
          <span className="sr-only">no definition</span>
        </td>
      ))}
    </tr>
  );
});
