import { memo } from "react";
import { cn } from "@/lib/utils";
import { PacketGridCell } from "@/components/features/socket/PacketGridCell";
import type { Row } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface PacketGridRowProps {
  row: Row;
  objects: SocketObject[];
  baselineKey: string;
  showFName: boolean;
  scopeKey: string | null;
  isSelected: boolean;
  rowIndex: number;
  onSelect: (name: string, scopeKey: string) => void;
}

/**
 * Memoized over a PRECOMPUTED row object, so filtering and search re-render
 * only the rows whose membership actually changed - which is what keeps 219
 * rows x 12 columns responsive without virtualization.
 */
export const PacketGridRow = memo(function PacketGridRow({
  row,
  objects,
  baselineKey,
  showFName,
  scopeKey,
  isSelected,
  rowIndex,
  onSelect,
}: PacketGridRowProps) {
  return (
    <tr
      role="row"
      aria-rowindex={rowIndex}
      aria-selected={isSelected}
      // Hover tint (prototype: `tr.row:hover td`) - a row is ~12 cells wide
      // and the frozen name column is far from the right-hand ones, so
      // without it you lose your place tracking one definition across.
      // Selection wins over hover; the cells' own state tints are
      // translucent, so both read through them.
      className={cn("group", isSelected ? "bg-accent" : "hover:bg-muted/50")}
    >
      {/* The definition name is the ONLY frozen column (FR-2.8). It scrolls
          over the other cells, so it carries an opaque background of its own
          and has to repeat the row's tint rather than let it show through. */}
      <th
        scope="row"
        role="rowheader"
        aria-colindex={1}
        className={cn(
          "bg-background sticky left-0 z-10 border-b border-r px-2 py-1 text-left text-sm font-medium",
          isSelected ? "bg-accent" : "group-hover:bg-muted/50",
        )}
      >
        <button
          type="button"
          className="text-left hover:underline"
          // FR-5.3: clicking the name leaves the scope on the baseline.
          onClick={() => onSelect(row.name, baselineKey)}
        >
          {row.name}
        </button>
      </th>

      {showFName && (
        <td
          role="gridcell"
          aria-colindex={2}
          className="text-muted-foreground border-b px-2 py-1 font-mono text-xs"
        >
          {row.fname ?? ""}
        </td>
      )}

      {objects.map((object, i) => (
        <PacketGridCell
          key={object.key}
          cell={row.cells.get(object.key)!}
          object={object}
          definitionName={row.name}
          isBaselineColumn={object.key === baselineKey}
          isScoped={isSelected && scopeKey === object.key}
          colIndex={(showFName ? 2 : 1) + i + 1}
          onSelect={(key) => onSelect(row.name, key)}
        />
      ))}
    </tr>
  );
});
