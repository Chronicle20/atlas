# Plan Audit — task-205-player-trade, meso-custody parity pass

**Plan Path:** docs/tasks/task-205-player-trade/parity-plan.md
**Audit Date:** 2026-08-11
**Branch:** task-205-player-trade
**Commit Range Audited:** `efec8332a~1..HEAD` (HEAD = `349748892`)
**Base Branch:** main

This audit covers the *parity-plan.md* pass only (prerequisites P1/P2 + Tasks
1–7 in that document), not the earlier `audit-plan-adherence.md` pass, which
is left untouched.

## Executive Summary

All seven tasks and both prerequisites were implemented, and every specific,
checkable claim in the plan's write-ups (row-lock vs `RETURNING`, relative
SQL adds vs read-then-assign, signed columns, named-test existence, mutation
behavior of the guards) held up against direct code reads and independent
mutation tests. `go build`/`go vet`/`go test -race` are clean in all three
changed modules (`atlas-trades`, `atlas-saga-orchestrator`,
`libs/atlas-database`); `docker buildx bake atlas-trades` succeeds; the
repo's redis-key, goroutine, buff-duration, skill-job-id, and
trade-contract-mirror guards are all clean.

One real, unflagged gap survives: **Task 4's own stated completeness claim
is false.** The plan says "every `SagaType()` comparison in the orchestrator
was checked," but `reverseWalkSagaTypes` (the list that drives the new
timeout switch) omits seven saga types — `PetEvolution`, `ItemTagUse`,
`SealingLockUse`, `IncubatorUse`, `PointReset`, `NoteSend`,
`SkillBookUse` — that `compensator.go`'s own step-failure dispatcher (the
same file, same pattern) already routes to bespoke reverse-walk
compensators. Every saga type gets the unconditional 30s timeout backstop
(`saga/processor.go:325`), so each of those seven has the exact "step
completes, next step stalls past 30s, nothing rolls back" failure mode that
this whole pass exists to close for `TradeStaging` — just not exercised by
task-205's own scenario. This is scope the plan's own membership rule
(stated in its `reverseWalkSagaTypes` doc comment) says belongs in the list,
and it doesn't ship. It is not blocking for a trade-focused PR, but the
"every comparison was checked" claim in the plan's Task 4 write-up
overstates what was actually verified — the audit only checked whether
`TradeStaging` was handled everywhere, not whether the class of bug was
closed for every other saga type already exposed to it.

A second, minor finding: the plan's mid-session working state shows a
concurrent, uncommitted modification landing in this same worktree during
this audit (`escrow/administrator.go`, `escrow/model.go`, plus a new
untracked `audit-backend-parity.md`) — see "Operational note" below. It does
not affect the audited commit range (all commits in `efec8332a~1..HEAD` are
already landed), but it means the working tree was not byte-for-byte
identical to `HEAD` for the whole duration of this audit.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| P1 | Isolation level = READ COMMITTED, no override | DONE | `libs/atlas-database/transaction.go:9-14` (`ExecuteTransaction` passes no `*sql.TxOptions`), `connection.go:54-56` (DSN has no `options=`). Tree-wide grep for `TxOptions\|Serializable\|default_transaction_isolation` outside the plan doc: zero matches. Confirmed independently. |
| P2 | `TRADE_CONFIRM` not excl-gated (v83 IDB) | DONE (unverifiable by re-derivation) | Client reverse-engineering claim; not independently re-derivable without IDA access to the v83 binary. Internally consistent: Task 3 gates at `settlementPayload` (settle time), not at confirm, exactly as P2's consequence requires — `trade/settlement.go:563-566` calls the gate inside the settlement loop, and grepping the confirm path (`settlement.go:176-271`) shows no meso-custody call there. |
| 1 | Defence A — net in-flight stake before submitting | DONE | `MesoStakeEntity` child table (`escrow/entity.go:269-283`) replaces the pending_* columns, migration backfills armed slots (`entity.go:337-397`). `CommitMesoStake` claims by `Delete` (CAS) + relative `Updates(... "amount": gorm.Expr("amount + ?", stake.Delta))` (`administrator.go:403-431`). `EffectiveMesoByOwner` = committed + `InFlightMesoDelta` (`provider.go:114-141`). `MesoEntity.Amount` is `int64`, signed (`entity.go:221`). `discardOrphanedMeso` discharges via relative `amount - ?` (`processor.go:1592-1599` → `administrator.go:519-527`), not read-then-assign. All six named tests exist verbatim; `TestArmMesoStakeSupersedesPriorStake` is genuinely gone (zero matches), not renamed with stale assertions. **Mutation-verified**: reverting `CommitMesoStake`'s relative add to an absolute assign makes `TestConcurrentMesoStakesConserve` fail with `"player was debited 1500 across two stakes; escrow holds 500 — 1000 meso destroyed"`; restored, green. |
| 2 | Defence B — durable claim latch for meso return | DONE | `ClaimMesoForReturn` (`administrator.go:479-507`) uses `SELECT ... FOR UPDATE` (row lock, not `RETURNING`), zeroes inside the same tx. All three unwind sites (`emitUnwind` settlement.go:713, `unwindRecord` settlement.go:1149, `unwindStranded` settlement.go:1511) call it. `retireClaimedMesos` (settlement.go:1382-1409) no longer zeroes — only the conditional `DeleteResolvedMeso`. Named tests exist; `TestClaimMesoForReturnIsExclusive` proves exclusivity via two *sequential* claims (honestly, per the plan's own stated sqlite `MaxOpenConns(1)` limitation — `libs/atlas-database/databasetest/testdb.go:37`), and a real double-refund would fail it. |
| 3 | Defence C — settlement-time custody check (BLOCKING per P2) | DONE | `assertMesoCustodyAgrees` (settlement.go:598-614) called once per side inside `OrderedParticipants()` loop (settlement.go:563-566); refuses on in-flight != 0 OR committed != what's about to be delivered. Confirmed gated at settle, not confirm. `TestSettlementRefusesWhileAMesoStakeIsInFlight` is table-driven both directions; **mutation-verified** — disabling the guard produces both "a settlement saga was submitted with -1500 meso still in flight..." and the +1500 case; restored, green. `TestSettlementSuccessKeepsARowWhoseStakeIsStillInFlight` still exists and still pins the DISCHARGE behavior as claimed. |
| 4 | Register `TradeStaging` in the timeout reverse-walk | **PARTIAL** (working code DONE; plan's completeness claim OVERSTATED) | `dispatchTimeoutRollbacks` is one `switch` over `reverseWalkSagaTypes` (`timer.go:159-200`) with `TradeStaging` now included. **Mutation-verified**: removing the `TradeStaging` case fails both `TestTradeStagingTimeoutDispatchesItsReverseWalk` and `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout` with the stated "compartment -1, escrow +0, item destroyed" message; restored, green. BUT: the plan's claim "every `SagaType()` comparison in the orchestrator was checked" is true only for `TradeStaging`'s own coverage, not for the class of bug. `compensator.go`'s bespoke step-failure dispatcher (`compensator.go:252-320`) also routes `PetEvolution` (259), `ItemTagUse`/`SealingLockUse`/`IncubatorUse` (267), `PointReset` (276), `NoteSend` (313), `SkillBookUse` (320) to bespoke reverse-walk compensators, none of which are in `reverseWalkSagaTypes` or the timeout switch — by the list's own doc-comment membership rule ("belongs here exactly when CompensateFailedStep routes it to a bespoke compensator"), all seven qualify and are missing. Every saga type gets the unconditional 30s timeout backstop (`saga/processor.go:325`), so each of these seven has the identical unrolled-back-timeout failure mode `TradeStaging` had. Additionally, `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout` only iterates `reverseWalkSagaTypes` itself, so it cannot catch a type omitted from the list (as these seven currently are) — only a type present in the list but missing from the switch. |
| 5 | Route a failed `trade_unwind` | DONE | `ItemEntity.ReturningTxId` (entity.go:165) stamped by `ClaimItemForReturn` (administrator.go:125-128). `MesoRefundEntity`/`trade_escrow_meso_refunds` (entity.go:41,303-319) written via `RecordMesoRefund` in the claiming tx. `UnwindFailed` (settlement.go:1686-1703) releases item latches and restores meso via relative `OnConflict...amount + r.Amount` (row re-created if retired). `UnwindSucceeded` (settlement.go:1715-1721) discards refund records. Fourth probe in `kafka/consumer/saga/consumer.go` confirmed ordered before the settlement probe in both `handleSagaFailed` (155 before 164) and `handleSagaCompleted` (102 before 111). Named tests read as claimed, including redelivery idempotency (`TestAFailedUnwindLeavesBothColumnsRecoverable`) and per-tx scoping (`TestReleaseItemReturnClaimsUnlatchesOnlyItsOwnTransaction`). |
| 6 | Fence `RestoreItem` against redelivery | DONE | Exact condition confirmed: `Where("tenant_id = ? AND id = ? AND (returning_tx_id IS NULL OR returning_tx_id = ?)", tenantId, id, txId)` (administrator.go:273) — not a bare "is claimed" check, matching the plan's own stated correction of its first (wrong) attempt. The Task-5-introduced latent bug (`returnOrphanedStage` stamping the claim with one id and submitting the unwind with a different one) is fixed to a single shared `unwindTxId` (processor.go:1136-1157), confirmed against the commit diff. Three named tests cover the three distinct cases (same-saga restores, different-saga inert, unclaimed release unaffected). **Mutation-verified**: dropping the tx-id predicate from the `WHERE` clause makes `TestRestoreItemCannotResurrectAReturnedRow` fail with "a redelivered restore resurrected a row whose item was already returned..."; restored, green. |
| 7 | Re-run both audits, close findings, full gate | DONE | F-8 guard (a negative-delta payout while anything is in-flight) lives in `trade/processor.go`'s `addMeso` (~line 1359-1362) — note the plan's prose never actually names the file, so this isn't a plan inaccuracy, just worth pinning down. **Mutation-verified**: disabling the guard makes `TestLoweringTheMesoBoxNeverPaysOutAgainstAnUnconfirmedStake` fail with "a credit of 800 was submitted while a debit was still in flight"; restored, green. `DispatchTradeStagingRollbacks`'s new `AwardMesos` arm (compensator.go:2473-2495) matches the existing late-completion inverse (compensator.go:2273-2279) call-for-call; `TestStagingReverseWalkReversesItsMesoDebit` asserts the actual reversed amount, not just that a function fired. The `01106e56e` cache-isolation fix is a genuine race fix, not a cover-up: reverting it and running `go test -race -count=1 ./saga/...` 4× reproduced a real detector failure once; with the fix applied, 8/8 clean. `trade_unwind`'s `SagaType` (`TradeTransaction`, not a distinct type) and `DeleteResolvedMeso`'s `amount = 0` (not `<= 0`) both confirmed as described. |

**Completion Rate:** 9/9 prerequisite+task items (100%), with one item (Task 4)
carrying a downgraded completeness claim — the code shipped for the task's
own defect is complete and correctly mutation-verified, but the plan's
"every comparison checked" sentence oversells the audit that produced it.

**Skipped without approval:** 0
**Partial implementations:** 1 (Task 4 — see above; not a missing fix, a
missing generalization the plan claims was done and wasn't)

## Skipped / Deferred Tasks

None skipped. Task 4 is the sole item flagged PARTIAL above, and it is a
completeness-of-audit issue rather than a missing fix for the task's stated
defect (`TradeStaging` timeout rollback, which is correctly implemented and
mutation-verified). Impact if left as-is: any saga using `PetEvolution`,
`ItemTagUse`, `SealingLockUse`, `IncubatorUse`, `PointReset`, `NoteSend`, or
`SkillBookUse` that stalls past its 30s backstop between steps will silently
skip its reverse walk, identical in shape to the bug this pass exists to
close — just outside the trade domain, so outside this branch's blast
radius for the reported bug, but inside the class of bug the plan explicitly
set out to close "not by adding a fourth `if`."

## Build & Test Results

| Service | Build | Vet | Tests (`-race -count=1`) | Notes |
|---------|-------|-----|---------------------------|-------|
| atlas-trades (`services/atlas-trades/atlas.com/trades`) | PASS | PASS | PASS | All packages ok, including `escrow` and `trade`. |
| atlas-saga-orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`) | PASS | PASS | PASS | All packages ok, including `saga`. |
| libs/atlas-database | PASS | PASS | PASS | Both `atlas-database` and `atlas-database/databasetest` ok. |

Additional repo-level checks run directly (not taken on trust):

| Check | Result |
|---|---|
| `docker buildx bake atlas-trades` | PASS (image built and exported) |
| `tools/redis-key-guard.sh` | PASS (exit 0; no findings in changed modules) |
| `tools/goroutine-guard.sh` | PASS (exit 0; no findings in changed modules) |
| `tools/buff-duration-guard.sh` | PASS (exit 0; not applicable to this diff) |
| `tools/skill-job-id-guard.sh` | PASS ("clean (14 divergent const(s) checked)") |
| `tools/trade-contract-mirror-guard.sh` | PASS ("the trade contract mirror matches its owner"; "the trade escrow custody contract mirror matches its owner") |
| `tools/service-registration-guard.sh` | Not run — no services.json/k8s/docker-bake/go.work/db-bootstrap files changed in this diff, so it is not gated per CLAUDE.md item 7 |
| `tools/template-*-guard.sh` (opcode order, duplicate binding, movement types) | Not run — no tenant socket-config templates changed in this diff |
| `tools/lint.sh --check` | Ran in background, exceeded the 300s foreground window; not confirmed clean within this audit's time budget (see Action Items) |
| `atlas-saga-orchestrator` / `libs/atlas-database` `go.mod` | Neither touched relative to `main` — `docker buildx bake` not mandated for either per CLAUDE.md item 4 |

Note: `go.mod` for `atlas-trades` was introduced earlier in the base
task-205 branch (commits `e7a427993`..`67fa13517`, before this parity pass
started at `efec8332a`), not within the audited commit range itself. The
plan's Task 7 claim "the branch moved its go.mod" is accurate for the
task-205 branch as a whole (which is what the bake step gates on), not a
misstatement.

## Operational note — concurrent uncommitted edits observed during this audit

Partway through this audit, `git status` in this same worktree showed
uncommitted modifications to `services/atlas-trades/atlas.com/trades/escrow/administrator.go`
and `.../escrow/model.go`, plus a new untracked
`docs/tasks/task-205-player-trade/audit-backend-parity.md`. These are not
edits made by this audit or its sub-agents (all sub-agents confirmed clean
`git status` at their own handoff, aside from noting this same external
change). The diff itself (converting `RestoreMesoRefunds`'s raw entity
field reads to the immutable-model pattern via a new `MesoRefundModel` /
`MakeMesoRefund`) looks like an in-flight DOM-guideline fix from a
concurrently running backend-guidelines-reviewer pass in the same worktree,
not a defect. It is called out here only because:

1. It means the working tree was not byte-identical to `HEAD` for the full
   duration of this audit — the build/test runs recorded above were captured
   at a point in time and are not guaranteed to reflect the exact same tree
   state as the mutation-test captures taken by the parallel sub-agents.
2. Anyone reading this report should re-run `git status` before merging to
   confirm this concurrent work has since been committed or reverted; it
   should not be silently folded into task-205's history without its own
   review.

This audit did not touch, stage, or revert these files.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — every task's own stated defect is
  fixed and mutation-verified; one task's own completeness claim (Task 4)
  overstates what was actually checked, leaving a same-shaped gap for seven
  other saga types.
- **Recommendation:** NEEDS_FIXES (narrow) — the trade-specific fixes in
  this PR are ready; Task 4's "every comparison checked" claim should either
  be corrected in the plan text to scope it explicitly to `TradeStaging`, or
  the seven other saga types should be added to `reverseWalkSagaTypes` before
  merge if the intent was genuinely a class fix. Everything else in this
  report is READY_TO_MERGE quality.

## Action Items

1. **Task 4 scope.** Either (a) narrow the plan's Task 4 write-up to state
   explicitly that only `TradeStaging` was added and the other seven
   bespoke-compensator saga types (`PetEvolution`, `ItemTagUse`,
   `SealingLockUse`, `IncubatorUse`, `PointReset`, `NoteSend`,
   `SkillBookUse`) remain unregistered in `reverseWalkSagaTypes` — filed as a
   known follow-up, not "checked" — or (b) add those seven to
   `reverseWalkSagaTypes` and extend `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout`'s
   iteration source to something that can catch a type omitted from the list
   itself (e.g. derive the expected set from `compensator.go`'s dispatcher
   directly, or add a second test that cross-checks the list against every
   `SagaType()` arm in `CompensateFailedStep`). Given CLAUDE.md's "no
   deferring producible work" rule, (b) is the compliant choice, but this is
   the PR author's call to make explicitly rather than have it discovered
   later.
2. **Confirm `tools/lint.sh --check` finishes clean.** It was still running
   when this audit's time budget closed; re-run it standalone
   (`tools/lint.sh --check` from repo root) and confirm exit 0 before
   claiming the branch's full CLAUDE.md gate is green — this audit could not
   independently confirm it within its window.
3. **Resolve the concurrent uncommitted diff** noted above
   (`escrow/administrator.go`, `escrow/model.go`,
   `audit-backend-parity.md`) — either commit it as its own reviewed change
   or discard it, and re-run `go build`/`go vet`/`go test -race` for
   `atlas-trades` afterward so the final gate reflects the actual tree that
   will be merged.
