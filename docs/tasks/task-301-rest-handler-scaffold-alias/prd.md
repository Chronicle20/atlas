# REST Handler Scaffold Consolidation — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04
---

## 1. Overview

61 services define a `rest/handler.go`. 40 of them reduce that file to a set of
type aliases over `libs/atlas-rest/server` — `type HandlerDependency =
server.HandlerDependency` and siblings — leaving only genuinely
service-specific path-parameter helpers. The other 21 hand-roll the whole
scaffolding: their own `HandlerDependency` struct, their own `HandlerContext`,
their own `GetHandler`/`InputHandler` function types, their own `ParseInput`,
and their own `RegisterHandler`/`RegisterInputHandler` middleware chain.

The correlation with service age is only half the story. Six of the ten oldest
services are in the 21, which is expected drift. But so are
`atlas-portal-actions`, `atlas-reactor-actions`, and `atlas-map-actions`, all
created in 2026-01/02 — long after `libs/atlas-rest/server` existed. New
services are being copied from an old service rather than from the shared
library, so the fleet is still growing this debt while paying it down
elsewhere. `docs/architectural-improvements.md:89` already records the
duplication ("~4,500 lines total across the fleet") and recommends the
extraction at line 93; the library half of that recommendation is already
done, and this task consumes it.

This is not the mechanical alias swap it appears to be, and the difference
matters for both effort and risk. The hand-rolled `HandlerDependency` in 20 of
the 21 carries a third field, `db *gorm.DB`, with a `DB()` accessor that the
shared two-field type does not have; ~174 non-test `d.DB()` call sites depend
on it. And the hand-rolled `RegisterHandler` chains only `RetrieveSpan →
ParseTenant`, omitting the `ParseEnvironment` step that the shared
`server.RegisterHandler` has chained since task-232. Converting these services
therefore changes observable behavior in two specific, intended ways
(FR-4). Both are improvements, but neither is a no-op, so neither may be
described as one.

## 2. Goals

Primary goals:

- Reduce all 21 non-conforming `rest/handler.go` files to the alias form the
  other 40 already use, so a change to the handler contract touches
  `libs/atlas-rest/server` and nothing else.
- Move `*gorm.DB` off `HandlerDependency` and into the handler constructor
  closure, matching the pattern already in `atlas-configurations`,
  `atlas-inventory`, and every other conforming DB-backed service.
- Bring the 21 services onto the shared middleware chain, which means they
  gain the `ParseEnvironment` gate they currently lack.
- Update the scaffolding checklist so the next service created is copied from
  the current best practice rather than from a 2023 service.

Non-goals:

- Any regression guard, lint rule, or `verify.sh` check for this pattern. The
  user explicitly declined a guard; the scaffolding-checklist item (FR-5) is
  the only regression control in scope.
- Adding a `db`-carrying variant to `libs/atlas-rest/server`. The library's
  `HandlerDependency` stays exactly as it is at
  `libs/atlas-rest/server/context.go:14`.
- Any change to the 40 already-conforming services.
- Any change to REST route paths, request shapes, response shapes, or status
  codes beyond the two behavior deltas enumerated in FR-4.
- Any change to `atlas-ui`, to Kafka handlers, or to the processor/entity
  layers of the affected services. Handler signatures change; processor
  signatures do not.
- Aliasing type families other than the `Handler*` scaffolding.

## 3. User Stories

- As a backend engineer changing the REST middleware chain, I want to edit one
  file in `libs/atlas-rest/server` and have all 61 services pick it up, so that
  a contract change is not a 50-file sweep.
- As a backend engineer creating a new service, I want the scaffolding
  checklist to tell me what `rest/handler.go` must look like, so that I do not
  copy a stale handler from whichever service I happened to open.
- As an operator of a sparse ephemeral environment (task-232), I want every
  REST service to honor the `ENVIRONMENT` header, so that a request naming an
  unknown or deactivated environment is rejected at the edge rather than served
  from baseline data.
- As a client of these APIs, I want a malformed JSON:API request body to come
  back with a JSON:API error document rather than an empty 400, so that the
  failure is machine-readable.

## 4. Functional Requirements

### FR-1: The target `rest/handler.go` shape

**FR-1.1** Each converted `rest/handler.go` declares the scaffolding as
aliases, exactly as `services/atlas-guilds/atlas.com/guilds/rest/handler.go`
does today:

```go
type HandlerDependency = server.HandlerDependency
type HandlerContext = server.HandlerContext
type GetHandler = server.GetHandler
type InputHandler[M any] = server.InputHandler[M]

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return server.ParseInput[M](d, c, next)
}

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

**FR-1.2** A service that never calls `ParseInput`, `InputHandler`, or
`RegisterInputHandler` omits those three declarations rather than adding a
dead alias. `atlas-messages` and `atlas-mounts` declare no `InputHandler`
today and must not gain one.

**FR-1.3** Service-specific path-parameter helpers (`ParseCharacterId`,
`ParseNpcId`, `ParseScriptId`, `ParseConversationId`, …) stay in the service's
`rest` package. Where the helper is a plain integer or UUID path segment, it
should delegate to the shared parser — `server.ParseIntId[uint32](l,
"characterId", next)` — as `atlas-guilds` and `atlas-marriages` already do,
rather than keeping a hand-rolled `strconv.Atoi` body. Helpers whose parsing is
not a bare int/UUID keep their existing body unchanged.

**FR-1.4** After conversion, no file under `services/*/rest/handler.go` contains
`type HandlerDependency struct`. The grep
`grep -rl "type HandlerDependency struct" --include=handler.go services` must
return zero files.

**FR-1.5** After conversion, no `services/*/rest/handler.go` imports
`gorm.io/gorm`.

### FR-2: Moving `*gorm.DB` off the dependency

**FR-2.1** Every handler that today reads `d.DB()` is restructured so the
`*gorm.DB` is closed over at construction:

```go
// before
func handleGetFame(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	... NewProcessor(d.Logger(), d.Context(), d.DB()) ...
}

// after
func handleGetFame(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		... NewProcessor(d.Logger(), d.Context(), db) ...
	}
}
```

`services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`
is the reference implementation.

**FR-2.2** `InitResource` keeps its existing
`func(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`
signature. The `db` it already receives is what gets passed to the handler
constructors. No service's `main.go` wiring changes.

**FR-2.3** Registration call sites drop the `(db)` curry level:
`rest.RegisterHandler(l)(db)(si)` becomes `rest.RegisterHandler(l)(si)`, and
`rest.RegisterInputHandler[M](l)(db)(si)` becomes
`rest.RegisterInputHandler[M](l)(si)`.

**FR-2.4** `d.Logger()` and `d.Context()` call sites are unchanged — both
methods exist on the shared type with identical semantics.

**FR-2.5** After conversion, `grep -rn "d\.DB()" --include='*.go' services`
returns zero matches. (`db.DB()` on a `*gorm.DB` — used in test helpers to
reach the underlying `sql.DB` — is a different expression and is not affected.)

### FR-3: Dead handler packages

**FR-3.1** `services/atlas-fame/atlas.com/fame/rest/handler.go` has zero
consumers: no file in `atlas-fame` imports the `rest` package, and the service
defines no `InitResource`. Delete the file (and the `rest` package directory if
it becomes empty) rather than converting it.

**FR-3.2** `services/atlas-events/atlas.com/events/rest/handler.go` likewise
has zero consumers — `atlas-events` already calls `server.RegisterHandler(l)(si)`
directly at
`services/atlas-events/atlas.com/events/event/definition/resource.go:32` and
`.../occurrence/resource.go:30`. Delete it on the same grounds.

**FR-3.3** Deletion must be justified by a fresh grep at implementation time,
not by this document alone. If either service has gained a consumer since this
PRD was written, convert it per FR-1 instead of deleting it.

### FR-4: Intended behavior changes

These two deltas are consequences of adopting the shared chain. They are
intended and must be stated in the PR description; they are the reason this
task is not purely mechanical.

**FR-4.1 — `ParseEnvironment` is added.** The hand-rolled `RegisterHandler` in
all 21 services chains `RetrieveSpan → ParseTenant`. The shared
`server.RegisterHandler` (`libs/atlas-rest/server/register.go:11`) chains
`RetrieveSpan → ParseEnvironment → ParseTenant`. After conversion these
services will, for the first time: reject with 400 a request whose
`ENVIRONMENT` header names an environment the registry does not know or knows
as DEACTIVATING/DELETED; add an `environment` field to the request log line;
and run `env.Reconcile` inside `ParseTenant`, which 400s when the header
disagrees with the tenant's environment. A request with no `ENVIRONMENT`
header is unaffected — that is the legacy path and passes through unchanged
(see the `ParseEnvironment` doc comment in
`libs/atlas-rest/server/handler.go`).

**FR-4.2 — Malformed-body 400s gain a response body.** The hand-rolled
`ParseInput` writes a bare `w.WriteHeader(http.StatusBadRequest)` on an
unreadable body. The shared `server.ParseInput`
(`libs/atlas-rest/server/context.go:47`) calls `writeBadRequestJSONAPI`, which
emits a JSON:API error document. Status code is unchanged at 400; the body
goes from empty to populated.

**FR-4.3** No other behavior change is acceptable. In particular: the tracing
span name and the `{"originator": ..., "type": "rest_handler"}` log fields are
identical in both chains and must stay identical.

**FR-4.4** `server.RegisterHandler` is the correct target for all 21, not
`server.RegisterSimpleHandler`. The `Simple*` variants omit `ParseTenant`, and
every one of the 21 currently parses the tenant.

### FR-5: Scaffold documentation

**FR-5.1** `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md`
gains a checklist item covering `rest/handler.go`. The file currently has no
handler item at all — `SCAFFOLD-04` is about `deploy/shared/routes.conf`, not
the handler — so this is a new entry, not an edit. The item must be written in
the same verifiable `| ID | command | expectation |` form as its neighbours,
and its command must be a grep an implementer can actually run, e.g. asserting
that `rest/handler.go` contains `= server.HandlerDependency` and does not
contain `type HandlerDependency struct`.

**FR-5.2** The new item states the `*gorm.DB` rule explicitly: a DB-backed
service closes the `*gorm.DB` over the handler constructor and does not put it
on the dependency.

**FR-5.3** `docs/adding-a-new-service.md` points at the checklist for
code-level scaffolding at line 18 and needs no structural change; verify that
pointer still resolves and leave it otherwise alone.

**FR-5.4** `docs/architectural-improvements.md` lines 87–93 are updated to
record that the per-service handler duplication is resolved, in the same
"**Done so far:**" style the document already uses for task-208 at line 53.
Do not delete the entry; amend it.

## 5. API Surface

No endpoint is added, removed, or re-pathed. No request or response schema
changes. The only externally visible deltas are the two in FR-4.1 and FR-4.2.

## 6. Data Model

No entity, column, index, or migration changes. `tenant_id` scoping is
unaffected — the tenant still reaches the processor via
`tenant.FromContext(d.Context())`, and `ParseTenant` remains in the chain.

## 7. Service Impact

19 services are converted; 2 have a dead handler package deleted. Counts are
from grep over `main` at 31a791e3a and must be re-derived at implementation
time rather than trusted from this table.

| Service | `rest.Register*` sites | `rest.HandlerDependency` refs | `d.DB()` | `rest.Parse*` | Action |
|---|---|---|---|---|---|
| atlas-character | 16 | 28 | 26 | 21 | convert |
| atlas-mts | 12 | 24 | 18 | 14 | convert |
| atlas-npc-conversations | 7 | 21 | 20 | 13 | convert |
| atlas-reward-pools | 7 | 15 | 16 | 11 | convert |
| atlas-npc-shops | 11 | 11 | 12 | 10 | convert |
| atlas-ban | 5 | 11 | 15 | 6 | convert |
| atlas-party-quests | 3 | 12 | 12 | 8 | convert |
| atlas-account | 5 | 11 | 9 | 8 | convert |
| atlas-quest | 4 | 9 | 3 | 15 | convert |
| atlas-notes | 2 | 7 | 7 | 5 | convert |
| atlas-map-actions | 2 | 6 | 6 | 4 | convert |
| atlas-parcel | 3 | 6 | 6 | 4 | convert |
| atlas-portal-actions | 2 | 6 | 6 | 4 | convert |
| atlas-reactor-actions | 2 | 6 | 6 | 4 | convert |
| atlas-drop-information | 3 | 4 | 4 | 2 | convert |
| atlas-pets | 4 | 4 | 4 | 3 | convert |
| atlas-rankings | 1 | 4 | 3 | 1 | convert |
| atlas-mounts | 1 | 1 | 1 | 1 | convert (no `InputHandler`) |
| atlas-messages | 1 | 1 | 0 | 0 | convert (no DB, no `InputHandler`) |
| atlas-events | 0 | 0 | 0 | 0 | **delete** handler.go (FR-3.2) |
| atlas-fame | 0 | 0 | 0 | 0 | **delete** handler.go (FR-3.1) |

Also touched: `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md`,
`docs/architectural-improvements.md`. `libs/atlas-rest/` is **not** touched.

## 8. Non-Functional Requirements

- **Performance.** Neutral. `ParseEnvironment` adds one header read and one
  registry lookup per request; the registry is in-process.
- **Multi-tenancy.** Unchanged, and strengthened: `ParseTenant` stays, and
  `env.Reconcile` now runs for these 21 services, so a tenant/environment
  mismatch is caught rather than silently served.
- **Observability.** Span name and existing log fields must be byte-identical
  (FR-4.3). The `environment` log field is added when the header is present.
- **Security.** No auth surface changes. FR-4.1 narrows what is accepted; it
  never widens it.
- **Reviewability.** The conversion is uniform across 19 services. Each service
  should be an independently compiling commit so a bisect lands on one service.

## 9. Open Questions

1. `atlas-quest` shows 15 `rest.Parse*` references against only 3 `d.DB()` — a
   ratio unlike its peers. Design should confirm its handlers are structured
   the same way before assuming the FR-2.1 transform applies unchanged.
2. FR-1.3 (delegating simple id parsers to `server.ParseIntId`) is a
   nice-to-have that expands the diff. If it proves noisy in review it may be
   dropped to a follow-up without affecting FR-1.4 or FR-1.5.
3. Whether the 19 conversions land as 19 commits on one branch or as a smaller
   number of grouped commits is a planning decision, not a spec decision.

## 10. Acceptance Criteria

- [ ] `grep -rl "type HandlerDependency struct" --include=handler.go services`
      returns nothing.
- [ ] `grep -rl "= server.HandlerDependency" --include=handler.go services`
      returns 59 files (61 today minus the 2 deleted).
- [ ] `grep -rn "d\.DB()" --include='*.go' services` returns nothing.
- [ ] No `services/*/rest/handler.go` imports `gorm.io/gorm`.
- [ ] `services/atlas-fame/atlas.com/fame/rest/handler.go` and
      `services/atlas-events/atlas.com/events/rest/handler.go` are deleted, and
      a fresh grep at implementation time confirmed neither had consumers.
- [ ] Every converted service's `RegisterHandler` resolves to
      `server.RegisterHandler` (not `RegisterSimpleHandler`).
- [ ] No service `main.go` changed.
- [ ] No file under `libs/atlas-rest/` changed.
- [ ] `scaffolding-checklist.md` has a new, runnable `rest/handler.go` item
      covering both the alias form and the `*gorm.DB` rule.
- [ ] `docs/architectural-improvements.md` lines 87–93 amended to record
      completion.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Existing tests pass unchanged; no new tests are required (behavior is
      preserved apart from FR-4.1/FR-4.2, neither of which any current test
      asserts against — confirm that claim by running the suite, do not assume
      it).
- [ ] Code review run before the PR, per `docs/review-protocol.md`.
- [ ] The PR description states FR-4.1 and FR-4.2 as intended behavior changes.
