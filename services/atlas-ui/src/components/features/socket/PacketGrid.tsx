import { useCallback, useRef, type KeyboardEvent } from "react";
import { cn } from "@/lib/utils";
import { PacketGridGapRow } from "@/components/features/socket/PacketGridGapRow";
import { PacketGridRow } from "@/components/features/socket/PacketGridRow";
import { isGapRow, type GridRow } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface GridSelection {
  name: string;
  /** The object the drawer's actions are scoped to (FR-5.2). */
  scopeKey: string;
}

export interface PacketGridProps {
  /** Definition rows, optionally interleaved with `withOpcodeGaps` blanks. */
  rows: GridRow[];
  objects: SocketObject[];
  baselineKey: string;
  showFName: boolean;
  selection: GridSelection | null;
  onSelect: (selection: GridSelection) => void;
}

/**
 * A plain semantic table with a sticky header and a sticky first column.
 *
 * Deliberately NOT @tanstack/react-table: the columns here are dynamic objects
 * and every sort/filter predicate is cross-column, so its column model would
 * add a layer owning none of the semantics. Deliberately NOT virtualized
 * either: 219 x 12 renders in one pass, and virtualization fights both the
 * frozen column and the deep-link scroll-to-row path. Virtualize only if
 * measurement shows jank.
 */
export function PacketGrid({
  rows,
  objects,
  baselineKey,
  showFName,
  selection,
  onSelect,
}: PacketGridProps) {
  const bodyRef = useRef<HTMLTableSectionElement>(null);

  // Position of the baseline column among a gap row's spanned cells, or -1
  // when the baseline isn't among the visualized objects (nothing to mark).
  const baselineIndex = objects.findIndex((o) => o.key === baselineKey);
  const gapBaselineIndex =
    baselineIndex < 0 ? -1 : baselineIndex + (showFName ? 1 : 0);

  const handleSelect = useCallback(
    (name: string, scopeKey: string) => onSelect({ name, scopeKey }),
    [onSelect],
  );

  // Arrow keys move between rows; Enter opens the drawer on the focused row.
  const onKeyDown = useCallback((e: KeyboardEvent<HTMLTableSectionElement>) => {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    const buttons =
      bodyRef.current?.querySelectorAll<HTMLButtonElement>("tr > th button");
    if (!buttons || buttons.length === 0) return;
    const current = document.activeElement;
    const index = [...buttons].findIndex(
      (b) => b === current || b.contains(current),
    );
    const next = e.key === "ArrowDown" ? index + 1 : index - 1;
    if (next < 0 || next >= buttons.length) return;
    e.preventDefault();
    buttons[next]!.focus();
  }, []);

  if (rows.length === 0) {
    return (
      <p className="text-muted-foreground p-8 text-center text-sm">
        No definitions match the current filters.
      </p>
    );
  }

  return (
    // Fills whatever height the page's frame gives it: the frame is the
    // scroll boundary, not an arbitrary viewport fraction. A hard max-height
    // here left the legend and the frame's bottom border floating away from
    // a short grid.
    <div className="relative min-h-0 flex-1 overflow-auto">
      <table role="grid" className="w-full border-collapse text-left">
        <thead className="bg-background sticky top-0 z-20">
          <tr role="row" aria-rowindex={1}>
            <th
              scope="col"
              aria-colindex={1}
              className="bg-background sticky left-0 z-30 border-b border-r px-2 py-2 text-sm"
            >
              Definition
            </th>
            {showFName && (
              <th
                scope="col"
                aria-colindex={2}
                className="border-b px-2 py-2 text-sm"
              >
                FName
              </th>
            )}
            {objects.map((o, i) => (
              <th
                key={o.key}
                scope="col"
                aria-colindex={(showFName ? 2 : 1) + i + 1}
                className={cn(
                  "border-b px-2 py-2 text-sm whitespace-nowrap",
                  o.key === baselineKey &&
                    "bg-muted/40 border-x border-primary/40",
                )}
              >
                <span>{o.label}</span>
                {o.key === baselineKey && (
                  <span className="bg-primary/10 text-primary ml-2 rounded px-1 text-[10px] uppercase">
                    baseline
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody ref={bodyRef} onKeyDown={onKeyDown}>
          {rows.map((row, i) =>
            isGapRow(row) ? (
              <PacketGridGapRow
                key={`gap-${row.opCodeValue}`}
                opCodeValue={row.opCodeValue}
                columnCount={objects.length + (showFName ? 1 : 0)}
                baselineColumnIndex={gapBaselineIndex}
                rowIndex={i + 2}
                {...(row.boundBy !== undefined ? { boundBy: row.boundBy } : {})}
                {...(row.boundByOpCodeValue !== undefined
                  ? { boundByOpCodeValue: row.boundByOpCodeValue }
                  : {})}
              />
            ) : (
              <PacketGridRow
                key={row.name}
                row={row}
                objects={objects}
                baselineKey={baselineKey}
                showFName={showFName}
                scopeKey={
                  selection?.name === row.name ? selection.scopeKey : null
                }
                isSelected={selection?.name === row.name}
                rowIndex={i + 2}
                onSelect={handleSelect}
              />
            ),
          )}
        </tbody>
      </table>
    </div>
  );
}
