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
 * An explicit `{}` (real corpus fact: gms_95_1 MiniRoom) counts as supplying
 * NO options. A key whose value is an empty array - e.g. `{ types: [] }`, the
 * gms_92_1/gms_95_1 CharacterMovement and gms_87_1/gms_95_1/jms_185_1
 * PetMovement shape - DOES count as supplying options: the key itself is
 * present, the list it names just happens to have zero entries. Those are two
 * different facts and only the first is "no options".
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
 * The "state" key has no single Row-level state to compare (a Row carries one
 * Cell per selected object) - it uses each row's FIRST selected object's cell,
 * i.e. the same object across every row, so the ordering is consistent and
 * deterministic across the whole result.
 */
export function sortRows(
  rows: Row[],
  key: SortKey,
  dir: SortDirection,
): Row[] {
  const sign = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    if (a.inBaseline !== b.inBaseline) return a.inBaseline ? -1 : 1;

    let cmp = 0;
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
      const aFirstKey = [...a.cells.keys()][0];
      const bFirstKey = [...b.cells.keys()][0];
      const as =
        (aFirstKey !== undefined ? a.cells.get(aFirstKey)?.state : undefined) ??
        "undefined";
      const bs =
        (bFirstKey !== undefined ? b.cells.get(bFirstKey)?.state : undefined) ??
        "undefined";
      cmp = STATE_ORDER[as] - STATE_ORDER[bs];
      if (cmp === 0) cmp = a.name.localeCompare(b.name);
    }
    return cmp === 0 ? a.name.localeCompare(b.name) : cmp * sign;
  });
}

/**
 * FR-4.3/4.4. State, hasOptions and service filters are evaluated against the
 * BASELINE object's cell, which is the object the row is ordered by and the
 * one the drawer defaults its scope to.
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
