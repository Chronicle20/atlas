import type {
  Binding,
  DefinitionKind,
  OptionsShape,
  SocketObject,
} from "@/lib/socket/model";
import { entriesOf, stateOf } from "@/lib/socket/model";
import { classifyOptions, deepEqual } from "@/lib/socket/options";

/**
 * How one Tenant Definition relates to the same Definition in its ancestor
 * Template (FR-8.3).
 */
export type AncestryClass =
  "same" | "modified" | "tenant-only" | "missing" | "unsupported";

/**
 * FR-8.1. The ancestor is inferred by exact match on Region, Major Version
 * and Minor Version - no Template id is stored anywhere on the tenant, so
 * there is no foreign key to follow.
 *
 * FR-8.2: zero matches returns null. The caller (Task 19) then renders a
 * SINGLE column with ancestry features ABSENT entirely - not a disabled
 * control, not an empty second column.
 *
 * When more than one template matches the same (region, major, minor)
 * triple, the FIRST match in `templates` array order wins (`Array.find`
 * semantics) - deterministic given the caller's ordering, never arbitrary.
 * Each seed template is expected to represent one distinct version, so this
 * is not expected to occur in practice; it is exercised in
 * `ancestry.test.ts` purely to pin the resolution rule rather than leave it
 * to whatever `Array.prototype.find` happens to do.
 */
export function inferAncestor(
  tenant: SocketObject,
  templates: SocketObject[],
): SocketObject | null {
  return (
    templates.find(
      (t) =>
        t.region === tenant.region &&
        t.majorVersion === tenant.majorVersion &&
        t.minorVersion === tenant.minorVersion,
    ) ?? null
  );
}

/**
 * FR-8.3/8.4. Classifies one Definition (a `kind` + `name` pair, which may
 * hold several bindings - `NoOpHandler` carries four in gms_95_1,
 * `ServerListRequestHandle` two in nine templates) against the same
 * Definition in the inferred ancestor.
 *
 * An explicit Unsupported marking on the tenant (FR-9.5) is the tenant's own
 * statement about this Definition and outranks whatever the ancestor
 * carries - `stateOf` already resolves "defined wins over unsupported" when
 * both are stored, so this check only fires when the tenant genuinely has no
 * live bindings for the name.
 */
export function classifyAgainstAncestor(
  tenant: SocketObject,
  ancestor: SocketObject,
  kind: DefinitionKind,
  name: string,
): AncestryClass {
  if (stateOf(tenant, kind, name) === "unsupported") return "unsupported";

  const tenantBindings = entriesOf(tenant, kind).get(name) ?? [];
  const ancestorBindings = entriesOf(ancestor, kind).get(name) ?? [];

  // `name` absent from BOTH sides falls into this branch too and is labeled
  // "missing" - unreachable under the expected caller pattern (Task 19 only
  // ever classifies a name drawn from the union of tenant/ancestor names, so
  // at least one side always has bindings), not a real class of its own.
  if (tenantBindings.length === 0) return "missing";
  if (ancestorBindings.length === 0) return "tenant-only";

  return sameBindingSet(tenantBindings, ancestorBindings) ? "same" : "modified";
}

/**
 * Compares the SET of bindings, not a single one, and not by stored index:
 * `templatesService` re-sorts both arrays by opcode on read, so position is
 * not meaningful. Keyed by the NORMALIZED opcode value (FR-8.4), so "0xB8"
 * and "0x0B8" collapse onto the same key. A differing binding COUNT for one
 * name is `modified`, never `same`.
 */
function sameBindingSet(
  tenantBindings: Binding[],
  ancestorBindings: Binding[],
): boolean {
  if (tenantBindings.length !== ancestorBindings.length) return false;

  const byOpcode = (bindings: Binding[]) => {
    const m = new Map<number | null, Binding>();
    for (const b of bindings) m.set(b.opCodeValue, b);
    return m;
  };
  const t = byOpcode(tenantBindings);
  const a = byOpcode(ancestorBindings);
  // Unequal Map size despite equal array length means one side stored a
  // duplicate (post-normalization) opcode the other didn't - not the same set.
  if (t.size !== a.size) return false;

  for (const [opcode, tb] of t) {
    const ab = a.get(opcode);
    if (!ab) return false;
    if (!sameBinding(tb, ab)) return false;
  }
  return true;
}

/**
 * FR-8.4: opcode, validator, services and options - `fname` NEVER
 * participates (FR-10.4), it is informational only.
 */
function sameBinding(tenant: Binding, ancestor: Binding): boolean {
  if (tenant.opCodeValue !== ancestor.opCodeValue) return false;
  if ((tenant.validator ?? "") !== (ancestor.validator ?? "")) return false;
  if (!sameServiceSet(tenant.services, ancestor.services)) return false;
  return sameOptionsValue(tenant.options, ancestor.options);
}

/** `services` is compared as a set: storage order carries no meaning here. */
function sameServiceSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sa = [...a].sort();
  const sb = [...b].sort();
  return sa.every((v, i) => v === sb[i]);
}

/** Shapes that carry no real options content - see `classifyOptions`. */
const NO_CONTENT: ReadonlySet<OptionsShape> = new Set(["absent", "empty"]);

/**
 * Reuses Task 10's `classifyOptions` and its `deepEqual` comparison rather
 * than a second implementation. Absent and an explicit `{}` both classify as
 * "no content" and compare equal - a tenant is not `modified` merely because
 * a PATCH round-trip materialised an empty object (mirrors FR-3.2's
 * treatment in `matrix.ts`'s `suppliesOptions`). Otherwise the two values
 * must share the same structural shape (list vs. map) and be deep-equal.
 */
function sameOptionsValue(a: unknown, b: unknown): boolean {
  const shapeA = classifyOptions(a);
  const shapeB = classifyOptions(b);
  if (NO_CONTENT.has(shapeA) && NO_CONTENT.has(shapeB)) return true;
  if (shapeA !== shapeB) return false;
  return deepEqual(a, b);
}

/**
 * FR-9.1. Definitions defined in the ancestor Template and undefined in the
 * Tenant. A name the tenant explicitly marked Unsupported is NOT undefined,
 * so it is excluded here (FR-9.5 - the tenant already made a deliberate
 * choice to omit it; the operator opts it back in separately rather than
 * having it re-offered as a gap to backfill).
 */
export function missingFromTenant(
  tenant: SocketObject,
  ancestor: SocketObject,
  kind: DefinitionKind,
): string[] {
  const out: string[] = [];
  for (const [name, bindings] of entriesOf(ancestor, kind)) {
    if (bindings.length === 0) continue;
    if (stateOf(tenant, kind, name) === "undefined") out.push(name);
  }
  return out.sort();
}
