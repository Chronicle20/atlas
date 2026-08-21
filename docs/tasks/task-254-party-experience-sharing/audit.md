# Backend Audit — task-254-party-experience-sharing

- **Service Path:** services/atlas-monster-death, services/atlas-monsters
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** All packages PASS (0 failures), including `-race` on all changed packages
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-monster-death/atlas.com/monster && go build ./...   # exit 0, no output
cd services/atlas-monster-death/atlas.com/monster && go test ./... -count=1
  ok  	atlas-monster-death	0.092s
  ok  	atlas-monster-death/map	0.171s
  ok  	atlas-monster-death/monster	6.463s
  ok  	atlas-monster-death/monster/drop	0.184s
  ok  	atlas-monster-death/monster/information	0.090s
  ok  	atlas-monster-death/party	0.087s
  ok  	atlas-monster-death/quest	0.177s
  ok  	atlas-monster-death/rates	0.153s
  ok  	atlas-monster-death/system_message	0.147s
  (all others: no test files)

cd services/atlas-monsters/atlas.com/monsters && go build ./...      # exit 0, no output
cd services/atlas-monsters/atlas.com/monsters && go test ./... -count=1
  ok  	atlas-monsters	1.297s
  ok  	atlas-monsters/monster	20.143s
  ok  	atlas-monsters/monster/information	15.556s
  (all others: ok or no test files)
```

`-race` re-run on every changed package in both modules (`monster/...`, `system_message/...`,
`party/...`, `rates/...`, `map/...` in atlas-monster-death; `monster/...` in atlas-monsters) — all
green, no data races reported for the new `sync.WaitGroup` fan-out in
`monster/processor.go:224-239`.

`tools/goroutine-guard.sh` (repo-wide, 89 modules, 8 parallel) — exit 0.

## Applicability

| Family | Fired? | Evidence |
|---|---|---|
| FILE (FILE-01..06) | Yes | Every changed package runs this family unconditionally. |
| DOM structure (DOM-01,02,03,04,05,11,16) | Yes | `party/model.go`, `party/rest.go`, `rates/model.go`(pkg has model.go), `monster/model.go`, `information/model.go` present in scope. |
| REST (DOM-06..09,12..15,17..19,32) | Yes (narrowly) | Changed packages have `processor.go` (map, monster, party, rates, system_message, information); none of the atlas-monster-death changed packages have `resource.go`, so DOM-07/08/09/12-19/32 are N/A per their own file-specific triggers. |
| Constants reuse (DOM-21) | Yes | New env-keyed constants (`EnvEnforceMobLevelRange`, `EnvLevelInterval`, `EnvLeachInterval`) and new types (`ExperienceConfig`, `Recipient`, etc.) declared in `monster/config.go`, `monster/experience.go`. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes many `_test.go` files and adds methods to several `Processor` interfaces (map, rates, information — all newly introduced in this diff). |
| Cache (DOM-29) | Yes | `system_message.Throttle` is cached/singleton state held as a `ProcessorImpl` field (`ht`). |
| Messaging (DOM-30) | Yes | `system_message/producer.go` calls `producer.ProviderImpl`/`producer.SingleMessageProvider`. |
| Multi-tenancy (DOM-31) | Yes | `party/rest.go` changed; `monster/processor.go` reads `tenant.MustFromContext`. |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and `libs/atlas-*`. |
| Deploy & topics (DOM-22,23) | N/A (checked anyway per task brief) | No `libs/atlas-*` module added; the three new env vars (`LEACH_INTERVAL`, `LEVEL_INTERVAL`, `USE_ENFORCE_MOB_LEVEL_RANGE`) are plain config, not `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` topic vars, so DOM-23's own trigger does not fire — verified wiring anyway (see Checklist Results). |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed throughout. `tools/goroutine-guard.sh` exit 0. |
| Channel wire values (DOM-25) | N/A | Diff does not touch `atlas-channel` or `atlas-packet`; `system_message` emits a semantic hint command (`SHOW_HINT`), not a raw client byte. |
| Resilience (DOM-27,28) | Yes (DOM-28 only) | DOM-27 N/A — atlas-monster-death has no `database.Connect` call (no DB). DOM-28 fires — `monster/processor.go` adds new remote-fetch fallback paths (party lookup, character lookup) that branch on `err`. |
| External clients (EXT-01..04) | Yes | `party`, `information`, `map`, `rates` packages call `requests.*` for other atlas services; `party` and `information` have changed `rest.go`/`processor.go` files in this diff. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, no `routes.conf` change. |
| Security (SEC-01..04) | N/A | Neither service handles auth/tokens/redirects/secrets. |
| patterns-provider.md (foundational) | Opened, no separate finding | `information/processor.go` composes the pre-existing `GetById` provider; no new provider defined. |
| patterns-functional.md (foundational) | Opened, no separate finding | `ProcessorOption` is the functional-options DI idiom, not a new curried constructor; no new curried constructor introduced in this diff. |

## Checklist Results

### monster (domain package — atlas-monster-death)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` + validating `Build()` | N/A | `monster/builder.go` deleted in this diff (`DamageDistributionBuilder` removed along with `DamageDistributionModel`); the package's only remaining `Model`, `DamageEntryModel`, has no builder and none is required by any caller — no `Build()`-shaped constructor exists to grade. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `monster/processor.go:43` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-26 | goroutines via `routine.Go` | PASS | `monster/processor.go:231,235` both goroutines spawned via `routine.Go(p.l, p.ctx, func(...) {...})`; `tools/goroutine-guard.sh` exit 0. |
| DOM-28 | Fallible enrichment paths degrade loudly (`model.ErrDecorator`+`degrade.Observe`) | **FAIL** | `monster/processor.go:273-280` — party lookup (`p.pp.GetByMemberId`) is a remote-data fetch that branches on `err` and degrades to a solo fallback with only `p.l.WithError(err).Warnf(...)`; no `degrade.Observe(...)` call, no `atlas_enrichment_degraded_total` increment. Same shape again at `monster/processor.go:345-351` (`soloInputFor`, character lookup fallback on `err`, `Errorf` only, no metric). `libs/atlas-rest/degrade/degrade.go` exists and is not imported anywhere in `atlas-monster-death`. |
| DOM-29 | Caches are singleton, reached via `GetCache()`-style accessor | PASS | `system_message/throttle.go:66-78` `hintThrottle`/`hintThrottleOnce`/`GetHintThrottle()`; `monster/processor.go:53` constructor takes only a reference: `ht: system_message.GetHintThrottle()`. Overridable for tests via `WithHintThrottle` (`processor.go:98-102`), never constructed inline. |
| DOM-33 | Every mock updated for interface method changes | PASS | New `map.Processor`/`rates.Processor`/`information.Processor` interfaces each ship their `mock/processor.go` in the same diff (`map/mock/processor.go`, `rates/mock/processor.go`, `information/mock/processor.go`); `monster.Processor`'s pre-existing `DistributeExperience([]DamageEntryModel)` signature is unchanged, so `monster/mock/processor.go` correctly stays untouched. `go build ./...` green in-module. |
| FILE-01 | Processor content lives in `processor.go` | PASS | All `ProcessorImpl` methods are in `monster/processor.go` (lines 43-376); `experience.go` and `interval.go` hold pure helper functions only, called by the processor, not `ProcessorImpl` methods themselves. |
| FILE-05 | Domain Model in `model.go` | PASS | `monster/model.go:3-21` — `DamageEntryModel`, private fields + accessors. |
| FILE-06 | No package-named catch-all | PASS | Non-test files: `config.go`, `experience.go`, `interval.go`, `model.go`, `processor.go` — none named `monster.go`; `experience.go`/`interval.go`/`config.go` are each single-purpose (pure planning math, interval-union math, env config) and hold no Processor/RestModel/requests/Entity content. |
| DOM-21 | No redeclared type/const already in `libs/atlas-constants` | PASS | Grepped `libs/atlas-constants/monster`, `.../constants` for `Interval`/`LevelRange`/`MobLevel` — zero matches; `ExperienceConfig`, `Recipient`, `computeAward`, etc. are new domain-specific types with no existing library equivalent. |
| DOM-20 | Table-driven tests | PASS | `experience_test.go` uses `tests := []struct{...}` + `t.Run` for `TestComputeAward`, `TestComputeAward_NonFiniteIsGuarded`, `TestPlanDistribution`, `TestAggregateDamageEntries`; single-scenario tests (`TestLevelGateHintText`, `TestPlanDistribution_IsDeterministicUnderShuffledInput`, etc.) match the guideline's own single-assertion example (`testing-guide.md:24-29`). |
| DOM-24 | Emit-path tests stub the producer | PASS | `grep` across every changed `_test.go` for `AndEmit(`/`message.Emit(`/`producer.Produce(` — zero matches; `processor_experience_test.go` tests inject full mocks (`systemmessagemock`, `charactermock`, `partymock`, `mapmock`, `informationmock`, `ratesmock`) so no real emit path is reached. |

### information (sub-package of monster — REST client)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Target REST model implements `SetToOneReferenceID` + `SetToManyReferenceIDs` (even as no-ops) | **FAIL** | `information/rest.go` (changed in this diff, lines 1-93) defines `RestModel` with `GetName`/`GetID`/`SetID` only — no `SetToOneReferenceID`, no `SetToManyReferenceIDs`, no `GetReferences`/`GetReferencedIDs`/`GetReferencedStructs` at all. |
| EXT-02 | httptest-backed integration test with a representative JSON:API fixture | **FAIL** | `information/rest_test.go` (new in this diff) only constructs `RestModel{...}` directly in Go and calls `Extract`/`NewBuilder` — no `httptest.Server`, no marshaled JSON:API fixture. `grep -rl httptest information/` → no match. |
| EXT-03 | Only 404 maps to not-found; transport/decode/5xx bubble | N/A | `information/provider.go` (unchanged) does not do any `errors.Is(err, requests.ErrNotFound)` collapsing — it just returns whatever `requests.Provider` yields, so no masking to grade. |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor` | PASS | `information/requests.go:16` `requests.RootUrlFor(ctx, "DATA")` (unchanged file, read to settle EXT-04). |
| DOM-33 | Mock matches new `information.Processor` interface | PASS | `information/mock/processor.go:7-18` implements `GetById` matching `information/processor.go:15-18` exactly, nil-check default returns `information.Model{}, nil`. |
| FILE-01 | Processor content in `processor.go` | PASS | `information/processor.go:14-26` — interface + `ProcessorImpl` + `NewProcessor` all together, new file. |

### party (REST-client package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Target REST model implements `SetToOneReferenceID` + `SetToManyReferenceIDs` | **FAIL** | `party/rest.go` (changed, full file read) defines `SetToManyReferenceIDs` (line 69) but never defines `SetToOneReferenceID` on `RestModel`, and `MemberRestModel` (lines 141-165 of the new content) implements neither method at all. |
| EXT-02 | httptest-backed integration test | **FAIL** | `party/rest_test.go` (new) only builds `RestModel`/`MemberRestModel` literals and calls `Extract`/`SetToManyReferenceIDs` directly — no `httptest.Server` fixture. `grep -rl httptest party/` → no match. |
| EXT-04 | URL via `RootUrl`/`RootUrlFor` | PASS | `party/requests.go:16` (unchanged) `requests.RootUrlFor(ctx, "PARTIES")`. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | N/A (documented exception) | `party/rest.go` has no `Transform` — this is the documented consumer-only `rest.go` shape from `cross-service-implementation.md:492-522` (`PartyRestModel`/`Extract`-only client pattern); party never serializes its own model outward, it only deserializes another service's response. |
| DOM-31 | Tenant/trace only in context | PASS | `party/rest.go` `RestModel`/`MemberRestModel` carry no tenant/trace field (grepped full file, none present). |
| FILE-05 | Builder in `builder.go`, Model in `model.go` | PASS | `party/builder.go:1-92` (`Builder`, `MemberBuilder`); `party/model.go:8-57` (`Model`, `MemberModel`, private fields + accessors). |

### rates (domain-ish package, DI wrapper added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Ctor takes `logrus.FieldLogger` | PASS | `rates/processor.go:16` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-33 | Mock matches new `rates.Processor` | PASS | `rates/mock/processor.go:14-19` matches `rates/processor.go:12-13` exactly. |
| DOM-01 | `builder.go` + validating `Build()` | PASS (no invariant to enforce) | `rates/builder.go:1-47` — new file, `NewBuilder()`/fluent setters/`Build()`; `Model`'s four rate multipliers carry no documented invariant (all default to 1.0, any float is legal), so a vacuous `Build()` is consistent — no invariant exists to check. |
| FILE-01 | Processor content in `processor.go` | PASS | `rates/processor.go:12-28`, new file, holds interface + impl. |

### map (DI wrapper added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Ctor takes `logrus.FieldLogger` | PASS | `map/processor.go:17` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-33 | Mock matches new `_map.Processor` | PASS | `map/mock/processor.go:13-19` matches `map/processor.go:11-13`. |

### system_message (new support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Ctor takes `logrus.FieldLogger` | PASS | `system_message/processor.go:27` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-30 | DB write + events via `AndEmit`/`message.Buffer` | N/A (documented exception) | `system_message.ShowHint` (`processor.go:37-38`) has no DB write on any path (no `entity.go`/`administrator.go` in this package) — matches the "operations over non-DB state" exception in `patterns-kafka.md` (the `atlas-chairs` precedent cited there); a direct `producer.ProviderImpl(...)` call is correct. |
| DOM-24 | Emit-path tests stub the producer | PASS | `producer_test.go` (`TestShowHintCommandProvider_WireShape`) calls `ShowHintCommandProvider(...)` directly (the pure message-provider function) — it never reaches `producer.Produce`/a live Kafka call, so no stub is required. |
| DOM-33 | Mock matches `system_message.Processor` | PASS | `system_message/mock/processor.go:11-22` matches `processor.go:15-17` (`ShowHint`). |
| FILE-06 | No package-named catch-all | PASS | `kafka.go` (env const + `Command[E]`/`ShowHintBody` envelope types), `processor.go`, `producer.go`, `throttle.go` — each single-purpose; none is `system_message.go`. |
| DOM-29 | Singleton reached via accessor | PASS | See "monster" section above — `GetHintThrottle()` is defined here (`throttle.go:73-78`). |

### atlas-monsters / monster (only `builder.go` touched)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Writes in `builder.go` correct location, aggregation logic | PASS | `builder.go:139-152` `AddDamageEntry` now aggregates by `characterId` before appending — new behavior, correctly placed in the existing `builder.go`. |
| DOM-20 | Table-driven tests | PASS | `builder_test.go:95-127` `TestAddDamageEntry_AggregatesByCharacter` is table-driven (`tests := []struct{...}` + `t.Run`); other single-scenario tests match the guideline's own example. |
| Correctness cross-check | `AddDamageEntry` aggregation matches `Registry.ApplyDamage` | PASS | `registry.go:436-461` sums into an existing `storedDamageEntry` by `characterId` the same way — both write paths agree, which is exactly what the death service's `aggregateDamageEntries` (FR-1.4 defense) assumes is already true. |

## Security Review

Not applicable — neither `atlas-monster-death` nor `atlas-monsters` handles authentication, authorization, tokens, redirects, or secrets in the changed surface. SEC-* family not triggered.

## Not evaluable from the diff

- FILE-01..06 (full-package grading) for `atlas-monsters/atlas.com/monsters/monster`: only `builder.go` changed in a 20+ file package (`resource.go`, `rest.go`, `processor.go`, `registry.go`, etc. all unchanged); grading the whole package's file layout for FILE-06 catch-alls would require reading files outside the diff's surface.
- EXT-01/02/04 for `map` and `rates` REST-client shape: `map/rest.go` and `rates/rest.go` are unchanged in this diff (only new `processor.go` DI wrappers were added); would need to read those unchanged files to grade the actual `RestModel`.
- DOM-04/05 for `information/rest.go`: `Transform`/`TransformSlice` are absent, consistent with a consumer-only rest.go, but I did not locate an explicit exception citation for `information` the way `cross-service-implementation.md:492-522` documents for `party` — recorded here rather than asserted as PASS or FAIL.

## Summary

### Blocking (must fix)
- DOM-28 — `services/atlas-monster-death/atlas.com/monster/monster/processor.go:273-280` and `:345-351`: party-lookup and character-lookup fallback paths fetch remote data, branch on `err`, and degrade to a fallback value with only a `Warn`/`Error` log — no `degrade.Observe(...)` call, no `atlas_enrichment_degraded_total` increment, as DOM-28 requires for every fallible enrichment/fallback.
- EXT-01 — `services/atlas-monster-death/atlas.com/monster/party/rest.go`: neither `RestModel` nor `MemberRestModel` implements `SetToOneReferenceID` (only `SetToManyReferenceIDs` at line 69 exists).
- EXT-01 — `services/atlas-monster-death/atlas.com/monster/monster/information/rest.go`: `RestModel` implements neither `SetToOneReferenceID` nor `SetToManyReferenceIDs`.
- EXT-02 — `services/atlas-monster-death/atlas.com/monster/party/rest_test.go` and `.../monster/information/rest_test.go`: no `httptest`-backed integration test serving a representative JSON:API fixture for either REST client; both test files only construct Go structs directly.

### Non-Blocking (should fix)
- None identified beyond the blocking items above.

### Not evaluable
- 3 items — see "Not evaluable from the diff" above.
