# Plan Audit — task-228-water-of-life-pet-revive

**Plan Path:** docs/tasks/task-228-water-of-life-pet-revive/plan.md
**Audit Date:** 2026-08-14
**Branch:** task-228-water-of-life-pet-revive
**Merge base:** 314ff8ad0 · **Head:** 6faf71aff (23 commits)

## Executive Summary

All 15 tasks are implemented and verifiable by file:line evidence in the branch diff; nothing was silently skipped or stubbed. Every affected Go module (`libs/atlas-constants`, `libs/atlas-saga`, `libs/atlas-packet`, `atlas-asset-expiration`, `atlas-data`, `atlas-pets`, `atlas-inventory`, `atlas-saga-orchestrator`, `atlas-channel`) builds and tests clean (`go build ./...` / `go test ./... -count=1`), and every project guard relevant to this branch (template opcode order, duplicate binding, movement types, redis-key, goroutine, buff-duration, skill-job-id, trade/mist/npc-shop contract mirrors, `tools/lint.sh --check`) passes with exit 0. The two controller-ruled deviations (matrix name `PetWaterOfLife`, REST path `data/cash/items/%d`) are correctly reflected in the code, not the plan's illustrative text. All eight logged deferred minors are present exactly as described and are cosmetic/logging-only — none violate a Global Constraint or leave a functional gap.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `ClassificationWaterOfLife` constant | DONE | `libs/atlas-constants/item/constants.go:88` adds `Classification(518)`; `constants_test.go:611-618` round-trips `GetClassification(5180000)`; bare literal replaced at `services/atlas-channel/.../character_cash_item_use.go:1051` (`category == item.ClassificationWaterOfLife`) — `grep -n '518'` on that file returns nothing |
| 2 | Pets survive expiration (FR-1.1–1.4) | DONE | `expiration/checker.go:28-35` `IsReapable` gates by classification, not id; applied at all 3 sweep sites `character/processor.go:72,96,116`; negative-control test `processor_reap_test.go:177` (`TestCheckAndExpireEmitsForExpiredNonPetRequiresStubs`) proves the stub URLs aren't a false pass |
| 3 | `atlas-data` parses `info/life` | DONE | `cash/reader.go:81` reads `life` days; `cash/rest.go:55-59` documents units; `cash/reader_test.go` covers it; `go test ./cash/...` green |
| 4 | `WATER_OF_LIFE` serverbound codec | DONE | `libs/atlas-packet/pet/serverbound/water_of_life.go` — empty `Encode`/`Decode`, `packet-audit:fname` comment, no version gates (matches D3); round-trip test with `packet-audit:verify` markers for all 5 versions in `water_of_life_test.go` |
| 5 | Route handler in 5 templates | DONE | `WaterOfLifeHandle` present in `template_gms_{83,84,87,92,95}_1.json`, each with non-empty `LoggedInValidator`; absent from all 6 n-a templates (verified by grep); `template-opcode-order-guard.sh` and `template-duplicate-binding-guard.sh` both exit 0 |
| 6 | `libs/atlas-saga` `revive_pet` action | DONE | `model.go:41-43,101` (`PetRevive` Type, `RevivePet` Action), `payloads.go:289-297` (`RevivePetPayload`, no expiration field per D6), `unmarshal.go:186-191` unmarshal arm, `unmarshal_test.go` covers it |
| 7 | Expired pet not summonable (FR-1.5) | DONE | `pet/processor.go:429-434` `ErrPetExpired`; gate at `processor.go:452` inside `Spawn`, beside `ErrTooManySpawnedPets`/`ErrNeedMultiPetSkill` |
| 8 | `atlas-pets` cash-data client + contracts | DONE | `data/cash/{model,builder,rest,requests,processor}.go` + `mock/processor.go` all present; `ResetPetExpirationCommandBody`/`ChangeTemplateCommandBody` field-for-field identical to atlas-inventory's copy (verified by diff of both files from `package` clause) |
| 9 | `atlas-pets` `REVIVE` command | DONE | `pet/processor.go:996-1049` `Revive` implements D7's 3-state table exactly (redelivery re-cascade at line 1013-1018, live-pet rejection at 1022-1025, else revive); every rejection buffers `REVIVE_FAILED` via `mb.Put(...)` then `return` (not a Go `error`) — trap avoided; `updateOnRevive(p.db)` at line 1039 uses the tx-bound `p.db` set by `p.With(WithTransaction(tx))` in `ReviveAndEmit`, not a fresh pool connection; `pet/administrator.go:114-130` writes only `expiration` + `revive_transaction_id`, deliberately not reusing `updateOnEvolve` |
| 10 | `atlas-inventory` `RESET_PET_EXPIRATION` | DONE | `compartment/processor.go:2144-2183` re-derives `serverCap` from its own `cashProcessor.GetById`, rejects (not clamps) beyond it, locks + walks assets for `IsPet() && PetId()==petId` exactly as `ChangeTemplate` does; consumer arm `kafka/consumer/compartment/consumer.go:435-450` |
| 11 | `atlas-saga-orchestrator` `revive_pet` step | DONE | `event_acceptance.go:56-57,157,369-370` (EventKind consts, acceptance table, outcome table); `event_acceptance_test.go:22` includes `sharedsaga.RevivePet` in the hand-maintained `allActions` list — coverage test would fail without it; `handler.go:1419-1430` `handleRevivePet`; `model.go:42,99,268,1413-1414` aliases + local payload switch; consumer arms `kafka/consumer/pet/consumer.go:38,41,102,125` |
| 12 | `atlas-channel` `WaterOfLifeHandle` handler | DONE | `socket/handler/water_of_life.go` — top-level handler (not a `character_cash_item_use` arm), `findWaterOfLife` resolves by classification + lowest slot, `selectRevivableTarget` picks latest-expired with pet-id tiebreak, pre-flight `cd.Life == 0` check, two-step saga (`destroy_water_of_life` → `revive_pet`); `main.go:924` registers `handlerMap[petsb.WaterOfLifeHandle]`; **no `EnableActions` call anywhere in the file** (grep confirms only the doc comment mentions it) |
| 13 | Async revive-failure announce | DONE | `kafka/consumer/pet/consumer.go:477-498` `handleReviveFailed` re-announces `WaterOfLifeFailedMessage` via `IfPresentByCharacterId`; registered at consumer.go:98 |
| 14 | Promote 5 matrix cells | DONE | `docs/packets/audits/status.json:19121-19172` — `packet: "pet/serverbound/PetWaterOfLife"` (correct per ruled deviation #1), `gms_v{83,84,87,92,95}` all `"state": "verified"`, all 6 non-applicable cells `"state": "n-a", "opcode": -1`; `STATUS.md:666` mirrors it |
| 15 | Full verification sweep | DONE | See Build & Test Results below; all guards green |

**Completion Rate:** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task is `SKIPPED`, `PARTIAL`, or `DEFERRED`.

## Deferred-Minors Triage

All eight items pre-logged in the task brief were re-verified present in the current diff. Assessment: **all ship as-is** — none is a functional defect or a Global Constraint violation; all are documentation/logging granularity gaps.

| # | Item | Verdict | Reasoning |
|---|------|---------|-----------|
| 1 | `checker.go:322` (now `:32`) names literal `5180000` in a doc comment | SHIP | Prose only; the code itself uses `item.GetClassification` — no literal in logic |
| 2 | T2 per-sweep coverage via negative control, not asserted in code | SHIP | `TestCheckAndExpireEmitsForExpiredNonPetRequiresStubs` is a legitimate, arguably stronger, proof pattern than an in-code assertion |
| 3 | `data/cash/mock/processor.go:19` unconfigured `GetByIdFunc` returns zero-value success | SHIP | Mock-only; `Life()==0` zero-value is caught by the real code's `cd.Life() == 0` reject path, so a test that forgets to configure the mock fails loudly rather than silently passing |
| 4 | `pet/processor_test.go:1303` stub later extended by T9 | SHIP | Confirmed extended — `fakeInventoryProcessor.ResetPetExpiration` at `processor_test.go:1311` is used by the T9 cascade tests |
| 5 | T9 rejection text doesn't distinguish "never revived" vs "revived once, still alive" | SHIP | Cosmetic; both cases correctly reject via the same `"pet has not dried up"` message and both correctly refund via the saga compensator — behavior is correct, only the string is coarse |
| 6 | `data/cash/{rest,model}.go` package doc comments not updated | SHIP | The `Life` field itself carries a full doc comment (`rest.go:55-59`); only the package-level comment is stale |
| 7 | `TestResetPetExpirationRejectsOverCap` asserts only `err != nil` | SHIP | Weaker than ideal but still exercises the reject-not-clamp path and asserts the asset is left unchanged (per the test's own doc comment) |
| 8 | `handleReviveFailed` logs `Reason` but not `TransactionId` | SHIP | Confirmed at `kafka/consumer/pet/consumer.go:492` — `l.Warnf` omits `e.Body.TransactionId`. Cosmetic logging gap only; does not affect the refund path (the saga, not this log line, drives the compensation) |

No new findings beyond this pre-logged list were discovered during the sweep.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-constants` | PASS | PASS | via downstream service builds; `item/constants_test.go` covers the new constant |
| `libs/atlas-saga` | PASS | PASS | via downstream builds; `unmarshal_test.go` covers `RevivePet` |
| `libs/atlas-packet` | PASS | PASS | via downstream builds; `water_of_life_test.go` round-trip |
| `atlas-asset-expiration` | PASS | PASS | `go test ./... -count=1` all green (expiration, character, inventory, storage packages) |
| `atlas-data` | PASS | PASS | `go build ./...` clean; `go test ./cash/...` green |
| `atlas-pets` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all green including `pet` (4.07s) and `data/cash` |
| `atlas-inventory` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all green including `compartment` and `kafka/consumer/compartment` |
| `atlas-saga-orchestrator` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all green including `saga` |
| `atlas-channel` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all green including `socket/handler` (0.88s) |

**Guards (repo root):**

| Guard | Result |
|---|---|
| `tools/template-opcode-order-guard.sh` | OK — 22 template arrays ascending |
| `tools/template-duplicate-binding-guard.sh` | OK — no duplicate bindings |
| `tools/template-movement-types-guard.sh` | OK — 54 move handlers across 11 templates valid |
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/buff-duration-guard.sh` | exit 0 |
| `tools/skill-job-id-guard.sh` | clean (14 divergent consts checked) |
| `tools/trade-contract-mirror-guard.sh` | OK |
| `tools/mist-contract-mirror-guard.sh` | OK |
| `tools/npc-shop-contract-mirror-guard.sh` | OK — all three copies identical |
| `tools/lint.sh --check` | OK (0 errors; 9 pre-existing atlas-ui warnings unrelated to this task) |

No `go.mod` or `go.work` changes on the branch (`git diff 314ff8ad0...HEAD -- '**/go.mod' 'go.work'` empty) — matches the plan's Global Constraint; `docker buildx bake` per-service is therefore not required by CLAUDE.md item 4's own trigger condition (no `go.mod` touched).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required before merge. Optional, non-blocking polish (do not block PR):

1. Add `TransactionId` to the `l.Warnf` call in `handleReviveFailed` (`services/atlas-channel/.../kafka/consumer/pet/consumer.go:492`) for easier redelivery tracing.
2. Tighten `TestResetPetExpirationRejectsOverCap` to assert the specific cap-rejection error string rather than `err != nil`.
3. Refresh the stale package-level doc comments in `atlas-inventory/data/cash/{rest,model}.go`.

## Backend Guidelines Review

- **Service Path(s):** libs/atlas-constants, libs/atlas-packet, libs/atlas-saga, services/atlas-asset-expiration, services/atlas-data, services/atlas-pets, services/atlas-inventory, services/atlas-saga-orchestrator, services/atlas-channel
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-14
- **Build:** PASS (all 9 changed modules, `go build ./...`)
- **Tests:** PASS (all 9 changed modules, `go test ./... -count=1`, zero `FAIL`/`panic` across the full output)
- **Overall:** NEEDS-WORK (two Important structural findings; build/tests are clean)

### Build & Test Results

`go build ./...` and `go test ./... -count=1` were run per changed module (libs/atlas-constants, libs/atlas-packet, libs/atlas-saga, atlas-asset-expiration, atlas-data, atlas-pets, atlas-inventory, atlas-saga-orchestrator, atlas-channel). All clean, no `FAIL` or `panic` lines in any module's test output. `tools/goroutine-guard.sh`, `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`, `tools/redis-key-guard.sh`, `tools/buff-duration-guard.sh`, `tools/skill-job-id-guard.sh` all exit 0 against the diff.

### File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor methods live in `processor.go` (or `processor_<group>.go`) | **FAIL** | `services/atlas-pets/atlas.com/pets/inventory/command.go:17-20,39-49` defines `(p *ProcessorImpl) ChangeTemplate` and `(p *ProcessorImpl) ResetPetExpiration` — both `Processor` interface methods declared at `inventory/processor.go:18-19` — but implemented in a bare-topic-named file (`command.go`), not `processor.go`. `ResetPetExpiration` is new code added by this task (task-228 extended a pre-existing violation by adding a second method to the same wrongly-named file). Fix: move both methods into `processor.go`, and move the two `*CommandProvider` functions (`changeTemplateCommandProvider`, `resetPetExpirationCommandProvider`) into a `producer.go` matching the sibling `pet/producer.go` convention used elsewhere in this same service for exactly this kind of Kafka-message-builder function. |
| FILE-02/03/04/05/06 | rest.go / requests.go / entity.go / builder.go / model.go / no catch-all file | PASS | `services/atlas-pets/atlas.com/pets/data/cash/{model,builder,processor,requests,rest}.go` and `services/atlas-inventory/atlas.com/inventory/data/cash/{model,builder,rest}.go` each place `RestModel`+`Transform`/`Extract`, `Processor`, and request funcs in their designated files; no `<pkg>.go` catch-all introduced |

### Domain Checklist Results — `atlas-pets/pet` (domain package, `model.go` present, materially touched: model.go, builder.go, entity.go, administrator.go, processor.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-02 | `ToEntity()` method on `Model` | **FAIL** | `grep -rn "ToEntity" services/atlas-pets/atlas.com/pets/pet/*.go` returns nothing. `pet/administrator.go:15-38`'s `create()` builds `&Entity{...}` manually field-by-field instead of calling `m.ToEntity()`. Pre-existing gap (not introduced by this diff), but the package is in-scope for this review since model.go/builder.go/entity.go/administrator.go were all modified to add the revive fields, and the new `reviveTransactionId` field was added to the manual entity-construction path (`administrator.go`'s `create()` does not even set `ReviveTransactionId` — acceptable since it's nil-by-default on creation, but underscores that the manual-construction path has no single source of truth). |
| DOM-03 | `Make(Entity) (Model, error)` | PASS | `pet/entity.go:44-60` |
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` validation | PASS | `pet/builder.go:108-146` validates templateId/ownerId/name/level/fullness/slot before returning |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `pet/processor.go:117` `NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-21 | No literal 500/518; reuse atlas-constants | PASS | `libs/atlas-constants/item/constants.go:88` adds `ClassificationWaterOfLife = Classification(518)`; `expiration/checker.go:34` uses `item.ClassificationPet` (pre-existing `Classification(500)`); `character_cash_item_use.go:1053` bare-literal `518` replaced with `item.ClassificationWaterOfLife`; `water_of_life.go:154` resolves by `item.GetClassification(...) != item.ClassificationWaterOfLife`. Every remaining `500`/`518` occurrence in the diff is inside a comment, a test literal, or the constant declaration itself — `grep` swept and confirmed. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `pet/processor_test.go` uses the `outbox.EmitProvider` pattern exclusively for `*AndEmit` (writes to an outbox table via `tx`, never calls the real Kafka producer directly), so no `producertest.InstallNoop()` is needed there. The one place in this diff that DOES call the real `producer.ProviderImpl` directly — `atlas-saga-orchestrator/pet/processor.go:37` `p.p = producer.ProviderImpl(l)(ctx)`, exercised transitively by `saga/handler.go:1419` `handleRevivePet` — has its stub at `saga-orchestrator/saga/testmain_test.go:7-11` (pre-existing `TestMain`, no `t.Cleanup(producer.ResetInstance)` reverting it). `saga-orchestrator/kafka/consumer/pet/testmain_test.go` (new file) also correctly calls `producertest.InstallNoop()`. |

### Transaction Correctness — `pet.ProcessorImpl.Revive` (specifically requested)

| Check | Status | Evidence |
|-------|--------|----------|
| No second pooled DB connection opened from inside the open transaction | PASS | `ReviveAndEmit` (`pet/processor.go:973-979`) opens the transaction once via `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)` and calls `p.With(WithTransaction(tx)).Revive(mb)(...)`. `Revive` itself (`:996-1056`) never calls `database.ExecuteTransaction` again — `p.GetById(petId)` and `updateOnRevive(p.db)(...)` (`:1043`, `administrator.go:116-132`) both resolve through `p.db`, which is the SAME `tx` bound by `WithTransaction`. No nested transaction, no second connection requested. |
| Minor: external HTTP call held open inside the DB transaction | Minor finding | `pet/processor.go:1025` `cd, err := p.cdp.GetById(sourceTemplateId)` is a synchronous HTTP round-trip to atlas-data, executed AFTER the outer transaction is already open (it is inside `Revive`, which only ever runs nested in `ReviveAndEmit`'s `ExecuteTransaction` closure). This holds the pooled DB connection for the duration of an external network call. Compare `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:2148` (`ResetPetExpiration`), which fetches the equivalent cash data BEFORE opening its own transaction at `:2167` — the safer ordering used on the sibling side of the same feature. Not a deadlock (no second connection is acquired), but a pool-hold/latency risk if atlas-data is slow, most consequential at the low pool sizes this codebase has hit before (see `bug_tx_scoped_reader_not_rebound_deadlocks.md`). Fix: move the `p.cdp.GetById(sourceTemplateId)` call in front of the `Revive` closure (e.g., resolve `cd` in `ReviveAndEmit` before calling `database.ExecuteTransaction`, or restructure `Revive` to take the already-resolved cash data), mirroring atlas-inventory's ordering. |

### Idempotency — REVIVE_TRANSACTION_ID gate (specifically requested)

| Check | Status | Evidence |
|-------|--------|----------|
| Redelivery vs. genuine second request | PASS | `pet/processor.go:1013-1019`: redelivery (same `transactionId` stored on the row) re-buffers the cascade using the STORED `pe.Expiration()` (not a recomputed value) and returns without a write. `:1020-1023`: a genuinely new attempt on an already-live pet (different/no matching stored transactionId, `Expiration()` now in the future) hits `"pet has not dried up"` and is rejected — refunded via the saga's generic `DestroyAsset` reverse-walk compensator (`saga/compensator.go:1463-1476`, no bespoke pet-revive compensator needed). Covered by `pet/processor_revive_test.go:221` (`TestReviveRedeliveryIsIdempotent`) and `:271` (`TestReviveRejectsSecondDifferentTransactionOnLivePet`). |

### Security / Trust Boundary (specifically requested)

| Check | Status | Evidence |
|-------|--------|----------|
| atlas-inventory re-derives its own expiration cap, doesn't trust the channel/pets-computed value | PASS | `compartment/processor.go:2144-2161` `ResetPetExpiration` independently calls `p.cashProcessor.GetById(sourceTemplateId)`, computes `serverCap := time.Now().Add(...)`, and REJECTS (`errors.New(...)`, not clamps) any requested expiration beyond it. Covered by `processor_reset_pet_expiration_test.go:164` (`TestResetPetExpirationRejectsOverCap`). |
| Ownership enforced at both layers | PASS | `pet/processor.go:1005-1008` rejects `pe.OwnerId() != actorId`; `compartment/processor.go:2177-2182` scopes the pet-asset walk to the given `characterId`'s own cash compartment, so a `petId` not owned by that character resolves as "not found," not a cross-character write. `actorId`/`characterId` originate server-side from the authenticated session (`water_of_life.go:60-61` `s.CharacterId()`), never from client-controlled packet fields (the packet body is empty). |
| No `EnableActions` / exclusive-request latch on this path | PASS | `water_of_life.go:45-49` — comment plus code inspection confirm no call to `SetExclRequestSent`/`EnableActions` anywhere in the handler; `grep` for `EnableActions` in the file returns only the doc comment. |

### Other Observations (Minor)

- `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life.go:107` discards the error from `saga.NewProcessor(l, ctx).Create(saga.Saga{...})` via `_ = ...`. This matches an existing repo-wide idiom (10 other call sites in the same package do the same, e.g. `character_cash_item_use.go:170,228,300,400,494`), so it is not a regression introduced by this task, but per the "prevalence is not compliance" rule it is recorded: if saga creation fails here, the player's Water of Life is not consumed (no step ran yet) but also receives no rejection message — a silent no-op rather than a silent loss. Low impact, but worth a `l.WithError(err).Warnf(...)` at minimum for observability.
- `services/atlas-channel/atlas.com/channel/socket/handler/pet_spawn.go:38,40` — `pet.NewProcessor(l, ctx).Spawn(...)`'s new `ErrPetExpired` return value is silently dropped (`_ = ...`), same as the two pre-existing spawn errors (`ErrTooManySpawnedPets`, `ErrNeedMultiPetSkill`). A player double-clicking a dried-up pet in their cash inventory gets no client-side message explaining why nothing happened. Pre-existing pattern, not a regression, but the new error path inherits the same UX gap.

### Summary

#### Blocking (must fix)

- FILE-01: `services/atlas-pets/atlas.com/pets/inventory/command.go:17-20,39-49` — `ChangeTemplate`/`ResetPetExpiration` `ProcessorImpl` methods belong in `processor.go`, not a bare-topic-named file. Move the two Provider-builder functions to a `producer.go` split, matching the sibling `pet/producer.go` convention in this service.
- DOM-02: `services/atlas-pets/atlas.com/pets/pet/` has no `ToEntity()` method on `Model`; `administrator.go`'s `create()` hand-builds the `Entity` struct instead. Pre-existing, but the package is materially touched by this diff (new `reviveTransactionId` field flows through the same manual-construction path) — worth closing while the file is open rather than deferring again.

#### Non-Blocking (should fix)

- `pet/processor.go:1025` — move the `p.cdp.GetById(sourceTemplateId)` HTTP call ahead of the open DB transaction in `Revive`, mirroring atlas-inventory's `ResetPetExpiration` ordering, to avoid holding a pooled connection across a cross-service network round-trip.
- `water_of_life.go:107` — log (don't just discard) a `saga.Create` failure for the Water of Life saga, for parity with `note_send.go:87`'s pattern of checking the error.
- `pet_spawn.go:38,40` — consider surfacing `ErrPetExpired` (and the two pre-existing spawn errors) to the player as a system message, rather than a silent no-op.
