# Gift notice — fix report

Status: **PARTIAL**. Steps 1 and 3 of the brief's `## Fix` are implemented and
tested. Step 2 is withheld — the mandatory producer sweep found a real
transaction-tagged wallet producer with no `CashQueryResult` refresh of its
own, and the brief's controller ruling forbids guessing a narrowing in that
case.

## Producer sweep (mandatory gate, done before writing step 2)

Every caller of `UpdateWithTransaction` / `UpdateAndEmitWithTransaction` /
`AdjustCurrencyWithTransaction`:

- `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go:214` —
  `AdjustCurrencyWithTransaction` calls `UpdateAndEmitWithTransaction`
  internally (not an external caller).
- `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/wallet/consumer.go:50`
  — `handleAdjustCurrencyCommand` calls
  `proc.AdjustCurrencyWithTransaction(c.TransactionId, c.AccountId, c.CurrencyType, c.Amount)`.
  This consumer handles `AdjustCurrencyCommand` on the `wallet_command` topic.
  Producer: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/cashshop/producer.go:14`
  (`AdjustCurrencyProvider`), used by the task-227 name-change/world-transfer
  sagas per `docs/tasks/task-227-cash-name-change-world-transfer/design.md`
  (line 513: "Player uses a `5400xxx`/`5401xxx` coupon..."). This flow
  supplies a real, non-`uuid.Nil` transaction id.
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go:131`
  — the gift path, now converted to `UpdateWithTransaction` by this task's
  step 1 (was `Update`, untagged, before this change).

`CashScene`-relevant consumer paths on the wallet status topic
(`EVENT_TOPIC_WALLET_STATUS`): only one —
`services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go:65`
(`handleWalletUpdated`), which is exactly the handler step 2 would modify.

**Finding:** `handleAdjustCurrencyCommand` (the task-227 saga currency
debit) emits only the wallet `UPDATED` status event with a non-nil
`TransactionId` — it emits **no cashshop status event of its own**, and
there is no other consumer that re-announces `CashQueryResult` for this
flow after the fact (unlike gift, purchase, inventory-capacity-increase,
and coupon-redeemed, which all have a cashshop status handler that either
already announces `CashQueryResult` or now does, per step 3).
`docs/tasks/task-227-cash-name-change-world-transfer/design.md` line 174
("whether the client accepts `CANCEL_*` outside the cash shop") implies the
coupon-consumption step normally happens while the player is inside the
cash shop scene (`session.CashSceneCashShop`), which is exactly the
condition `handleWalletUpdated` gates its announce on.

Under the brief's proposed broad rule ("skip the `CashSceneCashShop`
`CashQueryResult` announce when `e.Body.TransactionId != uuid.Nil`"), this
producer's wallet refresh would be silently dropped with no substitute
announce anywhere in the codebase. Per the explicit controller ruling —
"If any transaction-tagged producer would be left without a
`CashQueryResult` refresh under the broad skip rule, do NOT guess a
narrowing: report PARTIAL with the specific producer named" — I stopped
here instead of picking a narrowing (e.g. keying the skip off something
more specific than "non-nil transaction id", or giving the
`AdjustCurrencyCommand` flow its own `CashQueryResult`-announcing status
handler in atlas-channel, which is out of this task's file inventory and
would need its own design decision about which topic/event carries the
saga's completion).

**Producer named for the continuation:**
`services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/wallet/consumer.go:50`
(`handleAdjustCurrencyCommand` → `AdjustCurrencyWithTransaction`), driven by
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/cashshop/producer.go:14`
(`AdjustCurrencyProvider`), consumed as `EVENT_TOPIC_WALLET_STATUS` `UPDATED`
by `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go:65`
(`handleWalletUpdated`).

## What was implemented

### Step 1 — tag the gift wallet debit with the gift's transaction id

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go:131`

```go
_, err = walP.UpdateWithTransaction(buf)(transactionId)(sender.AccountId())(w.Credit())(w.Points())(w.Prepaid())
```

(was `walP.Update(buf)(...)`, untagged). The wallet `UPDATED` status event
for a gift now carries the gift's real transaction id instead of
`00000000-0000-0000-0000-000000000000`.

### Step 3 — `handleStatusEventGiftPurchased` owns the CashQueryResult order

`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
— `handleStatusEventGiftPurchased` now announces `CashShopGiftDoneBody`
(GIFT_SUCCESS), then fetches the sender's wallet and announces
`CashQueryResult`, in that order, mirroring
`handleStatusEventInventoryCapacityIncreased` (`:136-166`). Errors from
either announce return early / are logged, matching the existing pattern.

### Step 2 — NOT implemented

`services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go`
(`handleWalletUpdated`) is unchanged. Without step 2, the racy early
`CashQueryResult` from the wallet-status consumer (the actual root cause of
the ordering race) can still arrive before `GIFT_SUCCESS` on a live gift, so
**the live bug is not yet fixed** — step 3 alone adds the correct-order
announce but does not suppress the incorrect-order one. This is why the
overall status is PARTIAL rather than DONE_WITH_CONCERNS: the fix is
incomplete until step 2's design question is resolved.

## Tests

- Updated `TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem`
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`)
  to assert `announcedWriters() == [CashShopOperationWriter, CashQueryResultWriter]`
  in that order (was asserting exactly `[CashShopOperationWriter]`,
  pre-fix).
- Updated `decodeGiftDone` to search for the `CashShopOperationWriter`
  announcement by writer name rather than assuming it is the *last*
  announcement (it no longer is, now that `CashQueryResult` follows it) —
  same pattern already used by `decodeUseCouponDone`.
- No wallet-consumer test was added (step 2, the brief's second
  verification bullet, is not implemented).

## Commands run

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Result: `ok` for every package, including
`ok  	atlas-channel/kafka/consumer/cashshop	0.041s`.

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
Result: `ok` for every package with tests (build clean, no failures).

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go` —
  step 5 wallet debit now transaction-tagged.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
  — `handleStatusEventGiftPurchased` now announces `CashQueryResult` after
  `GIFT_SUCCESS`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`
  — updated existing gift test for the new two-announce contract; updated
  `decodeGiftDone` helper to search rather than assume "last announced".

## Self-review

- Step 1 and step 3 match the brief's exact prescription and the existing
  `handleStatusEventInventoryCapacityIncreased` pattern (same
  wallet-fetch-then-announce shape, same error handling/logging style).
- `Do not touch` respected: `GiftDone` codec, `GIFT_SUCCESS` mode table, and
  seed templates are untouched — only call sites and ordering changed.
- No new domain types/constants introduced; nothing to check against
  `libs/atlas-constants/`.
- Tree is clean aside from the pre-existing untracked
  `bug-round-2-defect-f-report.md` / `review-bug-round-2*.md` files, which
  were left alone as instructed.

## Remaining work for the continuation

**File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go`
(`handleWalletUpdated`, `:65-87`).

**Decision needed before any code changes:** how to keep the
`CashSceneCashShop` `CashQueryResult` announce for the task-227
`AdjustCurrencyCommand` flow (accountId-scoped, no transaction-owning status
handler of its own) while still suppressing it for the gift flow (which now
has its own correctly-ordered announce via step 3). Two directions the
controller may want to weigh:

1. Give the `AdjustCurrencyCommand` flow its own status/consumer path in
   atlas-channel that announces `CashQueryResult` after whatever confirms
   the name-change/world-transfer coupon consumption completed, then apply
   the brief's original broad skip rule
   (`e.Body.TransactionId != uuid.Nil`) safely.
2. Narrow the skip to something that distinguishes "a status handler already
   owns this transaction's `CashQueryResult`" from "no one does" — e.g. a
   per-operation marker on the wallet event, rather than presence of a
   transaction id alone. This needs its own design decision; nothing in the
   current wire contracts already carries that distinction.

**Exact next step:** once the direction is chosen, implement it in
`handleWalletUpdated`, add the wallet-consumer test the brief's
`## Verification` section specifies (`CashSceneCashShop` + non-nil
`TransactionId` → nothing announced; `uuid.Nil` → `CashQueryResult`
announced; MTS session unaffected either way), and re-run
`go test ./kafka/consumer/wallet/... ./kafka/consumer/cashshop/...` in
atlas-channel. At that point the live re-test (gift an item, confirm "All
the gifts have been sent...") becomes meaningful — it is not yet, since
step 2 is the piece that removes the racy early announce.
