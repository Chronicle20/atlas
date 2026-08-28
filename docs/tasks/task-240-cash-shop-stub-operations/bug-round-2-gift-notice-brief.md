# Gift notice — root cause and fix brief

Resolves the "Unresolved symptom" in `bug-cash-shop-live-testing-round-2.md`
(reported item 1: recipient receives the item, sender sees **"The gifts could
not be sent."**). Root cause is **established**, from the v83 IDB plus the live
log. Branch `task-240-cash-shop-stub-operations`.

## Root cause: a cross-topic ordering race, decided against us by ~99 ms

The v83 gift flow is a **batch state machine**, not a single request/response.
Decompiled from the v83 IDB (`GMS/v83_Me/MapleStory_dump.exe.i64`):

`CCashShop::SendGiftsPacket @ 0x46f940` keeps four fields:
- `this+1280` — array of recipient names for this batch
- `this+1284` — send cursor
- `this+1300` — array of recipients **confirmed** by the server
- `this+1288` — "batch in progress" flag (`this[322]`)
- `this+1292` — accumulated NX spent

Each call does one of two things:
- cursor != len(recipients) → encode and send **one** gift packet, cursor++;
- cursor == len(recipients) → emit the **final notice**, then reset the batch
  (`RemoveAll(1280)`, `RemoveAll(1300)`, `*(this+1288) = 0`):
  - `len(1300) == len(1280)` → `SP_561` "All the gifts have been sent…"
  - `1300` non-empty but short → `SP_563` "…could not be sent to several"
  - **`1300` EMPTY → `SP_562` "The gifts could not be sent."** ← what the tester saw

`CCashShop::OnCashItemResGiftDone @ 0x47a856` only **appends** to `this+1300`
(when `*(this+1288)` is set). **It never advances the batch.**

The batch is advanced by exactly three callers (`xrefs_to 0x46f940`):
`CCashShop::OnGift` (starts it), `CCashShop::OnCashItemResGiftFailed`, and —
critically — **`CCashShop::OnQueryCashResult @ 0x478f81`**, which ends with:

```c
if ( this[322] )                       // this+1288, the batch flag
    CCashShop::SendGiftsPacket(this);
```

So **`CASH_QUERY_RESULT` is the packet that drives the gift batch to its final
notice.** The client's required order is:
`GIFT_SUCCESS` (records the confirmation) **then** `CASH_QUERY_RESULT`
(resolves the batch).

We send them in the opposite order, on two different Kafka topics with no
cross-topic ordering guarantee. Live proof from `atlas-channel`:

```
19:22:20.744  EVENT_TOPIC_WALLET_STATUS      {"accountId":1,"type":"UPDATED","body":{"credit":86700,...,"transactionId":"00000000-0000-0000-0000-000000000000"}}
19:22:20.843  EVENT_TOPIC_CASH_SHOP_STATUS   {"characterId":1,"type":"GIFT_PURCHASED",...}
```

The wallet event won by **99 ms**. Sequence at the client:
1. `CASH_QUERY_RESULT` arrives → `OnQueryCashResult` → `this[322]` set →
   `SendGiftsPacket` → cursor == count, `this+1300` still **empty** → `SP_562`
   **"The gifts could not be sent."**, batch reset, `*(this+1288) = 0`.
2. `GIFT_SUCCESS` arrives 99 ms later. `*(this+1288)` is now 0, so it takes the
   `else` branch and shows the per-gift success text instead of appending.

This is why the recipient keeps the item and only the sender's notice is wrong,
and why it is timing-dependent rather than deterministic.

Producer side: `processor_gift.go` step 5 calls plain
`walP.Update(buf)(...)` — no transaction id — which is why the live wallet event
carried `transactionId: 00000000-…`. The `GIFT_PURCHASED` put happens later in
the same buffer, but the two land on different topics.

## Why the other arms were unaffected

`handleStatusEventInventoryCapacityIncreased`
(`kafka/consumer/cashshop/consumer.go:157`) and `handleStatusEventPurchase`
(`:315`) **already** announce `CashQueryResult` themselves after their arm. Those
flows have no batch state machine, so the duplicate `CASH_QUERY_RESULT` (one
from the wallet topic, one from the handler) is harmless — which is exactly why
this ordering bug stayed hidden until a batch-driven arm exercised it.

## Fix — the machinery already exists

`wallet.ProcessorImpl.UpdateWithTransaction` /
`UpdateAndEmitWithTransaction` (`services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go:137-165`)
already thread a transaction id into the wallet status event, and
`StatusEventUpdatedBody.TransactionId`
(`services/atlas-channel/atlas.com/channel/kafka/message/wallet/kafka.go:20`)
already carries it. The gift path simply does not use them.

1. `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go` step 5
   — call `walP.UpdateWithTransaction(buf)(transactionId)(...)` instead of
   `walP.Update(buf)(...)`, so the wallet event is tagged with the gift's
   transaction.
2. `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go:76-85`
   (`handleWalletUpdated`) — skip the `CashSceneCashShop` `CashQueryResult`
   announce when `e.Body.TransactionId != uuid.Nil`: that wallet movement
   belongs to a cash-shop operation whose own status handler owns the ordering.
   Leave the `CashSceneMts` arm alone.
3. `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
   `handleStatusEventGiftPurchased` — after announcing `CashShopGiftDoneBody`,
   fetch the wallet and announce `CashQueryResult`, mirroring
   `handleStatusEventInventoryCapacityIncreased` (`:150-159`). This is what
   drives the client's batch to `SP_561`.

### Decide before implementing step 2 — do not guess

Skipping on *any* non-Nil `TransactionId` is the simple rule, but it is only safe
if every other producer of a transaction-tagged wallet `UPDATED` also has a
status handler that announces `CashQueryResult` itself. **Sweep every producer
of `UpdateWithTransaction` / `UpdateAndEmitWithTransaction` /
`AdjustCurrencyWithTransaction` and every `CashScene`-relevant consumer path
before committing to it** — the name-change and world-transfer flows (task-227)
use transaction ids and must not silently lose their wallet refresh. If any
producer would be left without a refresh, either give it one on its own status
handler or narrow the skip. Report PARTIAL rather than guessing.

## Verification

- A consumer test pinning the ordering contract: a `GIFT_PURCHASED` status event
  announces `GIFT_SUCCESS` **followed by** `CashQueryResult`, in that order.
- A wallet-consumer test: a `CashSceneCashShop` session with a non-Nil
  `TransactionId` announces nothing; with `uuid.Nil` it still announces
  `CashQueryResult`; an MTS session is unaffected either way.
- Module-local `go build` / `go test` for atlas-cashshop and atlas-channel only.
- Formatting authority is `tools/lint.sh` (no flags = fix mode); bare `gofumpt`
  disagrees with the repo config and will fail the gate.
- Live re-test is the real proof: gift an item and confirm the sender sees
  "All the gifts have been sent…" rather than "The gifts could not be sent."

## Do not touch

The `GiftDone` codec, the `GIFT_SUCCESS` mode table, or any seed template. The
v83 arm is tier-1 fixture-verified
(`docs/packets/evidence/gms_v83/cash.clientbound.CashGiftDone.yaml`, mode `0x5E`)
and the decompile above confirms the body byte-for-byte. The packet was always
correct; only its ordering relative to `CASH_QUERY_RESULT` was wrong.
