# Tenant Template Drift Detection & Reset — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-02
---

## 1. Overview

`atlas-configurations` owns two sibling resources with near-identical documents:
`templates` (the version-specific blueprint, one row per region/major/minor) and
`tenants` (the live per-deployment configuration a running cluster actually
serves). A tenant configuration is created by cloning a template — the Atlas UI's
`onboardTenant` (`services/atlas-ui/src/services/api/onboarding.service.ts:105-122`)
reads a template and POSTs its attributes to `POST /configurations/tenants` with
the new tenant's UUID.

After that clone there is no link of any kind between the two rows, and no way to
answer the question an operator actually has: *has my deployed tenant diverged
from the template it came from, and can I put it back?* Every subsequent edit —
a socket opcode remapped through the grid, a character preset added, a MapleLife
class tweaked — silently accumulates. The only recovery available today is
`DELETE` + re-create, which re-rolls the tenant UUID and therefore orphans every
row of live game data in the other 70+ services (accounts, characters,
inventories, guilds — all keyed by tenant id).

Task-201 solved exactly this problem one level up the chain: a template now
carries read-only `shippedRevision` / `storedRevision` / `seedDrift` attributes
computed by `templates.Revision()`, plus an operator-triggered
`POST /configurations/templates/{id}/reseed`. It explicitly deferred the tenant
half ("Tenant configurations … out of scope here"). This task is that deferred
half, and it completes the chain: **shipped seed file → template → tenant**. Each
hop reports its own drift and offers its own reset, so an operator can see at
which hop a divergence was introduced.

The reset in this task rewrites exactly one database row — the `tenants` row in
`atlas-configurations`. It touches no other service, and by construction cannot
affect live game data: the tenant UUID, the `atlas-tenants` registry row, and
every tenant-scoped record in every other service are untouched.

## 2. Goals

Primary goals:

- Detect, per tenant configuration, whether its stored document diverges from the
  template it derives from, and expose that as read-only attributes of the tenant
  resource.
- Report drift **per section** (`socket`, `characters`, `npcs`, `cashShop`,
  `mapleLife`), not as one opaque boolean, so that a silently-dropped opcode is
  distinguishable from an intentional character-preset edit.
- Provide an explicit reset that restores a tenant's configuration to its
  template's content, preserving the tenant's UUID, region/version identity,
  environment, and tenant-owned sections — available both whole-document and
  per-section.
- Surface both in the Atlas UI: a drift indicator on the tenants list and the
  tenant detail header, and a reset action with a destructive-action confirm
  dialog.
- Guarantee, by test, that a tenant freshly created through the UI's onboarding
  flow reports **no** drift in any section.

Non-goals:

- **Automatic reconcile.** Nothing in this task ever rewrites a tenant document
  without an explicit operator action. There is no boot-time or event-driven
  convergence.
- **Re-rolling the tenant.** Delete-and-recreate semantics are precisely what this
  feature exists to avoid. The tenant UUID is invariant across a reset.
- **Touching live game data.** No service other than `atlas-configurations` is
  written to. No character, account, inventory, or world state is read or changed.
- **`worlds` drift.** World configuration (server message, event message, exp /
  meso / drop / quest-exp rates) is tenant-owned by design — see D3.
- **`diagnostics` drift.** The tenant-only `tracePackets` switch has no template
  counterpart and never participates.
- **Template drift.** A template's own divergence from its shipped seed file is
  task-201's feature and is unchanged here.
- **Bulk reset.** No "reset every drifted tenant" action. One tenant per action.
- **Arbitrary import.** Resetting from an uploaded JSON document remains deferred
  by `task-199-config-json-export`.
- **Field-level diff preview.** Section-level granularity is the resolution this
  task delivers; a value-by-value diff renderer is a separate effort.
- **Alerting.** Drift is advisory and relative to the current template row, which
  itself changes. It is not an alertable signal (see NFR-4).

## 3. User Stories

- As an operator, I want to see which of my deployed tenants have diverged from
  their template, so that I learn about a configuration divergence before it
  causes a gameplay bug rather than after.
- As an operator, I want to know *which section* diverged, so that I can tell a
  dangerous socket divergence apart from a deliberate character-preset edit
  without reading two JSON documents side by side.
- As an operator, I want to reset a tenant back to its template baseline without
  re-rolling its id, so that correcting a botched configuration edit does not
  destroy every account and character in that tenant.
- As an operator, I want to reset just the socket section, so that I can pick up a
  newly-added opcode mapping without discarding the character presets I authored
  by hand.
- As a developer who adds a socket handler to a template, I want existing tenants
  to visibly report that they are behind, so that shipping the code and shipping
  the configuration do not silently diverge.

## 4. Functional Requirements

### FR-1 — Baseline resolution

- **FR-1.1** A tenant's baseline is the **template row** whose
  `(region, majorVersion, minorVersion)` equals the tenant's, resolved through the
  same environment scoping and baseline fallback that
  `templates.Processor.GetByRegionAndVersion` already applies (a template visible
  to the caller's environment, falling back to the environment's registered
  baseline).
- **FR-1.2** The baseline is the template's content **as it stands at read time**,
  not a snapshot captured at clone time. No new column is added to the `tenants`
  entity and no migration is required (D1).
- **FR-1.3** When no template matches the tenant's region/version in the caller's
  environment or its baseline, the tenant reports **no baseline**:
  `baselineRevision` is the empty string and every section drift flag is `false`.
  This is the tenant analogue of task-201's "this image ships no seed file"
  state — an unknown, never a `true`.
- **FR-1.4** Baseline resolution failure is never fatal to a read. A tenant list or
  detail request succeeds and returns the tenant with the no-baseline state; it
  does not 404 or 500 because a template is missing.

### FR-2 — Revision computation

- **FR-2.1** There is exactly **one** function that produces a comparable revision
  for a given section, and both sides of every comparison call it. The template
  side and the tenant side must not be two definitions that happen to agree
  (this is the explicit lesson recorded in `templates/revision.go`).
- **FR-2.2** A revision is computed **per comparable section** over the sections
  named in FR-2.3, plus one aggregate revision over all of them together.
- **FR-2.3** The comparable sections are exactly: `usesPin`, `socket`,
  `characters`, `npcs`, `cashShop`, `mapleLife`.
- **FR-2.4** These fields are **excluded** from every revision:
  - `id` — differs by definition between the two rows.
  - `environment` — server-owned; a tenant and its template may legitimately live
    in different environments via baseline fallback.
  - `region`, `majorVersion`, `minorVersion` — the join key, equal by construction.
  - `worlds` — tenant-owned (D3).
  - `diagnostics` — tenant-only; has no template counterpart.
- **FR-2.5** `socket` is normalized (via the existing `socket.Normalize`) on both
  sides before hashing, so a stored document that omits an empty collection does
  not report drift against one that carries `[]`.
- **FR-2.6** The revision function must be tolerant of the template and tenant
  section types being distinct Go types with identical JSON shapes: the hash is
  taken over the marshaled JSON of the normalized section, so two structurally
  identical documents produce an identical hash regardless of declaring package.
- **FR-2.7** Adding a new field to the tenant or template document must not require
  a corresponding edit to the drift code for that field to participate in drift.
  Sections are opted **out** by name (FR-2.4), never opted in field by field.

### FR-3 — Read model

- **FR-3.1** `GET /configurations/tenants` and `GET /configurations/tenants/{id}`
  return, in addition to today's attributes:
  - `baselineTemplateId` — the UUID of the resolved baseline template; empty when
    no baseline resolves.
  - `baselineRevision` — the aggregate revision of the baseline template's
    comparable sections; empty when no baseline resolves.
  - `storedRevision` — the aggregate revision of the tenant's own comparable
    sections.
  - `templateDrift` — `true` when a baseline resolved **and**
    `baselineRevision != storedRevision`; always `false` when no baseline resolved.
  - `sectionDrift` — an object keyed by section name (`usesPin`, `socket`,
    `characters`, `npcs`, `cashShop`, `mapleLife`) whose values are booleans. All
    `false` when no baseline resolved.
- **FR-3.2** These attributes are **read-only and computed**. They are never
  persisted into the stored document, and the write model bound by `POST`/`PATCH`
  must not carry them — following task-201's `ViewRestModel` split exactly, for
  the same reason (a JSON-tagged field on the write model would be persisted,
  read back, and folded into the next revision, producing permanent phantom
  drift).
- **FR-3.3** A `PATCH` request body containing any of these attributes ignores them
  by omission from the bound model, and does not error.
- **FR-3.4** The list endpoint computes drift for every row in the page. Baseline
  templates are resolved once per distinct region/version key per request, not
  once per row.

### FR-4 — Reset

- **FR-4.1** `POST /configurations/tenants/{tenantId}/reset` replaces the tenant's
  stored content for the requested scope with the baseline template's content.
- **FR-4.2** The request body names the scope. An absent or empty body means the
  whole document (all comparable sections). A body naming one or more sections
  resets exactly those sections and leaves the rest of the document byte-identical
  to what was stored.
- **FR-4.3** A named section must be one of the comparable sections (FR-2.3). Any
  other name — including `worlds`, `diagnostics`, `region`, `id`, or `environment`
  — is rejected with `400`.
- **FR-4.4** These are **never** written by a reset, at any scope: the tenant's
  `id`, `region`, `majorVersion`, `minorVersion`, `environment`, `worlds`, and
  `diagnostics`. Their stored values survive verbatim.
- **FR-4.5** The reset is rejected with `409` when no baseline template resolves
  for the tenant (FR-1.3), and with `404` when the tenant id does not exist.
- **FR-4.6** The reset is subject to the same environment write-authorization rule
  as `PATCH`: rejected unless the caller's environment matches the target row's
  environment; a caller with no environment is always authorized.
- **FR-4.7** The reset creates a `tenant_history` record **before** modifying the
  row, exactly as `UpdateById` does, so the pre-reset document is recoverable.
- **FR-4.8** The reset enqueues the tenant status outbox message (the
  `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` envelope) in the same transaction as
  the write, so running services observe the new configuration by the same path
  they observe a `PATCH`.
- **FR-4.9** The resulting document is validated by the same validators a `PATCH`
  runs (preset validation, socket validation). A baseline whose content fails
  tenant validation is rejected with the collected validation errors rather than
  persisted.
- **FR-4.10** The response is the tenant's post-reset view model (FR-3.1). After a
  whole-document reset with no concurrent template change, `templateDrift` in that
  response is `false` and every `sectionDrift` value is `false`.
- **FR-4.11** The reset is idempotent: applying it twice with no intervening edit
  produces the same stored document (the second application is a no-op write that
  still records history).

### FR-5 — Clone fidelity

- **FR-5.1** The tenant onboarding flow must copy **every** comparable section from
  the template. Today it omits `mapleLife` entirely
  (`onboarding.service.ts:105-118` builds the attributes object from
  `region`, `majorVersion`, `minorVersion`, `usesPin`, `characters`, `npcs`,
  `socket`, `worlds`, and a conditional `cashShop`).
- **FR-5.2** A tenant created through that flow against an unmodified template
  reports `templateDrift: false` and all-`false` `sectionDrift` on its first read.
  This is the tenant analogue of task-201's freshly-seeded-template guarantee, and
  it must be covered by an automated test rather than asserted by inspection.
- **FR-5.3** Existing tenants created before this change are not migrated,
  backfilled, or auto-corrected. They will report whatever drift they actually
  have, which is the point.

### FR-6 — UI

- **FR-6.1** The tenants list (`TenantsPage` / `tenants-columns.tsx`) shows a drift
  indicator per row, consistent with how the templates list presents `seedDrift`.
- **FR-6.2** The tenant detail layout (`TenantDetailLayout.tsx`) shows the drift
  state in its header alongside the existing header actions, listing which
  sections diverge.
- **FR-6.3** A "Reset to template" action in the detail header performs a
  whole-document reset behind a destructive confirm dialog. The dialog states
  explicitly that (a) edits made through the UI to the reset sections will be lost,
  (b) the tenant's id, region, version, world configuration and diagnostics are
  unchanged, and (c) no game data is affected.
- **FR-6.4** Per-section reset is reachable from the section's own view — the
  operator resets `socket` from the handlers/writers area, `characters` from the
  character templates/presets area — each behind the same confirm-dialog pattern
  scoped to that section.
- **FR-6.5** The reset action is disabled, with a tooltip explaining why, when the
  tenant has no resolvable baseline template (FR-1.3).
- **FR-6.6** Confirm dialogs follow the existing convention established by
  `TemplateReseedButton`: Cancel renders first so it holds default focus, and the
  destructive action is never what Enter triggers.
- **FR-6.7** Errors surface through `createErrorFromUnknown` so the server's
  JSON:API error detail reaches the toast; on failure the dialog stays open and the
  displayed tenant is untouched.
- **FR-6.8** After a successful reset the tenant query cache is invalidated so the
  drift indicator and the displayed document both reflect the new state without a
  manual reload.

## 5. API Surface

All endpoints are JSON:API, under `atlas-configurations`.

### 5.1 `GET /configurations/tenants` (modified)

Unchanged request. Each `data[]` element's `attributes` gains:

```json
{
  "baselineTemplateId": "3f1c…",
  "baselineRevision": "9ab3…",
  "storedRevision": "9ab3…",
  "templateDrift": false,
  "sectionDrift": {
    "usesPin": false,
    "socket": false,
    "characters": false,
    "npcs": false,
    "cashShop": false,
    "mapleLife": false
  }
}
```

Pagination envelope unchanged.

### 5.2 `GET /configurations/tenants/{tenantId}` (modified)

Same five added attributes on the single resource.

### 5.3 `POST /configurations/tenants/{tenantId}/reset` (new)

Request — whole document:

```json
{ "data": { "type": "tenants", "attributes": { "sections": [] } } }
```

An absent body, an absent `sections` key, and an empty `sections` array are all
equivalent and mean "every comparable section".

Request — scoped:

```json
{ "data": { "type": "tenants", "attributes": { "sections": ["socket"] } } }
```

Response `200` — the tenant's view model, identical in shape to §5.2.

Errors:

| Status | Condition |
|---|---|
| `400` | `sections` contains a name that is not a comparable section (FR-4.3) |
| `403` | Caller's environment does not match the tenant row's environment (FR-4.6) |
| `404` | No tenant with that id visible to the caller |
| `409` | No baseline template resolves for the tenant's region/version (FR-4.5) |
| `422` | The baseline content fails tenant validation (FR-4.9); body carries the collected validation errors |

### 5.4 `PATCH /configurations/tenants/{tenantId}` (unchanged)

Continues to bind the write model. The computed attributes are absent from that
model and are ignored if a client sends them (FR-3.3).

## 6. Data Model

**No schema migration.** `tenants.Entity` and `tenants.HistoryEntity` are
unchanged; no `template_id` column, no revision column. Every added attribute is
computed at read time from the two documents (D1).

New Go types in `services/atlas-configurations/atlas.com/configurations/tenants`:

- `ViewRestModel` — embeds `RestModel`, adds `BaselineTemplateId`,
  `BaselineRevision`, `StoredRevision`, `TemplateDrift`, `SectionDrift`. Mirrors
  `templates.ViewRestModel` and exists for the same reason (FR-3.2).
- `SectionDrift` — a map or struct keyed by the six comparable section names.
- A shared revision helper covering both `templates.RestModel` and
  `tenants.RestModel` sections (FR-2.1, FR-2.6). Where it lives is a design-phase
  decision; the binding constraint is that one definition serves both sides and
  that `templates.Revision()`'s existing behavior for the template-vs-shipped
  comparison is not altered.

`tenant_history` gains no columns; a reset writes an ordinary history record with
the pre-reset document (FR-4.7), indistinguishable in shape from one written by a
`PATCH`.

## 7. Service Impact

**`atlas-configurations`** — the only service written to.
- `tenants/`: new view model, revision/drift computation, reset processor method,
  reset route + handler, section-name validation.
- `templates/`: the revision function may move or gain a shared sibling
  (FR-2.1). Template-vs-shipped drift behavior must be byte-identical after the
  change — its existing tests are the guard.
- `docs/domain.md` and `docs/rest.md`: updated for the new attributes and endpoint.

**`atlas-ui`** — read and write of the above.
- `onboarding.service.ts`: fix the lossy clone (FR-5.1).
- `tenants-columns.tsx` / `TenantsPage.tsx`: list drift indicator.
- `TenantDetailLayout.tsx`: header drift summary + whole-document reset action.
- Section pages (handlers/writers, character templates/presets, MapleLife, cash
  shop, NPCs): per-section reset action.
- `lib/hooks/api/` + `services/api/`: reset mutation, tenant view types.

**Every other service** — unaffected. No new Kafka topic, no new event type, no
contract change. A reset re-emits the existing tenant status envelope on the
existing topic (FR-4.8), which consumers already handle for `PATCH`.

## 8. Non-Functional Requirements

- **NFR-1 (Performance).** Drift computation is a hash over an in-memory document.
  The list endpoint must resolve each distinct baseline template at most once per
  request (FR-3.4) — a per-row template query on a paged list is not acceptable.
- **NFR-2 (Multi-tenancy).** The `templates` and `tenants` tables are control-plane
  tables, not tenant-scoped ones. Baseline resolution obeys the existing
  **environment** scoping and baseline fallback (FR-1.1); it must never read a row
  from an environment the caller cannot see.
- **NFR-3 (Safety).** The reset is the most destructive operation this service
  exposes to an operator. It is gated by an explicit confirm dialog, records
  history before writing (FR-4.7), and can never widen its blast radius beyond the
  single `tenants` row (FR-4.4).
- **NFR-4 (Advisory only).** `templateDrift` is relative to a template row that is
  itself mutable. A template edit makes its tenants report drift with no tenant
  change. The flag is an operator-facing signal, not an alertable condition, and
  nothing may page or fail a deploy on it.
- **NFR-5 (No false positives).** The freshly-onboarded tenant must report zero
  drift (FR-5.2). A drift indicator that is always on is worse than none, because
  it trains the operator to ignore it.
- **NFR-6 (Observability).** A reset logs at info with the tenant id, the resolved
  baseline template id, and the sections reset.
- **NFR-7 (Security).** The reset endpoint carries the same environment
  authorization as the existing write paths (FR-4.6). It exposes no new data:
  every byte it can write was already readable through the templates API.

## 9. Open Questions

- **OQ-1.** `usesPin` is a scalar, not a section. Treating it as its own
  "section" in `sectionDrift` is proposed for uniformity (FR-2.3), but folding it
  into a `properties` section alongside any future scalars is equally defensible.
  Design phase decides.
- **OQ-2.** Where the shared revision function lives — promoted into a small
  package both `templates` and `tenants` import, or kept in `templates` and
  imported by `tenants`. Constrained by the repo convention against calling
  another layer's internals across a boundary. Design phase decides.
- **OQ-3.** Whether `TenantsMtsConfigPage` corresponds to a document section that
  should be comparable. It appears in the UI but not in the tenant `RestModel`
  enumerated in `tenants/rest.go`; its storage needs to be confirmed during design
  and either added to FR-2.3 or explicitly excluded in FR-2.4.
- **OQ-4.** Whether the per-section reset UI (FR-6.4) is best expressed as an
  action on each section page or as a multi-select inside the single header
  dialog. The API supports both; this is a UX call for the design phase.

## 10. Acceptance Criteria

- [ ] `GET /configurations/tenants/{id}` returns `baselineTemplateId`,
      `baselineRevision`, `storedRevision`, `templateDrift`, and `sectionDrift`.
- [ ] `GET /configurations/tenants` returns the same five attributes for every row
      in the page, resolving each distinct baseline template once per request.
- [ ] A tenant whose region/version has no visible template returns empty
      revisions, empty `baselineTemplateId`, `templateDrift: false`, and all-`false`
      `sectionDrift` — and the request succeeds.
- [ ] Editing one section of a tenant flips exactly that section's flag and the
      aggregate flag; every other section flag stays `false`.
- [ ] Editing a tenant's `worlds` or `diagnostics` flips no flag.
- [ ] `POST /configurations/tenants/{id}/reset` with no body restores every
      comparable section from the template and returns a view model with
      `templateDrift: false`.
- [ ] `POST …/reset` with `sections: ["socket"]` restores `socket` only; a
      concurrently-drifted `characters` section still reports `true` afterward.
- [ ] `POST …/reset` preserves the tenant's `id`, `region`, `majorVersion`,
      `minorVersion`, `environment`, `worlds`, and `diagnostics` byte-for-byte.
- [ ] `POST …/reset` with an unknown or excluded section name returns `400`.
- [ ] `POST …/reset` on a tenant with no baseline returns `409`.
- [ ] `POST …/reset` from a mismatched environment returns `403`.
- [ ] `POST …/reset` writes a `tenant_history` row containing the pre-reset
      document, and enqueues the tenant status outbox message in the same
      transaction.
- [ ] Applying the same reset twice produces an identical stored document.
- [ ] A `PATCH` body carrying the five computed attributes succeeds and does not
      persist them into the stored document; a subsequent read shows no phantom
      drift.
- [ ] The template-vs-shipped drift behavior from task-201 is unchanged —
      `templates` existing tests pass without modification to their expectations.
- [ ] A tenant created through `onboardTenant` against an unmodified template
      reports `templateDrift: false` and all-`false` `sectionDrift` on first read,
      asserted by an automated test.
- [ ] `mapleLife` is copied by the onboarding flow.
- [ ] The tenants list shows a per-row drift indicator; the tenant detail header
      shows the drift state and names the diverging sections.
- [ ] The whole-document reset action is present in the detail header behind a
      confirm dialog whose Cancel button holds default focus.
- [ ] The reset action is disabled with an explanatory tooltip when no baseline
      resolves.
- [ ] A failed reset surfaces the server's JSON:API error detail in a toast, leaves
      the dialog open, and leaves the displayed tenant unchanged.
- [ ] A successful reset invalidates the tenant query cache.
- [ ] `services/atlas-configurations/docs/domain.md` and `docs/rest.md` document the
      new attributes, invariants, processor methods, and endpoint.
- [ ] `tools/verify.sh` (flagless) exits 0.
