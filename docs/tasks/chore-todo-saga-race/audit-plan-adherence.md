# Plan Audit — chore-todo-saga-race

**Plan Path:** `docs/tasks/chore-todo-saga-race/plan.md`
**Audit Date:** 2026-05-16
**Branch:** `chore/todo-saga-race` @ `d4af6ee07f4eb9e0343d76c83e6aa09bab221c2c`
**Base:** `9ae8c3fcbcc42ecc039ee4ebc996513419b3c501`
**Scope:** 12 tasks across `libs/atlas-saga`, `atlas-inventory`, `atlas-saga-orchestrator`, `atlas-character-factory`.

## Executive Summary

All 12 tasks in the plan were implemented faithfully. Every Action constant, payload, EventKind, consumer, handler, dispatch case, builder edit, and test cited in the plan exists in the diff with the specified semantics. One intentional deviation (dropping `omitempty` on `StatusEvent.TransactionId`) is recorded in a follow-up commit (`1d701522d`) as a correctness fix — `uuid.UUID` is a struct, not a pointer, so `omitempty` was a no-op and would have misled future readers. The Task 12 integration test was adapted to the real Builder/cache API (the plan explicitly authorized that adaptation). Plan-required test counts in pre-existing `processor_test.go` cases were updated for the new step. No promised work was silently skipped.

## Task Completion

| # | Task | Status | Evidence |
|---|---|---|---|
| 1 | `AwaitInventoryCreated` Action + payload + unmarshal + tests in `libs/atlas-saga` | DONE | `libs/atlas-saga/model.go:136`; `payloads.go:622-627`; `unmarshal.go:402-407`; `unmarshal_test.go:222,250` (both required test funcs present); commit `976015f4e` |
| 2 | Add `TransactionId`, `CREATION_FAILED`, body struct to `atlas-inventory` wire format | DONE (with intentional deviation) | `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go:1-33`; const + struct present; `omitempty` later dropped in commit `1d701522d` as ineffective (uuid.UUID is a struct value, never serialized as null/empty) |
| 3 | Producer signature carries `transactionId` + new `CreationFailedEventStatusProvider` | DONE | `services/atlas-inventory/atlas.com/inventory/inventory/producer.go:12,23,34` — three providers match the plan exactly |
| 4 | `Create` threads `transactionId` and emits CREATION_FAILED on rollback | DONE | `services/atlas-inventory/atlas.com/inventory/inventory/processor.go:77-118` — fresh-buffer emit via `message.Emit(producer.ProviderImpl(p.l)(p.ctx))` on `txErr` path matches the plan verbatim |
| 5 | Character consumer uses `e.TransactionId` on reactive create; deleted path retained | DONE | `services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go:48` uses `e.TransactionId`; line 60 still uses `uuid.New()` for deleted path as the plan instructed |
| 6 | `EventKindInventoryCreated`/`EventKindInventoryCreationFailed` + acceptance entry + tests | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:67-68,167`; `event_acceptance_test.go:34,165-174` (allActions updated; `TestStepAcceptsEvent_AwaitInventoryCreated` present with all three assertions) |
| 7 | Mirror inventory kafka message struct in orchestrator | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/inventory/kafka.go:1-30` matches plan (with same intentional `omitempty` drop for consistency with atlas-inventory) |
| 8 | Orchestrator inventory consumer + handlers + smoke tests | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go:18-69` (`InitConsumers`, `InitHandlers`, both event handlers with type guard + `AcceptEvent` + `StepCompleted`); `consumer_test.go:16,30` covers both type-guard tests |
| 9 | Wire consumer into `main.go` | DONE | `main.go:13` (import alias `inventoryConsumer`); `main.go:98` (InitConsumers registration); `main.go:129` (InitHandlers registration with Fatal on err) |
| 10 | `GetHandler` case + no-op `handleAwaitInventoryCreated` | DONE | `saga/handler.go:753-754` (dispatch case); `handler.go:2868-2870` (no-op implementation with godoc comment matching plan) |
| 11 | Insert `await_inventory_created` step in both character-factory builders + tests | DONE | `factory/processor.go:213` (`buildCharacterCreationSaga`); `processor.go:383` (`buildPresetCharacterCreationSaga`); `processor_test.go:1522` (`TestBuildCharacterCreationSaga_HasAwaitInventoryCreatedStep`); `processor_preset_test.go:164` (`TestBuildPresetCharacterCreationSaga_HasAwaitInventoryCreatedStep`); existing step-count assertions in `processor_test.go` (lines 60, 170, 212, 343, 429) updated to account for the new step |
| 12 | Integration test for race-condition repro | DONE (with documented adaptation) | `saga/await_inventory_created_integration_test.go:22,76` — both `TestAwaitInventoryCreated_BlocksAwardAssetUntilEvent` and `TestAwaitInventoryCreated_FailEventCompensates` present. Test uses the real Builder API (`NewBuilder().SetTransactionId(...).AddStep(...).Build()`) and `GetCache().Put(ctx, sg)` rather than the literal-struct construction shown in the plan — the plan explicitly authorized this adaptation in Task 12 Step 2 |

**Completion Rate:** 12 / 12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviations From Plan (all intentional, all evidence-backed)

1. **`omitempty` dropped from `StatusEvent.TransactionId`** in both atlas-inventory (`kafka.go:16`) and the orchestrator mirror (`kafka.go:16`). Commit `1d701522d` justifies this: `uuid.UUID` is `[16]byte`, so encoding/json never treats it as a zero value to omit — the tag was misleading. Wire compatibility is preserved (the field is always serialized; consumers ignore `uuid.Nil`).

2. **Task 12 test setup uses Builder + GetCache().Put** instead of the literal `sharedsaga.Saga{...}` struct and `p.Create(sg)` shown in the plan. The plan Step 2 explicitly authorizes this: *"If the test surfaces assumptions about `NewProcessor`/`Create`/`AcceptEvent`/`StepCompleted` signatures that don't match this codebase, read those methods … and adjust the test setup — DO NOT change the production code."* The test still asserts every behavior the plan required (await step blocks AwardAsset until INVENTORY_CREATED arrives; result-forwarding substitutes CharacterId=42 into both await and award payloads; failure path puts the saga into Failing state).

3. **Local re-exports added in two services** beyond the plan's literal file list:
   - `services/atlas-character-factory/atlas.com/character-factory/saga/model.go` — adds `AwaitInventoryCreatedPayload` type alias and `AwaitInventoryCreated` const re-export, required by `factory/processor.go` which references `saga.AwaitInventoryCreated` (the local re-export package).
   - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — same re-exports, required by `await_inventory_created_integration_test.go` (in `package saga`) and by the result-forwarding `case AwaitInventoryCreatedPayload:` arm in `processor.go:1458`.

   These are mechanical consequences of the project's "re-export sharedsaga in the service's local saga package" pattern; the plan didn't enumerate them but they're trivially necessary for the named symbols to resolve.

4. **`forwardCharacterCreationResult` learned about `AwaitInventoryCreatedPayload`** (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1458-1460`). Not enumerated in the plan, but required for the design's contract ("the result-forward already injects characterId into subsequent steps' payloads via the existing forwarding mechanism") to actually apply to the new payload type. The Task 12 integration test (`assert.Equal(t, uint32(42), awaitPl.CharacterId)`) covers this.

## Build & Test Results

Dispatcher reported the following clean (not re-run during audit):
- `go test -race`, `go vet`, `go build` in `libs/atlas-saga`, `atlas-inventory`, `atlas-saga-orchestrator`, `atlas-character-factory`.
- `docker build` clean for `atlas-inventory`, `atlas-saga-orchestrator`, `atlas-character-factory`.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All plan tasks are implemented with evidence; deviations are either explicitly authorized by the plan or are mechanical necessities of the codebase's existing patterns and are covered by tests.
