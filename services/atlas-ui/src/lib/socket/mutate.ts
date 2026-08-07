import type {
  SocketConfig,
  SocketHandlerEntry,
  SocketUnsupported,
  SocketWriterEntry,
} from "@/types/models/socket";
import type { DefinitionKind } from "@/lib/socket/model";
import { formatOpcode, parseOpcode } from "@/lib/socket/opcode";

/**
 * Thrown whenever a mutation cannot address exactly one binding: either the
 * `(name, opCodeValue)` pair matches nothing (someone else already changed
 * it) or it matches more than one entry (the document already holds a
 * duplicate). Both are concurrent-edit / data-integrity conditions the
 * caller must surface and stop on - never guess which entry was meant.
 */
export class MutationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "MutationError";
  }
}

/** The editable fields of one binding. `validator` is meaningful for handlers only. */
export interface BindingInput {
  opCode: string;
  validator?: string;
  services: string[];
  options?: unknown;
  fname?: string;
}

type AnyEntry = SocketHandlerEntry | SocketWriterEntry;

/**
 * Deep-clones a JSON-shaped value. Every function in this module reads
 * `cfg` and returns a disjoint object graph - nothing in the return value
 * may share a mutable array/object with the input, so a caller mutating the
 * result (or a caller that later calls another mutate.ts function on a
 * frozen/cached input) can never corrupt the other's copy.
 */
function cloneJson<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

function nameOf(entry: AnyEntry, kind: DefinitionKind): string {
  return kind === "handler"
    ? (entry as SocketHandlerEntry).handler
    : (entry as SocketWriterEntry).writer;
}

/** Always returns both lists, even when `cfg.unsupported` was absent (FR-1's backwards-compat default of "both empty"). */
function normalizedUnsupported(cfg: SocketConfig): SocketUnsupported {
  return {
    handlers: [...(cfg.unsupported?.handlers ?? [])],
    writers: [...(cfg.unsupported?.writers ?? [])],
  };
}

function dropName(list: string[], name: string): string[] {
  return list.filter((n) => n !== name);
}

function requireOpcodeValue(opCode: string): number {
  const value = parseOpcode(opCode);
  if (value === null) {
    throw new MutationError(
      `"${opCode}" is not a valid opcode. Use 0x followed by 1-4 hex digits.`,
    );
  }
  return value;
}

function buildHandlerEntry(
  name: string,
  input: BindingInput,
): SocketHandlerEntry {
  return {
    opCode: input.opCode,
    validator: input.validator ?? "",
    handler: name,
    ...(input.services.length > 0 ? { services: [...input.services] } : {}),
    ...(input.fname !== undefined && input.fname !== ""
      ? { fname: input.fname }
      : {}),
    ...(input.options !== undefined
      ? { options: cloneJson(input.options) }
      : {}),
  };
}

function buildWriterEntry(
  name: string,
  input: BindingInput,
): SocketWriterEntry {
  return {
    opCode: input.opCode,
    writer: name,
    ...(input.services.length > 0 ? { services: [...input.services] } : {}),
    ...(input.fname !== undefined && input.fname !== ""
      ? { fname: input.fname }
      : {}),
    ...(input.options !== undefined
      ? { options: cloneJson(input.options) }
      : {}),
  };
}

/** Builds a fresh, standalone entry - `input.options` is deep-cloned so the result never aliases the caller's object (FR-6.5). */
function buildEntry(
  kind: DefinitionKind,
  name: string,
  input: BindingInput,
): AnyEntry {
  return kind === "handler"
    ? buildHandlerEntry(name, input)
    : buildWriterEntry(name, input);
}

function collectionOf(cfg: SocketConfig, kind: DefinitionKind): AnyEntry[] {
  return cloneJson(kind === "handler" ? cfg.handlers : cfg.writers);
}

/** Rebuilds the whole config from one freshly-computed collection, deep-cloning the untouched sibling collection so nothing is shared with `cfg`. */
function withCollection(
  cfg: SocketConfig,
  kind: DefinitionKind,
  entries: AnyEntry[],
  unsupported: SocketUnsupported,
): SocketConfig {
  return {
    handlers:
      kind === "handler"
        ? (entries as SocketHandlerEntry[])
        : cloneJson(cfg.handlers),
    writers:
      kind === "writer"
        ? (entries as SocketWriterEntry[])
        : cloneJson(cfg.writers),
    unsupported,
  };
}

/**
 * Finds the single array position holding `(name, opCodeValue)`. Index is
 * NEVER an input and never the resolution key: `templatesService.getById`
 * re-sorts both arrays by opcode on read, so a fetched index does not match
 * the entry's position here. Name alone is equally unsafe - `NoOpHandler` is
 * bound to four opcodes in gms_95_1, so a name-only match would be
 * ambiguous by construction. Zero or multiple hits both throw rather than
 * pick one.
 */
function resolveOne(
  entries: AnyEntry[],
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
): number {
  const hits: number[] = [];
  entries.forEach((e, i) => {
    if (nameOf(e, kind) === name && parseOpcode(e.opCode) === opCodeValue)
      hits.push(i);
  });
  if (hits.length === 0) {
    throw new MutationError(
      `No ${kind} named "${name}" at opcode ${formatOpcode(opCodeValue)} was found. It may have been changed or removed by another session - reload and try again.`,
    );
  }
  if (hits.length > 1) {
    throw new MutationError(
      `"${name}" resolves to ${hits.length} bindings at opcode ${formatOpcode(opCodeValue)}. Resolve the duplicate opcode before editing it here.`,
    );
  }
  return hits[0]!;
}

/**
 * FR-6.1/FR-1.2. Appends a new binding for `name` and clears any Unsupported
 * marker for it - a name cannot be simultaneously Defined and Unsupported by
 * this path. Rejects an opcode already used by the same name (by normalized
 * value, so "0x17" and "0x017" collide) rather than creating a duplicate.
 */
export function addBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  input: BindingInput,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const value = requireOpcodeValue(input.opCode);

  const collision = entries.some(
    (e) => nameOf(e, kind) === name && parseOpcode(e.opCode) === value,
  );
  if (collision) {
    throw new MutationError(
      `"${name}" is already bound to opcode ${formatOpcode(value)}.`,
    );
  }

  entries.push(buildEntry(kind, name, input));

  const unsupported = normalizedUnsupported(cfg);
  if (kind === "handler")
    unsupported.handlers = dropName(unsupported.handlers, name);
  else unsupported.writers = dropName(unsupported.writers, name);

  return withCollection(cfg, kind, entries, unsupported);
}

/**
 * FR-6.2. Replaces exactly the binding addressed by `(name, opCodeValue)`.
 * The name itself is not editable through this function - renaming is not a
 * supported operation. When the new opcode differs from the old one, it is
 * rejected if it collides with another binding already using that name.
 */
export function editBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
  input: BindingInput,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const at = resolveOne(entries, kind, name, opCodeValue);
  const newValue = requireOpcodeValue(input.opCode);

  if (newValue !== opCodeValue) {
    const collision = entries.some(
      (e, i) =>
        i !== at &&
        nameOf(e, kind) === name &&
        parseOpcode(e.opCode) === newValue,
    );
    if (collision) {
      throw new MutationError(
        `"${name}" is already bound to opcode ${formatOpcode(newValue)}.`,
      );
    }
  }

  entries[at] = buildEntry(kind, name, input);
  return withCollection(cfg, kind, entries, normalizedUnsupported(cfg));
}

/**
 * FR-6.3 "Remove definition". Removes exactly the one binding addressed by
 * `(name, opCodeValue)`, leaving the Definition Undefined - it does NOT add
 * an Unsupported marker (FR-1.4). Marking Unsupported is a separate,
 * explicit choice the dialog offers alongside this one.
 */
export function deleteBinding(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  opCodeValue: number,
): SocketConfig {
  const entries = collectionOf(cfg, kind);
  const at = resolveOne(entries, kind, name, opCodeValue);
  entries.splice(at, 1);
  return withCollection(cfg, kind, entries, normalizedUnsupported(cfg));
}

/**
 * FR-6.4/FR-1.1. Removes EVERY binding of `name`, necessarily: `unsupported`
 * is name-scoped while bindings are opcode-scoped, so a name cannot be
 * half-marked. Marking `NoOpHandler` unsupported in gms_95_1 removes all
 * four of its routes. The dialog states this in as many words before
 * confirming. Idempotent: marking an already-unsupported name does not
 * duplicate the list entry.
 */
export function markUnsupported(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
): SocketConfig {
  const entries = collectionOf(cfg, kind).filter(
    (e) => nameOf(e, kind) !== name,
  );
  const unsupported = normalizedUnsupported(cfg);
  if (kind === "handler") {
    if (!unsupported.handlers.includes(name)) unsupported.handlers.push(name);
  } else if (!unsupported.writers.includes(name)) {
    unsupported.writers.push(name);
  }
  return withCollection(cfg, kind, entries, unsupported);
}

/** FR-1.3. Returns the Definition to Undefined. A no-op when `name` was not marked. */
export function clearUnsupported(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
): SocketConfig {
  const unsupported = normalizedUnsupported(cfg);
  if (kind === "handler")
    unsupported.handlers = dropName(unsupported.handlers, name);
  else unsupported.writers = dropName(unsupported.writers, name);
  return withCollection(cfg, kind, collectionOf(cfg, kind), unsupported);
}

/**
 * FR-6.5/FR-1.2. Adds every supplied binding for `name` in one pass (each
 * via `addBinding`, so every binding also clears any Unsupported marker and
 * is rejected on an opcode collision) and deep-clones every input value, so
 * the result shares no mutable structure with either `cfg` or `inputs`.
 */
export function copyBindings(
  cfg: SocketConfig,
  kind: DefinitionKind,
  name: string,
  inputs: BindingInput[],
): SocketConfig {
  let out = cfg;
  for (const input of inputs) out = addBinding(out, kind, name, input);
  return out;
}

export interface AncestorAddition {
  name: string;
  bindings: BindingInput[];
}

/**
 * FR-9.4/FR-9.6. Applies the whole selection as ONE returned document, and
 * NEVER overwrites an already-Defined Tenant Definition - a name is skipped
 * if it already has at least one binding in the (progressively updated)
 * result, so a name that gained a definition between the ancestor scan and
 * this apply is skipped rather than clobbered.
 */
export function copyMissingFromAncestor(
  cfg: SocketConfig,
  kind: DefinitionKind,
  additions: AncestorAddition[],
): SocketConfig {
  let out = cfg;
  for (const addition of additions) {
    const currentEntries = kind === "handler" ? out.handlers : out.writers;
    const alreadyDefined = currentEntries.some(
      (e) => nameOf(e as AnyEntry, kind) === addition.name,
    );
    if (alreadyDefined) continue;
    out = copyBindings(out, kind, addition.name, addition.bindings);
  }
  return out;
}

/**
 * FR-11.4 bulk remediation. The server rejects (400) any whole-document save
 * carrying an empty handler validator, so a single-definition edit can never
 * repair one - the live gms_95 tenant carries 32 of them, and every PATCH
 * that would fix one is itself rejected by the same rule. This fills every
 * blank or whitespace-only handler validator in one document write. Writers
 * are untouched: they carry no validator field.
 */
export function fillMissingValidators(
  cfg: SocketConfig,
  validator: string,
): SocketConfig {
  const handlers = cloneJson(cfg.handlers).map((h) =>
    (h.validator ?? "").trim() === "" ? { ...h, validator } : h,
  );
  return {
    handlers,
    writers: cloneJson(cfg.writers),
    unsupported: normalizedUnsupported(cfg),
  };
}
