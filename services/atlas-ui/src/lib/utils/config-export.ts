/**
 * Pure projection + naming for the Template / Tenant configuration export.
 *
 * The export owns its own output contract rather than inheriting it from the
 * service layer: templates.service's sortTemplate normalises null collections
 * and sorts the socket tables, but tenants.service's sortTenantConfig only
 * sorts - so on the tenant path there is nothing to inherit. Doing both here
 * keeps the exported document's shape independent of a service-layer detail
 * that could be refactored away.
 */

export type ConfigExportKind = "template" | "tenant";

/** One socket table entry, reduced to the field the export orders by. */
interface OpCodedEntry {
  opCode: string;
}

/**
 * The structural subset of TemplateAttributes / TenantConfigAttributes the
 * projection reads. Deliberately loose: everything else is passed through
 * untouched (FR-2.7), so this module never has to track a key list.
 */
export interface ExportableConfigAttributes {
  region?: string | undefined;
  majorVersion?: number | undefined;
  minorVersion?: number | undefined;
  npcs?: unknown[] | null;
  worlds?: unknown[] | null;
  socket?: {
    handlers?: OpCodedEntry[];
    writers?: OpCodedEntry[];
  } | null;
}

export interface ConfigExportMeta {
  id: string;
  region?: string | undefined;
  majorVersion?: number | undefined;
  minorVersion?: number | undefined;
}

function byOpCode(a: OpCodedEntry, b: OpCodedEntry): number {
  return parseInt(a.opCode, 16) - parseInt(b.opCode, 16);
}

/**
 * Project a JSON:API resource's `attributes` into the exported document.
 *
 * Spreading reproduces the seed-file key order for free: JSON.parse preserves
 * insertion order for non-integer-like keys, the server emits them in the
 * order templates/rest.go declares them, and re-assigning an existing key
 * keeps its position. No explicit key-order table is introduced - that would
 * be a second source of truth that goes stale the next time RestModel gains a
 * field.
 *
 * Note: normalising a key that is present-but-null keeps its position;
 * normalising an ABSENT key appends it at the end. The API never omits npcs
 * or worlds (neither Go field carries omitempty), so that case cannot arise
 * against the real server; the output is still valid if it ever does.
 */
export function toConfigExportPayload<T extends ExportableConfigAttributes>(
  attributes: T,
): T {
  // Built as a record because assigning to a property of a generic T is not
  // expressible in TypeScript; the assertion is confined to the return. The
  // spread source is explicitly typed as the (non-generic) structural
  // interface so the object literal has a known shape to assert from -
  // spreading the bare generic T leaves TypeScript unable to see that every
  // property value is index-signature-compatible.
  const out: Record<string, unknown> = {
    ...(attributes as ExportableConfigAttributes),
  } as Record<string, unknown>;
  // The ONLY place this module's "everything else is passed through untouched,
  // so we never track a key list" principle is knowingly broken. These three
  // are COMPUTED by atlas-configurations (task-201), not configured: they are
  // not part of the document's shape at all, and the exported file exists to
  // be promoted into seed-data/templates/. A committed seed file carrying a
  // stale hash of itself is exactly the noise task-201 exists to remove. The
  // server drops them on parse either way, so this is hygiene, not a fix.
  delete out.shippedRevision;
  delete out.storedRevision;
  delete out.seedDrift;
  // task-289's tenant-side equivalents. Deleted unconditionally: `out` is
  // an untyped record, and deleting a key the template path never had is
  // a no-op, so this needs no branch on `kind`. `storedRevision` is
  // deliberately shared — one delete covers both sides.
  delete out.baselineTemplateId;
  delete out.baselineRevision;
  delete out.templateDrift;
  delete out.sectionDrift;

  out.npcs = attributes.npcs ?? [];
  out.worlds = attributes.worlds ?? [];

  const socket = attributes.socket;
  if (socket) {
    out.socket = {
      ...socket,
      handlers: [...(socket.handlers ?? [])].sort(byOpCode),
      writers: [...(socket.writers ?? [])].sort(byOpCode),
    };
  }

  return out as T;
}

function sanitise(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]/g, "_");
}

/**
 * `template_gms_83_1.json` / `tenant_gms_83_1.json`, matching the seed-data
 * naming convention. Falls back to `<kind>_<id>.json` whenever the
 * region/version metadata is unusable, so the name is never malformed.
 */
export function configExportFilename(
  kind: ConfigExportKind,
  meta: ConfigExportMeta,
): string {
  const region = meta.region ? sanitise(meta.region.trim()) : "";
  const major = meta.majorVersion;
  const minor = meta.minorVersion;
  const versioned =
    region.replace(/_/g, "") !== "" &&
    typeof major === "number" &&
    Number.isFinite(major) &&
    typeof minor === "number" &&
    Number.isFinite(minor);

  if (!versioned) {
    return `${kind}_${sanitise(meta.id)}.json`;
  }
  return `${kind}_${region}_${major}_${minor}.json`;
}
