# Gift notice step 2 — ruling

Continuation of `bug-round-2-gift-notice-brief.md`. Steps 1 and 3 landed in
`7addf1508`. Step 2 was held PARTIAL pending the brief's
"Decide before implementing step 2" gate. This file closes that gate.

## The first sweep's blocker was wrong

The round-1 implementer reported that task-227's name-change / world-transfer
coupon flows debit NX via `AdjustCurrencyCommand`, and would therefore lose
their wallet refresh under a broad `TransactionId != uuid.Nil` skip. **That is
refuted.** Those flows go through the ordinary cash-shop purchase pipeline:

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:362`
(`handleBuyNameChange`; `:416` `handleBuyWorldTransfer`) calls
`cashshop.NewProcessor(...).RequestPurchase(...)` → `COMMAND_TOPIC_CASH_SHOP` →
`handleCommandRequestPurchase`
(`services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go:91`)
→ `cashshop.Processor.Purchase`
(`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:170-239`),
which debits in-process and emits a **cash-shop purchase status event**.
`handleStatusEventPurchase` already self-announces `CashQueryResult`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:315`).
They never touch `EVENT_TOPIC_WALLET_STATUS`.

## What the broad skip would actually cost

A repo-wide sweep of every construction site of an `AwardCurrency` saga step —
the only route to `AdjustCurrencyCommand` → transaction-tagged wallet `UPDATED`
— found three producers with no cash-shop status event of their own:

| Producer | Trigger | Scene at fire time |
|---|---|---|
| MTS auction settle, seller credit (`services/atlas-mts/atlas.com/mts/listing/processor.go:1319`) | unattended periodic `Sweep` (`task/periodic.go:89`) | unconstrained |
| `MtsSettlePurchase` seller-credit leg (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:2418-2428`) | buyer's saga completes; seller passive | unconstrained |
| GM `@award` (`services/atlas-messages/atlas.com/messages/command/character/currency_commands.go:85`) | GM chat command | target's scene, unconstrained |

Buyer legs are `CashSceneMts`, which the proposed skip does not guard.
Saga-orchestrator ingress (`saga/resource.go:22` REST, `kafka/consumer/saga/consumer.go:52`)
imposes no step-type allow-list, so an arbitrary `AwardCurrency` is also possible.

Exposure is therefore **stale displayed NX** for a passive credit recipient who
happens to have the cash shop open — not a broken flow. Real, but marginal.

## Ruling: explicit ownership flag, not a TransactionId heuristic

Do **not** key the skip off `TransactionId`. Add an explicit field to the wallet
`UPDATED` body meaning *"the originating operation's own status handler
announces the scene refresh."* Only the gift path sets it. Every other producer
— MTS settle, GM `@award`, arbitrary saga ingress — keeps its existing refresh
with zero behavior change.

Rationale: `TransactionId` answers "was this correlated to a saga," which is not
the question being asked. Overloading it silently couples an unrelated concern
to every future transaction-tagged producer. The flag states the actual
invariant and is additive, so it is trivially reversible.

### Files

1. `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go` —
   add the field to `StatusEventUpdatedBody`, `omitempty`.
2. `services/atlas-channel/atlas.com/channel/kafka/message/wallet/kafka.go` —
   the same field, same JSON tag. The contract is duplicated per service by
   convention; keep both in step.
3. `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/wallet/` (the
   package providing `UpdateStatusEventProvider` /
   `UpdateStatusEventWithTransactionProvider`, referenced from
   `wallet/processor.go:118` and `:151`) — a provider variant that sets it.
   Follow the existing provider naming; do not add a bare bool parameter to the
   existing exported providers if that breaks their call sites.
4. `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go` — thread it
   through whichever `Update*` variant the gift path uses. `processor_gift.go:131`
   currently calls `UpdateWithTransaction`; the transaction tag added in step 1
   is still wanted (the saga orchestrator ignores unknown transaction ids —
   `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/cashshop/consumer.go:54`
   gates on `AcceptEvent`), so keep it and add the flag alongside.
5. `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go`
   (`handleWalletUpdated`, `:76-86`) — in the `CashSceneCashShop` arm only,
   return nil when the flag is set. **Leave the `CashSceneMts` arm untouched.**

### Naming

`SceneRefreshOwned` is a suggestion, not a mandate. Pick whatever reads best
against the surrounding code and document the invariant in a comment on the
field. Check `libs/atlas-constants/` before introducing any new type.

## Verification

- Wallet-consumer test: `CashSceneCashShop` + flag set → nothing announced;
  `CashSceneCashShop` + flag unset → `CashQueryResult` announced; `CashSceneMts`
  unaffected either way. A non-Nil `TransactionId` with the flag unset must
  still announce — that pins the ruling against the rejected heuristic.
- The existing gift ordering test from `7addf1508` must still pass:
  `GIFT_PURCHASED` announces `CashShopOperationWriter` then
  `CashQueryResultWriter`, in that order.
- Module-local `go build` / `go test` for atlas-cashshop and atlas-channel only.
- Formatting authority is `tools/lint.sh` (no flags = fix mode); bare `gofumpt`
  disagrees with the repo config.

## Still binding from the original brief

`## Do not touch` — the `GiftDone` codec, the `GIFT_SUCCESS` mode table, and all
seed templates. The v83 arm is tier-1 fixture-verified; only ordering was wrong.
