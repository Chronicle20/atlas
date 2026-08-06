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

export type SortKey = "opcode" | "name" | "state";
export type SortDirection = "asc" | "desc";

export interface GridFilters {
  query: string;
  /** Empty means every state. Evaluated against the baseline object's cell. */
  states: DefinitionState[];
  /** Only rows where some cell carries the FR-3.2 marker. */
  optionsMissingOnly: boolean;
  /** null = don't care; true/false = the baseline cell supplies options or not. */
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
      if (av === null && bv === null) cmp = a.name.localeCompare(b.name);
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
 * FR-4.3/4.4. State and hasOptions are evaluated against the BASELINE
 * object's cell - the object the row is ordered by and the one the drawer
 * defaults its scope to. The service filter is NOT baseline-scoped: like the
 * free-text search, it matches if ANY selected object's binding for this row
 * lists that service, so a row stays visible while comparing which versions
 * carry a given service even when the baseline itself doesn't.
 */
export function filterRows(
  rows: Row[],
  filters: GridFilters,
  baselineKey: string,
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

    const baseline = row.cells.get(baselineKey);

    if (filters.states.length > 0) {
      if (!baseline || !filters.states.includes(baseline.state)) return false;
    }

    if (filters.optionsMissingOnly) {
      if (![...row.cells.values()].some((c) => c.optionsMissing)) return false;
    }

    if (filters.hasOptions !== null) {
      const has = suppliesOptions(baseline?.bindings ?? []);
      if (has !== filters.hasOptions) return false;
    }

    if (filters.services.length > 0) {
      const hit = [...row.cells.values()].some((c) =>
        c.bindings.some((b) =>
          b.services.some((s) => filters.services.includes(s)),
        ),
      );
      if (!hit) return false;
    }

    return true;
  });
}
