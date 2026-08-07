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
 * An opcode inside the baseline's range that the baseline does not bind.
 *
 * Laid out like the definition rows around it rather than as one run-on
 * sentence in the name column: the Definition column says what is missing
 * ("No Definition"), and the opcode itself sits in the BASELINE cell - the
 * same column every other row shows that number in - so the baseline's opcode
 * table still reads as one unbroken column of numbers with visible holes.
 *
 * Nothing to click - there is no definition here to scope a drawer to. The
 * accessible name is pinned on the row so it stays "0x1B — no definition"
 * regardless of which column ends up holding the number (and regardless of
 * the baseline not being among the visualized columns at all).
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
      aria-label={`${formatOpcode(opCodeValue)} — no definition`}
      className="bg-muted/20"
      data-testid="opcode-gap-row"
    >
      <th
        scope="row"
        role="rowheader"
        aria-colindex={1}
        className="bg-background text-muted-foreground/70 sticky left-0 z-10 border-b border-r px-2 py-1 text-left text-sm font-normal italic"
      >
        No Definition
      </th>
      {Array.from({ length: columnCount }, (_, i) => (
        <td
          key={i}
          role="gridcell"
          aria-colindex={i + 2}
          className={cn(
            "text-muted-foreground/70 border-b px-2 py-1 text-sm tabular-nums",
            i === baselineColumnIndex && "border-x border-primary/40",
          )}
        >
          {i === baselineColumnIndex && formatOpcode(opCodeValue)}
        </td>
      ))}
    </tr>
  );
});
