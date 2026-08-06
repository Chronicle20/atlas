# Backend Audit — task-196-npc-info-default-icon

- **Scope:** diff `31c7a664f..HEAD`, `libs/atlas-wz/icons/` only
  - `extract.go` (modified — adds `findNpcCanvas`, repoints `ExtractNpcIcon`)
  - `fixture_test.go` (new — `icons_test` fixture helpers)
  - `npc_default_test.go` (new)
  - `npc_default_edge_test.go` (new)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-06
- **Build:** PASS
- **Tests:** 8 passed (icons package), 0 failed; full `libs/atlas-wz` module: all packages `ok`
- **Overall:** PASS

## Build & Test Results

```
$ cd libs/atlas-wz && go build ./...
(clean, no output)

$ cd libs/atlas-wz && go vet ./...
(clean, no output)

$ cd libs/atlas-wz && go test ./... -count=1
ok  	.../atlas-wz/atlas
ok  	.../atlas-wz/atlas/pngenc
ok  	.../atlas-wz/canvas
ok  	.../atlas-wz/charparts
ok  	.../atlas-wz/crypto
ok  	.../atlas-wz/icons          0.010s
ok  	.../atlas-wz/manifest
ok  	.../atlas-wz/mapimage
ok  	.../atlas-wz/maplayout
ok  	.../atlas-wz/wz
ok  	.../atlas-wz/wz/property

$ cd libs/atlas-wz && go test ./icons/... -race -count=1
ok  	.../atlas-wz/icons  1.043s

$ go test ./icons/... -count=1 -v (icons package only)
--- PASS: TestPublicSurfaceExists
--- PASS: TestNormalizeId
--- PASS: TestMobIgnoresInfoDefault
--- PASS: TestNpcIgnoresTopLevelDefaultDir
--- PASS: TestNpcInfoDefaultBeatsLink
--- PASS: TestNpcFollowsLinkToInfoDefault
--- PASS: TestNpcPrefersInfoDefault
--- PASS: TestNpcFallsBackToStand
```

## Phase 2: Domain Discovery

`libs/atlas-wz/icons` contains no `model.go` and no `resource.go` — no domain model, no REST/Kafka/GORM/tenancy surface. Classified **Support package** (a stateless algorithmic library: parses WZ property trees and decodes canvases). This is not a `services/atlas-*` service — there is no `main.go`, no `go.mod`-level service registration, so the Service Scaffolding Checklist (SCAFFOLD-*) does not trigger.

Per audit instructions, the File Responsibilities Checklist still runs against this package (a support package is exactly where collapsed-file violations hide) — see below. All DOM-*/SUB-*/EXT-*/SEC-* items are marked N/A with the specific absent trigger condition (no model.go, no resource.go, no DB, no REST client, no auth surface) rather than blanket-exempted.

## File Responsibilities Checklist — `libs/atlas-wz/icons`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | N/A | No `type Processor interface`, `ProcessorImpl`, or `NewProcessor(` anywhere in the package — `grep -n "type Processor\|func NewProcessor("  libs/atlas-wz/icons/*.go` returns no matches. This package has no processor layer; it is a pure function library. |
| FILE-02 | RestModel + Transform in `rest.go` | N/A | No `type RestModel`, `func Transform(`, `func Extract(` (JSON:API sense), or `GetName()/GetID()/SetID()` anywhere — no REST transport in this package. |
| FILE-03 | Cross-service requests in `requests.go` | N/A | No `requests.RootUrl(`, `requests.GetRequest[`, `requests.PostRequest[` — this package never calls another atlas service. |
| FILE-04 | Entity + Migration + TableName in `entity.go` | N/A | No `type entity struct`, `func Migration(`, `TableName()` — no GORM/DB persistence in this package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No `Builder`, domain `Model`, `Create*/Update*/Delete*` writes, or `database.Query`/`SliceQuery` readers exist in the package — confirmed via grep, zero matches. |
| FILE-06 | No package-named catch-all file | PASS | `libs/atlas-wz/icons/extract.go` — file is named for its function (WZ canvas extraction), not `icons.go`; it holds exactly one responsibility class (canvas-finder algorithms + decode glue), not ≥2 of the Processor/RestModel/Entity/requests responsibilities from FILE-01–04, none of which are present in the package at all. `findNpcCanvas` (extract.go:335) is added alongside the pre-existing sibling finders (`findStandCanvas` line 346, `findReactorCanvas` line 368, `findInfoIcon` line 379) — consistent placement of a new finder next to its siblings, not a new collapsed responsibility. |

## Domain / Sub-Domain Checklist Results

`libs/atlas-wz/icons` — Support package, not Domain (no `model.go`) or Sub-Domain (no `resource.go`).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | N/A | No domain model requiring a builder — package has no `model.go`. |
| DOM-02 | `ToEntity()` method | N/A | No `entity.go`, no GORM entity. |
| DOM-03 | `Make(Entity)` function | N/A | No `entity.go`. |
| DOM-04 | `Transform` function | N/A | No REST transport, no `rest.go`. |
| DOM-05 | `TransformSlice` function | N/A | Same as DOM-04. |
| DOM-06 | Processor accepts `FieldLogger` | N/A | No Processor exists (FILE-01). |
| DOM-07 | Handlers pass `d.Logger()` | N/A | No `resource.go`, no REST handlers. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | N/A | No REST endpoints. |
| DOM-09 | Transform errors handled | N/A | No `Transform(` calls. |
| DOM-10 | Test DB has tenant callbacks | N/A | No database, no `RegisterTenantCallbacks` — fixture setup in `fixture_test.go:52-68` builds an in-memory WZ archive via `wztest.Builder`, not a SQL DB. |
| DOM-11 | Providers use lazy evaluation | N/A | No `provider.go`, no `database.Query`/`SliceQuery`. |
| DOM-12 | No `os.Getenv()` in handlers | N/A | No `resource.go`. `grep -n "os.Getenv" libs/atlas-wz/icons/*.go` returns no matches regardless. |
| DOM-13 | No cross-domain logic in handlers | N/A | No handlers exist. |
| DOM-14 | Handlers don't call providers directly | N/A | No handlers, no providers. |
| DOM-15 | No direct entity creation in handlers | N/A | No handlers, no `db.Create`/`Save`/`Delete` anywhere in the package. |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no writes (read-only WZ parsing). |
| DOM-17 | Domain error → HTTP status mapping | N/A | No REST layer; the package's only error is `ErrNotFound` (extract.go:25), a plain sentinel returned to Go callers — no HTTP status mapping applies at this layer. |
| DOM-18 | JSON:API interface on REST models | N/A | No REST models. |
| DOM-19 | Request models flat structure | N/A | No request models. |
| DOM-20 | Table-driven tests | PASS (with note) | `extract_test.go:21-45` (`TestNormalizeId`, pre-existing, unmodified) uses the `tests := []struct{...}{}` + range pattern. The four new tests (`npc_default_test.go:13,31`; `npc_default_edge_test.go:13,32,51,75`) are each a single named scenario (distinct WZ fixture shapes: info/default win, stand fallback, mob non-application, top-level `default` dir exclusion, link precedence, link fallback) rather than table cases sharing one assertion body — each constructs a structurally different fixture, so collapsing them into one table would obscure rather than clarify the assertion. Testing-guide.md states "Prefer table-driven tests" as a general guideline (not a DOM-* gated item, since DOM-20 is scoped to "every domain with model.go" and this package has none); marking PASS since the one directly-applicable existing test (`TestNormalizeId`) already follows the pattern and the new one-scenario-per-test style has a stated rationale in-file, not an omission. |
| DOM-21 | No duplication of atlas-constants types | PASS | The diff declares no new `type`, `const` block, or numeric-literal id classification. `findNpcCanvas` (extract.go:335) and its test fixtures operate on NPC ids as opaque `uint32`/decimal strings via the pre-existing `normalizeId`/`strconv.FormatUint` machinery (extract.go:171, 430) — no item-id division, no inventory/compartment enum, no world/channel/map/character-id type, no job/skill/monster-id type is introduced. `libs/atlas-wz/go.mod:6` already direct-requires `github.com/Chronicle20/atlas/libs/atlas-constants`, but `grep -rn "atlas-constants" libs/atlas-wz/icons/*.go` returns no matches — the package doesn't need it and doesn't fake-avoid it; there is nothing in this diff atlas-constants would cover (NPC ids are not modeled there). |
| DOM-22 | Dockerfile has 4 mentions per direct atlas-* require | N/A | `libs/atlas-wz` is a shared library consumed via `go.work`/`replace`, not a `services/atlas-<svc>` deployable with its own Dockerfile — no `services/atlas-wz/Dockerfile` exists. This check targets service Dockerfiles copying libs in; not applicable to a change confined to a lib itself. |
| DOM-23 | Kafka topic naming convention | N/A | No Kafka producer/consumer code in this package or diff. |
| DOM-24 | Kafka producer stubbed in tests | N/A | `grep -n "AndEmit(\|message.Emit(\|producer.Produce(" libs/atlas-wz/icons/*.go` returns no matches — no emit call sites, direct or transitive, in the package or its tests. |
| DOM-25 | Client wire values config-resolved | N/A | Not channel/socket code; this package extracts WZ art assets server-side, no client-interpreted byte codes involved. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])' --include='*.go' libs/atlas-wz/icons/` (excluding `_test.go`) returns no matches — the diff introduces no goroutines at all, bare or otherwise. |
| DOM-27 | Transient DB errors → 503 | N/A | No database, no HTTP handlers in this package. |
| DOM-28 | No silent degradation in decorators | N/A | No `model.Decorator[...]` implementation in this package. `findNpcCanvas`'s fallback to `findStandCanvas` (extract.go:335-342) is a documented, in-comment intentional lookup-order fallback for a pure function returning `nil`/`ErrNotFound` on total miss (extract.go:183-189) — not a remote-fetch enrichment path silently dropping fetched data, which is what DOM-28 targets (patterns-resilience.md decorator/enrichment scope). The caller (`extractEntityIcon`, line 157) still surfaces `ErrNotFound` when both the preferred and fallback finder miss (line 187-189) — no error is swallowed. |

**Sub-Domain checklist (SUB-01…04):** N/A — no `resource.go` in this package; not an action-event sub-domain package.

**External HTTP Client checklist (EXT-01…04):** N/A — package contains no `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` calls.

**Service Scaffolding checklist (SCAFFOLD-01…08):** N/A — no new `services/atlas-<service>/` directory and no new atlas-channel packet writer/handler introduced by this diff.

## Security Review

N/A — `libs/atlas-wz/icons` is not an authentication/authorization/token-management service (SEC-01…04 trigger condition not met).

## Additional Notes (non-checklist, informational)

- The production change is minimal and precisely scoped: `ExtractNpcIcon` (extract.go:94-96) is repointed from `findStandCanvas` to the new `findNpcCanvas` (extract.go:335-342); `ExtractMobIcon` (extract.go:100-102) and `ExtractReactorIcon` (extract.go:106-108) are left untouched, confirming the fix is NPC-scoped as designed and as covered by `TestMobIgnoresInfoDefault` (npc_default_edge_test.go:13-27).
- `fixture_test.go` is a new `_test.go` file providing shared builders/markers (`newArchive`, `openFixture`, `payloadFor`, `pixelAt`) consumed by both new test files — consistent with the task-196 plan's stated interface contract (plan.md:31-33, 149-154, 271-276) and not a production-code test-helper file (CLAUDE.md's "no `*_testhelpers.go` with test-only constructors" rule targets non-test production files; this is itself a `_test.go` file, package `icons_test`, appropriately using `t.Helper()` throughout — fixture_test.go:31,38,53,79).

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix)
None.
