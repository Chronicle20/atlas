import type {
  Binding,
  DefinitionKind,
  DefinitionState,
  SocketObject,
} from "@/lib/socket/model";
import { entriesOf, stateOf, unsupportedOf } from "@/lib/socket/model";
import { matchesOpcodeQuery } from "@/lib/socket/opcode";

/** One object's view of one Definition. */
export interface Cell {
  objectKey: string;
  state: DefinitionState;
  /** Every binding of this Definition in this object, in stored order. */
  bindings: Binding[];
  /**
   * FR-3.2. True when this object supplies NO options for a Definition that at
   * least one other selected object DOES supply options for. This is the only
   * options signal in the grid: structural divergence between versions is the
   * expected state and is never marked (FR-3.1).
   */
  optionsMissing: boolean;
  /** Two bindings whose parsed opcodes are equal - "0xB8" and "0x0B8". */
  hasDuplicateOpcode: boolean;
  /**
   * The lowest parsed opcode among this cell's bindings, irrespective of
   * their stored order - null when the cell has no binding with a parseable
   * opcode. FR-2.6: a multi-binding Definition displays its lowest opcode,
   * not whichever binding happens to be first in the array.
   */
  lowestOpCodeValue: number | null;
}

export interface Row {
  name: string;
  kind: DefinitionKind;
  /** First fname supplied by any object. Display and search only (FR-10.4). */
  fname?: string;
  cells: Map<string, Cell>;
  inBaseline: boolean;
  /** The baseline's lowest opcode for this row; null when not in the baseline. */
  baselineOpCodeValue: number | null;
  /**
   * The baseline object's state for this row - "undefined" when the baseline
   * has no cell at all (e.g. `baselineKey` doesn't match any selected
   * object). Drives the "state" SortKey so sorting always follows the same
   * object the row is ORDERED by, regardless of the order objects were
   * selected in.
   */
  baselineState: DefinitionState;
}

/**
 * A placeholder for an opcode inside the baseline's range that no Definition
 * row is ORDERED at. Rendered as a blank row so a contiguous opcode range
 * reads as contiguous, and the holes in it are the candidate slots for
 * definitions that have not been written yet.
 *
 * Two kinds, distinguished by `boundBy`:
 *
 * - absent (`undefined`) - the baseline does not bind this number at all.
 *   A genuine hole: "No Definition".
 * - present - the baseline DOES bind this number, but only as the non-lowest
 *   opcode of a Definition that FR-2.6 orders at its lowest one instead
 *   (real corpus fact: gms_95_1's `ServerListRequestHandle` binds both 0x04
 *   and 0x0B, so its row sits at 0x04 and 0x0B had no row at all). Without
 *   this second kind the number vanished from the column entirely - neither
 *   a row nor a gap - which read as a silently skipped opcode. It is NOT a
 *   hole, so it must not be labelled one; the row points back at the owning
 *   Definition and the opcode it is ordered at.
 */
export interface GapRow {
  gap: true;
  opCodeValue: number;
  /** Definition names the baseline binds here, when they sort elsewhere. */
  boundBy?: string[];
  /** The lowest opcode of `boundBy`'s first Definition - where its row is. */
  boundByOpCodeValue?: number;
}

/** What the grid actually renders: Definition rows interleaved with gaps. */
export type GridRow = Row | GapRow;

export function isGapRow(row: GridRow): row is GapRow {
  return "gap" in row;
}

export type SortKey = "opcode" | "name" | "state";
export type SortDirection = "asc" | "desc";

export interface GridFilters {
  query: string;
  /**
   * Empty means every state. Evaluated across the WHOLE row: a row survives
   * when ANY visualized object's cell is in one of these states, so filtering
   * on "Undefined" surfaces every Definition some template is missing rather
   * than only the ones the baseline itself lacks.
   */
  states: DefinitionState[];
  /** Only rows where some cell carries the FR-3.2 marker. */
  optionsMissingOnly: boolean;
  /** null = don't care; true/false, evaluated across the whole row - see `filterRows`. */
  hasOptions: boolean | null;
  /** Empty means every service. */
  services: string[];
}

export function emptyFilters(): GridFilters {
  return {
    query: "",
    states: [],
    optionsMissingOnly: false,
    hasOptions: null,
    services: [],
  };
}

/** True when `filters` would narrow the row set at all. */
export function hasActiveFilters(filters: GridFilters): boolean {
  return (
    filters.query.trim() !== "" ||
    filters.states.length > 0 ||
    filters.optionsMissingOnly ||
    filters.hasOptions !== null ||
    filters.services.length > 0
  );
}

/**
 * True when a bindings list supplies a non-empty options object.
 *
 * An explicit `{}` (real corpus fact: gms_95_1 MiniRoom writer, opCode 0xB8)
 * counts as supplying NO options - same as an absent `options` key entirely,
 * which is what gms_87_1 PetMovement, gms_92_1 CharacterMovement, gms_95_1
 * CharacterMovement + PetMovement, and jms_185_1 PetMovement actually do
 * (verified directly against the seed templates - no version omits `types`
 * via an EMPTY ARRAY; the key is simply not there). Those five cells
 * therefore classify as "supplies no options" and correctly trip the FR-3.2
 * absence marker against any sibling that does supply (e.g. gms_79_1
 * PetMovement, a populated 23-entry `types` table).
 *
 * Hypothetically, if a version DID store `{ types: [] }` - the key present
 * but the array empty - this function would treat that as SUPPLYING options
 * (the key exists; `Object.keys` sees it), not omitting them. No such shape
 * has been observed in the corpus; this is documented behavior for the case,
 * not a claim that the case occurs.
 */
function suppliesOptions(bindings: Binding[]): boolean {
  return bindings.some((b) => {
    const o = b.options;
    return (
      o !== undefined &&
      o !== null &&
      typeof o === "object" &&
      Object.keys(o as Record<string, unknown>).length > 0
    );
  });
}

export function buildRows(input: {
  objects: SocketObject[];
  kind: DefinitionKind;
  baselineKey: string;
}): Row[] {
  const { objects, kind, baselineKey } = input;

  // FR-2.5: the row set is the union of Defined and Unsupported names.
  const names = new Set<string>();
  for (const o of objects) {
    for (const n of entriesOf(o, kind).keys()) names.add(n);
    for (const n of unsupportedOf(o, kind)) names.add(n);
  }

  const rows: Row[] = [];
  for (const name of names) {
    // Decide the FR-3.2 marker per definition, which needs every object's
    // answer before any single cell can be classified.
    const supplying = new Set<string>();
    for (const o of objects) {
      const b = entriesOf(o, kind).get(name);
      if (b && suppliesOptions(b)) supplying.add(o.key);
    }
    const someoneSupplies = supplying.size > 0;

    const cells = new Map<string, Cell>();
    let fname: string | undefined;

    for (const o of objects) {
      const bindings = entriesOf(o, kind).get(name) ?? [];
      const state = stateOf(o, kind, name);

      if (fname === undefined) {
        const withFName = bindings.find(
          (b) => b.fname !== undefined && b.fname !== "",
        );
        if (withFName) fname = withFName.fname;
      }

      const values = bindings
        .map((b) => b.opCodeValue)
        .filter((v): v is number => v !== null);

      cells.set(o.key, {
        objectKey: o.key,
        state,
        bindings,
        // Only a DEFINED cell can be omitting options. An undefined cell has
        // no definition to attach options to, so marking it would be noise.
        optionsMissing:
          state === "defined" && someoneSupplies && !supplying.has(o.key),
        hasDuplicateOpcode: new Set(values).size !== values.length,
        lowestOpCodeValue: values.length > 0 ? Math.min(...values) : null,
      });
    }

    const baselineCell = cells.get(baselineKey);
    const baselineValues = (baselineCell?.bindings ?? [])
      .map((b) => b.opCodeValue)
      .filter((v): v is number => v !== null);

    rows.push({
      name,
      kind,
      ...(fname !== undefined ? { fname } : {}),
      cells,
      inBaseline: baselineCell?.state === "defined",
      baselineOpCodeValue:
        baselineValues.length > 0 ? Math.min(...baselineValues) : null,
      baselineState: baselineCell?.state ?? "undefined",
    });
  }
  return rows;
}

const STATE_ORDER: Record<DefinitionState, number> = {
  defined: 0,
  unsupported: 1,
  undefined: 2,
};

/**
 * FR-4.1/4.2 and FR-2.11. Rows absent from the baseline always sort AFTER
 * baseline-defined rows, in both directions - the direction toggle orders
 * within each group, it does not promote non-baseline rows to the top.
 *
 * The "state" key compares each row's `baselineState` - the SAME object
 * `buildRows` was given as `baselineKey`, regardless of where that object
 * falls in the `objects` array the caller selected. This mirrors
 * `filterRows`'s `states` filter, which is also baseline-anchored.
 *
 * A tie (equal opcode, or equal state) always breaks by ascending name,
 * unaffected by `dir` - the direction toggle reorders the primary key only,
 * it does not reverse the tie-break.
 */
export function sortRows(rows: Row[], key: SortKey, dir: SortDirection): Row[] {
  const sign = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    if (a.inBaseline !== b.inBaseline) return a.inBaseline ? -1 : 1;

    let cmp: number;
    if (key === "opcode") {
      const av = a.baselineOpCodeValue;
      const bv = b.baselineOpCodeValue;
      // Both null is a tie on the primary key: leave cmp at 0 so the shared
      // tie-break below runs unsigned (name compare must not be multiplied
      // by `sign` a second time).
      if (av === null && bv === null) cmp = 0;
      else if (av === null) cmp = 1;
      else if (bv === null) cmp = -1;
      else cmp = av - bv;
    } else if (key === "name") {
      cmp = a.name.localeCompare(b.name);
    } else {
      cmp = STATE_ORDER[a.baselineState] - STATE_ORDER[b.baselineState];
    }
    return cmp === 0 ? a.name.localeCompare(b.name) : cmp * sign;
  });
}

/**
 * FR-4.3/4.4. EVERY predicate aggregates across the row - a row survives when
 * ANY visualized object satisfies it, never only the baseline. State used to
 * be baseline-scoped, which made "Undefined" answer the much narrower question
 * "which definitions is the baseline missing?" and hid the case the matrix
 * exists for: one template out of eleven missing a definition the other ten
 * carry. `hasOptions: false` is read as "some DEFINED cell supplies no
 * options" (an undefined cell trivially supplies none, and counting those
 * would match nearly every row).
 *
 * `baselineKey` is retained in the signature - callers pass it and the sort
 * still anchors on it - but no predicate reads it any more.
 */
export function filterRows(
  rows: Row[],
  filters: GridFilters,
  _baselineKey: string,
): Row[] {
  const q = filters.query.trim().toLowerCase();

  return rows.filter((row) => {
    if (q !== "") {
      const nameHit = row.name.toLowerCase().includes(q);
      const fnameHit = (row.fname ?? "").toLowerCase().includes(q);
      const opcodeHit = [...row.cells.values()].some((c) =>
        c.bindings.some(
          (b) =>
            b.opCodeValue !== null &&
            matchesOpcodeQuery(filters.query, b.opCodeValue),
        ),
      );
      if (!nameHit && !fnameHit && !opcodeHit) return false;
    }

    const cells = [...row.cells.values()];

    if (filters.states.length > 0) {
      if (!cells.some((c) => filters.states.includes(c.state))) return false;
    }

    if (filters.optionsMissingOnly) {
      if (!cells.some((c) => c.optionsMissing)) return false;
    }

    if (filters.hasOptions !== null) {
      const hit = filters.hasOptions
        ? cells.some((c) => suppliesOptions(c.bindings))
        : cells.some(
            (c) => c.state === "defined" && !suppliesOptions(c.bindings),
          );
      if (!hit) return false;
    }

    if (filters.services.length > 0) {
      const hit = cells.some((c) =>
        c.bindings.some((b) =>
          b.services.some((s) => filters.services.includes(s)),
        ),
      );
      if (!hit) return false;
    }

    return true;
  });
}

/**
 * Interleaves a blank row for every opcode inside the baseline's range that
 * the BASELINE does not bind.
 *
 * Both the range and the "is it bound?" test are the baseline's own: the
 * baseline is the version whose opcode table you are reading down, so a
 * number it does not use is a hole in THAT table regardless of what any other
 * column happens to put there. Testing across every visualized object instead
 * (the earlier behavior) silently swallowed most of the holes the grid exists
 * to show - with gms_95_1 as the baseline, 0x02 and 0x03 produced no gap row
 * because gms_12_1 and gms_48_1 bind those numbers for entirely unrelated
 * definitions, and 0x10/0x11 disappeared from the 0x0F..0x13 run for the same
 * reason. It also made the matrix disagree with the single-object Tenant and
 * Template pages, where the two tests coincide.
 *
 * A definition some OTHER column binds inside the hole is not lost: it is a
 * row the baseline lacks, so `sortRows` parks it in the non-baseline tail.
 * The gap marks the baseline's empty slot; the tail lists what fills it
 * elsewhere.
 *
 * One opcode namespace, not one per service: a login handler at 0x1D and a
 * channel handler at 0x1D in the same baseline both suppress the 0x1D gap.
 * This under-reports (a login-range hole hidden by a channel handler at the
 * same number stays invisible) and is the deliberate trade for a quieter grid.
 *
 * Only meaningful under `sortRows(rows, "opcode", …)`: a gap has no name and
 * no state, so it cannot be placed in a name- or state-ordered list. Callers
 * gate on that, and on there being no active filter - a blank row cannot
 * match a search or a service filter, so mixing the two would misreport a
 * filtered view as a contiguous range.
 */
export function withOpcodeGaps(
  rows: Row[],
  input: {
    objects: SocketObject[];
    kind: DefinitionKind;
    baselineKey: string;
    direction: SortDirection;
  },
): GridRow[] {
  const { objects, kind, baselineKey, direction } = input;
  const baseline = objects.find((o) => o.key === baselineKey);
  if (!baseline) return rows;

  // Every number the baseline binds, and - separately - the numbers a row is
  // actually ORDERED at. The two differ for a Definition with several
  // bindings: FR-2.6 orders it at its LOWEST opcode, so its other opcodes are
  // bound but unoccupied. Scanning `bound` alone (the earlier behavior) left
  // those numbers with neither a row nor a gap.
  const bound = new Set<number>();
  const occupied = new Set<number>();
  const owners = new Map<number, string[]>();
  const lowestOf = new Map<string, number>();
  for (const [name, bindings] of entriesOf(baseline, kind)) {
    const values = bindings
      .map((b) => b.opCodeValue)
      .filter((v): v is number => v !== null);
    if (values.length === 0) continue;
    for (const v of values) {
      bound.add(v);
      const list = owners.get(v);
      if (list) list.push(name);
      else owners.set(v, [name]);
    }
    const lowest = Math.min(...values);
    lowestOf.set(name, lowest);
    occupied.add(lowest);
  }
  if (bound.size < 2) return rows;

  const min = Math.min(...bound);
  const max = Math.max(...bound);
  const gaps: GapRow[] = [];
  for (let v = min; v <= max; v++) {
    if (occupied.has(v)) continue;
    const boundBy = owners.get(v);
    if (boundBy === undefined) {
      gaps.push({ gap: true, opCodeValue: v });
      continue;
    }
    const sorted = [...boundBy].sort((a, b) => a.localeCompare(b));
    gaps.push({
      gap: true,
      opCodeValue: v,
      boundBy: sorted,
      boundByOpCodeValue: lowestOf.get(sorted[0]!)!,
    });
  }
  if (gaps.length === 0) return rows;
  if (direction === "desc") gaps.reverse();

  const before = (gap: number, rowValue: number) =>
    direction === "asc" ? gap < rowValue : gap > rowValue;

  const out: GridRow[] = [];
  let i = 0;
  let flushed = false;
  for (const row of rows) {
    const rv = row.baselineOpCodeValue;
    if (rv === null) {
      // sortRows parks every row the baseline doesn't order (no opcode, or
      // not in the baseline at all) at the tail. Gaps belong to the ordered
      // prefix, so drain them before the tail begins.
      if (!flushed) {
        flushed = true;
        while (i < gaps.length) out.push(gaps[i++]!);
      }
    } else if (!flushed) {
      while (i < gaps.length && before(gaps[i]!.opCodeValue, rv)) {
        out.push(gaps[i++]!);
      }
    }
    out.push(row);
  }
  while (i < gaps.length) out.push(gaps[i++]!);
  return out;
}
