# Task 14 Review — `atlas-saga-orchestrator` `AwardCraftedAsset` compensation

Commit range: `c96dd1f..a1d635c` (single commit `a1d635cc4`)

## Scope confirmed

Diff touches exactly the two files the brief named:

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` (+125)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go` (+212)

No other file changed. `saga/rest.go` untouched (per the standing ruling). Matches the brief's three-part wiring (reverse-walk case, `lateCompensableActions` entry, `dispatchLateInverse` case) plus tests.

## Design call 1 — routing through the plain per-action switch

**Claim to verify:** `compensateSelectGachaponReward` is the sole existing saga-type-agnostic bespoke compensator, and it is the correct model for `AwardCraftedAsset` because no dedicated `Craft`/`Maker` `SagaType` exists.

**Verified:**

- `grep -n "AwardCraftedAsset\|SagaType" libs/atlas-saga/model.go` confirms `AwardCraftedAsset` is only an `Action` constant (`model.go:257`) — there is no `Craft`/`Maker` entry in the `Type` enum anywhere in the repo.
- Read `CompensateFailedStep` in full (`saga/compensator.go:331-540`). It is a chain of `if s.SagaType() == X { return ... }` gates (CharacterCreation, PetEvolution, ItemTagUse family, PointReset, MesoSackUse, PetNameTagUse, MtsOperation, TradeTransaction, TradeStaging, NoteSend, SkillBookUse, WorldTransfer, ParcelSend, ParcelReceive), falling through to a `switch failedStep.Action()` only when the saga's type matches none of those. `SelectGachaponReward` and the new `AwardCraftedAsset` case both live in that trailing action-keyed switch (`compensator.go:511-531`), confirming the switch fires regardless of `SagaType` as long as it isn't one of the explicitly gated types — which craft sagas, having no dedicated type, never are.
- Read `compensateSelectGachaponReward` (`compensator.go:1106-1179`): it walks `s.Steps()` for `Completed` steps keyed by `step.Action()`, not gated by `s.SagaType()` anywhere in its body. This is a genuine precedent for the shape used.

**Judgment: routing is correct.** Since craft sagas are built via the generic `saga.NewBuilder()` (design §4.5.2) with no dedicated `SagaType`, the per-action switch is the only place a compensation for `AwardCraftedAsset` can be reached — a `SagaType`-gated reverse walk would need a `SagaType` to gate on, which doesn't exist. No path is silently missed: every craft saga necessarily falls through the entire `if s.SagaType() == X` chain (none of those types apply to a craft saga) to the action-keyed switch, where `case AwardCraftedAsset:` now sits.

## Design call 2 — the unreachable inner arm

**Claim to verify:** the inner `case AwardCraftedAsset:` destroy arm inside `compensateAwardCraftedAsset`'s per-completed-step switch (`compensator.go:1206-1220`) cannot fire in production because no craft sequence places a step after `AwardCraftedAsset` (design §4.5.2, always terminal).

**Verified:** grep of `AwardCraftedAsset` usage across the repo shows the only saga-construction sites for it are in `saga/handler.go` (dispatch of the step itself) and the test file; there is no code anywhere in this diff's surface that builds a craft saga with a step after `AwardCraftedAsset`. The claim is consistent with what's in-tree; nothing in this task's scope contradicts it. This is a design-level fact from §4.5.2 that this review cannot re-derive beyond the diff surface, but the implementer's characterization matches the codebase as it exists (no counter-example found).

**Judgment: keeping the arm is reasonable and low-risk** — it mirrors `AwardAsset`/`CreateAndEquipAsset`'s existing reverse-walk arms for symmetry, uses the same `RequestDestroyItem(templateId, quantity, removeAll=false)` shape already established at `compensator.go:1379-1388` (the `AwardAsset` arm) and `:1742-1743`, and is unit-tested directly (`TestCompensateAwardCraftedAssetDestroysTheAsset`, `compensator_test.go:1275-1327`). It is dead in production today by design, not by omission, and costs nothing beyond the ~15 lines it occupies.

**More importantly — the reachable path.** The actually-reachable path is `CompensateFailedStep` → `case AwardCraftedAsset: return c.compensateAwardCraftedAsset(...)`, which fires when `AwardCraftedAsset` **is** the failed step (its real, designed role as the terminal step of the craft sequence). That path is exercised end-to-end by `TestCraftSagaFullyCompensatesOnFinalStepFailure` (`compensator_test.go:1379-1478`), which builds the full mode-1 sequence — `AwardMesos` (completed, negative) → two `DestroyAssetFromSlot` (completed) → `AwardCraftedAsset` (failed) — calls `compensateAwardCraftedAsset` directly, and asserts:
- both consumed materials are re-created (`createCalls` == 2, with correct `TemplateId`s),
- the mesos charge is reversed exactly once with the correct positive magnitude (`mesosCalls[0] == mesoCost`),
- `totalDispatches == s.GetCompletedStepCount()` — the FR-3.7 no-partial-compensation invariant, verified against `GetCompletedStepCount()` (`model.go:658-666`, counts only `Completed` steps) which correctly excludes the `Failed` `AwardCraftedAsset` step itself.

This test would genuinely fail without the change (no `case AwardCraftedAsset` in `CompensateFailedStep`'s switch existed before this commit — `git diff` confirms the case is net-new at `compensator.go:530-531`), and it exercises the real reachable arm, not the unreachable symmetric one. This satisfies the review's core ask: the test passing on the unreachable arm (`TestCompensateAwardCraftedAssetDestroysTheAsset`) does not carry the weight of proving correctness — `TestCraftSagaFullyCompensatesOnFinalStepFailure` does that, and it targets the live path.

One minor gap: none of the four new tests invoke `CompensateFailedStep` itself (the public entry point that performs the `switch failedStep.Action()` dispatch) — they all call `compensateAwardCraftedAsset` or `CompensateLateStep` directly. The 2-line dispatch (`case AwardCraftedAsset: return c.compensateAwardCraftedAsset(s, failedStep)`) is trivial and low-risk, and `go build` + `go vet` would catch a typo'd case, but no test exercises the wiring from `CompensateFailedStep` all the way through. Non-blocking — the risk surface of a 2-line `case` statement is minimal and the underlying function is thoroughly tested.

## Late-compensation pairing (the failure mode this task exists to prevent)

Checked all three wiring points are present and paired, not just declared:

1. `lateCompensableActions[AwardCraftedAsset] = {}` — `compensator.go:3115-3118`.
2. `dispatchLateInverse`'s `case AwardCraftedAsset:` — `compensator.go:3266-3275`, dispatches `RequestDestroyItem(payload.CharacterId, payload.TemplateId, payload.Quantity, false)`. Payload type-asserts and returns an error (not silently continues) on mismatch, consistent with sibling arms.
3. `TestDispatchLateInverseAwardCraftedAsset` (`compensator_test.go:1339-1367`) exercises this **through `CompensateLateStep`** (the real production entry point — `c.CompensateLateStep(s, step)`, not `dispatchLateInverse` called directly), which internally calls `isLateCompensable` (consults the map from point 1) then `dispatchLateInverse` (point 2) after a `claimLateCompensation` guard (`compensator.go:3192`, read to confirm the call chain). This is a genuine end-to-end test of the late path, not just a table lookup — it would fail if either the map entry or the switch arm were missing.
4. `TestAwardCraftedAssetIsLateCompensable` additionally pins direct map membership (`compensator_test.go:1332-1336`), catching a future accidental removal even if the end-to-end test's behavior were somehow preserved by another arm.

No pairing gap found: an entry without a matching switch arm (or vice versa) would fail `TestDispatchLateInverseAwardCraftedAsset` — with an entry but no arm, `dispatchLateInverse` would hit its `default:` (need to confirm behavior, but regardless `destroyed` would stay 0 and the assertion `assert.Equal(t, 1, destroyed)` would fail); with an arm but no entry, `isLateCompensable` would return false and `compensated` would be `false`, failing `assert.True(t, compensated)`.

## Build / test verification

- `go build ./...` from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` — clean.
- `go test ./saga/... -count=1` — `ok atlas-saga-orchestrator/saga`, `ok atlas-saga-orchestrator/saga/mock`.

## Findings

None blocking.

Non-blocking:
- No test drives `CompensateFailedStep` end-to-end for the `AwardCraftedAsset` case (all four new tests call the compensator's internal methods or `CompensateLateStep` directly, skipping the outer `switch failedStep.Action()` dispatch point). Low risk given the triviality of the dispatch line, but it is the one seam in this diff not directly exercised by a test.

## Verdict rationale

Both self-flagged design calls hold up under independent verification: the routing precedent (`compensateSelectGachaponReward`) is real and genuinely saga-type-agnostic, and no `SagaType`-gated walker exists that a craft saga could fall into instead — so nothing is silently missed. The intentionally-unreachable inner arm is honestly labeled as such in both code comments and the report, is harmless, and — critically — the *reachable* path (`CompensateFailedStep` → `compensateAwardCraftedAsset` when `AwardCraftedAsset` is the failed step) is covered by a real FR-3.7 acceptance test that asserts full compensation with no partial state, and that test would fail without this commit's change. The late-compensable triad (map entry, switch arm, end-to-end test through `CompensateLateStep`) is complete and paired — the specific failure mode this task exists to prevent (a declared-but-undispatchable late inverse) is caught by `TestDispatchLateInverseAwardCraftedAsset` on the real call path, not a mock of it.
