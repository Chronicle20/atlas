import { buildOptionsMatrix } from "@/lib/socket/options";
import type { OptionsEntryCellState } from "@/lib/socket/options";
import type { DefinitionKind, SocketObject } from "@/lib/socket/model";
import { cn } from "@/lib/utils";

export interface OptionsMatrixTableProps {
  objects: SocketObject[];
  kind: DefinitionKind;
  name: string;
  baselineKey: string;
}

/**
 * Non-baseline cells whose state is not "same" get an aria-label naming the
 * divergence, so a screen reader (and `getByLabelText` in tests) can find
 * the interesting cells without relying on the colour classes below. The
 * baseline column is the reference every other column is compared against,
 * so it is never itself annotated.
 */
function cellAriaLabel(
  state: OptionsEntryCellState,
  objectLabel: string,
  isBaseline: boolean,
): string | undefined {
  if (isBaseline) return undefined;
  switch (state) {
    case "same":
      return `same as baseline in ${objectLabel}`;
    case "differs":
      return `differs in ${objectLabel}`;
    case "missing":
      return `missing in ${objectLabel}`;
    case "extra":
      return `only in ${objectLabel}`;
  }
}

/**
 * A non-baseline cell whose value is "same" renders an equality glyph
 * instead of repeating the value verbatim.
 *
 * Justification, from FR-3.3-3.5 forward (not retrofitted from a test): the
 * matrix exists so a reviewer can scan potentially the full selected column
 * set (up to the ~12 objects PacketGrid itself supports) for where a Definition's
 * options DIVERGE. A "same" cell is, by definition, not informative - the
 * value is already visible one column over, in the baseline, which is
 * pinned in place with its own highlight/badge (the same convention
 * PacketGrid's header row already uses for the baseline column). Printing
 * the identical string in every agreeing column adds nothing while
 * increasing the scanning cost across every row exactly where FR-3.1
 * already establishes the precedent that expected/structural agreement is
 * not itself a signal worth marking, and FR-2.6 already compresses a
 * multi-binding cell in the outer grid to "lowest opcode + count" rather
 * than enumerating every binding. The equality glyph applies the same
 * economy at the per-entry level: state is still fully recoverable (the
 * cell keeps its `aria-label`, e.g. "same as baseline in GMS v87.1", so nothing
 * is hidden from assistive tech or from a test scoped to that cell), only
 * the redundant repetition of the value is removed.
 *
 * The baseline column always shows its own value (it IS the reference); a
 * "missing" cell (absent at this index/key) renders an em dash.
 */
function cellText(
  cell: { value: unknown; state: OptionsEntryCellState },
  isBaseline: boolean,
): string {
  if (cell.state === "missing") return "—";
  if (!isBaseline && cell.state === "same") return "=";
  return String(cell.value);
}

/**
 * FR-3.3-3.5. Rows are option entries - the ARRAY INDEX for ordered lists, the
 * key name for maps - and columns are the same selected objects as the outer
 * grid. Positional comparison is not a detail: a list-shaped `types`/
 * `statistics` value compares by index, never by name, because the same name
 * can recur at different indices within one object's own list.
 *
 * Exported as `OptionsMatrixTable` (not `OptionsMatrix`) so it does not
 * collide with the `OptionsMatrix` TYPE exported from `lib/socket/options`.
 *
 * No comparison or classification logic lives here - `buildOptionsMatrix`
 * already decided every cell's `state`; this component only renders it.
 */
export function OptionsMatrixTable({
  objects,
  kind,
  name,
  baselineKey,
}: OptionsMatrixTableProps) {
  const matrix = buildOptionsMatrix({ objects, kind, name, baselineKey });

  if (matrix.rows.length === 0) {
    return (
      <p className="text-muted-foreground p-4 text-sm">
        No object supplies options for this definition.
      </p>
    );
  }

  return (
    <div className="overflow-auto">
      <p className="text-muted-foreground px-2 py-1 text-xs">
        {matrix.shape === "list"
          ? "Ordered list — the row number is the wire value, compared positionally."
          : "Keyed map — the row name is the option key."}
      </p>
      <table className="w-full border-collapse text-left text-sm">
        <thead>
          <tr>
            <th scope="col" className="border-b px-2 py-1">
              {matrix.shape === "list" ? "Index" : "Key"}
            </th>
            {objects.map((o) => (
              <th
                key={o.key}
                scope="col"
                className={cn(
                  "border-b px-2 py-1 whitespace-nowrap",
                  o.key === baselineKey && "bg-muted/40",
                )}
              >
                {o.label}
                {o.key === baselineKey && (
                  <span className="text-muted-foreground ml-1 text-[10px] uppercase">
                    baseline
                  </span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {matrix.rows.map((row) => (
            <tr key={row.key}>
              <th scope="row" className="border-b px-2 py-1 font-mono">
                {row.label}
              </th>
              {objects.map((o) => {
                const cell = row.cells.get(o.key)!;
                const isBaseline = o.key === baselineKey;
                return (
                  <td
                    key={o.key}
                    aria-label={cellAriaLabel(cell.state, o.label, isBaseline)}
                    className={cn(
                      "border-b px-2 py-1",
                      isBaseline && "bg-muted/40",
                      cell.state === "differs" &&
                        "text-amber-600 dark:text-amber-400",
                      cell.state === "extra" &&
                        "text-sky-600 dark:text-sky-400",
                      cell.state === "missing" && "text-muted-foreground",
                    )}
                  >
                    {cellText(cell, isBaseline)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
