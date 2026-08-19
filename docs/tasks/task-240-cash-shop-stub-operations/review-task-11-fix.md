# Review: task-11 fix round 1 (task-240-cash-shop-stub-operations)

**Fix commit:** `9f044e1f9` (on top of `669272766`, not amended)
**Prior review:** `docs/tasks/task-240-cash-shop-stub-operations/review-task-11.md`
**Scope:** the fix commit only — `git diff 669272766 9f044e1f9`, four files:
`cashshop/inventory/asset/entity.go`, `cashshop/processor.go`, `cashshop/rebate.go`,
`cashshop/rebate_test.go`.

## Blocking finding this fix was meant to close

Prior review: a BUY_NORMAL purchase (wire currency `0`) debited **prepaid** but
its rebate credited **credit/NX**, because `Currency == 0` was overloaded
between "legacy row" and "genuine prepaid purchase."

**Disposition: CLOSED.** Verified independently, not by reading and reasoning:

- Built and ran `TestRebateFixRound1CurrencyNormalization` unmodified: both
  subtests PASS. The first subtest drives a real `PurchaseAndEmit(..., currency=0,
  ...)` → `RebateAndEmit(...)` round trip and asserts `w.Prepaid() == price` /
  `w.Credit() == 0` after the rebate — i.e. it re-creates my original throwaway
  disproof and the asymmetry is gone.
- Reproduced the implementer's claimed RED: temporarily replaced
  `effectivePurchaseCurrency` with an identity function (`return currency`,
  simulating pre-fix behavior), re-ran the same test. Result:
  `expected: 0x3 actual: 0x0` on the "persists prepaid and rebates prepaid"
  subtest — exact match to the implementer's report
  (`.superpowers/sdd/plan/task-11-report.md`). Restored the file afterward;
  `git diff` on `cashshop/processor.go` is clean, `go build ./...` passes.
- This closes the discriminating-test gap noted in the original review: the
  pre-existing `TestRebate` round-trip subtest only ever exercised
  `Currency == 2` and would not have caught this bug; the new subtest exercises
  wire currency `0` specifically, which is the value that was broken.

## Item-by-item verification

**1. Original disproof re-run.** Done above — PASS on the fix, FAIL (RED) with
the normalization stubbed out. The asymmetry is genuinely closed, not just
argued away.

**2. Debit line left on raw wire currency — is that behaviour-preserving?**
Confirmed by reading `wallet/model.go`:
- `Balance(currency)`: `if currency==1 {credit} else if currency==2 {points} else {prepaid}`
- `Purchase(currency, amount)`: identical `if/else if/else` shape, decrementing `prepaid` in the `else` arm
- `Award(currency, amount)`: `switch currency { case 1: ...; case 2: ...; default: prepaid }`

All three treat `0` and `3` identically (both land in the `else`/`default`
prepaid arm). `w.Purchase(0, …)` (debit, still uses raw wire currency per
`cashshop/processor.go`) and a hypothetical `w.Purchase(3, …)` produce the same
result. No wallet path distinguishes `0` from `3` — confirmed by grep across
`wallet/*.go` for any `switch`/`case 0`/`case 3` construct; none exists.

**3. Do the two new subtests actually discriminate?** Yes — reproduced RED and
GREEN myself (item 1). Not evaluated on the implementer's say-so.

**4. Does `3` leak anywhere it shouldn't?**
- `asset/rest.go` (`RestModel.Currency uint32 \`json:"currency"\`` at
  `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest.go:14`)
  does project `Currency` on the cashshop-service REST resource for an asset.
  This is a real widening of the value space that field can carry (previously
  0/1/2, now 0/1/2/3), but I checked the one known consumer:
  `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/{model,rest,builder,processor,requests}.go`
  has **zero** references to `currency`/`Currency` anywhere — the channel-side
  asset consumer does not read or render this field at all today. Not a
  regression this fix introduces; the field was already present and already
  capable of holding `1`/`2` which the channel side also ignores. Non-blocking,
  noted for awareness only.
- `kafka/message/cashshop/kafka.go:249` (`LockerRebatedBody.Currency`) and
  `kafka/producer/cashshop/producer.go:100/109`
  (`LockerRebatedStatusEventProvider`) emit the rebate's resolved currency
  (`1`/`2`/`3`, never a raw `0` after this fix, since `rebate.go:139-141`
  still normalizes a stored `0` to `1` before it reaches the emit). Grepped
  `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/*.go` for
  any consumer of a `LockerRebated`-shaped event: none found. No consumer
  exists yet to receive a `3` it can't interpret. Non-blocking.
- No other Kafka event body in the diff's blast radius carries the asset's
  `Currency` field.

Conclusion: `3` does not leak anywhere with a live consumer that would
misrender it. Flagged as non-blocking awareness items in case either
projection later gets a real reader.

**5. Two implementer claims, verified independently:**
- `PurchaseInventoryIncrease` (`cashshop/processor.go`, function starting
  around line 320) — read the full function body; it never calls
  `p.astP.Create`/`CreateWithCashId` or touches the `asset` package at all
  (`grep -n "astP\|asset\."` over its body returns nothing). Confirmed: it
  creates no asset row, so `effectivePurchaseCurrency`'s write-site fix
  correctly has no bearing on this path.
- `libs/atlas-constants/` — swept every subpackage (`asset/`, `channel/`,
  `character/`, `constants/`, `coupon/`, `field/`, `gen/`, `inventory/`,
  `invite/`, `item/`, `job/`, `map/`, `merchant/`, `miniroom/`, `monster/`,
  `pet/`, `point/`, `skill/`, `stat/`, `summon/`, `world/`) for
  `currency|credit|prepaid|wallet` (case-insensitive). Only hit:
  `item/constants.go:100` — `ClassificationCurrencySack = Classification(520)`,
  an unrelated item-classification constant, not a wallet-bucket enum.
  Confirmed: no existing wallet-currency enum was available to reuse; the new
  local `walletCurrencyCredit`/`walletCurrencyPoints`/`walletCurrencyPrepaid`
  constants in `cashshop/processor.go` do not violate the constants-reuse
  convention.

## Regression check

`go build ./... && go test ./...` for `atlas-cashshop` (module-local, per the
implementer-budget convention) — all packages PASS, none skipped, none
newly failing.

## Verdict

The fix closes the blocking finding from `review-task-11.md` correctly and
minimally: normalizing at the write site (`effectivePurchaseCurrency`) makes a
stored `0` unambiguous, the debit path is provably unaffected (same wallet
dispatch arm for `0` and `3`), the new tests genuinely discriminate (RED/GREEN
reproduced independently), and the retained `0`→credit rebate default for
legacy rows is preserved and still covered by its own subtest. No new
regressions found in the fix's own surface.

Two non-blocking awareness notes are recorded above (REST/Kafka projection of
`Currency` now includes `3`, with no live consumer today) — they do not block
this fix but are worth a mental note if either projection gains a reader.
