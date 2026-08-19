# Review: Task 16 — Package purchase (`atlas-cashshop`)

Commit range: `863252f..e2f5aaece` (single commit `e2f5aaece`)
Brief: `.superpowers/sdd/plan/task-16-brief.md`
Report: `.superpowers/sdd/plan/task-16-report.md`

## Scope

```
services/atlas-cashshop/atlas.com/cashshop/cashshop/package.go         | 238 ++++++++++++
services/atlas-cashshop/atlas.com/cashshop/cashshop/package_test.go    | 397 +++++++++++++++++++++
services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go       |  56 +--
services/atlas-cashshop/atlas.com/cashshop/data/cashpackage/model.go   |  14 +
services/atlas-cashshop/atlas.com/cashshop/data/cashpackage/processor.go |  31 ++
services/atlas-cashshop/atlas.com/cashshop/data/cashpackage/requests.go |  25 ++
services/atlas-cashshop/atlas.com/cashshop/data/cashpackage/rest.go    |  32 ++
services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go |  18 +
services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go     |  40 +++
services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go |  24 ++
```

Matches the brief's Files list plus the two declared deviations (`cashshop/processor.go` field/interface wiring, and `producer.go`'s new `PackagePurchasedStatusEventProvider` rather than a one-liner). Both deviations are correctly justified and both are declared in the report under "Declared deviations." Nothing outside `services/atlas-cashshop/atlas.com/cashshop/` is touched. `go.work.sum` in the working tree is a pre-existing local drift unrelated to this commit (present before this review began, and outside the commit's diff). **Scope confirmed: matches the brief.**

## Findings

### 1. Price source (FR-PKG-5) — PASS, verified by independent mutation

`package.go:90-102` resolves the package commodity (`pkgCommodity`) by serial number first, then resolves the `CashPackage` by `pkgCommodity.ItemId()`. The debit at `package.go:191` (`w.Purchase(currency, pkgCommodity.Price())`) and the emitted `Price` (`package.go:226`) both read `pkgCommodity.Price()`. Member commodities are stored in an anonymous struct (`serialNumber`, `itemId`, `count`) that deliberately does not carry `Price` at all — the "sum of members" mistake is structurally unavailable in this code.

Mutation performed independently (not trusting the report's quoted output): replaced the debit amount with a hardcoded `3600` (= 3 × 1200, the sum of the three member prices) at `package.go:191`. Result: RED —
```
--- FAIL: TestPurchasePackage/creates_one_asset_per_member
--- FAIL: TestPurchasePackage/gift_mode_delivers_to_the_recipient
--- FAIL: TestPurchasePackage/replay_delivers_once
```
Reverted; `diff` against a pre-mutation backup confirmed byte-identical restoration.

### 2. Atomicity ordering — PASS on data integrity, but the "proves ordering" framing is not correct (non-blocking)

Mutation performed independently: moved the capacity check (`package.go:169-176`, `if ccm.Capacity() < uint32(len(ccm.Assets())+len(memberCommodities))`) to run **after** the wallet debit block, i.e. debit-then-check instead of check-then-debit.

Result: **GREEN** — `capacity is checked against the full member count before charging` and every other subtest still passed with the check reordered.

Root cause, confirmed by reading `libs/atlas-database/transaction.go:9-14`: `database.ExecuteTransaction` calls `db.Transaction(fn)` (a real GORM/SQL transaction) whenever `db` is not already inside one. `PurchasePackageAndEmit`'s `reject(reason)` closure returns the sentinel `errPackageRejected` (non-nil), so GORM rolls back the entire transaction — every write made inside the closure up to that point, including the wallet debit that ran before the (now misplaced) capacity check, is undone. `w.Purchase(...)` and `walP.Update(buf)(...)` are both `tx`-bound, so the rollback genuinely reverts the debit. The two observable postconditions the test checks — zero assets, unchanged credit — hold regardless of intra-closure step order, because the whole closure is one atomic unit.

This means the code is **not incorrect** — atomicity is real and holds under this specific reordering mutation — but the test (and the report's and code's own comments, e.g. `package.go:169-172` "the check that proves this precedes the debit rather than failing partway through") overstate what is being proven: the test does not actually discriminate check-before-debit from check-after-debit, because GORM's transaction wrapper already guarantees the debit is reverted either way. The genuine risk this ordering guards against (a non-transactional side effect executing before the capacity check, e.g. a bare Kafka publish outside the outbox) is not present in this closure — every side effect that matters (`ledger.Claim`, wallet, asset, `purchaserecord.Record`) is `tx`-bound, and the only non-transactional action (`rejectEmit`, the direct-producer error path) fires identically regardless of when the rejection occurs.

Reverted; restored to byte-identical.

Recommendation (non-blocking): either adjust the comment/test-name framing to stop claiming this test "proves ordering," or add a distinct assertion that genuinely depends on ordering if the intent is defense-in-depth against a future non-transactional side effect being inserted before the capacity check.

### 3. "Unresolvable member aborts everything" — PASS, verified by independent mutation

Mutated `package.go:117` (`return reject("UNKNOWN_ERROR")` inside the member-resolution loop) to `continue`, independently reproducing the report's claimed mutation rather than trusting its quoted output.

Result: RED —
```
--- FAIL: TestPurchasePackage/an_unresolvable_member_creates_nothing
--- FAIL: TestPurchasePackage/capacity_is_checked_against_the_full_member_count_before_charging
--- FAIL: TestPurchasePackage/an_unknown_package_creates_nothing
--- FAIL: TestPurchasePackage/gift_mode_delivers_to_the_recipient
```
(The `capacity`/`unknown_package`/`gift_mode` subtests failed collaterally because `continue` on the member commodity fetch failure changes downstream `mc` handling incorrectly under `go vet`-safe semantics for this fixture — confirms the check is load-bearing well beyond the one subtest it was designed to catch.) Reverted; byte-identical restoration confirmed.

### 4. Non-tx-bound compartment processor bug fix — PASS

`package.go:162` resolves the target compartment via `cicP := compartment.NewProcessor(p.l, p.ctx, tx)`, constructed locally inside the transaction closure and bound to `tx` — exactly matching `gift.go:107`'s established pattern (the landed precedent). All other DB-writing calls inside the closure are likewise `tx`-bound: `ledger.Claim(p.ctx, tx, ...)` (`package.go:79`), `wallet.NewProcessor(p.l, p.ctx, tx)` (`package.go:180`), `asset.NewProcessor(p.l, p.ctx, tx)` (`package.go:200`), `purchaserecord.Record(tx, ...)` (`package.go:214`, `219`). Read-only REST calls (`p.comP.GetById`, `p.dataCashPkgP.GetById`, `p.chaP.GetById()`) are not tx-bound, which is correct — they are not database calls and carry no deadlock risk. The fix genuinely resolves the reported SQLite deadlock (a fresh, non-tx connection blocking on a file lock held by the open outer transaction) without weakening the transaction boundary — the compartment resolution, capacity check, debit, and asset creation are all still inside the single `database.ExecuteTransaction` closure.

### 5. Gift mode / error-operation swap detection — PASS, verified by independent mutation

`package.go:60-63` selects `operation` once at the top: `ErrorOperationBuyPackage` when `recipientCharacterId == 0`, `ErrorOperationGiftPackage` otherwise. The test table pins `ErrorOperationBuyPackage` on four buy-for-self failure subtests (`an unresolvable member`, `capacity is checked`, `an unknown package`) and pins `ErrorOperationGiftPackage` on the gift-mode subtest's second (insufficient-funds) attempt (`package_test.go:340-346`).

Mutated `package.go:60,62` to swap the two constants (default `GiftPackage`, flip to `BuyPackage` on `recipientCharacterId != 0` — i.e., inverted). Result: RED in both directions —
```
--- FAIL: TestPurchasePackage/an_unresolvable_member_creates_nothing        (buy-mode arm expected BuyPackage, got GiftPackage)
--- FAIL: TestPurchasePackage/capacity_is_checked_against_the_full_member_count_before_charging
--- FAIL: TestPurchasePackage/an_unknown_package_creates_nothing
--- FAIL: TestPurchasePackage/gift_mode_delivers_to_the_recipient            (gift-mode arm expected GiftPackage, got BuyPackage)
```
Confirms the swap is caught in both directions across the two mode's subtests, satisfying the "assert `!= theOther`" concern even though no single subtest does a literal `require.NotEqual`. Reverted; byte-identical restoration confirmed.

### 6. Replay safety and purchase records — PASS

`replay delivers once` (`package_test.go:349-373`) reruns `PurchasePackageAndEmit` with the same `transactionId` and asserts exactly 3 assets and unchanged credit (2000) after the second call — relies on `ledger.Claim`'s `ErrAlreadyProcessed` short-circuit (`package.go:79-85`), same pattern as `gift.go`/`rebate.go`. `records the package and every member` (`package_test.go:375-396`) asserts `purchaserecord.Get` returns `1` for all four serial numbers (package SN 50000 and member SNs 10000/10001/10002), matching `purchaserecord.Record` being called once for the package serial number (`package.go:214`) and once per member serial number (`package.go:218-223`).

### 7. Currency 3 (prepaid) — PASS

`package.go` never special-cases currency. `currency` flows unmodified into `w.Balance(currency)` (`package.go:186`) and `w.Purchase(currency, ...)` (`package.go:191`); the created assets' persisted currency column uses `effectivePurchaseCurrency(currency)` (`package.go:199`), an existing, unmodified helper (`cashshop/processor.go:56-90`) already used identically by `Purchase()`/`GiftAndEmit`. No new 0/1/2-only switch is introduced by this commit — confirmed by grep across the diff for `walletCurrency`/`effectivePurchaseCurrency`, both pre-existing.

### 8. Scope — PASS

All ten changed files match the brief's list plus the two declared, justified deviations (`cashshop/processor.go`, and the non-one-line `producer.go` change). `data/cashpackage/{model,rest,requests,processor}.go` mirror `data/pet/` file-for-file. No files outside `services/atlas-cashshop/atlas.com/cashshop/` are touched by this commit.

## Verification performed

- `go build ./... && go test ./...` in `services/atlas-cashshop/atlas.com/cashshop`: all packages GREEN, no failures.
- Independently reproduced (not trusted from the report) four mutations against `cashshop/package.go`, each followed by `diff` against a pre-mutation backup to confirm byte-identical restoration and `git diff --stat` returning clean for this commit's files:
  1. Price source → sum-of-members: RED.
  2. Capacity check moved after the debit: **GREEN** (see Finding 2 — not a functional defect, but the test/comment overstate what is proven).
  3. Member-resolution-failure `return reject(...)` → `continue`: RED.
  4. Operation constant assignment swapped (buy↔gift): RED in both directions.
- `git diff --stat` after each restoration confirmed only the pre-existing, unrelated `go.work.sum` drift remains — no leaked mutation state.

## Not evaluable

None — the full review surface (the one commit's diff, plus `gift.go`/`rebate.go` as the cited precedents, plus `libs/atlas-database/transaction.go` for the transaction-rollback semantics that Finding 2 depends on) was read and independently exercised.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-task-16.md
scope_confirmed: reviewed the full diff of e2f5aaece (data/cashpackage/*, cashshop/package.go + package_test.go, cashshop/processor.go, kafka/message|producer|consumer/cashshop) against task-16-brief.md; cross-checked cashshop/gift.go and libs/atlas-database/transaction.go as the load-bearing precedents/contracts
blocking: 0
non_blocking: 1
  - services/atlas-cashshop/atlas.com/cashshop/cashshop/package.go:169-176,187-197 (and package_test.go:211-241) — the "capacity is checked... before charging" test does not actually discriminate check-before-debit from check-after-debit; independent mutation moving the capacity check after the wallet debit left the whole suite GREEN, because `database.ExecuteTransaction`'s GORM transaction rolls back the debit regardless of order. Atomicity holds either way (not a data-integrity bug), but the code comment and report both claim the test "proves" ordering, which it does not. Either soften the claim or add an assertion that would genuinely require the documented step order.
not_evaluable: 0
