import type {
  DefinitionKind,
  OptionsShape,
  SocketObject,
} from "@/lib/socket/model";
import { entriesOf } from "@/lib/socket/model";

/**
 * `options` is free-form, but two structural families occur in practice:
 *
 *   list - a single key whose value is a JSON array ("types", "statistics").
 *          The ARRAY INDEX is the wire value and the name is NOT unique:
 *          gms_95_1 CharacterMovement carries UNKNOWN at six separate
 *          indices.
 *   map  - one or more top-level keys, each wrapping a name -> wire-value
 *          object ("operations", "failedReasonCodes", "codes", and others).
 *          Measured across all eleven templates: of 543 options objects, 495
 *          carry exactly one such key, 39 carry two to five (e.g. ServerIP's
 *          `codes` + `modes`, CharacterInteraction's five groups), and 9 are
 *          the literal empty object (classified "empty", not "map"). No
 *          options object anywhere in the corpus mixes a list-shaped group
 *          with a map-shaped one.
 *
 *          Group COUNT is not fixed per Definition either: CharacterInteraction
 *          carries 5 groups in gms_48_1/61_1/72_1/79_1/83_1/84_1/87_1/95_1, 4
 *          groups in jms_185_1 (missing `enterError`), and just 1 group
 *          (`operations`) in gms_92_1 - all measured directly against the
 *          seed templates. Because the group count varies PER OBJECT for the
 *          same Definition, `buildOptionsMatrix` cannot decide whether to
 *          qualify a row key by looking at one object in isolation: doing so
 *          would key the same wire value ("operations.INVITE") differently
 *          depending on which template happened to supply it, and two
 *          otherwise-identical values would never compare `same`. See
 *          `mapEntriesOf` / `buildOptionsMatrix` for the fix: `key` is always
 *          the fully-qualified `group.entry` form, independent of which
 *          columns are selected; only `label` (display only) collapses to
 *          the bare entry name when the whole compared set agrees on a
 *          single group.
 *
 * Anything else falls back to map over its top-level keys and renders read-only.
 */
export function classifyOptions(value: unknown): OptionsShape {
  if (value === undefined || value === null) return "absent";
  if (typeof value !== "object") return "map";
  const keys = Object.keys(value as Record<string, unknown>);
  if (keys.length === 0) return "empty";
  if (
    keys.length === 1 &&
    Array.isArray((value as Record<string, unknown>)[keys[0]!])
  ) {
    return "list";
  }
  return "map";
}

export type OptionsEntryCellState = "same" | "differs" | "missing" | "extra";

export interface OptionsMatrixRow {
  /**
   * Array index (as a string) for lists. For maps, the fully-qualified
   * "group.entry" string (or the bare top-level key when it has no nested
   * group to qualify with) - ALWAYS qualified this way regardless of how
   * many groups any single compared object happens to carry, so the same
   * wire value lands on the same row no matter which objects are selected.
   * See the `classifyOptions` doc comment for why a per-object decision
   * would break this.
   */
  key: string;
  /**
   * What the header cell shows. For lists, same as key. For maps: the bare
   * entry name ("INVITE") when every compared object's Definition carries a
   * single option group, or the qualified "group.entry" form ("operations.
   * INVITE") once any compared object supplies a second group - `key` stays
   * qualified either way; only display collapses.
   */
  label: string;
  cells: Map<string, { value: unknown; state: OptionsEntryCellState }>;
}

export interface OptionsMatrix {
  shape: OptionsShape;
  rows: OptionsMatrixRow[];
}

/** The options value of a definition's FIRST binding in this object. */
function optionsOf(
  obj: SocketObject,
  kind: DefinitionKind,
  name: string,
): unknown {
  return entriesOf(obj, kind).get(name)?.[0]?.options;
}

/** Pulls a list's array out of its single wrapping key. */
function listOf(value: unknown): unknown[] | null {
  if (classifyOptions(value) !== "list") return null;
  const k = Object.keys(value as Record<string, unknown>)[0]!;
  return (value as Record<string, unknown>)[k] as unknown[];
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/** One flattened map row's source: which group it came from (empty string
 * when the top-level key had no nested group to unwrap) and its entry name
 * within that group. */
interface MapEntry {
  group: string;
  entry: string;
  value: unknown;
}

/**
 * Row key for one flattened map entry. ALWAYS `group.entry` when there's a
 * real group; the bare entry name when there is none (the top-level key
 * itself is the entry - never observed with a real group alongside it in
 * the corpus, but a possible mix under the defensive "falls back to map for
 * anything else" path).
 */
function mapRowKey(entry: MapEntry): string {
  return entry.group === "" ? entry.entry : `${entry.group}.${entry.entry}`;
}

/**
 * Pulls a map's entries out of its options object, unwrapping every
 * top-level key that itself holds a name -> value object ("group"), e.g.
 * `{ operations: { OPEN: 5 } }`, `{ codes: {...}, modes: {...} }`
 * (ServerIP), or CharacterInteraction's five groups - `operations`,
 * `enterError`, `leaveReason`, `putStoneError`, `resultType` - all verified
 * directly against the seed templates. A top-level key whose value is not
 * itself a plain object (never observed in the corpus) has no group to
 * unwrap and becomes its own entry with an empty group - defensive
 * fallback only.
 */
function mapEntriesOf(value: unknown): MapEntry[] | null {
  if (classifyOptions(value) !== "map") return null;
  const obj = value as Record<string, unknown>;

  const entries: MapEntry[] = [];
  for (const group of Object.keys(obj)) {
    const groupValue = obj[group];
    if (isPlainObject(groupValue)) {
      for (const entry of Object.keys(groupValue)) {
        entries.push({ group, entry, value: groupValue[entry] });
      }
    } else {
      entries.push({ group: "", entry: group, value: groupValue });
    }
  }
  return entries;
}

/**
 * Flattens one object's map entries into a row-key -> entry lookup, and
 * guards against key collisions: two DIFFERENT (group, entry) pairs that
 * happen to produce the same "group.entry" string because a group or entry
 * name itself contains the "." separator. Not observed anywhere in the
 * corpus - every group/entry name measured is UPPER_SNAKE_CASE without
 * dots - but silently letting the second entry overwrite the first via
 * plain object-literal assignment would drop a row with no error, which is
 * the wrong failure mode for a comparison tool. Throwing surfaces the
 * ambiguity immediately instead.
 */
function flattenMapEntries(
  entries: MapEntry[],
  context: string,
): Map<string, MapEntry> {
  const rows = new Map<string, MapEntry>();
  for (const entry of entries) {
    const key = mapRowKey(entry);
    const prior = rows.get(key);
    if (prior && (prior.group !== entry.group || prior.entry !== entry.entry)) {
      throw new Error(
        `options.ts: ambiguous flattened options row key "${key}" in ${context} - ` +
          `("${prior.group}", "${prior.entry}") and ("${entry.group}", "${entry.entry}") ` +
          `both resolve to it. A group or entry name contains the "." separator.`,
      );
    }
    rows.set(key, entry);
  }
  return rows;
}

/**
 * FR-3.3-3.5. Rows are option entries; columns are the same selected objects as
 * the outer grid. Every cell is classified against the BASELINE at that
 * index/key.
 */
export function buildOptionsMatrix(input: {
  objects: SocketObject[];
  kind: DefinitionKind;
  name: string;
  baselineKey: string;
}): OptionsMatrix {
  const { objects, kind, name, baselineKey } = input;

  const values = new Map<string, unknown>();
  for (const o of objects) values.set(o.key, optionsOf(o, kind, name));

  // Prefer the baseline's shape; fall back to the first object that supplies
  // one, so a baseline with no options still renders its siblings' entries.
  const baselineShape = classifyOptions(values.get(baselineKey));
  let shape: OptionsShape = baselineShape;
  if (shape === "absent" || shape === "empty") {
    const supplier = objects.find((o) => {
      const s = classifyOptions(values.get(o.key));
      return s === "list" || s === "map";
    });
    shape = supplier
      ? classifyOptions(values.get(supplier.key))
      : baselineShape;
  }

  if (shape !== "list" && shape !== "map") return { shape, rows: [] };

  if (shape === "list") {
    const lists = new Map<string, unknown[]>();
    for (const o of objects) lists.set(o.key, listOf(values.get(o.key)) ?? []);
    const baseline = lists.get(baselineKey) ?? [];
    const extent = Math.max(0, ...[...lists.values()].map((l) => l.length));

    const rows: OptionsMatrixRow[] = [];
    for (let i = 0; i < extent; i++) {
      const cells = new Map<
        string,
        { value: unknown; state: OptionsEntryCellState }
      >();
      for (const o of objects) {
        const list = lists.get(o.key)!;
        const has = i < list.length;
        const baseHas = i < baseline.length;
        cells.set(o.key, {
          value: has ? list[i] : undefined,
          state: cellState(baseHas, has, baseline[i], list[i]),
        });
      }
      rows.push({ key: String(i), label: String(i), cells });
    }
    return { shape, rows };
  }

  // Per-object row maps. `key` is unconditionally the fully-qualified
  // "group.entry" form - computed independently per object, but from a rule
  // (mapRowKey) that depends only on that object's OWN group/entry names,
  // never on how many groups a DIFFERENT compared object happens to carry.
  // That is what makes the same wire value land on the same row regardless
  // of which columns are selected (FR-3.5) - see the classifyOptions doc
  // comment for the CharacterInteraction cross-cardinality case this fixes.
  const perObjectRows = new Map<string, Map<string, MapEntry>>();
  for (const o of objects) {
    const entries = mapEntriesOf(values.get(o.key)) ?? [];
    perObjectRows.set(
      o.key,
      flattenMapEntries(entries, `${kind} "${name}" in "${o.key}"`),
    );
  }
  const baselineRows = perObjectRows.get(baselineKey) ?? new Map();

  // `label` display rule: bare entry name when every compared object agrees
  // there is at most one real group for this Definition; the qualified form
  // once ANY compared object supplies a second group. This is evaluated
  // across the whole selected set, not per object, precisely so it can't
  // introduce the same per-object inconsistency `key` had to be fixed for -
  // it only affects what is DISPLAYED, never row identity.
  const groupNames = new Set<string>();
  for (const rowMap of perObjectRows.values()) {
    for (const entry of rowMap.values()) {
      if (entry.group !== "") groupNames.add(entry.group);
    }
  }
  const qualifyLabels = groupNames.size > 1;

  // Canonical (group, entry) per row key, for label formatting only - taken
  // from the first object (in selection order) that supplies the key. This
  // never affects cell values or classification, only what the row header
  // displays.
  const canonical = new Map<string, MapEntry>();
  for (const o of objects) {
    for (const [key, entry] of perObjectRows.get(o.key)!) {
      if (!canonical.has(key)) canonical.set(key, entry);
    }
  }

  const keys = new Set<string>();
  for (const rowMap of perObjectRows.values())
    for (const k of rowMap.keys()) keys.add(k);

  const rows: OptionsMatrixRow[] = [];
  for (const k of [...keys].sort()) {
    const cells = new Map<
      string,
      { value: unknown; state: OptionsEntryCellState }
    >();
    for (const o of objects) {
      const rowMap = perObjectRows.get(o.key)!;
      const entry = rowMap.get(k);
      const baseHas = baselineRows.has(k);
      cells.set(o.key, {
        value: entry !== undefined ? entry.value : undefined,
        state: cellState(
          baseHas,
          entry !== undefined,
          baselineRows.get(k)?.value,
          entry?.value,
        ),
      });
    }
    const meta = canonical.get(k)!;
    const label =
      meta.group === "" || !qualifyLabels
        ? meta.entry
        : `${meta.group}.${meta.entry}`;
    rows.push({ key: k, label, cells });
  }
  return { shape, rows };
}

function cellState(
  baselineHas: boolean,
  objectHas: boolean,
  baselineValue: unknown,
  objectValue: unknown,
): OptionsEntryCellState {
  if (!objectHas) return "missing";
  if (!baselineHas) return "extra";
  return deepEqual(baselineValue, objectValue) ? "same" : "differs";
}

/**
 * Structural equality over JSON-shaped values - object KEY ORDER never
 * matters, array ELEMENT order always does.
 *
 * This used to be a `JSON.stringify(a) === JSON.stringify(b)` comparison,
 * which is key-order-sensitive. That was safe as long as `deepEqual` only
 * ever compared one flattened leaf value at a time (`cellState` above, via
 * `mapEntriesOf`/`listOf`), but `ancestry.ts` (Task 11) applies it to a WHOLE
 * raw multi-key options object, and that exposed a real false-positive:
 * `encoding/json` on the Go side marshals map keys in SORTED order, so a
 * tenant configuration written back through the REST model comes out
 * alphabetized, while an ancestor Template seeded from file keeps its
 * original, non-alphabetical key order (e.g. `CharacterInteraction`'s five
 * groups are `operations, enterError, resultType, putStoneError,
 * leaveReason` in the seed templates - measured directly against them, not
 * alphabetical). A tenant byte-identical in meaning to its ancestor would
 * have compared `modified` the moment it was saved once.
 *
 * A recursive structural walk (rather than a canonicalizing/key-sorting
 * stringify) is used to fix this: it needs no intermediate string
 * allocation, and arrays vs. objects already need different key-order
 * handling either way, so a direct walk is the simpler of the two options.
 * `options` is `unknown` at the type level, so a pathological input (e.g. a
 * self-referential object) must not throw - the try/catch backstop is kept:
 * a circular structure now recurses until the engine raises a
 * `RangeError: Maximum call stack size exceeded` instead of `JSON.stringify`
 * throwing directly, but that error still propagates up through the call
 * stack into the same top-level catch, so the "must not throw" contract and
 * the conservative "falls back to not equal" answer are both unchanged. No
 * corpus value has been observed to need this backstop.
 *
 * Exported so `ancestry.ts` (Task 11) reuses this exact comparison for the
 * options half of FR-8.4 instead of writing a second implementation.
 */
export function deepEqual(a: unknown, b: unknown): boolean {
  try {
    return structuralEqual(a, b);
  } catch {
    return false;
  }
}

function structuralEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== typeof b) return false;
  if (a === null || b === null) return false;
  if (typeof a !== "object") return false;

  const aIsArray = Array.isArray(a);
  const bIsArray = Array.isArray(b);
  if (aIsArray !== bIsArray) return false;

  if (aIsArray && bIsArray) {
    if (a.length !== b.length) return false;
    return a.every((v, i) => structuralEqual(v, b[i]));
  }

  // Plain objects: compare by KEY SET + per-key value, never by insertion
  // order - this is the fix. Arrays above stay strictly positional.
  const ao = a as Record<string, unknown>;
  const bo = b as Record<string, unknown>;
  const aKeys = Object.keys(ao);
  const bKeys = Object.keys(bo);
  if (aKeys.length !== bKeys.length) return false;
  return aKeys.every(
    (k) =>
      Object.prototype.hasOwnProperty.call(bo, k) &&
      structuralEqual(ao[k], bo[k]),
  );
}
