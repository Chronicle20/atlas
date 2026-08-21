# Backend Audit — task-250-inner-portal-registration

- **Service Path:** services/atlas-channel, services/atlas-character, libs/atlas-packet, tools/packet-audit
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** All packages report `ok` (0 failures) across all four modules
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:  go build ./... -> clean; go test ./... -count=1 -> all ok
services/atlas-character/atlas.com/character: go build ./... -> clean; go test ./... -count=1 -> all ok
  (atlas-character/pending_change 301.889s — pre-existing slow package, no failures)
libs/atlas-packet: go build ./... -> clean; go test ./... -count=1 -> all ok
tools/packet-audit: go build ./... -> clean; go test ./... -count=1 -> all ok
```

## Applicability

| Family | Status | Trigger evidence |
|---|---|---|
| FILE placement | Fired | Every changed Go package audited (support and domain-shaped alike) |
| DOM structure (model.go/entity.go/rest.go/provider.go) | Fired (narrow) | No changed package has `model.go`+`entity.go`, or `rest.go`, except `data/portal/rest.go` (unchanged file, read for EXT-01 contract check) |
| REST (DOM-06..09,12..19,32) | Fired (DOM-06 only) | `processor.go` changed in `portal`, `movement`, `data/portal`; no `resource.go`/route registration in scope |
| SUB (action-event) | N/A | No changed package has `resource.go` with no `model.go` |
| Constants reuse (DOM-21) | Fired, no violation | New `position.Position{X,Y int16}`, `maxPortalEntryDistance` constant — no matching classification in `libs/atlas-constants/` |
| Testing (DOM-10,20,24,33) | Fired | `_test.go` files added across channel/character; `Processor`/mock interfaces changed |
| Cache (DOM-29) | Fired | `data/portal/processor.go` holds package-level `sync.Map` cache state; `position/registry.go` is a singleton registry |
| Messaging (DOM-30) | Fired, no violation | `movement/processor.go` `TeleportCharacter` calls `producer.ProviderImpl` directly — documented non-DB-state exception |
| Multi-tenancy (DOM-31) | Fired, no violation | Tenant travels via `tenant.MustFromContext`/`tenant.Model` params only, never serialized |
| Migration hygiene (DOM-34/35) | N/A | No symbols moved between a service and a `libs/atlas-*` module — `inner_portal.go` is net-new |
| Deploy & topics (DOM-22/23) | N/A | No new `libs/atlas-*` module; no new/renamed Kafka topic env var (`TeleportCharacter` reuses `EnvCommandCharacterMovement`) |
| Runtime safety (DOM-26) | Fired, no violation | `movement/processor.go:94` uses `routine.Go(p.l, p.ctx, ...)`; no bare `go` statements added |
| Channel wire values (DOM-25) | Fired, N/A within | `libs/atlas-packet`/`services/atlas-channel` touched, but no dispatcher-mode/sub-op/fail-reason byte is introduced — plain field decode |
| Resilience (DOM-27/28) | N/A | No handler writes `http.StatusInternalServerError`; no `model.Decorator` changed |
| External clients (EXT-01..04) | Fired | `data/portal/processor.go` calls `requests.SliceProvider`/`requestInMapFn` for the DATA service |
| Scaffolding (SCAFFOLD-01..09) | Fired (SCAFFOLD-07 only) | New channel `Handler` (`InnerPortalHandle`) registered — `.github` service scaffolding items N/A (no new service) |
| Security (SEC-01..04) | N/A | No token/auth/redirect/secret handling in the diff |

## Checklist Results

### libs/atlas-packet/portal/serverbound (support — codec package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted wire bytes resolved via tenant writer-options table | N/A | No dispatcher-mode/sub-op/fail-reason byte in `InnerPortal` — plain string/int16 fields; `encodesFieldKey` gates a *field presence*, not a client lookup code (`inner_portal.go:57-63`) |
| DOM-20 | Table-driven tests | PASS | `TestInnerPortalGoldenBytes` uses `cases := []struct{...}` + `t.Run` (`inner_portal_test.go:21-33,54-59`); `TestInnerPortalRoundTrip` iterates `pt.Variants` |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `inner_portal.go` is a single-purpose codec file (Model+Encode+Decode is the documented packet-codec shape, not a DOM package) |

### services/atlas-channel/channel/portal (domain-ish processor package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:54` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| FILE-01 | Processor interface/impl live in `processor.go` | PASS | `Processor` interface, `ProcessorImpl`, `EnterInner` all in `processor.go` |
| DOM-33 | Mock updated for interface change | PASS | `portal/mock/processor.go:12` adds `EnterInnerFunc`; `:39-45` adds `EnterInner` method with nil-check default |
| DOM-20 | Table-driven tests | PASS | `TestEnterInner` — `tests := []struct{...}` + `t.Run` (`processor_test.go:76-79,187`) |
| DOM-29 | Cache scope (singleton) | N/A | `portal` package holds no cache state itself (registry access is delegated to the `position` package's own singleton) |

### services/atlas-channel/channel/movement

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:59` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) Processor` |
| DOM-33 | Mock updated for interface change | PASS | `movement/mock/processor.go:11` adds `TeleportCharacterFunc`; `:47-52` adds `TeleportCharacter` |
| DOM-26 | Goroutine spawned via `routine.Go` | PASS | `processor.go:94` `routine.Go(p.l, p.ctx, func(_ context.Context) {...})` |
| DOM-30 | DB write + emit atomicity via `AndEmit`/`message.Buffer` | PASS (documented exception) | `TeleportCharacter` (`processor.go:93-102`) calls `producer.ProviderImpl(...)` directly with no DB write on any path — matches the "Operations over non-DB state" exception in `patterns-kafka.md` (in-memory `position.GetRegistry()` state, mirrors `atlas-chairs` precedent) |
| DOM-24 | Test package reaching an emit path installs `producertest` stub | **FAIL** | `movement/teleport_test.go:22-41` (`TestTeleportCharacter_WritesLastPosition`) and `:43-67` (`TestTeleportCharacter_NoClientboundBroadcast`) both call `p.TeleportCharacter(...)`, which transitively reaches `producer.ProviderImpl(...)` (`processor.go:97`) inside a spawned goroutine — neither test, nor any `TestMain` in the `movement` package, installs `producertest.InstallNoop()`/`InstallCapturing()`. Only the fourth test in the same new file, `TestTeleportCharacter_EmitsFhZeroOnWire` (`:109-111`), installs the stub. `TestForCharacter_WritesLastPosition` (`:70-105`) has the same gap for the pre-existing `ForCharacter` emit path. No `TestMain` exists anywhere under `movement/*_test.go` (`action_test.go`, `displacement_test.go`, `fold_test.go`, `processor_test.go`, `teleport_test.go` — grepped, zero matches for `func TestMain`) |
| DOM-20 | Table-driven tests | WARN | `teleport_test.go` uses 4 standalone `func TestXxx` (not `tests:=[]struct{}+t.Run`); each asserts a genuinely distinct property (position write, no broadcast, folded-position write, wire `Fh`), not parametrized variants of one input — consistent with the single-scenario `TestBuilderValidation` example in `testing-guide.md:24-28`. Not graded as a hard FAIL, but flagged since the audit checklist's literal DOM-20 pass criteria is the table form |

### services/atlas-channel/channel/position (support — singleton registry)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-29 | Singleton cache reached via accessor, never per-instance | PASS | `registry.go:35-40` `GetRegistry()` uses `sync.Once`; `mutex sync.RWMutex` guards `Put`/`Lookup`/`Clear` (`:43-48,51-56,59-63`) — a singleton under a different accessor name (`GetRegistry` not `GetCache`) still passes DOM-29 per its own scope-grading rule |
| FILE-06 | No catch-all bundling ≥2 responsibilities | PASS | `registry.go` bundles the cache-shaped `Key`/`Position` types with the singleton accessor in one file — this matches the *documented* `cache.go` pattern itself (`patterns-cache.md`'s worked example bundles `cacheEntry`/`Cache`/`GetCache` in one file), not a violation |
| DOM-20 | Table-driven tests | WARN | `registry_test.go` — 5 standalone `func TestRegistry_*` (PutThenLookup, LookupMiss, TenantIsolated, PutOverwrites, Clear), each exercising a different method/property, not table-driven. Same single-scenario reasoning as above |

### services/atlas-channel/channel/session

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant travels via context only | PASS | `clearLastPositionOnDestroy(ctx, characterId)` (`processor.go:472-476`) extracts tenant via `tenant.MustFromContext(ctx)`, passed only as an internal registry key — never serialized to a REST model or request body |
| DOM-20 | Table-driven tests | WARN | `position_hook_test.go:35,58` — two standalone functions (`_NonZeroCharacter_ClearsState`, `_ZeroCharacter_NoOp`) are a positive/negative pair of the same function, `clearLastPositionOnDestroy` — a natural table-driven candidate that was not consolidated |

### services/atlas-channel/channel/data/portal (support — REST-client + cache)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:27` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-29 | Cache is a singleton reached via `GetCache()` accessor | **FAIL** | `processor.go:52-53` declares `portalCache sync.Map` and `portalLoadMu sync.Map` as bare package vars, accessed directly from `getPortalsInMap` (`:56-78`) — no `CacheInterface`, no `GetCache()` accessor, no `sync.Once`-gated singleton construction as required by the "Standard Singleton Cache Pattern" (`patterns-cache.md` step 3 of the DOM-29 verification: "Confirm the singleton itself follows the shape... a `GetCache()` accessor"). The cache is *scoped* correctly (package-level, not per-request), but does not follow the documented accessor shape |
| FILE-06 | No file bundling ≥2 responsibilities | **FAIL** | `processor.go` carries both the `processor.go` responsibility (`Processor` interface, `ProcessorImpl`, `GetInMapByName`) and the `cache.go` responsibility (`cacheKey`, `portalCache`, `portalLoadMu`, `getPortalsInMap`'s load-and-cache logic) in one file (`processor.go:34-79`). Cf. `position/registry.go`, which passes because it is a *single-purpose* cache file with no competing `processor.go` in the same package — `data/portal` already has a `Processor`-responsibility file and layered cache logic into it instead of a separate `cache.go` |
| EXT-01 | Target `RestModel` implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** | `rest.go` (unchanged, but its contract is called by the changed `processor.go`) defines `RestModel` with only `GetName()`/`GetID()`/`SetID()` (`rest.go:19-31`) — no `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Sibling packages `data/consumable/rest.go:51,55` and `data/equipment/rest.go:37` implement these on their `RestModel` |
| EXT-02 | `httptest`-backed integration test with a representative JSON:API fixture | **FAIL** | No `httptest` server exists anywhere in `data/portal/*_test.go` — `grep -rn httptest` only matches a code comment (`processor.go:38`). `processor_test.go`'s three cache tests replace `requestInMapFn` with a hand-built Go closure (`staticRequest`, `processor_test.go:17-21`) returning `[]RestModel` directly — a `FakeClient`-style seam, which the rule explicitly says "does not satisfy this" |
| EXT-03 | Only genuine 404s map to "not found"; transport/5xx bubble unmodified | PASS | `getPortalsInMap` (`processor.go:56-79`) and `GetInMapByName` (`processor.go:90-100`) return the raw error from `requests.SliceProvider` unmodified on any transport failure; the only synthesized error (`fmt.Errorf("no portal named...")`, `processor.go:98`) fires only after a successful fetch when the name is absent from the result set — not a status-code reclassification |
| EXT-04 | URL composed via `requests.RootUrl`/`RootUrlFor`, not hardcoded DNS | PASS | `requests.go:16` `requests.RootUrlFor(ctx, "DATA")` |
| DOM-20 | Table-driven tests | WARN | `processor_test.go` — 4 standalone functions, no `tests:=[]struct{}+t.Run`; `_CachesWholeList`/`_TenantScoped`/`_NotFound` are variants of `GetInMapByName`'s caching behavior along different dimensions (call count, tenant, filter miss) and are a plausible table-driven candidate that was not consolidated |

### services/atlas-channel/channel/socket/handler (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | N/A trigger (no `processor.go` in this package) | N/A | `portal_inner.go` has no `Processor` definition |
| FILE-06 | No catch-all bundling ≥2 responsibilities | PASS | `portal_inner.go` is a single handler function, matching every sibling handler file's shape |

### services/atlas-character/character/character (domain package, seam only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | N/A (no `Processor`/`Provider`/`Administrator` interface re-signed) | N/A | `Move` signature unchanged; only its body branches differently (`processor.go:876-882`) |
| DOM-20 | Table-driven tests | WARN | `temporal_position_test.go:27,43,59` — three standalone functions, each a parametrized variant (fh=0-with-prior, fh≠0, fh=0-no-prior) of the same `Move`/`UpdatePosition` behavior along one dimension — a clean table-driven candidate that was not consolidated |

### tools/packet-audit/internal/idasrc (repo tool, not a service — tool-appropriate subset)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go` | N/A | No `go` statement added in `export.go`/`extract.go`/`export_test.go` (grepped; only comment-text matches) |
| DOM-20 | Table-driven tests | PASS (documented exception) | Both new tests (`export_test.go:448,537`) are single-scenario byte-level pins with an explicit RED/GREEN provenance note (`export_test.go:448-455`) — matches the `TestBuilderValidation` single-assertion shape endorsed in `testing-guide.md:24-28`, and this file already contains ~19 other pre-existing single-function pin tests of the same shape |
| DOM-21 | No redeclaration of an existing shared type/constant | PASS | `Selector`/`selectorJSON`/`exportFn.Note`/`Region`/`NoteUnderscore` are tool-internal JSON-shape types with no `libs/atlas-constants/` equivalent |

## Security Review

Not applicable — SEC-01..04 trigger ("service handles authentication, authorization, tokens, redirects, or secrets") did not fire on any file in this diff.

## Not evaluable from the diff

- DOM-11 (provider laziness) for `data/portal` — the package has no `provider.go`; `getPortalsInMap`'s "load once, cache" pattern is graded under DOM-29/FILE-06 instead, since the rule's own trigger (`provider.go` present) never fires.
- Whether `data/portal`'s missing `SetToOneReferenceID`/`SetToManyReferenceIDs` (EXT-01) is a pre-existing gap predating this task, versus something this diff should have closed while touching `processor.go` in the same package — not determinable from the diff alone; `rest.go` carries no history markers in the diff. Recorded as a FAIL regardless, per EXT-01's package-level trigger ("package calls another atlas service via `requests.*Request[T]`"), which fired because `processor.go` (changed) makes that call.
- Whether atlas-channel's `data/map/processor.go` (referenced in a comment at `data/portal/processor.go:43` as the precedent for `cacheKey`) itself passes DOM-29/FILE-06 — out of scope (unchanged file, not read); cited only as evidence of what the comment claims, not graded.

## Summary

### Blocking (must fix)
- DOM-24: `services/atlas-channel/atlas.com/channel/movement/teleport_test.go` — `TestTeleportCharacter_WritesLastPosition` (line 22), `TestTeleportCharacter_NoClientboundBroadcast` (line 43), and `TestForCharacter_WritesLastPosition` (line 70) transitively reach `producer.ProviderImpl` without installing `producertest.InstallNoop()`/`InstallCapturing()`; no `TestMain` in the `movement` package installs it either.
- DOM-29 / FILE-06: `services/atlas-channel/atlas.com/channel/data/portal/processor.go:34-79` — the new portal cache is a bare package-level `sync.Map`, with no `CacheInterface`/`GetCache()` singleton accessor, and its logic is co-located inside `processor.go` rather than a dedicated `cache.go`.
- EXT-01: `services/atlas-channel/atlas.com/channel/data/portal/rest.go` — `RestModel` does not implement `SetToOneReferenceID`/`SetToManyReferenceIDs`, unlike sibling `data/*` packages.
- EXT-02: `services/atlas-channel/atlas.com/channel/data/portal/processor_test.go` — no `httptest`-backed integration test exists for the DATA-service REST client; the new cache tests stub the request seam directly instead.

### Non-Blocking (should fix)
- DOM-20: `movement/teleport_test.go`, `position/registry_test.go`, `session/position_hook_test.go`, `data/portal/processor_test.go`, `character/temporal_position_test.go` are not table-driven; the `session/position_hook_test.go` positive/negative pair and `character/temporal_position_test.go`'s three fh-variant tests are the clearest table-driven candidates.

## Not evaluable notes count
notEvaluable: 3
