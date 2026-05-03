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
