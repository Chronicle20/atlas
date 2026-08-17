# Backend Audit — task-232-sparse-ephemeral-environments (Phases D/E/F, Go changes only)

- **Reviewed range:** `git diff 60631ae5e..HEAD -- '*.go'` (Task 41 onward). Earlier
  Tasks 1-40 were reviewed separately in `audit-backend-core.md`,
  `audit-backend-libs-tools.md`, `audit-backend-services.md` — not re-reviewed here.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-16
- **Build:** PASS (`services/atlas-configurations/atlas.com/configurations`: `go build ./...` clean)
- **Tests:** `go test ./environments/... ./services/...` — all packages PASS (no failures)
- **Overall:** NEEDS-WORK

## Scope note

The diff touches ~1,180 `.go` files, but it is overwhelmingly one mechanical,
uniform change repeated per-service: every `requests.go`'s `getBaseRequest()`
was migrated from bare `requests.RootUrl(domain)` to context-aware
`requests.RootUrlFor(ctx, domain)` (environment-scoped ingress routing), with
every caller in the corresponding `processor.go` threading `ctx` through and
checking the returned error. I verified this mechanically across the full
diff (not sampled) rather than reading all ~560 request/processor files
individually — see "Mechanical sweep results" below — then did close reads of
the genuinely new business logic the user's brief pointed at:
`services/atlas-configurations/atlas.com/configurations/services/` (Task 48,
`Environment` field) and `.../environments/` (the new environments resource).

## Mechanical sweep results (the ~560 requests.go/processor.go pair)

| Check | Command | Result |
|---|---|---|
| Every changed `requests.go` moved off bare `RootUrl` | `git grep -n "requests.RootUrl(" -- '*requests.go' \| grep -v RootUrlFor` | 0 hits — clean |
| No error silently discarded from `getBaseRequest(ctx)` | `git diff ... \| grep ", _ := getBaseRequest\|_, _ = getBaseRequest"` | 0 hits |
| No leftover bare `getBaseRequest()` call sites in changed `processor.go` files | grep across all changed processor.go | 0 hits |
| New bare `go` statements in changed non-test files | manual grep `^\s*go (func\|[A-Za-z_])` across 1,010 changed non-test files | 0 hits (DOM-26 PASS) |

`libs/atlas-rest/requests/url.go`'s `RootUrlFor` (lines 34-65) fails closed —
an unresolvable environment returns an error, never falls back to the
baseline namespace (FR-3.5/G4, confirmed by the code comment and by
`env.CurrentRegistry().EnvironmentNamespace` error propagation at line 48-51).
This is a library change, not scoped to a single domain checklist item, but
it's the mechanism every one of the ~560 call-site changes depends on being
correct, so I read it directly rather than trusting the call sites alone.

## Domain Checklist Results

### `services/atlas-configurations/atlas.com/configurations/environments` (new package, Task 18/19)

No `model.go` — `RestModel` doubles as the domain model (comment at
`environments/rest.go:3-8` states this explicitly: field names mirror
`libs/atlas-env.Record` exactly). Classified as a support/domain-adjacent
package; full DOM-01/02/03/04/05 (which gate on `model.go` existing) do not
strictly apply, so those are not scored FAIL — see file-responsibilities
placement checks instead.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-04 | Entity + Migration + TableName in `entity.go` | PASS | `environments/entity.go:17-30` |
| FILE-05 | Builder in `builder.go`; write funcs in `administrator.go`; providers in `provider.go` | PASS | `environments/builder.go`, `environments/administrator.go:13-48`, `environments/provider.go:14-32` |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `environments/processor.go:145` `func NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `environments/resource.go:54,72,90,115` all call `NewProcessor(d.Logger(), ...)` |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `environments/resource.go:27,29` |
| DOM-09 | Transform errors handled | N/A | package has no `Transform`/`TransformSlice` — `RestModel` is returned directly |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `environments/resource.go` — zero matches; `os.Getenv` only appears in `processor.go:109` (topic lookup, not a handler) |
| DOM-14/DOM-15 | Handlers don't call providers/DB directly | PASS | `environments/resource.go` calls only `NewProcessor(...).X` |
| DOM-17 | Domain error → HTTP status mapping | PASS | `environments/resource.go:92-98,117-123` map `isValidationError` (line 40-43) to 400 via `server.WriteBadRequest`; other errors go through `server.WriteErrorResponse` (transient/500-family) |
| **DOM-19 (RestModel/PATCH shape)** | **Non-pointer fields on a RestModel that is also the PATCH input, without a merge-with-existing step** | **FAIL — Critical** | See "Critical Finding 1" below |

### `services/atlas-configurations/atlas.com/configurations/services` and `services/service` (Task 48 — new `Environment` field)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Consistency with the 3 sibling resources the field was modeled on | `Environment string` field shape matches `tenants/rest.go:28` and `templates/rest.go:28` exactly | PASS | `services/service/rest.go:20,66,94` all declare `Environment string \`json:"environment"\`` — same non-pointer/no-`omitempty` shape as `tenants` and `templates` |
| Environment is server-owned, never read from client input on write | `Make()` (read path) always overwrites from `e.Environment` after unmarshal; `Create`/`UpdateById` (write path) take `service.InputRestModel`, which has **no** `Environment` field at all | PASS | `services/processor.go:117-149` (`Make`, three branches all do `rm.Environment = e.Environment`); `services/service/rest.go:39-44` (`InputRestModel` struct has no `Environment` field, so it cannot be set from a client body even accidentally) |
| DOM-18 JSON:API interface | `GetName()/GetID()/SetID()` present on all three new RestModel variants | PASS | `services/service/rest.go:23-34,46-57,69-80,101-108` |
| Write path escapes the "any other caller with this shape" risk the brief flagged | `UpdateById` (services) takes `InputRestModel`, a **separate type** from the read-side `RestModel`/`GenericRestModel`/`LoginRestModel`/`ChannelRestModel` that carry `Environment` — so this package's write path was never exposed to the zeroing bug the environments package has | PASS (by construction, not by an explicit merge step) | `services/processor.go:174-200` (`UpdateById` signature takes `service.InputRestModel`) |

DOM-21 (atlas-constants duplication): no new numeric/id classification type
was added in either package; `Environment string` is a plain string field,
not a new domain type. PASS — no grep hit for anything overlapping
`libs/atlas-constants/` in this diff's scope.

### `libs/atlas-env` (new library, Task 18)

Not a domain package (no REST surface) — File Responsibilities checklist
doesn't apply in the DOM sense, but I checked DOM-21 directly since it
defines a new `Id` type:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | `env.Id` doesn't duplicate an existing atlas-constants type | PASS | `grep -il environment libs/atlas-constants/README.md` — no hits. `env.Id` (a deployment/environment identifier: `"main"`, `"pr-123"`) is not an id-space atlas-constants indexes (item/world/channel/character/job/skill/monster) |

## Critical Finding 1 — `environments.RestModel`'s non-pointer fields cause a partial PATCH to silently zero columns

**File:** `services/atlas-configurations/atlas.com/configurations/environments/rest.go:9-17`
**File:** `services/atlas-configurations/atlas.com/configurations/environments/processor.go:221-254` (`UpdateByName`)

`RestModel` is used as both the GET response shape and the PATCH input type
(`rest.RegisterInputHandler[RestModel]` at `resource.go:29`). Every field
except `Id` is a plain non-pointer type with no `omitempty`:

```go
type RestModel struct {
	Id        string            `json:"-"`
	Name      string            `json:"name"`
	Baseline  string            `json:"baseline"`
	Namespace string            `json:"namespace"`
	Tenant    string            `json:"tenant"`
	Overrides map[string]string `json:"overrides"`
	Phase     string            `json:"phase"`
}
```

This directly violates the documented guideline `ai-guidance.md:208` /
`patterns-rest-jsonapi.md:228` — "Use pointer fields for optional attributes
with `omitempty`."

`UpdateByName` (processor.go:221-254) fetches the `existing` record only to
validate the phase transition (line 228-234) — it never merges `existing`'s
`Baseline`/`Namespace`/`Tenant`/`Overrides` back into `input` before writing:

```go
func (p *ProcessorImpl) UpdateByName(name string, input RestModel) (RestModel, error) {
	...
	existing, err := p.GetByName(name)          // fetched...
	...
	if err := validatePhaseTransition(existing.Phase, input.Phase); err != nil {
		return RestModel{}, err
	}
	input.Name = name
	...
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := update(p.ctx, name, input.Baseline, input.Namespace, input.Tenant, overrides, input.Phase)(db); err != nil {
		//                       ^^^^^^^^^^^^^^ never merged with `existing` — only `existing` ...used for Phase check
```

Consequence: a client that sends `PATCH /configurations/environments/pr-123`
with body `{"data":{"attributes":{"phase":"ACTIVE"}}}` — a legitimate,
minimal lifecycle-transition PATCH, exactly what a phase-only client would
send — has `Baseline`/`Namespace`/`Tenant`/`Overrides` decode to their Go
zero values (`""`, `""`, `""`, `nil`) because the JSON body omitted them, and
those zero values get written straight to the database, overwriting the
real values.

This is not a hypothetical: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
carries an explicit code comment describing exactly this bug and working
around it client-side —

```
# services/atlas-pr-bootstrap/scripts/cleanup.sh:141-149
#  — PATCHes the environments record to <new_phase>, carrying the OTHER four
# attributes through unchanged. environments.RestModel's fields are
# non-pointer (environments/rest.go), so ParseInput unmarshals a PATCH body
# ... is zeroed, not left alone (environments/administrator.go's update() doc
# ... round 2 Critical 2). GET-then-PATCH-with-everything is the fix.
```

That comment confirms the bug was found once (fix round 2, Critical 2) and
patched **in the one known shell-script caller** by having the caller
GET-then-PATCH-with-everything. The Go backend itself was never fixed. Any
other caller of this PATCH endpoint — a future admin UI, an operator running
`curl` by hand, a different automation script — that sends a partial PATCH
body will silently zero `Baseline`/`Namespace`/`Tenant`/`Overrides` on a live
environment record. Given this record drives `RootUrlFor`'s namespace
resolution (`libs/atlas-rest/requests/url.go:48`) for every environment-scoped
inter-service call, a zeroed `Namespace` would break inter-service routing
for that entire PR environment.

No test in `environments/processor_test.go` exercises this path: every
`UpdateByName` call in the test file (lines 278-283, 320-325, 336-341,
352-357, 371-376) builds the input via `NewBuilder()...Build()` with every
field explicitly set, never a genuinely partial input. The HTTP layer
(`resource.go`'s `RegisterInputHandler[RestModel]` decode-from-JSON path) is
never exercised by a test that omits a field.

**Fix options** (not prescribing one, per audit scope): either (a) make
`RestModel`'s fields pointers with `omitempty` and have `UpdateByName` apply
only the non-nil ones on top of `existing` (matching the documented pattern),
or (b) have `UpdateByName` explicitly merge `input` onto `existing` for every
field before calling `update()`, mirroring what `cleanup.sh` now does at the
shell layer. Either closes the gap for every caller, not just the one that
happened to get patched.

## Sub-Domain Checklist Results

No sub-domain (action-event, no-`model.go`-no-`processor.go`) packages were
newly introduced in this diff range.

## Security Review

Not an auth-related service; SEC-* checks not applicable to this scope.

## Not evaluable from the diff

- **DOM-24 (Kafka producer stubbed in tests)** — I did not walk the full
  transitive call graph from every changed `*_test.go` file across all ~90
  affected services to check for un-stubbed emit paths; the diff's Kafka
  surface here is limited to header/gate plumbing in `libs/atlas-kafka` and
  `libs/atlas-outbox`, which already have their own `*_test.go` files in the
  diff, but I did not verify every downstream service's test package still
  has its producer stub wired after this header change. Would need a
  targeted grep across all changed `kafka/consumer/*/consumer_test.go` files,
  which is outside this diff (none were changed) — the risk is low since no
  consumer test files appear in the changed-file list.
- **EXT-01/EXT-02 (JSON:API relationship interfaces, httptest fixture)** — no
  new external HTTP client package was introduced in this diff range (only
  existing `requests.go` files were mechanically retrofitted with
  `RootUrlFor`); would need to confirm no *new* target struct was added
  anywhere in the ~560 changed files, which I did not enumerate file-by-file.
- **DOM-25 (client-interpreted byte values)** — no `atlas-channel`
  socket/dispatcher files appear in this diff range; not evaluated.
- **DOM-28 (decorator degradation)** — no `model.Decorator[...]`
  implementations appear among the files I read closely; not swept across
  the full 560-file mechanical change.

## Summary

### Blocking (must fix)

- **Critical** — `environments.RestModel`'s non-pointer PATCH-input fields
  cause `UpdateByName` to silently zero `Baseline`/`Namespace`/`Tenant`/
  `Overrides` on any partial PATCH body that isn't the one caller
  (`cleanup.sh`) that was patched to work around it.
  `services/atlas-configurations/atlas.com/configurations/environments/rest.go:9-17`,
  `services/atlas-configurations/atlas.com/configurations/environments/processor.go:221-254`.

### Non-Blocking (should fix)

- None identified in the reviewed surface.
