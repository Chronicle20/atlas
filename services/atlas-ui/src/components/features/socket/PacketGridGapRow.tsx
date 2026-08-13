import { memo } from "react";

import { formatOpcode } from "@/lib/socket/opcode";
import { cn } from "@/lib/utils";

export interface PacketGridGapRowProps {
  opCodeValue: number;
  /** Columns to span after the frozen name column (fname + one per object). */
  columnCount: number;
  baselineColumnIndex: number;
  rowIndex: number;
  /** Definitions the baseline binds here whose row sorts at a lower opcode. */
  boundBy?: string[];
  boundByOpCodeValue?: number;
}

/**
 * An opcode inside the baseline's range that no definition row is ordered at.
 *
 * Laid out like the definition rows around it rather than as one run-on
 * sentence in the name column: the Definition column says why nothing is
 * here, and the opcode itself sits in the BASELINE cell - the same column
 * every other row shows that number in - so the baseline's opcode table still
 * reads as one unbroken column of numbers with visible holes.
 *
 * With `boundBy` the number is NOT a hole: the baseline binds it as the
 * non-lowest opcode of a definition ordered further up, so the row names that
 * definition and where to find it instead of claiming "No Definition" - which
 * would be a false negative for a number that IS wired up.
 *
 * Nothing to click either way - a hole has no definition to scope a drawer
 * to, and an alias's definition already has its own clickable row. The
 * accessible name is pinned on the row so it survives regardless of which
 * column ends up holding the number (and regardless of the baseline not being
 * among the visualized columns at all).
 */
export const PacketGridGapRow = memo(function PacketGridGapRow({
  opCodeValue,
  columnCount,
  baselineColumnIndex,
  rowIndex,
  boundBy,
  boundByOpCodeValue,
}: PacketGridGapRowProps) {
  const names = boundBy ?? [];
  const alias = names.length > 0;
  const at =
    boundByOpCodeValue !== undefined ? formatOpcode(boundByOpCodeValue) : "";
  const label = alias
    ? `${formatOpcode(opCodeValue)} — also bound by ${names.join(", ")}, listed at ${at}`
    : `${formatOpcode(opCodeValue)} — no definition`;

  return (
    <tr
      role="row"
      aria-rowindex={rowIndex}
      aria-label={label}
      className="bg-muted/20"
      data-testid={alias ? "opcode-alias-row" : "opcode-gap-row"}
    >
      <th
        scope="row"
        role="rowheader"
        aria-colindex={1}
        title={alias ? label : undefined}
        className="bg-background text-muted-foreground/70 sticky left-0 z-10 border-b border-r px-2 py-1 text-left text-sm font-normal italic"
      >
        {alias ? `${names.join(", ")} — also at ${at}` : "No Definition"}
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
