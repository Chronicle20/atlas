# Template Re-seed Trigger — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07
---

## 1. Overview

`atlas-configurations` ships a set of configuration templates as JSON files baked
into its image at `/seed-data/templates/` (source:
`services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`,
11 files today). At boot, `seeder.Run` walks those files and imports each one that
does not already exist in the database.

The import is **create-if-absent**. `importTemplate`
(`services/atlas-configurations/atlas.com/configurations/seeder/seeder.go:216-225`)
looks up `(region, majorVersion, minorVersion)` and returns `"skipped"` when a row
is already present. It never updates. That is the correct default — templates are
editable through the Atlas UI, and a reconcile-on-boot would silently discard those
edits on every redeploy — but it means an edit to a checked-in seed file **never
reaches a database that already has that template row**. The row is frozen at
whatever the file contained the first time that database was seeded.

This has already produced live drift. Confirmed against the `atlas-main` cluster on
2026-08-07: the template rows for GMS 87.1, GMS 92.1, GMS 95.1 and JMS 185.1 are all
missing the `QuestActionHandle` socket handler, even though every one of those seed
files has carried it since commit `69bffe88d` (task-085, 2026-06-13) or `298ee6d73`
(task-145, 2026-08-07). Nothing surfaced this. It was found only by querying the API
by hand after a UI observation, and there is no mechanism today to correct it short of
deleting the row and restarting the service.

This task adds two things: a way to **see** that a template has diverged from the
file its image ships, and an explicit, operator-invoked way to **reset** it back.
Boot behaviour is deliberately unchanged.

## 2. Goals

Primary goals:

- Detect, per template, whether the stored row differs from the seed file baked into
  the running image, and expose that as a first-class read-only attribute of the
  template resource.
- Provide an explicit `POST /configurations/templates/{templateId}/reseed` trigger
  that resets one template's content to the shipped file, preserving the template's
  UUID.
- Surface both in the Atlas UI: a drift badge on the templates list, and a
  "Reset to shipped defaults" action on the template detail page.
- Guarantee, by test, that a freshly seeded template reports **no** drift — i.e. the
  drift computation is free of false positives arising from normalization.

Non-goals:

- **Tenant configurations.** Live tenant socket configs drift from seed templates by
  the same mechanism, and that drift is more damaging (an unmapped opcode is silently
  dropped by the channel dispatcher). It is out of scope here and remains owned by
  `task-189-tenant-config-seed-provisioning`. This task changes nothing about tenants.
- **Automatic reconcile.** Boot-time seeding stays create-if-absent. Nothing in this
  task ever overwrites a template without an explicit operator action.
- **Arbitrary import.** Uploading a JSON file to replace a template is out of scope
  and remains deferred by `task-199-config-json-export`. The only source this task can
  reset from is the file baked into the running image.
- **A diff preview.** The re-seed is a full replace behind a confirm dialog; there is
  no field-level preview of what will be lost.
- **Bulk re-seed.** No "reset all drifted templates" action. One template per action.
- **Merge/additive semantics.** No mode that only adds missing entries.
- **Alerting.** The drift flag is advisory and image-relative; it is not an alertable
  signal (see NFR-4).

## 3. User Stories

- As an operator, I want to see at a glance which templates have diverged from the
  configuration shipped in the deployed image, so that I find out about staleness
  before it causes a gameplay bug rather than after.
- As an operator, I want to reset one template to its shipped defaults from the UI, so
  that correcting drift does not require deleting a database row and restarting the
  service.
- As a developer who adds a socket handler to a seed template, I want existing
  environments to visibly report that they are behind, so that shipping the code and
  shipping the configuration do not silently diverge.
- As an operator who has intentionally hand-edited a template, I want my edits to
  survive every redeploy, so that the UI remains a trustworthy place to make changes.

## 4. Functional Requirements

### FR-1 Shipped-template catalog

- **FR-1.1** The service SHALL load every `*.json` file under
  `$SEED_DATA_PATH/templates` into an in-memory catalog at startup, keyed by
  `(region, majorVersion, minorVersion)`.
- **FR-1.2** The catalog SHALL be a singleton following the project's registry
  convention (`sync.Once` initialization, `sync.RWMutex` guarded access).
- **FR-1.3** Each catalog entry SHALL hold the source file name, the parsed
  `templates.RestModel`, and a **shipped revision**.
- **FR-1.4** The shipped revision SHALL be the lowercase hex SHA-256 of
  `json.Marshal` applied to the parsed model after `socket.Normalize` has been applied
  to its `Socket` field — the identical transformation
  `ProcessorImpl.Create` applies before persisting
  (`templates/processor.go:86-95`).
- **FR-1.5** A file that fails to parse SHALL be logged at ERROR and omitted from the
  catalog. It SHALL NOT prevent startup or prevent other files from loading.
- **FR-1.6** Two files that resolve to the same `(region, majorVersion, minorVersion)`
  key SHALL be logged at ERROR; the first in the existing deterministic sort order
  wins. This mirrors `discoverFiles`' existing `sort.Strings` ordering.
- **FR-1.7** The boot seeder SHALL read from this catalog rather than re-reading and
  re-parsing files, so exactly one parse path exists.

### FR-2 Drift detection

- **FR-2.1** The **stored revision** of a template SHALL be computed as the lowercase
  hex SHA-256 of `json.Marshal` applied to the `RestModel` produced by reading the row
  through the existing `Make` mapper (which already applies `socket.Normalize`). It
  SHALL NOT be computed over the raw `Entity.Data` bytes, so that a row written by an
  older code path or edited out-of-band is still compared on equal terms.
- **FR-2.2** `RestModel.Id` carries `json:"-"` and therefore does not participate in
  either revision. This is relied upon, not incidental; a test SHALL assert it.
- **FR-2.3** A template SHALL be reported as drifted when a catalog entry exists for
  its `(region, majorVersion, minorVersion)` and its stored revision differs from that
  entry's shipped revision.
- **FR-2.4** A template for which **no** catalog entry exists SHALL be reported as not
  drifted, with an empty shipped revision. Absence of a shipped file is not drift.
- **FR-2.5** Drift SHALL be computed on read. No drift state is persisted.

### FR-3 Re-seed trigger

- **FR-3.1** `POST /configurations/templates/{templateId}/reseed` SHALL replace the
  identified template's stored content with the catalog entry for its
  `(region, majorVersion, minorVersion)`.
- **FR-3.2** The template's UUID SHALL be preserved. The operation is an update of the
  existing row, never a delete-and-recreate — a new id would break UI links and any
  external reference.
- **FR-3.3** The region and version of the row SHALL NOT change. They are the lookup
  key; the catalog entry matched on them by construction.
- **FR-3.4** The write SHALL use `Create`'s validation and serialization semantics —
  `socket.Normalize` followed by `socketValidate` — applied against the existing row
  id. It SHALL NOT route through `UpdateById`.

  *Rationale (load-bearing).* `UpdateById`
  (`templates/processor.go:124-159`) additionally runs the preset validator, which
  reassigns `input.Characters.Presets` before marshalling. Re-seeding through it would
  persist bytes that differ from the shipped file, and the row would report drift
  again the instant it was re-seeded. Re-seed must produce a row byte-identical to
  what a fresh boot seed of the same file would produce; that is the definition of
  "reset to shipped defaults".
- **FR-3.5** The replacement SHALL execute inside a single transaction via the
  existing `database.ExecuteTransaction` helper.
- **FR-3.6** On success the endpoint SHALL return `204 No Content`, matching the
  existing PATCH handler's contract.
- **FR-3.7** The operation SHALL be idempotent — re-seeding an undrifted template
  SHALL succeed and leave the row byte-identical.
- **FR-3.8** No Kafka or outbox event SHALL be emitted. Templates do not participate
  in the outbox (`seeder.go:91-99` backfills only `services` and `tenants`), and the
  existing PATCH path emits nothing; re-seed stays consistent with that.

### FR-4 Boot behaviour is preserved

- **FR-4.1** `importTemplate` SHALL continue to return `"skipped"` for a template whose
  `(region, majorVersion, minorVersion)` already exists, regardless of whether the file
  content differs from the stored row.
- **FR-4.2** No code path invoked during startup SHALL modify an existing template row.

### FR-5 UI

- **FR-5.1** The templates list page SHALL render a badge on each row reported as
  drifted, labelled to convey "differs from the configuration shipped in this image".
- **FR-5.2** Rows with an empty shipped revision SHALL render no badge.
- **FR-5.3** The template detail page SHALL offer a "Reset to shipped defaults" action.
- **FR-5.4** The action SHALL be disabled, with an explanatory tooltip, when the
  template has an empty shipped revision.
- **FR-5.5** Activating the action SHALL open a confirmation dialog that states the
  template will be overwritten with the version shipped in the currently deployed
  image and that edits made through the UI will be lost. The destructive action SHALL
  NOT be the dialog's default focus.
- **FR-5.6** On confirmation the UI SHALL `POST` to the re-seed endpoint and, on
  success, invalidate the affected template queries so the view and the badge reflect
  the new state without a manual reload.
- **FR-5.7** A failed re-seed SHALL surface an error to the operator and leave the
  displayed template unchanged.

## 5. API Surface

### 5.1 Modified — template resource attributes

`GET /api/configurations/templates` and
`GET /api/configurations/templates/{templateId}` gain three read-only attributes:

| Attribute | Type | Meaning |
|---|---|---|
| `shippedRevision` | string | SHA-256 of the seed file baked into the running image for this region/version. Empty string when no such file ships. |
| `storedRevision` | string | SHA-256 of the persisted template content. |
| `seedDrift` | boolean | `true` when `shippedRevision` is non-empty and differs from `storedRevision`. |

These are computed, not stored. `POST /configurations/templates` and
`PATCH /configurations/templates/{templateId}` SHALL ignore them if present in a
request body — supplying them is not an error and does not affect the stored row.

They are inline on the template resource rather than on a separate endpoint so the
list page can render per-row badges without an additional request per template.

### 5.2 New — re-seed

```
POST /api/configurations/templates/{templateId}/reseed
```

No request body. No query parameters.

| Status | Condition |
|---|---|
| `204 No Content` | Template reset to the shipped file. |
| `404 Not Found` | No template with that id. |
| `409 Conflict` | The template exists but no seed file for its region/version ships in this image. Nothing to reset to. |
| `400 Bad Request` | The shipped file failed socket validation. JSON:API `errors` array, same shape as the existing create/update validation failure. Indicates a broken seed file and should not occur — the seed files are CI-guarded. |
| `500 Internal Server Error` | Persistence failure. |

The route registers on the existing `/configurations/templates` subrouter in
`templates/resource.go:InitResource`.

## 6. Data Model

**No schema change. No migration.**

The `templates` table is unchanged:

```go
type Entity struct {
    Id           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
    Region       string          `gorm:"not null"`
    MajorVersion uint16          `gorm:"not null"`
    MinorVersion uint16          `gorm:"not null"`
    Data         json.RawMessage `gorm:"type:json;not null"`
}
```

Re-seed rewrites `Data` on the existing row via the established `update` transaction
function in `templates/administrator.go`, leaving `Id`, `Region`, `MajorVersion` and
`MinorVersion` untouched.

Revisions are derived on demand and never persisted, so there is no cache to
invalidate and no state that can itself go stale.

Templates are global configuration and are not tenant-scoped; the `templates` table
has no `tenant_id` column and this task introduces none.

## 7. Service Impact

### `services/atlas-configurations`

| Area | Change |
|---|---|
| `seeder/` | New shipped-template catalog (FR-1). `importTemplate` reads from it instead of re-reading files; its create-if-absent behaviour is unchanged. |
| `templates/processor.go` | New replace-from-shipped operation implementing FR-3.4 — `Create`'s validation and marshalling against an existing row id. Revision computation for FR-2.1. |
| `templates/resource.go` | New `POST /{templateId}/reseed` route and handler. |
| `templates/rest.go` | Three read-only computed attributes on `RestModel`, excluded from write paths. |

### `services/atlas-ui`

| Area | Change |
|---|---|
| `services/api/templates.service.ts` | `reseedTemplate(id)`. |
| `lib/hooks/api/useTemplates.ts` | Re-seed mutation with query invalidation. |
| `types/models/template.ts` | The three new attributes. |
| `pages/templates-columns.tsx`, `pages/TemplatesPage.tsx` | Drift badge. |
| `pages/TemplateDetailPage.tsx` | "Reset to shipped defaults" action and confirm dialog. |

No other service is touched. `atlas-channel` is unaffected: it reads tenant
configuration, not templates, and no tenant data changes.

## 8. Non-Functional Requirements

- **NFR-1 Startup cost.** The catalog parses 11 files totalling roughly 1.4 MB once at
  boot. It SHALL NOT add a measurable delay to readiness, and SHALL NOT be rebuilt per
  request.
- **NFR-2 Read cost.** Listing templates computes one SHA-256 per row over its
  marshalled model. The default page size is 50 and the corpus is 11 rows; this is
  acceptable without caching. If the list endpoint's latency regresses measurably, the
  stored revision may be memoized per row content — but not in v1.
- **NFR-3 Observability.** A successful re-seed SHALL log at INFO with the template id,
  region, version, source file name, and the before/after revisions, so the change is
  reconstructable from logs alone.
- **NFR-4 Drift is image-relative.** A template is drifted *with respect to the
  currently deployed image*. During a rolling update the two replicas may briefly
  disagree. This is harmless — the flag is advisory and re-seed is idempotent — but it
  means the flag MUST NOT be treated as an alertable condition, and the UI copy should
  not imply an error state.
- **NFR-5 Multi-tenancy.** Templates are global. The re-seed endpoint is not
  tenant-scoped and SHALL NOT require the tenant headers.
- **NFR-6 Security.** The endpoint inherits the same exposure as the existing template
  PATCH and DELETE handlers; it introduces no new authentication or authorization
  surface. It is destructive to template content, which is why FR-5.5 requires an
  explicit confirmation in the only UI that offers it.

## 9. Open Questions

1. **Badge wording.** FR-5.1 fixes the meaning but not the label. "Outdated",
   "Differs from image", and "Behind seed" all read differently to an operator; pick
   one during design and use it consistently in the badge, the tooltip, and the confirm
   dialog.
2. **Detail-page action placement.** Whether "Reset to shipped defaults" belongs in the
   detail page header alongside the existing actions or in a separate destructive-action
   grouping. A UI convention question, to settle against existing detail pages.
3. **PATCH-induced drift.** The preset validator assigns ids to presets that lack them
   on PATCH. The checked-in seed files already carry preset ids, so a PATCH that does
   not touch presets should not perturb them — but this should be confirmed empirically
   during implementation, because if PATCH does rewrite preset content, every
   UI-edited template reports drift for a reason unrelated to what the operator
   changed. The behaviour would still be *correct* (the row genuinely differs from the
   file), but it affects how often the badge lights up and therefore how much operators
   trust it.

## 10. Acceptance Criteria

**Detection**

- [ ] Every one of the 11 files in `seed-data/templates/`, seeded into a fresh
      database and read back, reports `seedDrift: false`. Table-driven over the
      directory so a new version bring-up is covered automatically. *This is the
      load-bearing test: if `Create` or `Normalize` perturbs anything the revision
      sees, every template reports permanent phantom drift and the badge becomes
      noise.*
- [ ] A template whose stored content is mutated reports `seedDrift: true` with
      differing `storedRevision` and `shippedRevision`.
- [ ] A template with no corresponding seed file reports `seedDrift: false` and an
      empty `shippedRevision`.
- [ ] `RestModel.Id` does not affect either revision.

**Re-seed**

- [ ] Create a template from a seed file, PATCH one socket handler, confirm
      `seedDrift: true`, POST reseed → `204`; the template's UUID is unchanged, the
      PATCHed handler is gone, and `seedDrift` is `false`.
- [ ] Re-seeding an undrifted template returns `204` and leaves `storedRevision`
      unchanged.
- [ ] Re-seed against an unknown template id returns `404`.
- [ ] Re-seed against a template whose region/version ships no seed file returns `409`.
- [ ] Re-seed leaves `Region`, `MajorVersion` and `MinorVersion` untouched.

**Boot behaviour (regression guard)**

- [ ] Given an existing template row and a seed file whose content differs, running the
      boot seeder returns `"skipped"` and leaves the row byte-identical. *This is the
      test that protects "UI edits survive a redeploy" from being quietly broken by a
      later change.*

**UI**

- [ ] The drift badge renders only on rows reported as drifted, and never on rows with
      an empty `shippedRevision`.
- [ ] "Reset to shipped defaults" is disabled with a tooltip when no seed file ships
      for the template.
- [ ] Confirming the dialog issues the POST and refreshes the view; the badge clears
      without a manual reload.
- [ ] Dismissing the dialog issues no request.
- [ ] A failed re-seed surfaces an error and leaves the displayed template unchanged.

**Verification**

- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` clean in
      `services/atlas-configurations`.
- [ ] `tools/lint.sh --check` clean from the repo root.
- [ ] `services/atlas-ui` build and vitest clean.
- [ ] `docker buildx bake atlas-configurations` clean from the worktree root if
      `go.mod` changed.

**Remediation (post-merge, manual)**

- [ ] After deploy, the GMS 87.1, GMS 92.1, GMS 95.1 and JMS 185.1 template rows in
      `atlas-main` report drift, and re-seeding each one restores `QuestActionHandle`.
      Shipping this task does not fix those rows on its own — it makes them visible and
      makes the fix a button press. Call this out in the PR description.
