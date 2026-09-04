# task-289 — Tenant Template Drift Detection & Reset — Design

Status: Approved for planning
Created: 2026-09-02
Inputs: `prd.md` (v1), `decisions.md` (D1–D5)

This document resolves OQ-1 … OQ-4, fixes the architecture, and records the
tradeoffs. It does not contain an implementation plan — that is `/plan-task`.

---

## 1. Ground truth established during design

Everything below was read out of the worktree, not remembered.

| Fact | Evidence |
|---|---|
| `templates.Revision` is a whole-document hash over the struct, clearing `Id`/`Environment` and normalizing `Socket` | `services/atlas-configurations/atlas.com/configurations/templates/revision.go:31-42` |
| `templates.ViewRestModel` embeds `RestModel` and adds three computed keys; the PATCH path still binds `RestModel` | `templates/rest.go:59-64` |
| Template reads use **overlay/baseline fallback** scoping; tenant reads use **strict** environment scoping | `templates/provider.go:24-63`, `tenants/provider.go:16-51`, `scope/scope.go:29-52` |
| `tenants.update` already writes a `HistoryEntity` **before** `db.Save`, and calls `scope.AuthorizeWrite` first | `tenants/administrator.go:57-89` |
| `tenants.UpdateById` enqueues the status outbox inside the same transaction, re-reading the row so `Environment` is server-owned | `tenants/processor.go:127-175` |
| `tenants.Make` normalizes `Socket` and overwrites `Id`/`Environment` from the entity | `tenants/processor.go:99-113` |
| `templates.socket.Normalize` and `tenants.socket.Normalize` are byte-for-byte the same function over byte-for-byte the same JSON tags | `templates/socket/rest.go:30-44` vs `tenants/socket/rest.go:30-44` |
| `preset.Validator.Validate` **mutates** the input: it assigns a fresh `uuid.New()` to any preset with an empty `Id` | `tenants/characters/preset/validator.go:36-47` |
| `templates.ReseedById` deliberately routes through `canonicalBytes`, *not* `UpdateById`, precisely to dodge that mutation | `templates/processor.go:150-160`, `:236-241` |
| `tenants` does not import `templates` today; `templates` does not import `tenants` | grep over `tenants/**.go` |
| The tenants **list page** renders registry tenants (`useTenant()` context → atlas-tenants), not configuration documents | `services/atlas-ui/src/pages/TenantsPage.tsx:49`, `tenants-columns.tsx:11` |
| A hook for the configuration list already exists | `src/lib/hooks/api/useTenantConfigurations` (`useTenants.ts:212`) |
| `onboardTenant` builds the attribute object from 8 keys + conditional `cashShop`, dropping `mapleLife` | `src/services/api/onboarding.service.ts:104-118` |
| `updateTenantConfiguration` spreads the whole fetched `attributes` into the PATCH body | `src/services/api/tenants.service.ts:308-322` |
| `toConfigExportPayload` deletes the three template computed keys before export | `src/lib/utils/config-export.ts:80-83` |
| **MTS configuration is not in this document.** It is served by `atlas-tenants` as the `mts-configs` type under `/api/tenants/{id}/configurations/mts-configs` | `src/services/api/mts-config.service.ts:8-10, 35` |

---

## 2. Resolved open questions

### OQ-3 — `TenantsMtsConfigPage` is **out of scope**, and not by choice

The MTS knobs are a different resource in a different service
(`atlas-tenants`, type `mts-configs`). They are not a section of the
`atlas-configurations` tenant document, they have no template counterpart, and
`atlas-configurations` cannot read them. Nothing to add to FR-2.3, nothing to
exclude in FR-2.4 — the question is answered by the resource boundary. The
design records it here so the next reader does not re-derive it.

### OQ-1 — `usesPin` becomes the **`properties`** residual section (supersedes FR-2.3's literal name)

FR-2.7 is the binding requirement: *adding a new field to the document must not
require an edit to the drift code for that field to participate in drift.*

An enumerated `usesPin` section does not satisfy it. Section revisions need a
name → value mapping; a future top-level scalar (say `enableAutoRegister`)
would land in the aggregate hash but in no named section, producing the worst
possible output — `templateDrift: true` with all six section flags `false`, an
indicator that says "something is wrong, and I will not tell you what."

So `sectionDrift` is keyed by:

```
properties   socket   characters   npcs   cashShop   mapleLife
```

where **`properties` is defined by subtraction**: every comparable key not
claimed by one of the five named sections. Today that is exactly `usesPin`.
Tomorrow it is `usesPin` plus whatever was added, with no drift-code edit. It
also maps cleanly onto the UI's existing "Global Properties" page, which is
where an operator would look for it.

**This supersedes the literal `usesPin` key in PRD FR-2.3 and in three
acceptance criteria.** Substitute `properties` for `usesPin` when reading them.
No alias is accepted on the reset endpoint: `sections: ["usesPin"]` is a `400`
like any other unknown name. An alias would be a permanent second name for one
section that exists only to paper over a one-day-old PRD draft.

### OQ-2 — a new sibling package, `atlas-configurations/drift`

`templates` and `tenants` are peer domain packages of the same service. Making
`tenants` import `templates`' revision internals to hash a tenant is the exact
shape the repo convention warns about, and the reverse direction is no better.
A third, dependency-free package that neither owns is the only arrangement in
which "one definition serves both sides" (FR-2.1) is structurally true rather
than maintained by discipline.

`drift` imports nothing from `templates` or `tenants`. It operates on
**marshaled JSON**, never on either package's Go types, which is what makes
FR-2.6 fall out for free — `templates.maplelife.RestModel` and
`tenants.maplelife.RestModel` are distinct Go types with identical JSON tags,
and the hash never learns the difference.

### OQ-4 — one component, mounted twice (header = whole document, section page = its own section)

A single `TenantResetButton` takes an optional `sections` prop. The header
mounts it with no sections (whole document, FR-6.3); each section page mounts
it scoped to its own section (FR-6.4). Rejected: a multi-select inside the
header dialog. It puts a six-way form in front of the one-click common case,
and it asks the operator to name sections in the abstract rather than while
looking at the thing they want reverted.

---

## 3. Architecture

### 3.1 The `drift` package

New: `services/atlas-configurations/atlas.com/configurations/drift/`.

```go
package drift

// Doc is a canonical, package-neutral view of a configuration document.
type Doc map[string]json.RawMessage

// Excluded names the keys that never participate in drift or reset (FR-2.4).
var Excluded = []string{"environment", "region", "majorVersion", "minorVersion", "worlds", "diagnostics"}

// Named lists the sections that get their own key in the drift report.
// Everything comparable and unnamed falls into Properties (OQ-1).
var Named = []string{"socket", "characters", "npcs", "cashShop", "mapleLife"}
const Properties = "properties"

func Canonicalize(v any) (Doc, error)          // marshal → map → drop Excluded → prune
func Sections(d Doc) map[string]string          // per-section hex sha256, incl. "properties"
func Aggregate(d Doc) (string, error)           // one hex sha256 over the whole Doc
func Compare(base, stored Doc) (agg bool, per map[string]bool, err error)
func Merge(stored, base Doc, sections []string) (Doc, error) // reset primitive
func ValidateSections(sections []string) error  // ErrUnknownSection
```

`Id` never appears: it carries `json:"-"` on both models. `Excluded` deliberately
does not list it — a key that cannot be produced does not need excluding, and
listing it would imply the marshal is untrusted.

**Pruning is the interesting part.** `Canonicalize` recursively removes any
object key whose value is `null`, `[]`, or `{}` (after recursing into it). This
collapses the entire nil-vs-empty false-positive class in one generic rule
rather than one normalizer per slice field:

- `npcs: null` (a seed file that omits the key) ≡ `npcs: []` (what the UI sends
  after a round trip through `?? []`).
- `cashShop.commodities: null` ≡ `[]`, at any depth, without naming it.
- Socket is still `Normalize`d by both `Make`s before it reaches `Canonicalize`;
  pruning makes that normalization belt-and-braces rather than load-bearing.

Pruning cannot hide a real divergence: if one side has content and the other is
empty, the non-empty side keeps its key and the hashes differ. It only erases
distinctions between *ways of writing nothing*. `false`, `0`, and `""` are
**not** pruned — they are values, not absences.

Determinism: `json.Marshal` of a `map[string]json.RawMessage` emits keys in
sorted order, so a `Doc` has exactly one serialization regardless of how it was
built. This is also why §3.2's warning matters.

### 3.2 `drift` hashes and `templates.Revision` hashes are **not comparable**

`templates.Revision` hashes the *struct* — field-declaration order.
`drift.Aggregate` hashes a *map* — key-sorted order. Same document, different
bytes, different SHA-256. That is fine, because each is only ever compared
against itself:

| Comparison | Function | Both sides |
|---|---|---|
| template vs shipped seed file (task-201) | `templates.Revision` | shipped catalog entry, stored row |
| tenant vs baseline template (this task) | `drift.Aggregate` / `drift.Sections` | baseline template `RestModel`, tenant `RestModel` |

**`templates.Revision` is not touched, not moved, and not re-expressed in terms
of `drift`.** Refactoring it would change the bytes it hashes and silently
invalidate task-201's behavior; its existing tests are the guard the PRD names,
and they pass unmodified because the file is unmodified. A comment in
`revision.go` and a matching one in `drift/doc.go` will state the
non-comparability, because the failure mode if someone crosses them is a drift
flag that is permanently `true` for every row.

### 3.3 Read path — `tenants.ViewRestModel`

Mirrors `templates.ViewRestModel` exactly, for the reason its comment already
gives (FR-3.2):

```go
type ViewRestModel struct {
	RestModel
	BaselineTemplateId string          `json:"baselineTemplateId"`
	BaselineRevision   string          `json:"baselineRevision"`
	StoredRevision     string          `json:"storedRevision"`
	TemplateDrift      bool            `json:"templateDrift"`
	SectionDrift       map[string]bool `json:"sectionDrift"`
}
```

`SectionDrift` is a `map[string]bool`, not a struct. A struct would have to be
edited every time a section is added, which is the FR-2.7 trap again, one level
up. The map is always fully populated — all six keys present, all `false` when
no baseline resolved — so a client never has to distinguish "absent" from
"false".

The write model `RestModel` is untouched, so FR-3.3 holds by omission rather
than by code, exactly as it does for templates.

### 3.4 Baseline resolution and the N+1 (FR-1.1, FR-3.4)

`tenants.Processor` gains `WithTemplates(templates.Processor) Processor`,
following the established `WithValidator` / `WithCatalog` injection shape. An
unset templates processor degrades to "no baseline for anything" — every row
reports the FR-1.3 unknown state — so an un-wired processor is safe rather than
nil-panicking. This mirrors `WithCatalog`'s documented degradation.

Direction: `tenants` → `templates`. `templates` imports nothing from `tenants`,
so there is no cycle, and both are peer domain packages within one service.

Baseline lookup is `templates.Processor.GetByRegionAndVersion(region, major,
minor)`, which carries the overlay/baseline environment fallback for free —
that *is* FR-1.1, and re-implementing the query in `tenants` would be a second
definition of visibility. `gorm.ErrRecordNotFound` → no baseline (FR-1.3),
never an error to the caller (FR-1.4). Any other error is logged and also
degrades to no-baseline: a read must not 500 because the templates table
hiccupped.

For the list (FR-3.4, NFR-1) the view is built in **two explicit phases**, not
inside `ParallelMap`:

1. `AllProvider(page)` → `[]RestModel` (already parallel-mapped from entities).
2. Collect the distinct `{region, major, minor}` keys — realistically 1–3 per
   page — and resolve each baseline **once**, serially, into a plain map.
3. Decorate every row from that map.

A cache consulted from inside `ParallelMap` would need a mutex and would still
race two goroutines into the same query. Phase separation makes "once per
distinct key per request" a property of the control flow instead of a property
of a lock.

### 3.5 Reset

```go
ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error)
```

Sentinel errors, following `templates`' precedent of switching on sentinels in
the handler because `server.WriteErrorResponse` flattens everything to 500:

| Sentinel | Status |
|---|---|
| `ErrUnknownSection` (from `drift.ValidateSections`) | 400 |
| `scope.ErrCrossEnvironmentWrite` (already produced by `update`) | 403 |
| `ErrTenantNotFound` | 404 |
| `ErrNoBaselineTemplate` | 409 |
| `*validationFailureError` | 422 |

Order of operations:

1. **Validate section names first**, before any I/O. A 400 for a typo should not
   depend on the tenant existing.
2. Read the entity through `byIdEntityProvider` (environment-scoped) →
   `ErrTenantNotFound` on `gorm.ErrRecordNotFound`. Scoped, so a tenant in
   another environment is a 404, not a 403 — a caller who cannot read the row
   learns nothing about it.
3. Resolve the baseline from the **entity's** region/version columns, not the
   document's. Same reasoning as `templates.ReseedById`: the lookup key must
   come from the row, so a document/column mismatch can never rewrite the key.
   No baseline → `ErrNoBaselineTemplate` (409).
4. `stored := drift.Canonicalize(tenants.Make(e))`,
   `base := drift.Canonicalize(baselineTemplateRestModel)`,
   `merged := drift.Merge(stored, base, sections)`.
5. Unmarshal `merged` back into `tenants.RestModel`, then re-apply the
   entity-owned fields the merge never carried (`Id`, `Environment`, `Region`,
   `MajorVersion`, `MinorVersion`) and the excluded document fields (`Worlds`,
   `Diagnostics`) from the stored model. FR-4.4 is enforced twice: once because
   `Canonicalize` dropped those keys so `Merge` cannot see them, and once
   because they are re-applied from the stored row.
6. Validate (§3.6).
7. Write, in one transaction: `update(ctx, tenantId, e.Region, e.MajorVersion,
   e.MinorVersion, data)` — which gives history-before-write (FR-4.7) and
   `AuthorizeWrite` (FR-4.6) with no new code — followed by
   `enqueueTenantStatus` with the environment re-read from the persisted row
   (FR-4.8), byte-identical to what `UpdateById` does.
8. Log at info: tenant id, baseline template id, sections, before/after
   aggregate revision (NFR-6). Modeled on `ReseedById`'s log line, including
   its best-effort treatment of a before-revision that fails to compute.
9. Re-read through the view provider and return it (FR-4.10).

**Merge semantics.** `Merge` operates on `Doc`, so a section is replaced
wholesale — key-for-key, at the top level — never field-merged. `sections`
empty or nil means every comparable section. `properties` means "every
comparable key not in `Named`", computed from `base ∪ stored` so that a key
present on only one side is still handled: present in base and not stored → it
is added; present in stored and not base → it is removed. Both are correct
"restore to baseline" outcomes.

**Idempotence (FR-4.11)** is structural: the second application computes the
same `merged` from the same `base`, so `db.Save` writes identical bytes. It
still records a history row, which is the honest behavior — an operator did
perform an action.

**Field-shape drop.** Unmarshaling `merged` into `tenants.RestModel` discards any
key the template document has and the tenant model does not. There are none
today (the two models are key-identical minus `diagnostics`), and if one is ever
added the tenant simply does not gain a field it has no code to use. Worth a
one-line comment at the unmarshal, not worth a guard.

### 3.6 Validation, and the preset-mutation trap (FR-4.9 vs FR-4.10)

`preset.Validator.Validate` assigns a fresh UUID to any preset with an empty
`Id` (`validator.go:37-41`). If the reset persisted the validator's output, a
baseline whose presets carry empty ids would be written with *new* ids, the
tenant would differ from the template the instant it was reset, and FR-4.10
("`templateDrift: false` in the reset response") would fail intermittently — on
exactly the templates that were hand-authored rather than round-tripped through
a PATCH. This is the same trap `templates.ReseedById` documents at
`processor.go:150-160`, arriving from a different direction.

Resolution: **run the validator for detection, discard its mutation.** The reset
passes a copy of the merged presets to `Validate`, collects the errors, and
persists the merged document verbatim. Socket validation (`socketValidate`) is
pure and is applied to the merged document directly.

Consequence, accepted: a baseline preset with an empty id is persisted with an
empty id, and the *next* ordinary PATCH will assign one — at which point the
tenant genuinely has drifted, and the flag correctly says so.

Second consequence: the reset needs the same synthesized tenant context that
`handleUpdateConfigurationTenant` builds (`resource.go:82-92`), or the
atlas-data-backed preset rules silently skip. The reset handler synthesizes it
identically, from the URL tenant id and the **stored row's** region/version.

The PRD assigns 422 to reset validation failures while PATCH renders the same
`validationFailureError` as 400. The design follows the PRD. The inconsistency
is deliberate and worth naming: a 400 on PATCH means "your body is bad", while
a validation failure on reset means "the server's own baseline is unprocessable"
— the request was fine. 422 is the more truthful code, and this is a new
endpoint, so nothing regresses. `validationFailureError.AsJSONAPIErrors()` is
reused verbatim for the body.

### 3.7 Routing and handler wiring

```
POST /configurations/tenants/{tenantId}/reset
```

Registered in `tenants.InitResource` alongside the existing five routes. The
body is optional; an absent body, `{}`, an absent `sections` key, and
`sections: []` are all "whole document". This means the handler cannot use
`rest.RegisterInputHandler[T]` (which requires a JSON:API envelope) — it uses
`rest.RegisterHandler` and decodes the body itself, tolerating `io.EOF` as
"empty". A malformed non-empty body is a 400.

A `viewProcessor(d, db)` helper mirrors `templates/resource.go:39-42`: the
ordinary processor with the templates processor attached. The write paths
(`Create`, `UpdateById`, `DeleteById`) deliberately do not get it.

`GET` list, `GET` by id, and `POST` (create) all switch their marshal type from
`RestModel` to `ViewRestModel`. Create is changed to read back through the view
provider rather than echoing `input`, matching
`handleCreateConfigurationTemplate` — additive on the wire, and it means the
onboarding flow can assert FR-5.2 from the create response.

Sparse fieldsets (`?fields[tenants]=...`, used by `getSocketMatrix`) keep
working: `ViewRestModel` embeds `RestModel` anonymously, `encoding/json`
flattens it, and templates already prove the shape.

### 3.8 Mock processor

`tenants/mock/processor.go` gains `WithTemplates` and `ResetById` fields to keep
satisfying the interface.

---

## 4. Frontend design

### 4.1 Types and hygiene

`TenantConfigAttributes` gains the five computed keys as **optional**
(`baselineTemplateId?`, `baselineRevision?`, `storedRevision?`,
`templateDrift?`, `sectionDrift?: Record<string, boolean>`), so a response from
an older backend still type-checks.

Two existing pass-through sites must drop them, for the reason
`config-export.ts:75-83` already spells out:

- `toConfigExportPayload` — an exported file must not carry a stale hash of
  itself. Extend the existing `delete` block.
- `tenantsService.updateTenantConfiguration` — it spreads the whole fetched
  `attributes` into the PATCH body. The server ignores them (FR-3.3), so this
  is hygiene, not a fix; it keeps request bodies honest.

### 4.2 List indicator (FR-6.1)

`TenantsPage` renders **registry** tenants from `useTenant()` context, which have
no configuration attributes at all. The drift column therefore needs a second
source: the page calls `useTenantConfigurations()` (already exists,
`useTenants.ts:212`) and passes a `Map<id, TenantConfig>` into `getColumns`.
The cell mirrors `templates-columns.tsx:58-65` — render nothing unless
`templateDrift === true`, so an older backend that omits the key renders
nothing rather than a false positive.

A tenant present in the registry with no configuration row renders nothing,
which is already a visible problem elsewhere and is not this task's to solve.

### 4.3 Detail header (FR-6.2, FR-6.3, FR-6.5)

`TenantDetailLayout` already calls `useTenantConfiguration(id)` for the MapleLife
nav gate, so the drift data arrives with no extra request. The header gains:

- a drift summary naming the diverging sections (from `sectionDrift`), beside
  the existing `ConfigExportButton`;
- `<TenantResetButton id={id} />` — whole document.

`TenantResetButton` is `TemplateReseedButton` re-cut, keeping every convention
that component's comments establish and that FR-6.6/6.7 restate:

- the `Tooltip` root is always mounted and only its content is gated, so the
  button's DOM node is never remounted and focus is never dropped;
- `AlertDialogCancel` renders **first**, so Enter never fires the destructive
  action;
- errors route through `createErrorFromUnknown`, the dialog stays open, the
  displayed tenant is untouched;
- disabled with an explanatory tooltip when `baselineTemplateId` is empty or
  absent (FR-6.5) — the client-side mirror of the server's 409.

The dialog copy states all three facts FR-6.3 requires: UI edits to the reset
sections are lost; id, region, version, worlds and diagnostics are unchanged;
no game data is touched. The third is the one an operator actually needs, and
it is the whole reason this feature exists instead of delete-and-recreate.

### 4.4 Per-section reset (FR-6.4)

The same component with `sections={["socket"]}` (handlers, writers),
`["characters"]` (character templates, presets), `["mapleLife"]`,
`["cashShop"]`, `["npcs"]`, `["properties"]` (global properties). Mounted into
each page's `DetailActionBar`, so no page grows its own dialog. The dialog title
and copy interpolate the section label; everything else is shared.

Handlers and Writers are two pages over one `socket` section: both mount
`sections={["socket"]}` and their copy says "socket handlers and writers", so an
operator resetting from the Writers page is not surprised to find handlers
reverted too.

### 4.5 Cache invalidation (FR-6.8) and the clone fix (FR-5.1)

`useResetTenantConfiguration` invalidates, mirroring `useReseedTemplate`
(`useTenants.ts` gains what `useTemplates.ts:358-371` does):
`tenantKeys.configuration(id)`, `tenantKeys.configurations()`, and
`socketKeys.all` — the last because a socket reset changes what the socket
matrix and the handlers/writers grids show, and none of those clear on their
own.

`onboarding.service.ts` adds `mapleLife: template.attributes.mapleLife` to the
attribute object (D5, FR-5.1). It is copied unconditionally rather than behind
the `cashShop`-style `!== undefined` guard: `mapleLife` is a non-pointer struct
on both models, so it is always present in a template response, and a
conditional would reintroduce exactly the omission being fixed the first time a
template happened to serialize it as absent.

---

## 5. Testing strategy

The acceptance criteria are mostly already test-shaped. What matters is where
each lives.

**`drift` package (pure, no DB).** The pruning rules — `null` ≡ `[]` ≡ absent at
top level and nested; `false`/`0`/`""` preserved; content-vs-empty still differs.
Section partitioning: a `properties`-only edit flips only `properties`. `Merge`
semantics: replacement not field-merge; key-present-on-one-side-only in both
directions; empty `sections` ≡ all. `ValidateSections` rejects `worlds`,
`diagnostics`, `region`, `id`, `environment`, `usesPin`, and gibberish.

**Cross-type equality (FR-2.6).** One test that canonicalizes a
`templates.RestModel` and a `tenants.RestModel` built from the same JSON and
asserts identical aggregate and per-section hashes. This is the test that fails
if someone adds a field to one model and not the other, which is the whole
point of having it.

**`tenants` processor (sqlite, as the existing `processor_test.go` does).**
No-baseline state; per-section flip isolation; `worlds`/`diagnostics` edits flip
nothing; reset at each scope; the FR-4.4 preservation set byte-for-byte;
idempotence; the history row's content; the outbox enqueue; 403 on cross-env;
409 on no baseline. The list test asserts **one** template query for a page of
many same-version tenants — a counting stub around the templates processor, so
FR-3.4 is enforced rather than described.

**`templates` package.** Untouched, and its tests are run unmodified as the
guard the PRD names. If any template test needs editing, the change is wrong.

**FR-5.2 — the freshly-onboarded tenant.** Two halves, because the flow spans
two services and no single test can see both:
- Go: create a template, clone its full attribute set into a tenant through
  `tenants.Create`, read the view, assert `templateDrift: false` and all six
  section flags `false`.
- TS: assert `onboardTenant` puts every comparable key — `mapleLife` included —
  into the POST body, so the Go test's premise stays true.

Neither test alone is sufficient and both are cheap; the failure this pair
catches is precisely D5's.

**Frontend.** `tenants-columns` drift cell (true / false / absent);
`TenantResetButton` disabled-with-tooltip, Cancel-first focus order, error
toast leaves the dialog open, success invalidates; the two hygiene deletions.

---

## 6. Rejected alternatives

**Pin the baseline on the tenant row at clone time.** Rejected in D1 and not
reopened. Design adds one observation: it would also make the *reset* target a
frozen document, so "reset to template" would stop meaning "adopt the current
blueprint" — which is the operation FR-5's motivating story (a newly added
socket handler) actually needs.

**Reuse `templates.Revision` for the tenant comparison by exporting a
section-wise variant from `templates`.** Fewer files, but it makes `tenants`
depend on `templates`' hashing internals, and it puts the shared policy in the
package that has the *other* policy (template-vs-shipped excludes nothing but
id/environment; tenant-vs-template additionally excludes `worlds` and
`diagnostics`). Two exclusion sets in one file is how they get crossed.

**Struct-typed `SectionDrift`.** Better autocomplete, wrong shape: it must be
edited whenever a section is added, which is the FR-2.7 failure re-introduced at
the API layer.

**Field-level diff.** Out of scope per D2, and section granularity is genuinely
the resolution at which the reset decision is made.

**Reset through `UpdateById`.** Would have been three lines. It runs the preset
validator's mutation (§3.6) and would make the post-reset document differ from
the baseline, breaking FR-4.10 on exactly the inputs nobody tests with.

**A bulk `WHERE (region, major, minor) IN (...)` baseline query.** Fewer
round-trips than §3.4's loop, but it bypasses `templates`' overlay/baseline
fallback, which is per-key and not expressible as a single `IN`. Correct
visibility beats one saved query on a page with 1–3 distinct version keys.

---

## 7. Risks

- **`templates.Revision` and `drift.Aggregate` are both "the revision".** Same
  word, different bytes, never comparable (§3.2). Mitigated by comments on both
  sides; the failure mode is a permanently-`true` flag, i.e. NFR-5's exact
  nightmare.
- **Pruning is generous.** `{"a": {}}` and `{}` hash identically. Deliberate, and
  it is the price of killing the nil-vs-empty class generically instead of one
  field at a time. It can only ever produce a false *negative*, never the false
  positive NFR-5 forbids.
- **A template edit makes every derived tenant report drift.** True, accepted in
  D1, restated as NFR-4. The UI must not present it as an error state — the
  templates list precedent (render nothing when not drifted, a quiet badge when
  drifted) is the right register.
- **`properties` is defined by subtraction.** Adding a top-level *section* (an
  object, not a scalar) without adding it to `drift.Named` silently folds it
  into `properties`. That is a safe default — it participates in drift and reset
  — but the section will not get its own flag. The cross-type equality test does
  not catch this; a comment on `Named` will have to carry it.
