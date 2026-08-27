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

## Round 2 — step 2

Status: **DONE**. Implements the controller's ruling in
`bug-round-2-gift-notice-step2-ruling.md`: an explicit ownership flag on the
wallet `UPDATED` body, not a `TransactionId` heuristic — direction 2 above,
but as a boolean marker rather than the design decision I flagged as open.
The ruling's producer sweep (already recorded in that file) refutes the
task-227 blocker this report raised: name-change/world-transfer go through
the ordinary cash-shop purchase pipeline (`handleStatusEventPurchase`
already self-announces `CashQueryResult`) and never touch
`EVENT_TOPIC_WALLET_STATUS` at all, so they were never at risk from any
skip rule on that topic.

### What changed

1. `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go`
   — added `SceneRefreshOwned bool` (`json:"sceneRefreshOwned,omitempty"`)
   to `StatusEventUpdatedBody`, documented as: the originating operation's
   own status handler announces the scene refresh, so a
   `CashSceneCashShop` consumer must skip its own.
2. `services/atlas-channel/atlas.com/channel/kafka/message/wallet/kafka.go`
   — the identical field/tag/comment, keeping the duplicated contract in
   step per repo convention.
3. `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/wallet/producer.go`
   — added `UpdateStatusEventWithTransactionSceneRefreshOwnedProvider`, a
   sibling of the existing `UpdateStatusEventWithTransactionProvider` with
   `SceneRefreshOwned: true`. Did not add a bool parameter to the existing
   provider (it's also called from `UpdateStatusEventProvider` and would
   have forced every non-gift call site to pass `false`).
4. `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go` — added
   `UpdateWithTransactionSceneRefreshOwned` to the `Processor` interface and
   `ProcessorImpl`, a sibling of `UpdateWithTransaction` that calls the new
   provider. `AdjustCurrencyWithTransaction` / `UpdateAndEmitWithTransaction`
   (the task-227/MTS/GM-award path) are untouched and keep emitting the
   unflagged event.
5. `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go`
   step 5 — now calls `walP.UpdateWithTransactionSceneRefreshOwned(buf)(transactionId)(...)`
   instead of `UpdateWithTransaction`, keeping the transaction tag from step
   1 and adding the ownership flag.
6. `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go`
   (`handleWalletUpdated`) — in the `CashSceneCashShop` arm only, `return
   nil` when `e.Body.SceneRefreshOwned` is set, before the existing
   `CashQueryResult` announce. `CashSceneMts` is untouched.

No new domain type/constant was needed (`SceneRefreshOwned` is a `bool`
field on an existing struct); `libs/atlas-constants/` has nothing
comparable.

### Tests

Added `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer_test.go`
(no test file previously existed for this consumer). Four cases, matching
the ruling's `## Verification`:

- `TestHandleWalletUpdatedCashShopSceneRefreshOwnedSkipsAnnounce` —
  `CashSceneCashShop` + `SceneRefreshOwned: true` → nothing announced.
- `TestHandleWalletUpdatedCashShopSceneRefreshUnownedAnnounces` —
  `CashSceneCashShop` + `SceneRefreshOwned: false` → `CashQueryResult`
  announced.
- `TestHandleWalletUpdatedNonNilTransactionIdWithoutFlagStillAnnounces` — a
  non-Nil `TransactionId` with the flag unset still announces
  `CashQueryResult`. This is the test the ruling calls out explicitly to
  pin the rejection of the `TransactionId` heuristic.
- `TestHandleWalletUpdatedMtsSceneUnaffectedBySceneRefreshOwned` —
  `CashSceneMts` announces `MtsOperation2` regardless of the flag's value
  (table over `true`/`false`).

The existing `TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem`
(landed in `7addf1508`, unchanged by this task) still asserts
`GIFT_PURCHASED` announces `CashShopOperationWriter` then
`CashQueryResultWriter` in that order.

### Commands run

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Result: `ok` for every package, including
`ok  	atlas-channel/kafka/consumer/wallet	0.010s` and
`ok  	atlas-channel/kafka/consumer/cashshop	(cached)`.

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
Result: `ok` for every package with tests, including
`ok  	atlas-cashshop/kafka/producer/wallet	0.033s` and
`ok  	atlas-cashshop/wallet	0.069s` (build clean, no failures).

`gofmt -l` on every changed/added file: no output (all formatted).
`tools/lint.sh` was not run — it is the controller/verifier's gate per
Contract 2, not the implementer's.

### Files changed (round 2 step 2)

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go`
- `services/atlas-channel/atlas.com/channel/kafka/message/wallet/kafka.go`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/wallet/producer.go`
- `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer_test.go` (new)
- `docs/tasks/task-240-cash-shop-stub-operations/bug-round-2-gift-notice-step2-ruling.md`
  (added to version control alongside the report; it was provided as an
  input and documents the binding decision, mirroring how
  `bug-round-2-gift-notice-brief.md` is already tracked)

### Self-review

- Matches the ruling's file inventory and naming guidance exactly; only
  departure is a name (`SceneRefreshOwned`, the ruling's own suggestion) and
  the exact provider/method names, which the ruling explicitly leaves to
  implementer judgment.
- Did not touch `AdjustCurrencyWithTransaction` / `UpdateAndEmitWithTransaction`
  or any of their callers — the MTS/GM-award/saga-ingress producers the
  ruling's sweep found keep their existing unflagged refresh with zero
  behavior change, as required.
- `CashSceneMts` arm in `handleWalletUpdated` is untouched (test asserts
  this directly).
- `Do not touch` from the original brief (GiftDone codec, GIFT_SUCCESS mode
  table, seed templates) still respected — nothing in this diff touches
  those.
- Tree is clean aside from the pre-existing untracked
  `bug-round-2-defect-f-report.md` / `review-bug-round-2*.md` files, left
  alone as instructed.

### Concerns

None. The live-bug fix (steps 1–3 combined) is now complete: the gift's
wallet debit is transaction-tagged and flagged owned (step 1 + step 2),
`handleWalletUpdated` no longer races a second `CashQueryResult` in ahead of
`GIFT_SUCCESS` (step 2), and `handleStatusEventGiftPurchased` announces the
correct order itself (step 3). Live re-test (gift an item, confirm "All the
gifts have been sent…") is now meaningful but was not run — this was a
module-local implementation task, and the brief's live re-test bullet is
outside the module-local verification contract.
