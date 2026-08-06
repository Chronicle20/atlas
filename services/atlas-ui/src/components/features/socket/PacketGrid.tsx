import { useCallback, useRef, type KeyboardEvent } from "react";
import { cn } from "@/lib/utils";
import { PacketGridRow } from "@/components/features/socket/PacketGridRow";
import type { Row } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

export interface GridSelection {
  name: string;
  /** The object the drawer's actions are scoped to (FR-5.2). */
  scopeKey: string;
}

export interface PacketGridProps {
  rows: Row[];
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
    <div className="relative max-h-[70vh] overflow-auto rounded-md border">
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
              <th scope="col" aria-colindex={2} className="border-b px-2 py-2 text-sm">
                fname
              </th>
            )}
            {objects.map((o, i) => (
              <th
                key={o.key}
                scope="col"
                aria-colindex={(showFName ? 2 : 1) + i + 1}
                className={cn(
                  "border-b px-2 py-2 text-sm whitespace-nowrap",
                  o.key === baselineKey && "bg-muted/40 border-x border-primary/40",
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
          {rows.map((row, i) => (
            <PacketGridRow
              key={row.name}
              row={row}
              objects={objects}
              baselineKey={baselineKey}
              showFName={showFName}
              scopeKey={selection?.name === row.name ? selection.scopeKey : null}
              isSelected={selection?.name === row.name}
              rowIndex={i + 2}
              onSelect={handleSelect}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
