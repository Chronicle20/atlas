# Backend Audit — task-184-portal-enter-double-execute

- **Scope:** Diff-scoped audit, base `616f20675` → head `d70cb26b1` (9 commits), two services:
  - `services/atlas-portal-actions/atlas.com/portal/` — new `dedupe` package, `script/optable.go`, changes to `script/{executor,consumer,processor,model}.go`, `action/registry.go`, `kafka/consumer/saga/consumer.go`, `main.go`
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/` — `saga/{processor,event_acceptance,character_extractor}.go`, `saga/mock/processor.go`, `kafka/consumer/character/consumer.go`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS (controller attestation: `go build ./...` clean both modules)
- **Tests:** PASS (controller attestation: `go test -race ./...` clean both modules; `go vet`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` all clean; no go.mod/go.sum delta)
- **Overall:** NEEDS-WORK

## Build & Test Results

Per controller attestation (Phase 1 skipped per instructions): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both `atlas-portal-actions` and `atlas-saga-orchestrator`; `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` exit 0; `tools/lint.sh --check` 0 issues; working tree clean.

## Domain Discovery

- `services/atlas-portal-actions/atlas.com/portal/dedupe/` — support package (no `model.go`, no `resource.go`). Contains `gate.go` + `gate_test.go` only. Runs File Responsibilities checklist.
- `services/atlas-portal-actions/atlas.com/portal/action/` — support package (Redis-backed registry, pre-existing). `registry.go` + `registry_test.go`.
- `services/atlas-portal-actions/atlas.com/portal/script/` — domain package (`model.go`, `entity.go`, `builder.go`, `processor.go`, `rest.go`, `resource.go` all present, pre-existing). This diff touches `consumer.go`, `executor.go`, `model.go`, `optable.go` (new), `processor.go` — none of which is `resource.go`, so DOM-08/09/12 (REST-handler checks) are out of scope; the touched surface is Kafka consumer + business logic.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/` — domain package (pre-existing, huge). Diff touches `processor.go`, `event_acceptance.go` (a `state.go`-shaped enum/constants file), `character_extractor.go` (helper file), `mock/processor.go`.
- `services/atlas-saga-orchestrator/.../kafka/consumer/character/` — sub-domain/consumer package, `consumer.go` only touched (one new call-site argument).

## File Responsibilities Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `script/processor.go:20` (`Processor` interface), `script/processor.go:40` (`ProcessorImpl`), `saga/processor.go:128` (`NewProcessor`) — all in correctly-named files. `dedupe.Gate`/`redisGate` in `dedupe/gate.go` is not a "Processor" per the checklist's letter (different interface shape: a boolean predicate, not CRUD/business orchestration) — not a violation. |
| FILE-02 | RestModel/Transform in rest.go | N/A (no `rest.go` touched by diff) | — |
| FILE-03 | Cross-service requests in requests.go | N/A (no new `requests.go`/HTTP client package touched) | — |
| FILE-04 | Entity+Migration+TableName in entity.go | N/A (no `entity.go` touched by diff) | — |
| FILE-05 | Builder/Model/administrator/provider/state placement | PASS | `script/model.go` holds `ProcessResult` (plain result DTO, not a persisted domain object — appropriately not builder-shaped) plus pre-existing `PortalScript`/`Rule`/`RuleOutcome` (private fields + getters, unchanged shape). `saga/event_acceptance.go:1-45` is a `state.go`-shaped enum/constants file (EventKind, SkipReason*, acceptanceTable) — consistent with its pre-existing role. |
| FILE-06 | No package-named catch-all file | PASS | `dedupe/gate.go` is single-purpose (Gate interface + one impl); `action/registry.go` is single-purpose (Registry CRUD, pre-existing shape unchanged); `script/optable.go` is a single-purpose dispatch/classification table, not a bundle of Processor+RestModel+requests. No new catch-all introduced. |

## Domain Checklist Results (touched surface only)

### `script` (atlas-portal-actions)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `script/processor.go:50` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-13/14 | No cross-domain logic / no provider calls in handlers | PASS | `script/consumer.go:99-106` (dedupe gate check) and `:137-140` (unlock decision) call only `dedupe.GetGate()`/`enableActionsFn` and the `Processor` — no provider or cross-domain DB access in the Kafka handler. |
| DOM-20 | Table-driven tests | PASS | `script/consumer_test.go` `TestHandleEnterCommand_NonMovingOutcomesAllUnlock` uses a `map[string]ProcessResult` + `t.Run` table (diff line ~915 in review diff); `script/optable_test.go` `TestOpTable_MovingOperations`/`TestOpTable_StaticOperations` iterate slices with `t.Run`-free but table-shaped assertions. |
| DOM-21 | atlas-constants reuse | PASS | `script/optable.go:21-26` `opClass` (`opClassUnset`/`opClassStatic`/`opClassMoving`) is a local business classification (moving vs. static portal operation) with no equivalent in `libs/atlas-constants` — not a wire-level/item/inventory/job/skill classification the shared lib owns. No duplication found. |
| DOM-24 | Kafka producer stubbed in tests | PASS | `script/executor_test.go` new tests inject `fakeSagaProcessor` via `newOperationExecutorWithSaga` (diff, `newTestExecutor`) rather than exercising the real `portalsaga.Processor.Create` Kafka/REST path; `script/consumer_test.go` substitutes `newScriptProcessorFn`/`enableActionsFn`/`gateFn` package seams so no real Kafka or Redis producer path is exercised. No direct `AndEmit`/`message.Emit`/`producer.Produce` call in any new test in this package (verified by grep across `dedupe/`, `action/`, `script/`, `kafka/consumer/saga/`). |

### `saga` (atlas-saga-orchestrator)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `saga/processor.go:128` `func NewProcessor(logger logrus.FieldLogger, ctx context.Context) Processor` (pre-existing, unaffected by diff). |
| DOM-13/14 | Business logic in processor, not consumer | PASS | The character-id guard logic lives in `saga/processor.go:762-779` (`AcceptEvent`); `kafka/consumer/character/consumer.go:76-83` only supplies the `ForCharacter(e.CharacterId)` option and calls `p.AcceptEvent`/`p.StepCompleted` — no orchestration logic duplicated in the consumer. |
| DOM-24 | Kafka producer stubbed in tests | PASS | `saga/testmain_test.go:1-13` (pre-existing, package-wide) calls `producertest.InstallNoop()` in `TestMain`; the new `accept_event_test.go` and `character_extractor_test.go` tests run under this same `TestMain`, and none of them call `StepCompleted`'s emit path directly (they only call `AcceptEvent`, which does not itself emit) — but even if they did, the package-wide stub covers them. No `t.Cleanup(producer.ResetInstance)` found anywhere in the `saga` package (`grep -rn "ResetInstance"` = no hits). |
| DOM-9/interface change workflow | Mock kept in sync | PASS | `saga/mock/processor.go` `AcceptEventFunc` signature updated to `func(transactionId uuid.UUID, kind saga.EventKind, opts ...saga.AcceptOption) (saga.AcceptDecision, bool)` and the `AcceptEvent` method forwards `opts...` — matches the new `Processor.AcceptEvent` interface exactly (diff, `saga/mock/processor.go` hunk). |

## Multi-Tenancy Review

| Check | Status | Evidence |
|-------|--------|----------|
| `dedupe` gate keys are tenant-scoped via context, not a parameter | PASS | `dedupe/gate.go:98` `t := tenant.MustFromContext(ctx)`, composed into the Redis key at `dedupe/gate.go:99` via `redisKey(t, k)`, which in turn uses `atlas.TenantKey(t)` (`dedupe/gate.go:57-65`). `dedupe.Key` (`dedupe/gate.go:37-42`) deliberately has no `TenantId` field. |
| `action.Registry` writes/reads are tenant-scoped via context | PASS | `action/registry.go:55,64,70,80` all call `tenant.MustFromContext(ctx)` before touching the underlying `TenantRegistry`, for `Add`, `AddWithTTL` (new), `Get`, and `Remove` respectively. |
| No tenant-id parameter bypassing context anywhere in the diff | PASS | Grepped all changed files for a `tenantId`/`TenantId` parameter threaded manually into a lookup/store call; none found outside the standard `p.t.Id()` pattern in unrelated pre-existing code paths. |
| Character-id guard (`ForCharacter`) does not leak across tenants | PASS | `AcceptEvent` operates on a `Saga` already loaded by `p.GetById(transactionId)` (`saga/processor.go`), which is itself tenant-filtered via the standard GORM/context path; the new `ForCharacter` option only compares `characterId` values already inside that tenant-scoped saga's step payload — no new tenant-crossing surface introduced. |

## libs/atlas-redis Usage Review

| Check | Status | Evidence |
|-------|--------|----------|
| `dedupe` uses the shared lib abstraction, not raw keyed go-redis | PASS | `dedupe/gate.go:29` `atlas.NewLock(client, lockNamespace)` and `dedupe/gate.go:103` `g.lock.AcquireWithTTL(ctx, rk, enterGateTTL)` — both from `libs/atlas-redis/lock.go` (`Lock.AcquireWithTTL`, `lock.go:60-67`), which itself issues `SET NX EX` (`lock.go:62` `l.client.SetNX(ctx, rk, "1", ttl)`). No direct `goredis.Client` keyed command in `dedupe/gate.go` outside this call. |
| TTL/NX semantics match design | PASS | `AcquireWithTTL` uses `SetNX` (`lock.go:62`), which is Redis `SET key val NX EX ttl` semantics — first caller wins, TTL is the sole expiry, matching the 2s window claimed at `dedupe/gate.go:24` (`enterGateTTL = 2 * time.Second`) and the "TTL expiry IS the release" comment at `dedupe/gate.go:101-102`, verified by test `TestGate_AllowsAfterTTL` (`gate_test.go`, `mr.FastForward(enterGateTTL + time.Second)`). |
| Fail-open is real (Redis errors do not block rule evaluation) | PASS | `dedupe/gate.go:104-110`: on `err != nil` from `AcquireWithTTL`, logs a `Warn` and `return true` (allow) — confirmed by test `TestGate_FailsOpenOnRedisError` (`gate_test.go`, closes miniredis then asserts `Allow` returns `true`). |
| Redis errors are logged, not silently swallowed | **PARTIAL FAIL** | The **dedupe gate** logs correctly (`dedupe/gate.go:104-109`, `Warn` with tenant/character/portal fields). The **action registry**, however, does not — see Finding 1 below. |

## Findings

### Finding 1 (Important) — `action.Registry.AddWithTTL` and `Registry.Get` silently swallow Redis errors, with a real stuck-player failure mode now that this diff makes the registry entry load-bearing for unlock

- **File:** `services/atlas-portal-actions/atlas.com/portal/action/registry.go:63-66` (new in this diff, commit `2a6810ad7`)
  ```go
  func (r *Registry) AddWithTTL(ctx context.Context, sagaId uuid.UUID, a PendingAction, ttl time.Duration) {
  	t := tenant.MustFromContext(ctx)
  	_ = r.reg.PutWithTTL(ctx, t, sagaId, a, ttl)
  }
  ```
  and `services/atlas-portal-actions/atlas.com/portal/action/registry.go:69-76` (`Get`, pre-existing but now load-bearing):
  ```go
  func (r *Registry) Get(ctx context.Context, sagaId uuid.UUID) (PendingAction, bool) {
  	t := tenant.MustFromContext(ctx)
  	v, err := r.reg.Get(ctx, t, sagaId)
  	if err != nil {
  		return PendingAction{}, false
  	}
  	return v, true
  }
  ```
- **Failure scenario:** `script/executor.go:152-158` mints a `sagaId` and calls `action.GetRegistry().AddWithTTL(...)` *before* creating the warp saga, specifically so that — per the comment at `script/executor.go:148-151` and `script/consumer.go:130-132` — if the warp saga times out (`warpSagaTimeout = 5s`, `script/executor.go:28`), `kafka/consumer/saga/consumer.go`'s `handleStatusEventFailed` can look the entry up and call `character.EnableActions` (`kafka/consumer/saga/consumer.go:109-111`) to release the player. If the Redis `PutWithTTL` call transiently fails (network blip, Redis restart, connection pool exhaustion), `AddWithTTL` discards the error via `_ =` with **no log line, no metric** — nothing observable. Five seconds later the saga times out, `handleStatusEventFailed` calls `action.GetRegistry().Get(ctx, e.TransactionId)` (`kafka/consumer/saga/consumer.go:85`), gets `found == false` (because the registry write never landed), and takes the early-return path at `kafka/consumer/saga/consumer.go:86-89` ("Not a portal action saga, ignore") — **`character.EnableActions` is never called**. Combined with this same diff's change to stop eagerly unlocking on `CharacterMoved` (`script/consumer.go:137-140`), the player is now stuck with `m_bExclRequestSent` never cleared, with zero log trace explaining why, for as long as the client waits. Before this diff, the eager `EnableActions` call on every outcome path meant this specific registry-write failure was invisible/harmless; this diff makes it a real, silent stuck-player bug.
- **Why this is a diff-introduced regression, not merely pre-existing:** `AddWithTTL` is new code (commit `2a6810ad7`). Its sibling in the same file, `dedupe/gate.go:104-109` (same PR series, commit `4e026ce2b`), gets this exactly right — it logs a `Warn` with tenant/character/portal fields on the equivalent Redis failure. The asymmetry between the two new/touched Redis call sites in the same task is the tell: the dedupe gate's fail-open path is instrumented, the registry's is not, and the registry's failure mode is strictly worse (permanent stuck client vs. a merely-unsuppressed duplicate).
- **Fix shape (not prescribing exact code):** log a `Warn`/`Error` in `AddWithTTL` (and ideally `Add`, `Get`, `Remove`) on a non-nil error from the underlying `TenantRegistry` call, mirroring the pattern already established in `dedupe/gate.go:104-109`.

### Finding 2 (Minor) — `Registry.Add` (pre-existing, unchanged in this diff) has the identical silent-swallow shape

- **File:** `services/atlas-portal-actions/atlas.com/portal/action/registry.go:54-57`
  ```go
  func (r *Registry) Add(ctx context.Context, sagaId uuid.UUID, a PendingAction) {
  	t := tenant.MustFromContext(ctx)
  	_ = r.reg.Put(ctx, t, sagaId, a)
  }
  ```
  Used by `script/executor.go:466` (`executeStartInstanceTransport`, unchanged call shape, only a new `Kind: action.KindTransport` field added by this diff). Same silent-failure shape as Finding 1, same consequence for the pre-existing transport-failure-message path. Filed as Minor rather than folded into Finding 1 because this exact code was already present before task-184 and the transport path already relied on the same registry before this diff — the *risk* is pre-existing, not new. Called out here only because a full fix for Finding 1 should logically cover this sibling function too.

## Non-Findings Worth Recording (verified compliant, cited for completeness)

- **FR-2.3 "declared vs. dispatched" distinction is real and tested:** `script/executor.go:87-98` (`ExecuteOperations`) only sets `movedCharacter = true` *after* `ExecuteOperation` returns a nil error, so a warp that fails before its saga is created correctly reports `movedCharacter == false` and the caller still unlocks — verified by `script/executor_test.go` `TestExecuteOperations_WarpDispatchFailureReportsNotMoved` and `TestExecuteOperations_WarpParamErrorReportsNotMoved`.
- **`opTable` validation is fail-loud, not fail-silent:** `script/optable.go` `validateOpTable` + `init()` panics at process start if any table entry omits its `opClass` — a new operation type that forgets classification crashes the service at boot rather than silently defaulting to "does not move" (which would reintroduce the exact bug this task fixes). Verified by `optable_test.go` `TestValidateOpTable_RejectsUnsetClass`.
- **Character-id guard correctly no-ops for unconstrained payloads:** `saga/processor.go:767-768` only applies the mismatch check `if want := ExtractCharacterId(step); want != 0 && want != o.characterId` — a step whose payload doesn't carry a character id (`ExtractCharacterId` returns 0, `saga/character_extractor.go:82-83` default case) is left unconstrained, preserving behavior for the ~60 other `AcceptEvent` call sites that don't pass `ForCharacter`. Verified by `TestAcceptEvent_NoCharacterIdInPayloadIsUnconstrained` and `TestAcceptEvent_NoOptionIsUnconstrained`.
- **No panic risk in the character-id guard on a missing case:** `ExtractCharacterId` (`saga/character_extractor.go:5-84`) is an exhaustive type switch with a `default: return 0` (`:81-83`) — an unrecognized payload type cannot panic, it degrades to "unconstrained" (a design tradeoff, not a crash risk).
- **`WarpToSavedLocationPayload` case was correctly added to the extractor**, closing the gap that would have silently defeated the guard for that specific action: `saga/character_extractor.go:45-46`, tested by `TestExtractCharacterId_WarpToSavedLocationPayload`.
- **Interface change / mock sync workflow followed correctly:** `Processor.AcceptEvent`'s new variadic `opts ...AcceptOption` parameter (`saga/processor.go:712-713`) is mirrored exactly in `saga/mock/processor.go` (`AcceptEventFunc` field type and method both updated to accept and forward `opts ...saga.AcceptOption`).
- **No bare `go` statements introduced.** Grep of all changed non-test files for `go func`/`go <ident>` found none.
- **No DOM-21 (atlas-constants duplication).** `opClass` is a local, non-wire, non-shared-domain classification; no existing `libs/atlas-constants` type covers "does this operation move the character."
- **DOM-27 (transient DB error → 503) is N/A** — no `resource.go`/REST handler is touched by this diff in either service.
- **EXT-01..04 (external HTTP client checklist) is N/A** — no new `requests.GetRequest[T]`/`requests.PostRequest[T]` call site is introduced by this diff.
- **SEC-01..04 is N/A** — neither service is an auth/token service.

## Summary

### Blocking (must fix)
- **Finding 1** — `action.Registry.AddWithTTL` (new, `registry.go:63-66`) and `Registry.Get` (`registry.go:69-76`) silently swallow Redis errors. Given this diff makes the `PendingAction` registry entry the sole recovery path for unlocking a player whose warp saga times out (the eager `EnableActions` call this replaced is gone), a transient Redis write failure at exactly this call site now produces a silently, permanently stuck client with no log trace. Add error logging mirroring `dedupe/gate.go:104-109`'s pattern, at minimum to `AddWithTTL` and `Get`.

### Non-Blocking (should fix)
- **Finding 2** — `Registry.Add` (`registry.go:54-57`, pre-existing) has the same silent-swallow shape as Finding 1; fix in the same pass for consistency, though its risk profile is unchanged by this diff (transport path already had this exposure before task-184).
