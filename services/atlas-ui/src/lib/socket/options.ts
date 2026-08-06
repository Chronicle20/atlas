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
 *          with a map-shaped one. `mapOf` below flattens multi-group objects
 *          to one row per group.entry pair so a divergence inside a single
 *          group doesn't collapse into one opaque group-level "differs".
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
  /** Array index (as a string) for lists; the option name for maps. */
  key: string;
  /** What the header cell shows. Same as key; separate so lists can be relabelled. */
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

/**
 * Pulls a map's entries out of its options object, unwrapping its
 * single-key or multi-key group wrapper.
 *
 * Single-group shape (495/543 measured): one top-level key wrapping a
 * name -> value object, e.g. `{ operations: { OPEN: 5 } }`, `{ codes: {
 * NORMAL: 0, ... } }`, `{ failedReasonCodes: { BANNED: 2, ... } }` (verified
 * directly against template_gms_95_1.json). Unwrapped one level so rows are
 * keyed by entry name ("OPEN"), not the group name ("operations").
 *
 * Multi-group shape (39/543 measured, e.g. ServerIP's `{ codes: {...},
 * modes: {...} }` in gms_12_1/48_1/61_1/..., or CharacterInteraction's five
 * groups `operations`/`enterError`/`leaveReason`/`putStoneError`/
 * `resultType` in gms_48_1 - all verified directly against the seed
 * templates): every measured multi-group object is entirely map-shaped
 * groups, never mixed with a list. Flattened to "group.entry" rows, one per
 * entry across every group, so a divergence inside a single group (e.g. one
 * `operations` code) surfaces on its own row instead of the whole group
 * comparing as one opaque blob.
 *
 * A group whose value is not itself a plain object - not observed anywhere
 * in the corpus - is kept as a single raw row under its own top-level key;
 * defensive fallback only, never a corpus shape.
 */
function mapOf(value: unknown): Record<string, unknown> | null {
  if (classifyOptions(value) !== "map") return null;
  const obj = value as Record<string, unknown>;
  const keys = Object.keys(obj);

  if (keys.length === 1) {
    const inner = obj[keys[0]!];
    return isPlainObject(inner) ? inner : obj;
  }

  const flattened: Record<string, unknown> = {};
  for (const k of keys) {
    const group = obj[k];
    if (isPlainObject(group)) {
      for (const [entry, entryValue] of Object.entries(group)) {
        flattened[`${k}.${entry}`] = entryValue;
      }
    } else {
      flattened[k] = group;
    }
  }
  return flattened;
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

  const baselineValue = values.get(baselineKey);

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

  const maps = new Map<string, Record<string, unknown>>();
  for (const o of objects) maps.set(o.key, mapOf(values.get(o.key)) ?? {});
  const baseline = mapOf(baselineValue) ?? {};

  const keys = new Set<string>();
  for (const m of maps.values()) for (const k of Object.keys(m)) keys.add(k);

  const rows: OptionsMatrixRow[] = [];
  for (const k of [...keys].sort()) {
    const cells = new Map<
      string,
      { value: unknown; state: OptionsEntryCellState }
    >();
    for (const o of objects) {
      const m = maps.get(o.key)!;
      const has = Object.prototype.hasOwnProperty.call(m, k);
      const baseHas = Object.prototype.hasOwnProperty.call(baseline, k);
      cells.set(o.key, {
        value: has ? m[k] : undefined,
        state: cellState(baseHas, has, baseline[k], m[k]),
      });
    }
    rows.push({ key: k, label: k, cells });
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
 * Structural equality over JSON-shaped values. `JSON.stringify` is used
 * rather than a hand-rolled walk because option values are themselves
 * JSON-derived (arrays, flat objects, primitives) - never functions, Dates,
 * or Maps. The try/catch exists purely as a defensive backstop: `options` is
 * `unknown` at the type level, so a pathological input (e.g. a
 * self-referential object) must not throw. No corpus value has been observed
 * to need it; on that path this falls back to "not equal", which is the
 * conservative answer.
 */
function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== typeof b) return false;
  if (a === null || b === null) return false;
  if (typeof a !== "object") return false;
  try {
    return JSON.stringify(a) === JSON.stringify(b);
  } catch {
    return false;
  }
}
