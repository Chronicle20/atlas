# Plan Audit — task-240-cash-shop-stub-operations (Tasks 1-12)

**Plan Path:** docs/tasks/task-240-cash-shop-stub-operations/plan.md
**Audit Date:** 2026-08-19
**Branch:** task-240-cash-shop-stub-operations
**Base Branch:** main (range: merge-base `d9ec287b8`..`3bc7ebd21`)
**Scope:** Plan Tasks 1 through 12 only (a separate shard covers 13-24)

## Executive Summary

All 12 tasks in this shard's range were fully implemented, verified against
the code (not just the plan/reports), and pass their targeted tests. Two
tasks (6, 11) went through a documented `CHANGES_REQUIRED` → fix cycle
whose fix commits are present on the branch and independently re-verified
here (bounded SQL aggregate for the backfill boot query; currency
normalization for the rebate credit). One task (10) intentionally diverged
from the plan's literal file layout (a single shared `ledger.go` wrapping
the existing `libs/atlas-database` idempotency table, rather than a new
`cash_command_ledger` table with `entity.go`/`administrator.go`/
`duplicate.go`) — this is a disclosed, justified architectural
simplification that satisfies the same interface contract (`Claim`,
`ErrAlreadyProcessed`, uuid.Nil rejection), not a shortcut. No stubs,
`// TODO`s, or hard-coded success/failure responses were found introduced
by these 12 tasks. Both affected Go modules (`atlas-channel`,
`atlas-cashshop`) build clean and all tests pass.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Derive the four blocking client facts | DONE | `docs/tasks/task-240-cash-shop-stub-operations/derivation.md` — D1 (opcode 0x8F), D2a (empty body), D2b (UPDATE_WISHLIST 98), D3a/D3b, D4a/D4b all RESOLVED with cited decompilation. Prior review-task-1.md: APPROVED. |
| 2 | Register `CashShopOpen` for `gms_95` | DONE | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:3292-3298` — `{"opCode":"0x8F","writer":"CashShopOpen","fname":"CStage::OnSetCashShop","services":["channel"]}`, matches D1 exactly. Corpus-count guard test updated (`corpus_test.go:64`). |
| 3 | Failure routing — `ErrorEventBody.Operation` | DONE | `Operation` field + `ErrorOperation*` consts added to both atlas-cashshop and atlas-channel `kafka.go` copies; `ErrorStatusEventForOperationProvider` added (`producer.go:31`); `failureBodyForOperation` switch in `consumer.go:558-579` with correct default fallback and priority ordering after the pending-change branch (`consumer.go:591-623`). `TestFailureBodyForOperation` passes all 10 subtests. |
| 4 | The secondary-credential gate | DONE | `cash_shop_credential.go` implements `ErrCredentialMismatch`, `credentialMatches`, `verifySecondaryCredential` verbatim to the plan's interface signatures, using the `MajorAtLeast(95)` idiom and `remoteIpAddress(s)`. `TestCredentialMatches` passes all 8 subtests. |
| 5 | Purchase records | DONE | `purchaserecord/entity.go`, `administrator.go` (`Record`/`Get`) match the plan's interfaces exactly; `Record` called inside `Purchase`'s transaction at `cashshop/processor.go:288`, before the `mb.Put(...)`; `purchaserecord.Migration` wired into `main.go:65`. `TestRecordUpsertsAndCounts` passes. |
| 6 | Backfill purchase records | DONE (after fix) | Initial review-task-6.md was `CHANGES_REQUIRED`; fix commit `39d0239bf36` ("bound backfill's boot-time query to a single SQL aggregate") landed a Postgres single-aggregate path (`backfillGroupsSQL`) plus a documented sqlite fallback (`backfillGroupsSQLite`) for the driver limitation. `Backfill` wired at `main.go:71-75`, idempotency gate documented and tested (`TestBackfill*`, `TestBackfillCountsDuplicatesAndSoftDeletes/is_idempotent`) — all pass. |
| 7 | GET_PURCHASE_RECORD (mode 40/44) | DONE | atlas-cashshop `purchaserecord/resource.go` GET route returns 200 with `Purchased: false` on a miss (never 404); atlas-channel arm at `cash_shop_operation.go:251-268` reads through the client and announces `CashShopPurchaseRecordDoneBody`/`FailedBody` on the two arms, never a silent return. `TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag` and `TestPurchaseRecordFlag` pass. |
| 8 | APPLY_WISHLIST (mode 33/35) | DONE | Derivation D2a (empty body) correctly means no new serverbound codec was added — this is the plan's own conditional path, not a shortcut. Arm at `cash_shop_operation.go:212-243` answers `CashShopWishListUpdateBody` on success. Initial review-task-8.md found a genuine blocking defect (wrong failure arm, LOAD_WISH_FAILED instead of the latch-clearing SET_WISH_FAILED); fix commit landed `CashShopSetWishFailedBody`, confirmed present at line ~229 with a doc comment explaining the correction, and review-task-8-fix.md confirms the fix. `TestCashShopWishListBodyEncodesStoredSerials` passes. |
| 9 | BUY_NORMAL (mode 20/34) | DONE | Arm at `cash_shop_operation.go:159-169` requests purchase with `isPoints=false, currency=0` (cited derivation `shop_operation_buy_normal.go:23-28`), minting a fresh `transactionId` per click, threading `ErrorOperationBuyNormal` through. Consumer routes `BUY_NORMAL_SUCCESS`/`_FAILED` distinctly via the `Operation` discriminator (`consumer.go:222-243,562-563`). Initial review-task-9.md found two blocking issues (an invented SlotPos value, a non-discriminating test); fix commits (`795f370fe`, `79b0e08c5`) resolved both per review-task-9-fix.md. All relevant tests (`TestBuyNormalPurchaseCurrency`, `TestBuyNormalHandleEmitsPurchase`, `TestCashShopBuyNormalDoneBodyMode`, `TestCashShopBuyNormalFailedBodyMode`) pass. |
| 10 | The shared command ledger | DONE (deliberate deviation) | Implemented as `ledger/ledger.go` wrapping `libs/atlas-database`'s existing `IdempotencyEntity`/`database.IdempotencyMigration` rather than the plan's bespoke `cash_command_ledger` table — a disclosed simplification (commit message: "add the shared command ledger over the existing idempotency table"), not a shortcut: it still satisfies `Claim(...) error`, `ErrAlreadyProcessed`, and rejects `uuid.Nil` outright, all independently confirmed in code. `main.go` already carries `database.IdempotencyMigration` in `SetMigrations`. All 8 `TestClaim_*` subtests pass; review-task-10.md: APPROVED. |
| 11 | REBATE_LOCKER_ITEM — atlas-cashshop side | DONE (after fix) | `cashshop/rebate.go` `RebateAndEmit` claims the ledger first, resolves the asset account-scoped, rejects zero-commodity/expired/unowned assets identically (no distinguishing NOT_OWNED leak), deletes the asset, and credits the wallet. Initial review-task-11.md was `CHANGES_REQUIRED` on currency resolution; fix round 1 (commit `9f044e1f9`) normalizes at the write site and rebate now reads `am.Currency()` with a documented 0→credit legacy default. `TestRebate` (7 subtests) and `TestRebateFixRound1CurrencyNormalization` (2 subtests) pass; review-task-11-fix.md: `APPROVED_WITH_FINDINGS` (non-blocking). |
| 12 | REBATE_LOCKER_ITEM — atlas-channel side | DONE | `cashshop/producer.go` `RequestLockerRebateCommandProvider`, `processor.go` `RequestLockerRebate`, arm at `cash_shop_operation.go:174-192` runs `verifySecondaryCredential` first (Task 4), names the local variable `cashId` (not `unk`) as the plan required, mints `uuid.New()`. `handleStatusEventLockerRebated` announces `CashShopRebateDoneBody`. `TestCashShopRebateDoneBodyEncodes` passes; review-task-12.md: verdict rationale confirms all controller corrections independently verified. |

**Completion Rate:** 12/12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 12 tasks in this range are fully implemented and tested.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (services/atlas-channel/atlas.com/channel) | PASS | PASS | Full `go test ./...` clean, no FAIL lines. |
| atlas-cashshop (services/atlas-cashshop/atlas.com/cashshop) | PASS | PASS | Full `go test ./...` clean, no FAIL lines, including `purchaserecord`, `ledger`, `cashshop` (rebate) packages. |

Targeted task-specific tests independently re-run and confirmed passing:
`TestFailureBodyForOperation`, `TestCredentialMatches`, `TestRecordUpsertsAndCounts`,
`TestBackfill*`, `TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag`,
`TestPurchaseRecordFlag`, `TestCashShopWishListBodyEncodesStoredSerials`,
`TestBuyNormalPurchaseCurrency`, `TestBuyNormalHandleEmitsPurchase`,
`TestCashShopBuyNormalDoneBodyMode`, `TestCashShopBuyNormalFailedBodyMode`,
`TestClaim_*` (8), `TestRebate` (7 subtests), `TestRebateFixRound1CurrencyNormalization`,
`TestCashShopRebateDoneBodyEncodes`.

A repo-wide sweep for `TODO`/`FIXME`/`unimplemented` in the diff surfaced two hits,
both outside this shard's range (task 20's ring-pair status handler doc comment,
and a pre-existing unrelated `// TODO` on `ErrorEventBody` predating this branch) —
neither belongs to tasks 1-12.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; final merge readiness also depends on the sibling shard covering tasks 13-24 and the repo-wide Task 24 verification)

## Action Items

None. All 12 tasks in this range are complete, correctly implemented, tested, and their prior CHANGES_REQUIRED review cycles were genuinely closed by verified fix commits.
