import type {
  SocketHandlerEntry,
  SocketWriterEntry,
} from "@/types/models/socket";

/** The two collections are never mixed in a single view. */
export type DefinitionKind = "handler" | "writer";

/**
 * For any object, a Definition is in exactly one state.
 *   defined     - at least one binding exists
 *   unsupported - the name appears in socket.unsupported.<kind>s
 *   undefined   - neither of the above, inferred from absence
 *
 * A name can be BOTH defined and listed unsupported at once (the corpus can
 * produce this - e.g. a handler re-added after being marked unsupported,
 * where the unsupported entry was never cleaned up). "defined" wins: an
 * actual binding is a stronger, directly-observable fact than an
 * audit-time assertion that the packet doesn't exist, and callers that only
 * care whether a cell has real bindings (the grid, the drawer) need that
 * priority to render correctly.
 */
export type DefinitionState = "defined" | "unsupported" | "undefined";

/**
 *   absent - the options key is missing or null
 *   empty  - an explicit {}
 *   list   - a single key whose value is a JSON array; the ARRAY INDEX is the
 *            wire value and the name is not unique (gms_95_1 CharacterMovement
 *            carries UNKNOWN at six separate indices)
 *   map    - a flat object of name -> wire number (operations,
 *            failedReasonCodes, codes), and the fallback for anything else
 */
export type OptionsShape = "absent" | "empty" | "map" | "list";

/**
 * One entry of socket.handlers or socket.writers.
 *
 * A Definition holds one or MORE bindings. NoOpHandler is bound to four opcodes
 * in gms_95_1; ServerListRequestHandle to two in nine templates. A binding, not
 * a Definition, is the unit of Add / Edit / Delete.
 */
export interface Binding {
  /** As stored, e.g. "0x0B8". Never rewritten. */
  opCode: string;
  /** Parsed value, e.g. 184. null when the stored string is malformed. */
  opCodeValue: number | null;
  /** Handlers only; undefined for writers. */
  validator?: string;
  services: string[];
  /** As stored; undefined when the key was absent. */
  options?: unknown;
  fname?: string;
  /**
   * Position in this object's OWN array as fetched. Useful for display order
   * only - never use it to splice, because templatesService re-sorts both
   * arrays by opcode on read, so it does not match the stored index.
   */
  index: number;
}

/** A Template or a Tenant configuration, normalized for the domain layer. */
export interface SocketObject {
  /** Stable identity: the template or tenant uuid. Used as the column key. */
  key: string;
  /** Display label, e.g. "GMS v83.1". */
  label: string;
  source: "template" | "tenant";
  region: string;
  majorVersion: number;
  minorVersion: number;
  /** Implementation name -> its bindings, in stored order. */
  handlers: Map<string, Binding[]>;
  writers: Map<string, Binding[]>;
  unsupportedHandlers: Set<string>;
  unsupportedWriters: Set<string>;
}

export function entriesOf(
  obj: SocketObject,
  kind: DefinitionKind,
): Map<string, Binding[]> {
  return kind === "handler" ? obj.handlers : obj.writers;
}

export function unsupportedOf(
  obj: SocketObject,
  kind: DefinitionKind,
): Set<string> {
  return kind === "handler" ? obj.unsupportedHandlers : obj.unsupportedWriters;
}

/** The state of one Definition within one object. */
export function stateOf(
  obj: SocketObject,
  kind: DefinitionKind,
  name: string,
): DefinitionState {
  const bindings = entriesOf(obj, kind).get(name);
  if (bindings && bindings.length > 0) return "defined";
  if (unsupportedOf(obj, kind).has(name)) return "unsupported";
  return "undefined";
}

/** Narrowing helper so callers can share one code path over both entry types. */
export function nameOfEntry(
  entry: SocketHandlerEntry | SocketWriterEntry,
  kind: DefinitionKind,
): string {
  return kind === "handler"
    ? (entry as SocketHandlerEntry).handler
    : (entry as SocketWriterEntry).writer;
}
