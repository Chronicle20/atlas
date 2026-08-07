# Template Re-seed Trigger — Design

Status: Approved for planning
Created: 2026-08-07
PRD: [`prd.md`](prd.md)

---

## 1. Summary

Three pieces of machinery, in dependency order:

1. **A canonical revision function.** One exported function turns a
   `templates.RestModel` into a SHA-256. Both the shipped side and the stored
   side call it. The PRD words FR-1.4 and FR-2.1 as two separate definitions
   that happen to agree; this design makes them *one* definition, so they
   cannot drift apart.
2. **A shipped-template catalog.** Loaded once from `$SEED_DATA_PATH/templates`,
   keyed by `(region, majorVersion, minorVersion)`, holding the source file
   name, the parsed model, and its revision. Becomes the single parse path —
   the boot seeder reads from it instead of re-reading files.
3. **A read-only view model plus a re-seed operation.** The three computed
   attributes ride on a `ViewRestModel` that embeds `RestModel`; re-seed
   rewrites `Entity.Data` on the existing row using `Create`'s exact
   validation-and-marshal semantics.

The load-bearing property that everything else rests on is stated in §5 and
guarded by one table-driven test over all eleven seed files.

---

## 2. Architecture decisions

### D1 — Where the catalog lives: package `templates`, not package `seeder`

The PRD's Service Impact table puts the catalog under `seeder/`. That does not
compile.

`seeder` already imports `templates`
(`seeder/seeder.go:5`, and `templateExists` / `importTemplate` construct
`templates.NewProcessor`). Drift detection and re-seed both live in
`templates`. A catalog in `seeder` would force `templates` → `seeder`, closing
an import cycle.

| Option | Verdict |
|---|---|
| **A. Catalog in package `templates`** (new `templates/shipped.go`) | **Recommended.** No cycle: `seeder` → `templates` stays one-directional. The catalog's payload *is* `templates.RestModel`, so it belongs in the package that owns that type. |
| B. New subpackage `templates/shipped` | Needs `templates.RestModel`, so `templates/shipped` → `templates`; then `templates` cannot import it back for drift computation. Same cycle, one level down. |
| C. New neutral top-level package with its own model type | Duplicates the entire template REST tree, or degrades to `json.RawMessage` and loses the normalization that makes revisions comparable. Rejected. |

FR-1.7 ("the boot seeder SHALL read from this catalog") is therefore satisfied
by `seeder` calling into `templates`, which it already does.

### D2 — Catalog lifetime: explicit init in `main`, singleton accessor, injectable into the processor

FR-1.2 asks for the project's registry convention (`sync.Once` + `sync.RWMutex`).
Applied literally to a lazily-initialized singleton, that would make the
catalog's source directory implicit and the processor untestable without
touching `$SEED_DATA_PATH`. Split it:

- `LoadCatalog(l logrus.FieldLogger, dir string) Catalog` — **pure**, returns a
  value, no globals. Everything in the test suite uses this.
- `InitShippedCatalog(l, dir)` / `ShippedCatalog() Catalog` — the singleton
  wrapper, `sync.Once` on init and `sync.RWMutex` on access, satisfying FR-1.2.
- `ProcessorImpl.WithCatalog(c Catalog) Processor` — builder injection mirroring
  the existing `WithValidator` (`templates/processor.go:49`). Unset means an
  empty catalog, which reports "no shipped file" for every template — the FR-2.4
  behaviour, so an un-wired processor degrades safely rather than panicking.

`main.go` calls `InitShippedCatalog(l, filepath.Join(seedConfig.SeedPath, "templates"))`
**before** `s.Run()` and before route registration.

**`SEED_ENABLED=false` must not disable the catalog.** It gates *importing*, not
*knowing what ships*. An operator who has disabled seeding still needs the drift
badge and the reset button. The catalog load is therefore unconditional; only
`Seeder.Run`'s import loop keeps its `Enabled` check.

### D3 — Computed attributes: a view model, not three fields on `RestModel`

This is the highest-risk decision in the task.

`Create` persists `json.Marshal(input)` verbatim into `Entity.Data`
(`templates/processor.go:92-110`). Any field added to `RestModel` with a JSON
tag is therefore **written into the stored document**. `shippedRevision`,
`storedRevision` and `seedDrift` would be persisted, `Make` would read them
back, and the revision would be computed over bytes that contain a previous
revision. Self-reference, permanent phantom drift, and a stored document that no
longer matches the seed-file shape the task-199 export depends on.

| Option | Verdict |
|---|---|
| **A. `ViewRestModel` embedding `RestModel`, used only on read paths** | **Recommended.** The write model is untouched, so the contamination class does not exist rather than being defended against. api2go builds the `attributes` object with a plain `json.Marshal(element)` (`api2go@v1.0.4/jsonapi/marshal.go:241`), and `encoding/json` flattens anonymous embedded structs — so the wire shape is exactly `RestModel`'s attributes plus three keys, with no per-field restating. `GetID`/`GetName`/`SetID` promote from the embedded `RestModel`. |
| B. Three fields on `RestModel`, zeroed before every marshal | Works today and breaks the first time someone adds a write path and forgets the zeroing. The failure is silent and only surfaces as universal phantom drift. Rejected. |
| C. `GET /templates/{id}/drift` sidecar | Rejected by the PRD (§5.1): the list page would need one request per row to render badges. |

```go
// templates/rest.go
type ViewRestModel struct {
    RestModel
    ShippedRevision string `json:"shippedRevision"`
    StoredRevision  string `json:"storedRevision"`
    SeedDrift       bool   `json:"seedDrift"`
}
```

The PATCH path keeps `rest.RegisterInputHandler[RestModel]`. The UI does a
read-modify-write and will send the three attributes back; `encoding/json`
drops unmodelled keys, which is exactly FR-5.1's "ignore them if present in a
request body" — achieved by omission, not by code.

`POST /configurations/templates` also returns `ViewRestModel`, computed from the
model it just persisted, so the resource shape is identical on every read and
on create.

### D4 — Re-seed writes through `Create`'s semantics, not `UpdateById`

FR-3.4 already fixes this and the PRD's rationale is correct: `UpdateById`
(`templates/processor.go:124-158`) runs the preset validator, which reassigns
`input.Characters.Presets` before marshalling. Re-seeding through it could
persist bytes that differ from the shipped file, and the row would report drift
again the instant it was reset.

The shared code is small enough that a helper is cleaner than a flag:

```go
// canonicalBytes applies the write-path normalization and validation and
// returns the exact bytes Create would persist.
func canonicalBytes(input RestModel) (json.RawMessage, error)
```

`Create` and `ReseedById` both call it. `UpdateById` deliberately does not — it
has the extra preset step and must keep it.

`ReseedById` then reuses the existing `update` transaction function
(`templates/administrator.go:12`), passing the **entity's** `Region` /
`MajorVersion` / `MinorVersion` columns rather than the file's, satisfying FR-3.3
structurally. (They are equal by construction — the catalog is keyed on the
parsed file's own region/version fields — but reading them from the row means a
hypothetical mismatch cannot rewrite the key.)

### D5 — The seeder's `ConfigMetadata` / `extractMetadata` are deleted

Today the seeder parses each file twice: once into `ConfigMetadata`
(`seeder/seeder.go:39-44`, `166-182`) to get the key, once into
`templates.RestModel` (`seeder.go:235`) to create. FR-1.6's duplicate-key rule
and FR-1.5's parse-failure rule need the key anyway, and the catalog already
holds the full model. `importTemplate(filePath)` becomes
`importTemplate(entry CatalogEntry)`, and `extractMetadata` / `ConfigMetadata` /
`discoverFiles` move into `LoadCatalog`. One parse path, as FR-1.7 requires.

`seedTemplates` keeps its `SeedResult` counters and its `"imported"` /
`"skipped"` / `"failed"` outcomes verbatim — `seeder_test.go` asserts on them.

### D6 — Error mapping is done in the handler, from sentinel errors

`server.WriteErrorResponse` (`libs/atlas-rest/server/error.go:48`) maps
everything to 500 (or 503 for classified-transient). There is no `WriteNotFound`
or `WriteConflict` helper, and the existing template GET-by-id already returns
500 for an unknown id. This task does not fix that pre-existing behaviour, but
the new endpoint must honour the PRD's status table, so:

```go
var (
    ErrTemplateNotFound  = errors.New("template not found")   // wraps gorm.ErrRecordNotFound
    ErrNoShippedTemplate = errors.New("no shipped template")
)
```

The re-seed handler switches on these and writes the JSON:API error document
itself, reusing the shape already used by `validationFailureError.AsJSONAPIErrors`
(`templates/validation_error.go:22-27`): `{"errors":[{"status","title","detail"}]}`
with `Content-Type: application/vnd.api+json`. A small unexported
`writeJSONAPIError(w, status, title, detail)` in the `templates` package covers
404 and 409; the 400 path reuses the existing `validationFailureError` branch
verbatim, so validation failures render identically to create and update.

### D7 — Where the drift is computed: inside the processor, over the existing providers

The processor gains three view providers built by mapping over the ones that
already exist, so there is one query path, not two:

```go
ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel]
ViewByRegionAndVersionProvider(region string, major, minor uint16) model.Provider[ViewRestModel]
AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]
```

each `model.Map(p.makeView)` over its `RestModel` counterpart. `AllProvider`
already runs `model.ParallelMap()` (`processor.go:64`), so the per-row SHA-256
is parallel across the page for free — NFR-2 is satisfied without a cache.

The `RestModel`-returning methods stay on the interface: `seeder.templateExists`
uses `GetByRegionAndVersion` and has no use for revisions.
`templates/mock/processor.go` needs the four new members.

---

## 3. Component inventory — `services/atlas-configurations`

| File | Change |
|---|---|
| `templates/shipped.go` *(new)* | `CatalogEntry`, `Catalog`, `LoadCatalog`, `InitShippedCatalog`, `ShippedCatalog`. |
| `templates/revision.go` *(new)* | `Revision(rm RestModel) (string, error)`. |
| `templates/rest.go` | `ViewRestModel`. |
| `templates/processor.go` | `WithCatalog`; `canonicalBytes` extracted from `Create`; three view providers + `makeView`; `ReseedById`; sentinel errors. |
| `templates/resource.go` | Read handlers marshal `ViewRestModel`; new `POST /{templateId}/reseed`; `writeJSONAPIError`. |
| `templates/mock/processor.go` | New interface members. |
| `seeder/seeder.go` | `extractMetadata` / `ConfigMetadata` / `discoverFiles` removed; `seedTemplates` iterates `templates.ShippedCatalog()`; `importTemplate` takes a catalog entry. Create-if-absent behaviour byte-for-byte unchanged. |
| `main.go` | `InitShippedCatalog` before `s.Run()`, unconditional on `SEED_ENABLED`. |

No schema change, no migration, no Kafka/outbox participation (FR-3.8).

---

## 4. Request flow

**Read** (`GET /configurations/templates`)

```
rows → Make (unmarshal + socket.Normalize + Id)   [existing]
     → makeView:
         stored  = Revision(rm)
         entry, ok = catalog.Lookup(rm.Region, rm.MajorVersion, rm.MinorVersion)
         shipped = entry.Revision if ok else ""
         drift   = ok && shipped != stored
     → ViewRestModel
```

**Re-seed** (`POST /configurations/templates/{id}/reseed`)

```
byIdEntityProvider(id) ─── not found ──→ ErrTemplateNotFound → 404
        │
   catalog.Lookup(e.Region, e.MajorVersion, e.MinorVersion)
        │      ─── miss ──→ ErrNoShippedTemplate → 409
        │
   canonicalBytes(entry.Model)  ── socket issues ──→ validationFailureError → 400
        │
   ExecuteTransaction(update(ctx, id, e.Region, e.MajorVersion, e.MinorVersion, bytes))
        │
   INFO log: templateId, region, majorVersion, minorVersion, file, beforeRevision, afterRevision   [NFR-3]
        │
      204
```

Idempotence (FR-3.7) is free: `canonicalBytes` is a pure function of the
catalog entry, so re-running it produces the same bytes.

---

## 5. The canonical-bytes invariant

Everything about false-positive-free drift reduces to one chain. Write it down,
because every future change to `RestModel` has to preserve it.

Let `C = Marshal(Normalize(Unmarshal(fileBytes)))`.

1. **Shipped side.** `LoadCatalog` parses the file and calls `Revision`, which
   normalizes and marshals. Shipped revision = `sha256(C)`.
2. **Persist side.** `Create` normalizes then marshals the same parsed model
   (`processor.go:87-95`). `Entity.Data` = `C`, byte-identical.
3. **Stored side.** `Make` unmarshals `Data` and re-normalizes
   (`processor.go:67-76`); `Revision` marshals that. Stored revision =
   `sha256(Marshal(Unmarshal(C)))`.

Step 3 equals step 1 iff `Marshal ∘ Unmarshal` is the identity on `C` for this
struct tree. It is, and the reasons are worth recording because each is a real
hazard someone could reintroduce:

- **Unmodelled keys.** Dropped on parse — but dropped symmetrically, since `C`
  was produced by the same parse. A key in the file that `RestModel` does not
  declare is absent from both sides.
- **`map[string]interface{}` options** (`socket/handler/rest.go:18`,
  `socket/writer/rest.go:10`). Go marshals map keys in sorted order, so ordering
  is deterministic. Values arrive as `float64`; `encoding/json` formats a
  `float64` with `'f'` notation for `1e-6 ≤ |x| < 1e21`, which covers every
  opcode, id and count these tables carry — so `123` round-trips as `123`, not
  `1.23e+02`.
- **`nil` vs empty slices.** `socket.Normalize` (`templates/socket/rest.go:30`)
  coerces the four socket collections to `[]`. `NPCs`, `Worlds`,
  `Characters.Templates` and `Characters.Presets` carry no `omitempty`, so an
  absent key becomes `nil` and marshals as `null` on **both** sides. Five of the
  eleven seeds omit `npcs` (gms 12/87/92/95, jms 185); that is the documented
  `"npcs": null` the UI service layer already compensates for
  (`templates.service.ts` `sortTemplate`). Symmetric, therefore invisible to the
  revision.
- **`RestModel.Id`.** `json:"-"` (`templates/rest.go:12`). **Deviation from
  FR-2.2:** rather than *rely* on the tag, `Revision` clears `rm.Id` before
  marshalling. This is strictly stronger, costs one assignment, and keeps the
  FR-2.2 test (revision is id-invariant) meaningful instead of tautological.

The acceptance test in §8 is what actually holds this. Prose is not the guard.

---

## 6. UI design — `services/atlas-ui`

### Open Question 1 — badge wording: **"Differs from image"**

Chosen over "Outdated" and "Behind seed". NFR-4 says the flag is advisory and
image-relative and must not read as an error state; "Outdated" asserts the row
is wrong, which is false when an operator edited it deliberately. "Behind seed"
implies an ordering that does not exist — the row can differ in either
direction. "Differs from image" names the comparison and nothing more.

Used verbatim in all three places: badge label, badge tooltip
("Differs from the configuration shipped in this image"), and the confirm
dialog body. Badge variant: `secondary` (neutral), not `destructive`.

### Open Question 2 — action placement: **detail-layout header**

`TemplateDetailLayout.tsx:37` already hosts `ConfigExportButton` in a header
flex row, for the reason given in that component's own doc comment: living in
the layout puts it on every sub-tab with no per-page wiring. Re-seed has the
same property — it acts on the whole template document, not on the sub-tab
being viewed. A new `TemplateReseedButton` sits beside it. No separate
destructive-action grouping is introduced; one sibling button does not warrant
a new convention.

### Components

| File | Change |
|---|---|
| `types/models/template.ts` | `shippedRevision?: string; storedRevision?: string; seedDrift?: boolean` on `TemplateAttributes`. Optional because fixtures and any older API predate them; `exactOptionalPropertyTypes` is on, so read sites must handle `undefined` rather than assume `""`/`false`. |
| `services/api/templates.service.ts` | `reseed(id: string, options?: ServiceOptions): Promise<void>` → `POST ${BASE_PATH}/${id}/reseed`, no body, 204. `sortTemplate` / `validateTemplate` unchanged — `validateTemplate` whitelists required fields and ignores extras. |
| `lib/hooks/api/useTemplates.ts` | `useReseedTemplate()` mutation; on success invalidates `templateKeys.detail(id)` and `templateKeys.lists()` (FR-5.6). |
| `components/features/templates/TemplateReseedButton.tsx` *(new)* | Button + `AlertDialog`. Disabled with tooltip when `shippedRevision` is falsy (FR-5.4) or the query has no data. `AlertDialogCancel` renders first and holds default focus (FR-5.5). Errors surface via `toast.error(createErrorFromUnknown(...))`, matching `ConfigExportButton`. |
| `components/features/templates/TemplateDetailLayout.tsx` | Mount the button next to `ConfigExportButton`. |
| `pages/templates-columns.tsx` | Badge in the existing Minor column cell, or a dedicated narrow column; rendered only when `seedDrift === true`. FR-5.2 falls out — `seedDrift` is `false`/absent when `shippedRevision` is empty. |
| `lib/utils/config-export.ts` | **Strip the three keys from the export payload.** See below. |

### The export-leak fix (not in the PRD, in scope here)

`toConfigExportPayload` passes every attribute through untouched by design
(its doc comment: "everything else is passed through untouched … so this module
never has to track a key list"). Adding three computed attributes to the
resource therefore leaks `shippedRevision` / `storedRevision` / `seedDrift` into
the seed-shaped JSON that task-199's Export button produces — and that file
exists precisely to be promoted into `seed-data/templates/`. The server would
drop the keys on parse, so nothing breaks, but a committed seed file carrying a
stale hash of itself is exactly the kind of noise this task is trying to remove.

Delete the three keys in `toConfigExportPayload` with a one-line comment
pointing here. This is the only place the module's "no key list" principle is
knowingly broken, and the reason is that these keys are *computed*, not
*configured* — they are not part of the document's shape at all.

---

## 7. Open Question 3 — resolved empirically: PATCH does not perturb presets

`Validator.Validate` assigns a UUID only when `presets[i].Id == ""`
(`templates/characters/preset/validator.go:37-41`). Across the eleven shipped
seed files:

| Files | Presets | With ids |
|---|---|---|
| gms 48/61/72/79/83/84 | 10, 10, 12, 12, 12, 12 | all |
| gms 12/87/92/95, jms 185 | 0 | — |

Six files carry presets and every preset already has an id; the other five have
none, so the validator's loop is a no-op. **No shipped template can be perturbed
by the preset validator.** The badge therefore lights up only for changes an
operator actually made, and the concern raised in the PRD does not materialize.

This is a property of the current seed corpus, not an invariant — a future seed
file added with id-less presets would reintroduce it. The `presets-carry-ids`
condition is worth a line in `TEMPLATE_CONVENTIONS.md`, not a new CI guard: the
consequence is a badge that lights up spuriously on one template, not a
gameplay failure.

---

## 8. Testing strategy

**Go — `templates` package** (existing sqlite in-memory harness,
`processor_test.go:29`; seed files reachable at `../../../seed-data/templates`)

1. **`TestShippedSeedsReportNoDrift`** — the load-bearing test. Table-driven
   over every `*.json` in the seed directory (so a new version bring-up is
   covered without editing the test): load the catalog, `Create` from the
   catalog model, read back through the view provider, assert
   `seedDrift == false` and `storedRevision == shippedRevision`. Also assert the
   file count is non-zero, so a broken path silently passing zero cases fails.
2. `TestRevisionIgnoresId` — FR-2.2.
3. `TestDriftDetectedAfterMutation` — mutate stored data, expect
   `seedDrift == true` and differing revisions.
4. `TestNoCatalogEntryIsNotDrift` — unknown region/version →
   `shippedRevision == ""`, `seedDrift == false`.
5. `TestCanonicalBytesMatchesCreate` — `canonicalBytes(m)` equals the `Data`
   column `Create(m)` writes. Pins §5 step 2.
6. `TestReseed*` — restores mutated content; preserves `Id`, `Region`,
   `MajorVersion`, `MinorVersion`; idempotent (`storedRevision` unchanged on a
   second call); `ErrTemplateNotFound` on unknown id; `ErrNoShippedTemplate`
   when the catalog has no entry.
7. `TestLoadCatalog*` — unparseable file logged and omitted without failing the
   load (FR-1.5); duplicate key resolves to the sort-order-first file (FR-1.6).
   Fixtures under `templates/testdata/`.

**Go — `seeder` package**

8. `TestSeederSkipsExistingWithDifferentContent` — the PRD's regression guard:
   pre-create a row, run the seeder against a file with different content,
   assert outcome `"skipped"` and `Entity.Data` byte-identical.
9. Existing `seeder_test.go` cases keep passing against the refactored
   `seedTemplates` / `importTemplate`.

**Go — `resource`**

10. `POST /{id}/reseed` → 204; unknown id → 404; no shipped file → 409;
    response bodies are JSON:API error documents. Follows the existing
    `resource_no_content_test.go` pattern.

**TypeScript — vitest**

11. `templates.service` — `reseed` issues the POST and resolves on 204;
    surfaces an error on non-2xx.
12. `useReseedTemplate` — invalidates detail + list keys on success.
13. `TemplateReseedButton` — disabled + tooltip when `shippedRevision` is
    absent; dialog cancel issues no request; confirm issues the POST; failure
    raises a toast and leaves the rendered template unchanged.
14. `templates-columns` — badge renders only when `seedDrift === true`.
15. `config-export` — the three keys are absent from the exported payload.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| A future `RestModel` field breaks `Marshal ∘ Unmarshal` identity and every template reports phantom drift. | Test 1 fails on the whole corpus the moment it happens. §5 records why, so the fix is obvious rather than archaeological. |
| A write path is added that forgets the view/write split and persists a computed attribute. | Structurally impossible — the computed fields do not exist on `RestModel`. |
| The catalog is empty because `SEED_DATA_PATH` is wrong; every template silently reports "no shipped file" and the feature appears to do nothing. | `InitShippedCatalog` logs at INFO with the resolved directory and the entry count, and at WARN when the count is zero. |
| Re-seed silently reverts a deliberate operator edit. | Confirmation dialog (FR-5.5) plus the NFR-3 INFO log carrying before/after revisions, so the change is reconstructable. |
| `atlas-channel` behaviour changes. | It does not — templates are not tenant configuration, and no tenant row is touched. |

---

## 10. Out of scope (restating the PRD, unchanged)

Tenant-configuration drift (owned by task-189), automatic reconcile, arbitrary
JSON import (task-199), diff preview, bulk re-seed, merge semantics, alerting.

---

## 11. Verification

Per `CLAUDE.md`, before PR:

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `services/atlas-configurations`.
- `tools/lint.sh --check` clean from the repo root.
- `services/atlas-ui`: `npm run build` (type-checks tests) and vitest clean.
- `docker buildx bake atlas-configurations` only if `go.mod` changed — no new
  dependency is expected (`crypto/sha256` and `encoding/json` are stdlib), so
  this should be a no-op.
- Code review via `superpowers:requesting-code-review` before opening the PR.

The PR description must carry the PRD's remediation note: shipping this task
does not repair the GMS 87.1 / 92.1 / 95.1 / JMS 185.1 rows in `atlas-main`. It
makes them visible and makes the repair a button press.
