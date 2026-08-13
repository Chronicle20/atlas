# Template / Tenant Configuration JSON Export — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07
---

## 1. Overview

The Atlas UI exposes two near-identical configuration editors: **Template Details**
(`/templates/:id/*`) and **Tenant Details** (`/tenants/:id/*`). Both render the same
document — region/version metadata, `usesPin`, socket handlers, socket writers,
character templates, character presets, NPCs, worlds (and, for tenants, MTS/cash-shop
config) — spread across six or seven sidebar tabs. Today the only way to obtain the
whole document as one artifact is to hit the API by hand (`GET
/api/configurations/templates/{id}` or `GET /api/configurations/tenants/{id}`) and
strip the JSON:API envelope manually.

That gap costs real time in three recurring workflows: capturing a live tenant's socket
config so it can be reconciled back into a checked-in seed template under
`services/atlas-configurations/seed-data/templates/`; diffing a tenant against the
template it was provisioned from; and attaching a configuration snapshot to a bug
report or a task doc.

This feature adds a single **Export** button to the shared detail-page header of both
editors. Clicking it downloads the configuration currently being viewed as a
pretty-printed `.json` file, in the **exact shape of a seed-template file** — the bare
attributes object, no `id`, no JSON:API envelope — so the downloaded file can be dropped
straight into `seed-data/templates/` or used as a `POST` body without editing.

## 2. Goals

Primary goals:

- One-click download of the viewed template configuration as JSON from any Template
  Details sub-tab.
- One-click download of the viewed tenant configuration as JSON from any Tenant Details
  sub-tab.
- Emit the seed-template file shape byte-compatibly enough that the artifact is usable
  as a seed file: bare attributes, 2-space indentation, handlers/writers in ascending
  `opCode` order.
- Zero backend change — the export is composed entirely from data the UI already
  fetches.

Non-goals:

- **Import.** Uploading a JSON file to create or replace a template/tenant config is
  explicitly out of scope and will be a separate task (it needs schema validation,
  overwrite semantics, and parity with the seed-template CI guards).
- Exporting the tenant *identity* resource (`/api/tenants/{id}` — the `name` field).
  The tenant export is the configuration resource only, so its shape is identical to
  the template export.
- CSV or any non-JSON export format.
- Partial/per-tab export (e.g. "export just the writers").
- Bulk export of all templates or all tenants from the list pages. (Note: a list-level
  `templatesService.export()` already exists and is untouched by this task.)
- Server-side export endpoints, signed URLs, or export history.

## 3. User Stories

- As a server operator, I want to download a tenant's live configuration as JSON so
  that I can reconcile it against the checked-in seed template and open a PR for the
  drift.
- As a developer debugging a version-specific packet issue, I want to export the
  template for that version so that I can attach the exact socket handler/writer table
  to the task doc or bug report.
- As a developer provisioning a new version, I want to export an existing template as a
  seed-shaped file so that I can copy it as the starting point for the new version's
  seed file without hand-stripping a JSON:API envelope.
- As an operator, I want the export button visible on every Template/Tenant Details
  sub-tab so that I don't have to navigate back to Global Properties to trigger it.

## 4. Functional Requirements

### FR-1 — Export button placement

- **FR-1.1** `TemplateDetailLayout`
  (`services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx`) MUST
  render an **Export** button in the page header row, horizontally aligned with the
  "Template Details" heading and the `{id}` subtitle.
- **FR-1.2** `TenantDetailLayout`
  (`services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`) MUST
  render the same button aligned with the "Tenant Details" heading.
- **FR-1.3** Because the button lives in the layout, it MUST be present and functional
  on every sub-route: Global Properties, Character Templates, Character Presets, Socket
  Handlers, Socket Writers, Worlds, and (tenants only) MTS Configuration.
- **FR-1.4** The button MUST NOT interfere with the shared `DetailActionBar`
  (save/discard) rendered at the bottom of the layout. Export is read-only and always
  enabled regardless of the page's dirty state.
- **FR-1.5** The button MUST use the existing shadcn `Button` component with
  `variant="outline"` and `size="sm"`, labelled `Export` with a lucide `Download` icon,
  matching the visual weight of other secondary detail-page actions.

### FR-2 — Export payload shape

- **FR-2.1** The downloaded file MUST contain **only the resource's `attributes`
  object** — no `data` wrapper, no `type`, no `id`.
- **FR-2.2** For templates the payload keys are: `region`, `majorVersion`,
  `minorVersion`, `usesPin`, `socket` (`{ handlers, writers }`), `characters`
  (`{ templates, presets }`), `npcs`, `worlds`.
- **FR-2.3** For tenants the payload is the tenant *configuration* resource's
  attributes — the same key set as FR-2.2 plus `cashShop` when the tenant config
  carries one. The tenant's `name` from `/api/tenants/{id}` MUST NOT be included.
- **FR-2.4** JSON MUST be serialised with `JSON.stringify(payload, null, 2)` and a
  trailing newline, matching the formatting of the checked-in files under
  `services/atlas-configurations/seed-data/templates/`.
- **FR-2.5** `socket.handlers` and `socket.writers` MUST be emitted in **strictly
  ascending numeric `opCode` order**. Both services already apply this sort on read
  (`sortTemplate` in `templates.service.ts`, `sortTenantConfig` in
  `tenants.service.ts`); the export MUST consume the sorted form rather than re-fetching
  raw. This keeps exported files compatible with
  `tools/template-opcode-order-guard.sh`.
- **FR-2.6** `npcs` and `worlds` MUST be emitted as arrays. `atlas-configurations`
  serialises absent collections as `null` for several seeds; `sortTemplate` already
  normalises these to `[]` and the export MUST inherit that normalisation, never
  emitting `null` for either key.
- **FR-2.7** No field re-ordering, renaming, filtering, or value coercion beyond
  FR-2.5/FR-2.6. Whatever the API returned inside `attributes` is what ships.

### FR-3 — Filename

- **FR-3.1** Template exports MUST be named
  `template_<region-lowercase>_<majorVersion>_<minorVersion>.json` — e.g.
  `template_gms_83_1.json` — matching the seed-data naming convention exactly.
- **FR-3.2** Tenant exports MUST be named
  `tenant_<region-lowercase>_<majorVersion>_<minorVersion>.json` — e.g.
  `tenant_gms_83_1.json`. The `tenant_` prefix distinguishes a live-tenant snapshot from
  a seed template at a glance.
- **FR-3.3** Region MUST be lowercased and any character outside `[a-z0-9]` replaced
  with `_` so the filename is always filesystem-safe.
- **FR-3.4** If region/version metadata is unavailable for any reason, the filename MUST
  fall back to `template_<id>.json` / `tenant_<id>.json` rather than producing a
  malformed name.

### FR-4 — Download mechanism

- **FR-4.1** The download MUST be performed client-side via a `Blob` +
  `URL.createObjectURL` + synthetic `<a download>` click, following the existing pattern
  in `services/atlas-ui/src/components/features/npc/NpcShopCard.tsx` (lines ~263–274).
- **FR-4.2** The object URL MUST be revoked and the temporary anchor removed from the
  DOM after the click, in a `finally`-equivalent path so a throw mid-flow cannot leak.
- **FR-4.3** The blob MIME type MUST be `application/json`.
- **FR-4.4** The download logic MUST live in a single shared helper so template and
  tenant call sites do not duplicate it. Proposed:
  `services/atlas-ui/src/lib/utils/download-json.ts` exporting
  `downloadJson(filename: string, payload: unknown): void`.

### FR-5 — Data sourcing and loading states

- **FR-5.1** The export MUST source its payload from the same React Query cache the
  detail pages already populate — `templatesService.getById(id)` for templates,
  `tenantsService.getTenantConfigurationById(id)` for tenants — so the exported document
  reflects the *persisted server state*, not unsaved in-form edits.
- **FR-5.2** The button MUST be disabled while the underlying query is loading or in
  error, and enabled once data is present.
- **FR-5.3** The export MUST reflect the last saved state. If the page has unsaved
  changes, exporting MUST NOT silently include or exclude them ambiguously: the button
  MUST show a tooltip reading `Exports the last saved configuration` whenever the
  shared `DetailActionBar` reports `dirty === true`.
- **FR-5.4** The export MUST NOT trigger a new network request if the query cache
  already holds the resource; it consumes the existing hook.

### FR-6 — Feedback and error handling

- **FR-6.1** On success the UI MUST show a `toast.success` reading
  `Template exported` / `Tenant exported`, consistent with `NpcShopCard`'s
  `Shop exported`.
- **FR-6.2** If serialisation or the download throws, the UI MUST show a `toast.error`
  with a human-readable message and MUST NOT leave a dangling object URL.
- **FR-6.3** No navigation, route change, or page reload may occur as a side effect of
  export.

### FR-7 — Multi-tenancy

- **FR-7.1** Data sourcing goes through the existing `api` client, so the four tenant
  headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) are applied by
  `TenantProvider` as they already are. The export introduces no new request path and
  no new header handling.
- **FR-7.2** Switching the active tenant clears the React Query cache; the export button
  MUST therefore return to its disabled/loading state until the resource refetches,
  per FR-5.2. It MUST NOT export stale cross-tenant data.

## 5. API Surface

**No new or modified endpoints.** The feature is composed from endpoints the UI already
calls:

| Method | Path | Used for |
|---|---|---|
| `GET` | `/api/configurations/templates/{id}` | Template export payload (existing `templatesService.getById`) |
| `GET` | `/api/configurations/tenants/{id}` | Tenant export payload (existing `tenantsService.getTenantConfigurationById`) |

Both return a JSON:API single-resource document `{ data: { type, id, attributes } }`.
The export projects `data.attributes` and discards the rest.

New client-side surface:

```ts
// services/atlas-ui/src/lib/utils/download-json.ts
export function downloadJson(filename: string, payload: unknown): void;

// services/atlas-ui/src/lib/utils/config-export.ts (or colocated)
export function configExportFilename(
  kind: "template" | "tenant",
  meta: { id: string; region?: string; majorVersion?: number; minorVersion?: number },
): string;
```

## 6. Data Model

No database entities, columns, migrations, or Kafka messages are added or changed. No
Go service is touched. The only "model" concern is the exported document shape, which is
defined by the existing `TemplateAttributes` (`src/types/models/template.ts`) and
`TenantConfigAttributes` (`src/services/api/tenants.service.ts`) TypeScript types.

Reference for the target shape — `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`:

```json
{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [ { "opCode": "0x01", "validator": "...", "handler": "...", "fname": "...", "services": ["..."] } ],
    "writers":  [ { "opCode": "0x0F", "writer": "...", "fname": "...", "options": {} } ]
  },
  "characters": { "templates": [ ... ], "presets": [ ... ] },
  "npcs":   [ { "npcId": 9000000, "impl": "..." } ],
  "worlds": [ { "name": "...", "flag": "...", "serverMessage": "...", ... } ]
}
```

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-ui` | **All changes live here.** New `downloadJson` utility, new `ConfigExportButton` component, header wiring in `TemplateDetailLayout` and `TenantDetailLayout`, unit tests. |
| `atlas-configurations` | None. Read-only consumer of existing GET endpoints. |
| `atlas-tenants` | None. |
| All other Go services | None. |

Expected touched files:

- `services/atlas-ui/src/lib/utils/download-json.ts` (new)
- `services/atlas-ui/src/components/features/config/ConfigExportButton.tsx` (new — or
  colocated per-feature if the design phase prefers two thin wrappers)
- `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx`
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`
- `services/atlas-ui/src/lib/utils/__tests__/download-json.test.ts` (new)
- `services/atlas-ui/src/components/features/**/__tests__/ConfigExportButton.test.tsx` (new)

## 8. Non-Functional Requirements

**Performance.** Export is a pure client-side serialisation of an already-cached object.
The largest real configuration is a full socket table (hundreds of handler + writer
entries) — well under 1 MB serialised. No streaming, chunking, or worker offload is
warranted. Export MUST NOT introduce an additional network round-trip (FR-5.4).

**Security.** No credentials, secrets, or PII are present in a template/tenant
configuration document — it is server topology and packet routing metadata. The file is
written only to the user's own machine via the browser download mechanism; nothing is
uploaded anywhere. No new endpoint is exposed.

**Multi-tenancy.** Covered by FR-7. The export inherits `TenantProvider`'s header
injection and cache-clear-on-switch behaviour; it adds no independent tenant handling
that could drift.

**Observability.** No server-side telemetry. Failures surface as a user-visible
`toast.error` (FR-6.2) and route through the existing `errorLogger` if the shared error
path is used.

**Accessibility.** The button MUST have an accessible name (`Export`) and the icon MUST
be marked `aria-hidden`. It MUST be keyboard-reachable in the header's natural tab
order.

**Frontend conventions.** Named exports, `@/` alias imports, `import.meta.env.VITE_*`
only, no `next/*` imports, shadcn `Button` + lucide icons, `sonner` `toast` — per
`services/atlas-ui/CLAUDE.md` and the `frontend-dev-guidelines` skill. New test files
MUST use `vi.*`, not Jest-era `jest.*`.

## 9. Open Questions

1. **Component factoring.** One shared `ConfigExportButton` parameterised by
   `kind: "template" | "tenant"`, or two thin per-feature components over a shared
   hook? Deferred to the design phase; either satisfies the FRs. Leaning shared, since
   the two payload shapes are identical by FR-2.3.
2. **Dirty-state tooltip source.** FR-5.3 needs to read the shared action bar's `dirty`
   flag. `DetailActionBarContext` currently exposes `config` only through the
   provider's internal context value; whether the export button can read it without a
   new exported hook is a design-phase detail. If it can't be read cleanly, the fallback
   is an unconditional tooltip reading `Exports the last saved configuration`.
3. **Trailing newline.** FR-2.4 specifies one for seed-file parity. Confirm during
   implementation that the checked-in seed files actually end with a newline; if they
   don't, drop the requirement rather than diverging.

## 10. Acceptance Criteria

- [ ] An **Export** button renders in the Template Details header and is visible on all
      six template sub-tabs.
- [ ] An **Export** button renders in the Tenant Details header and is visible on all
      seven tenant sub-tabs.
- [ ] Clicking Export on a template downloads a file named
      `template_<region>_<major>_<minor>.json` (e.g. `template_gms_83_1.json`).
- [ ] Clicking Export on a tenant downloads a file named
      `tenant_<region>_<major>_<minor>.json`.
- [ ] The downloaded file's top-level object is the bare attributes object — it has no
      `data`, `type`, or `id` key.
- [ ] The downloaded template file's key set and formatting match a checked-in
      `seed-data/templates/template_gms_*.json` closely enough that a diff between an
      export of the seeded template and the seed file itself shows only intentional
      drift.
- [ ] `socket.handlers` and `socket.writers` in the exported file are in strictly
      ascending numeric `opCode` order.
- [ ] `npcs` and `worlds` are `[]`, never `null`, for a configuration that has none.
- [ ] The button is disabled while the underlying query is loading or errored.
- [ ] Exporting fires no additional network request when the resource is already cached.
- [ ] A success toast appears on export; an error toast appears on failure.
- [ ] No object URL leaks — `URL.revokeObjectURL` is called on both the success and
      failure paths.
- [ ] Unit tests cover: filename derivation (including the FR-3.3 sanitisation and the
      FR-3.4 fallback), envelope stripping, `null` → `[]` normalisation, and the
      disabled-while-loading state.
- [ ] `npm run test`, `npm run build` (which type-checks), and `tools/lint.sh --check`
      all pass from the worktree root.
- [ ] No Go module changed, so no `docker buildx bake` target is affected — confirmed by
      the branch diff touching only `services/atlas-ui/` and `docs/`.
