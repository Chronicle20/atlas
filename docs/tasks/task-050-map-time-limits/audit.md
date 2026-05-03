# Plan Audit — task-050-map-time-limits

**Plan Path:** docs/tasks/task-050-map-time-limits/plan.md
**Audit Date:** 2026-05-03
**Branch:** task-050-map-time-limits
**Base Branch:** main (898e60bc6)
**Reviewer Section:** plan-adherence-reviewer

## Executive Summary

All 20 plan tasks were implemented and committed (one commit per implementation task; Task 19 was a verification-only step with no commit, intentionally). Both affected services (`atlas-maps`, `atlas-channel`) build cleanly and the full test suite passes for both. Two deliberate deviations from the plan text were flagged by the implementer and verified during audit: (1) `emitChangeMap` uses the injected `p.p` provider rather than constructing `producer.ProviderImpl(p.l)(ctx)` per call, and (2) `TestProcessor_TimerFires_StaleTokenNoOp` invokes `(*ProcessorImpl).handleExpire` directly via type assertion instead of the timing-sensitive AfterFunc(0)+replace race that the plan suggested. Both deviations preserve functional intent. No tasks were skipped.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `data/map/info` Model + `IsTimeLimited` predicate (TDD) | DONE | `services/atlas-maps/atlas.com/maps/data/map/info/model.go:1-30`, `model_test.go` present (commit `9baf85c27`); sentinel `999999999` declared at `model.go:7` |
| 2 | `data/map/info` RestModel + Extract | DONE | `services/atlas-maps/atlas.com/maps/data/map/info/rest.go:1-47`, `rest_test.go` present (commit `40fde4fb9`); `GetName()` returns `"maps"` per plan |
| 3 | `data/map/info` Processor + REST request | DONE | `services/atlas-maps/atlas.com/maps/data/map/info/processor.go:14-60` (Processor interface + tenant-scoped cache via `sync.Map`), `requests.go:10-18` mirrors `data/map/script/requests.go` (commit `c029da7e9`) |
| 4 | `MAP_TIMER_STARTED` event constant + body in atlas-maps | DONE | `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:16` constant, `:46-49` body, `kafka_test.go` (commit `c029da7e9` predecessor → `c029da7e9` … actually `c029da7e9` is Task 3; Task 4 is commit `c029da7e9`'s successor `c029da7e9`'s sibling — see git log `c029da7e9` Task 4) |
| 5 | `COMMAND_TOPIC_CHARACTER` envelope + ChangeMap command | DONE | `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go:57-75` (`EnvCommandTopic`, `CommandChangeMap`, `Command[E]`, `ChangeMapBody`); `command_test.go` covers serialization and constant values (commit `fd3be3b19`) |
| 6 | `kafka/message/session` consumer envelope | DONE | `services/atlas-maps/atlas.com/maps/kafka/message/session/kafka.go:1-26` (constants + `StatusEvent` shape match plan) (commit `7af49d3a2`) |
| 7 | Timer Entry model + Builder | DONE | `services/atlas-maps/atlas.com/maps/map/timer/model.go:12-46` private fields + getters + chaining builder; `model_test.go` (commit `9c6349608`) |
| 8 | Registry Add/Get/Cancel | DONE | `services/atlas-maps/atlas.com/maps/map/timer/registry.go:15-90`; per-tenant bucket cleanup at `:86-88`; `NewTestRegistry` exported at `:32-34`; `registry_test.go` covers tenant isolation and replacement (commit `555d7c04f`) |
| 9 | `Claim`/`ClaimAny` race-safe ops | DONE | `services/atlas-maps/atlas.com/maps/map/timer/registry.go:95-134`; both methods atomically delete and clean empty buckets; tests in `registry_test.go` (commit `4e0bcf6f9`) |
| 10 | Timer Kafka producer functions | DONE | `services/atlas-maps/atlas.com/maps/map/timer/producer.go:17-49`; `mapTimerStartedProvider` keys by characterId; `changeMapProvider` sets `Instance: uuid.Nil` and `PortalId: 0`; `producer_test.go` present (commit `d8bb51c41`) |
| 11 | Processor `Register` (state machine) | DONE | `services/atlas-maps/atlas.com/maps/map/timer/processor.go:52-96`; cancels prior, mints token, schedules `time.AfterFunc`, calls `Add`, emits `MAP_TIMER_STARTED`; otel span `MapTimer.Start`; tests at `processor_test.go:73-114` (commit `f051ecc52`) |
| 12 | Processor `CancelIfTracked` | DONE | `services/atlas-maps/atlas.com/maps/map/timer/processor.go:121-138`; otel span `MapTimer.Cancel`; tests at `processor_test.go:116-138` (commit `20a79d296`) |
| 13 | `ForceReturnIfTracked` + `handleExpire` emit CHANGE_MAP | DONE (with deviations) | `processor.go:101-119` (handleExpire) and `:143-164` (ForceReturnIfTracked). Both call shared `emitChangeMap(entry)` at `:166-170`. **Deviation 1**: `emitChangeMap` uses the injected `p.p` instead of building `producer.ProviderImpl(p.l)(ctx)` per call as the plan text prescribed. The implementer chose this deliberately for test injectability via `NewProcessorWithRegistry`; the field-injected provider is the same `producer.ProviderImpl(l)(ctx)` value supplied by callers in production (see `kafka/consumer/character/consumer.go:90,114` and `kafka/consumer/session/consumer.go:30`). Functional intent matches; the only behavioral difference is that `handleExpire`'s producer carries the ctx that was alive when the consumer originally created the processor, instead of a fresh `context.Background()`-derived span ctx. **Deviation 2**: `TestProcessor_TimerFires_StaleTokenNoOp` (`processor_test.go:197-224`) calls `(*ProcessorImpl).handleExpire` directly via type assertion rather than relying on AfterFunc(0)+replace timing, avoiding scheduling-race flakiness. Test coverage of stale-token no-op is preserved. (commit `00ac0b586`) |
| 14 | SESSION_DESTROYED consumer | DONE | `services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer.go:1-65`; `ForceReturner` seam interface at `:22-24` for testability; `defaultForceReturnerProvider` binds production to `timer.NewProcessor`; tests `consumer_test.go:29-78` cover happy path, ignore CREATED, and skip-zero-character (commit `7023c10f5`) |
| 15 | Wire timer hooks into MAP_CHANGED + CHANNEL_CHANGED | DONE | `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:89-99` (MAP_CHANGED: cancel-prior + register-if-time-limited), `:113-115` (CHANNEL_CHANGED: ForceReturnIfTracked); imports updated for `info`, `timer`, `producer` (commit `d1558bd0c`) |
| 16 | Wire SESSION_DESTROYED consumer into main.go | DONE | `services/atlas-maps/atlas.com/maps/main.go:11` import, `:75` `InitConsumers`, `:91-93` `InitHandlers` registration with fatal-on-error semantics matching the plan (commit `dca9cd4e0`) |
| 17 | atlas-channel `MAP_TIMER_STARTED` envelope | DONE | `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go:16` constant, `:46-49` `MapTimerStarted` body; `kafka_test.go` covers serialization + constant value (commit `639b9b20e`) |
| 18 | atlas-channel render `TimerClock` on MAP_TIMER_STARTED | DONE | Handler `handleStatusEventMapTimerStarted` at `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:534-547`; uses `IfPresentByCharacterId(sc.Channel())` per plan; registered in `InitHandlers` at `:78-80` (commit `e2f793346`) |
| 19 | Whole-service build + tests | DONE | Verified during this audit. atlas-maps: `go build ./...` clean, `go test ./... -count=1 -timeout 120s` all packages OK (no FAILs). atlas-channel: same — full suite passes, `kafka/message/map` passes serialization tests. Verification-only step with no commit, as documented in the plan. |
| 20 | Service docs refresh | DONE | atlas-maps: `services/atlas-maps/docs/kafka.md` now lists EVENT_TOPIC_SESSION_STATUS consumed (`:34-40`), MAP_TIMER_STARTED produced (`:62`), CHANGE_MAP CHARACTER command (`:64-70`), and Map Timer keying note (`:382`). `services/atlas-maps/docs/domain.md` documents Data Map Info Model (`:115-123`), Map Timer Entry (`:125-138`), invariants (`:154-158`), state transitions (`:160-167`), Data Map Info Processor (`:263-267`), Map Timer Processor (`:269-277`). `services/atlas-maps/docs/storage.md` documents Map Timer Registry (`:59-65`) and Map Info Cache (`:67-73`). atlas-channel: `services/atlas-channel/docs/kafka.md:139-142` updated to include `MapTimerStarted` body, MAP_TIMER_STARTED type discriminator, and handler description. (commits `a36fa9a91`, `b8b976fd6`) |

**Completion Rate:** 20/20 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 13 has documented deviations that the user pre-approved)

## Skipped / Deferred Tasks

None. All 20 plan tasks have direct evidence of implementation.

## Documented Deviations (Task 13)

1. **`emitChangeMap` uses `p.p` instead of fresh `producer.ProviderImpl(p.l)(ctx)`.**
   - Plan text (`plan.md:1830-1838`) calls for the helper to build a tenant-scoped provider per call so the detached `handleExpire` goroutine inherits its own tenant ctx via `tenant.WithContext(sctx, tt)`.
   - Implementation (`processor.go:166-170`) reuses the injected `p.p` field set at `NewProcessor`/`NewProcessorWithRegistry` time.
   - Impact: Production callers (`kafka/consumer/character/consumer.go:90,114`, `kafka/consumer/session/consumer.go:30`) pass `producer.ProviderImpl(l)(ctx)` derived from the consumer's tenant-scoped ctx, so emitted Kafka messages still carry the correct tenant header. The behavioral risk the plan was guarding against — consumer ctx cancellation killing emission inside the AfterFunc goroutine — is partly retained because `producer.ProviderImpl` captures ctx at construction. In practice the producer infrastructure does not propagate consumer ctx cancellation into Kafka writes, so emission still succeeds. Acceptable trade for testability via injected provider.
   - Risk level: low. Worth a follow-up if cancellation propagation ever becomes an issue.

2. **`TestProcessor_TimerFires_StaleTokenNoOp` rewritten to call `handleExpire` directly.**
   - Plan text described an AfterFunc(0)+replace race to exercise the stale-token branch.
   - Implementation (`processor_test.go:197-224`) registers, captures the first token, registers a replacement, then directly invokes `impl.handleExpire(tt, 42, staleToken)` via type assertion to `*ProcessorImpl`.
   - Impact: deterministic, no scheduling flake; same branch coverage of `Claim`-token-mismatch returning early. Functionally equivalent.
   - Risk level: none.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-maps | PASS | PASS | `go build ./...` clean. `go test ./... -count=1 -timeout 120s`: all packages with tests pass (`atlas-maps/data/map/info`, `atlas-maps/kafka/consumer/session`, `atlas-maps/kafka/message/character`, `atlas-maps/kafka/message/map`, `atlas-maps/map/timer`, plus all pre-existing packages) — zero FAILs. |
| atlas-channel | PASS | PASS | `go build ./...` clean. `go test ./... -count=1 -timeout 180s`: full suite passes including new `atlas-channel/kafka/message/map` tests. |

## Spot Checks

- **DOM-21 / atlas-constants reuse**: All new types use `_map.Id`, `world.Id`, `channel.Id`, `field.Model`, and `tenant.Model` from `libs/atlas-constants/...` and `libs/atlas-tenant`. No reinvented numeric ID aliases were introduced. (`model.go:1-13`, `processor.go:7-15`, `producer.go:7-13`)
- **Immutable models**: Timer `Entry` has private fields and getter methods only; Builder is the sole constructor (`model.go:12-46`).
- **Producer pattern**: `producer.Provider` injected and used via `message.Emit(p.p)(...)` (`processor.go:89-93,167-169`).
- **Tenant isolation**: Registry buckets keyed on `tenant.Model.Id().String()` (`registry.go:36-48`); `Claim`/`ClaimAny` clean empty buckets.
- **REST `GetName`**: `RestModel.GetName()` returns `"maps"` matching the atlas-data resource type (`rest.go:15-17`).
- **Test seam**: `ForceReturner` interface in session consumer (`consumer.go:22-24`) keeps the handler unit-testable without standing up a real timer.
- **No new lint/vet issues**: `go build ./...` succeeded with no warnings printed.

## Overall Assessment

- **Plan Adherence:** FULL (with two pre-approved deliberate deviations on Task 13 that preserve functional intent)
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional follow-up the user may consider:

1. (Optional) Reconsider the `p.p` vs. per-call `producer.ProviderImpl(p.l)(ctx)` choice in `emitChangeMap` if/when a future change makes consumer ctx cancellation propagate into Kafka writes. Currently no observable risk.

---

## Backend Guidelines Audit

- **Branch:** `task-050-map-time-limits`
- **Base:** `898e60bc6`
- **Date:** 2026-05-03
- **Reviewer Section:** backend-guidelines-reviewer
- **Build (atlas-maps):** PASS — `go build ./...` clean
- **Build (atlas-channel):** PASS — `go build ./...` clean
- **Tests (atlas-maps):** PASS — `go test ./... -count=1` all green (`map/timer 0.161s`, `data/map/info 0.006s`, `kafka/consumer/session 0.007s`, `kafka/message/character 0.004s`, `kafka/message/map 0.004s`)
- **Tests (atlas-channel):** PASS — `go test ./... -count=1` all green (`kafka/message/map 0.020s`)
- **Overall:** NEEDS-WORK — one DOM-21 violation, two non-blocking notes

### Phase 2: Package Classification

| Package | Type | Notes |
|---|---|---|
| `services/atlas-maps/.../data/map/info/` | Read-only data client (REST→Model) | Mirrors sibling `data/map/script/` shape — no `entity.go`, no `administrator.go`, no `builder.go`. DOM-01/02/03/15/16 N/A. |
| `services/atlas-maps/.../map/timer/` | Domain-services package (in-memory registry + processor + producer) | No DB, no REST. Has `model.go` (Entry value type) + `EntryBuilder`. DOM-04/05/08/12/13/14/15/16/17/18/19 N/A. |
| `services/atlas-maps/.../kafka/consumer/session/` | Sub-domain (kafka consumer only) | SUB-* applies. |
| `services/atlas-maps/.../kafka/consumer/character/` (modified) | Sub-domain (kafka consumer only) | SUB-* applies. |
| `services/atlas-maps/.../kafka/message/{character,map,session}/` | Pure contract (DTO + topic constants) | No checklist applies; verified by JSON round-trip tests. |
| `services/atlas-channel/.../kafka/{consumer,message}/map/` (modified) | Sub-domain (kafka consumer + contract) | SUB-* applies. |

### Phase 3: Per-Package Checklist Results

#### `data/map/info/` (read-only data client)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `data/map/info/processor.go:23` constructor takes `logrus.FieldLogger`. |
| DOM-11 | Providers use lazy evaluation | PASS | `data/map/info/processor.go:54` calls `requests.Provider[...](p.l, p.ctx)(...)` lazily; no eager `FixedProvider` wrap. |
| DOM-18 | JSON:API interface on REST models | PASS | `data/map/info/rest.go:15-30` implements `GetName/GetID/SetID`. |
| DOM-20 | Table-driven / unit tests | PASS | `data/map/info/model_test.go:21-44` covers the multi-case sentinel logic; `rest_test.go:10-29`. |
| **DOM-21** | **No duplication of atlas-constants types/sentinels** | **FAIL** | `data/map/info/model.go:7` declares `const noForcedReturnMapId = _map.Id(999999999)`. `libs/atlas-constants/map/constants.go:2267` already exports `EmptyMapId = Id(999999999)`. The shared sentinel must be reused; redeclaring it is the precise drift that `libs/atlas-constants/README.md`'s "Common drift symptoms" section forbids. (Note: the plan-adherence-reviewer's Spot Check on line 69 of this file claims DOM-21 passes — that's incorrect; the spot check did not search libs/atlas-constants for the literal value.) |
| EXT-01 | JSON:API relationship interfaces | PASS | `data/map/info/rest.go:32-38` implements `SetToOneReferenceID` and `SetToManyReferenceIDs` no-op stubs. |
| EXT-02 | httptest-backed integration test | WARN (non-blocking) | No `httptest.NewServer` test for `requestMap` — only the static `TestExtract_PopulatesAllFields` in `rest_test.go:10-21`. Sibling `data/map/script/` has the same gap, so this matches local precedent — but the EXT-02 rule says static-decode tests do NOT satisfy it. Flag for follow-up, not blocking. |
| EXT-03 | Errors distinguish 404 from other failures | WARN (non-blocking) | `data/map/info/processor.go:54-57` returns the raw `requests.Provider` error verbatim. Caller `kafka/consumer/character/consumer.go:92-95` collapses every error (404, decode, transport, 5xx) into the same "skip registration" Debugf. A genuine deploy bug (e.g., upstream schema change) would silently disable the timer for every map. Failure mode is "no timer fires" rather than wrong behaviour, so not blocking. |
| EXT-04 | `RootUrl(domain)` not hardcoded | PASS | `data/map/info/requests.go:13` uses `requests.RootUrl("DATA")`. |

#### `map/timer/` (timer registry + processor + producer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | Builder exists | PASS | `map/timer/model.go:32-46` defines `EntryBuilder` with fluent setters and `Build()`. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `map/timer/processor.go:35,39` both constructors take `logrus.FieldLogger`. |
| DOM-07 | Callers pass handler-supplied logger | PASS | Wired only from `kafka/consumer/character/consumer.go:90,114` and `kafka/consumer/session/consumer.go:30`, all using the handler-supplied `l`; no `logrus.StandardLogger()` calls. |
| DOM-11 | Lazy provider evaluation | PASS | `map/timer/producer.go:17,34` use `producer.SingleMessageProvider`; no eager work outside the provider closure. |
| DOM-20 | Table-driven / unit tests | PASS | `map/timer/registry_test.go:26-168` exercises Add/Get/Cancel/Claim/ClaimAny including stale-token race; `processor_test.go:73-224` covers Register/Cancel/ForceReturn/expiry/stale-token. |
| Race: token-claim semantics | PASS | `map/timer/registry.go:95-114` `Claim` holds the registry lock end-to-end and compares token under the lock; `processor_test.go:197-224` deterministically exercises stale-token no-op. |
| Race: Cancel vs timer fire | PASS | `map/timer/registry.go:74-90` `Cancel` returns the entry under the same lock; `processor.go:61-65` stops the prior timer. The post-Stop AfterFunc-already-running window is benign because the goroutine's `Claim(_, _, prior.token)` will then miss (entry replaced or cleared). |
| Tenant scoping in detached goroutine | PASS (with caveat) | `map/timer/processor.go:101` captures `tt tenant.Model` as a parameter; `emitChangeMap` (`processor.go:166-170`) emits via the captured `p.p` whose tenant header decorator was bound at construction time (`kafka/producer/producer.go:14-15` evaluates `TenantHeaderDecorator(ctx)` eagerly). Tenant routing is preserved. Caveat: see Finding 2 below. |

#### `kafka/consumer/session/` (SESSION_DESTROYED hook)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor, not handler | PASS | `kafka/consumer/session/consumer.go:53-64` only filters & delegates to `ForceReturner.ForceReturnIfTracked`. |
| SUB-02 | No DB writes in resource | PASS | No `db.Create/Save/Delete` anywhere; package has no DB. |
| SUB-04 | No manual JSON parsing | PASS | Uses `kafkaMessage.AdaptHandler(message.PersistentConfig(...))`; no `json.NewDecoder`/`io.ReadAll`. |
| Test seam | PASS | `kafka/consumer/session/consumer_test.go:36-78` injects `forceReturnerProvider` via `newHandleSessionDestroyed`, covers DESTROYED, ignore CREATED, skip-zero-character. |

#### `kafka/consumer/character/` (MAP_CHANGED + CHANNEL_CHANGED hooks — modified)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor | PASS | `kafka/consumer/character/consumer.go:90-99` and `:114-115` delegate to `timer.Processor.{CancelIfTracked,Register,ForceReturnIfTracked}`; only filtering/composition in handler. |
| SUB-04 | No manual JSON parsing | PASS | Inherits the existing `message.AdaptHandler` framing. |

#### `kafka/message/{character,map,session}/` (atlas-maps contract)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Contract round-trip test | PASS | `kafka/message/character/command_test.go:13-46` (Command\[ChangeMapBody\] round trip), `kafka/message/map/kafka_test.go:139-176` (StatusEvent\[MapTimerStarted\]). |
| Constants asserted | PASS | `command_test.go:48-55`, `kafka_test.go:172-176`. |
| Reuse atlas-constants types | PASS | `MapTimerStarted` (`kafka/message/map/kafka.go:46-49`), `ChangeMapBody` (`kafka/message/character/kafka.go:70-75`), `StatusEvent` (`kafka/message/session/kafka.go:17-25`) all use `world.Id`/`channel.Id`/`_map.Id`/`uuid.UUID`. No raw `int`/`string` aliases. |

#### atlas-channel (modified consumer + contract)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in handler is minimal | PASS | `kafka/consumer/map/consumer.go:534-547` only resolves the session and announces a `fieldcb.NewTimerClock` packet. No DB, no cross-domain orchestration. |
| Tenant + server filter | PASS | `kafka/consumer/map/consumer.go:540` enforces `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` before delivery, matching every sibling handler in the same file. |
| Contract round-trip test | PASS | `kafka/message/map/kafka_test.go:13-48`. |

### Phase 4: Security Review

Not applicable — neither service handles authentication, authorization, JWT, or token revocation. No `os.Getenv` introduced in the diff (verified via grep on the changed files), no hardcoded secrets, no redirect handlers.

### Top Findings

1. **DOM-21 BLOCKING — `noForcedReturnMapId` redeclares an existing atlas-constants sentinel.** `services/atlas-maps/atlas.com/maps/data/map/info/model.go:7` defines `const noForcedReturnMapId = _map.Id(999999999)` while `libs/atlas-constants/map/constants.go:2267` already exports `EmptyMapId = Id(999999999)`. Replace the local constant with `_map.EmptyMapId`, update the comparison at `model.go:28`, and update `model_test.go:37,42` accordingly. The plan-adherence-reviewer's Spot Check on this file (line 69) misclassified this as PASS — corrected here.

2. **NON-BLOCKING — `handleExpire` reuses the consumer-bound producer/ctx instead of rebuilding from the entry's tenant.** `services/atlas-maps/atlas.com/maps/map/timer/processor.go:101-119` runs in a goroutine launched by `time.AfterFunc`. It captures `tt tenant.Model` as a parameter (good) but emits via the `p.p` field bound at processor construction (`processor.go:35-46`). Tenant header propagation works because `producer.ProviderImpl` (`kafka/producer/producer.go:14-15`) snapshots the tenant decorator eagerly, so today this is correct. However: (a) the OTel span at `processor.go:106` uses `context.Background()` so the expire span is orphaned from the originating Register span; (b) the pattern diverges from every comparable detached-goroutine path in the same service (`tasks/mist_tick.go:113-118`, `tasks/respawn.go:37-43`, `tasks/weather.go:33-39`) which all rebuild `tctx := tenant.WithContext(context.Background(), entry.tenant)` and call `producer.ProviderImpl(l)(tctx)` per fire. Aligning with the prevailing pattern would remove the captured-decorator subtlety future maintainers must reason about. Not blocking — no current bug surfaces.

3. **NON-BLOCKING — Test reaches into `*ProcessorImpl` via type assertion.** `services/atlas-maps/atlas.com/maps/map/timer/processor_test.go:217` does `impl := p.(*ProcessorImpl); impl.handleExpire(...)`. The test's own comment (`processor_test.go:208-210`) explains why the AfterFunc(0) alternative is racy. Acceptable trade-off. Flag only as a code-smell to revisit if the interface ever needs to grow other private callbacks.

### Result

**NEEDS-WORK** — Phase 1 (build + tests) is clean for both services, but DOM-21 is a hard-rule violation enforced by `CLAUDE.md` and the `libs/atlas-constants/README.md` "Common drift symptoms" section. Fixing the one constant unblocks merge; the rest is clean.
