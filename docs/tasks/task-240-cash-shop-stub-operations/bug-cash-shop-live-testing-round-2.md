# bug-cash-shop-live-testing-round-2

PR: atlas-pr-1426 · branch `task-240-cash-shop-stub-operations` · HEAD at triage `fc9ad271c`
Environment: namespace `atlas-pr-1426`, tenant `f3fc852d-555a-45b1-80d8-578ea3b9f401`, region GMS, client version **83.1** (confirmed from the tenant config broadcast in the `atlas-channel` log and the `ms.version":"83.1","region":"GMS"` log fields).
Pods read: `atlas-cashshop-88b475f55-rf45j`, `atlas-channel-5f8575f84d-dqcrr` (both started 19:06 UTC, i.e. AFTER the round-1 fixes `f64b454a3`/`bcb403bd7`/`e6dffd8f6`/`56108b790`; the round-1 Defect A/B/C fixes are live in this run).

Live session window: 19:21–19:31 UTC on 2026-08-26.

Five symptoms were reported. They resolve to **four** independent defects plus **one** unresolved symptom.

---

## Defect D — inventory-slot purchase by TYPE grants 8 slots, not the advertised 4

Covers reported item **2**.

### Reproduced
Live, 19:23:35 UTC. Cash Shop UI → inventory slot expansion for the USE tab (`CASHSHOP_OPERATION` mode 6 with `item = false`, i.e. `REQUEST_INVENTORY_INCREASE_BY_TYPE`).

### Observed
`atlas-cashshop`:
```
19:23:35.795 Character [1] attempting to purchase inventory [2] increase using currency [1]. Cost is [4000].
19:23:35.806 Character [1] purchased inventory [2] increase. New capacity will be [40].
```
`atlas-channel` consumed:
```
19:23:35.976 {"characterId":1,"type":"INVENTORY_CAPACITY_INCREASED","body":{"inventoryType":2,"capacity":40,"amount":8}}
```
Capacity moved 32 → 40, i.e. **+8**. The by-ITEM path taken 20 s earlier granted +4:
```
19:23:15.197 Character [1] attempting to purchase inventory [4] increase using currency [1]. Cost is [6800].
19:23:15.383 {"characterId":1,"type":"INVENTORY_CAPACITY_INCREASED","body":{"inventoryType":4,"capacity":28,"amount":4}}
```

### Expected
+4 slots, matching what the v83 Cash Shop UI advertises for the 4 000 NX by-type expansion (tester's live observation) and matching the by-ITEM path.

### Root cause
`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:354-360`,
`PurchaseInventoryIncreaseByTypeAndEmit`, hard-codes the amount:

```go
return NewProcessor(p.l, p.ctx, tx).PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, 4000, 8)
```

while `PurchaseInventoryIncreaseByItemAndEmit` (same file, line ~347) passes `4`.

**This is NOT a task-240 regression** — the literal `8` is present on `main` at the
task base `32d55cb21` (`services/atlas-cashshop/.../cashshop/processor.go:261`).
It is in scope here because the by-type path only became reachable/testable in
this PR's live pass.

The cost literal `4000` is left alone: the tester did not report a price
mismatch and there is no client evidence in-repo pinning it.

---

## Defect E — buy-a-package-for-yourself answers with the GIFT arm, so the client never receives the item blobs

Covers reported item **3** ("I bought the robot package, it awarded me the items, but I didn't see them appear in my cash inventory until I left/re-joined the cash shop").

### Reproduced
Live, 19:24:23 UTC. Cash Shop → buy package `9101376` for self (`CASHSHOP_OPERATION` mode 30, `BUY_PACKAGE`).

### Observed
`atlas-channel` consumed:
```
19:24:23.935 {"characterId":1,"type":"PACKAGE_PURCHASED","body":{"transactionId":"18b4d48b-...","compartmentId":"3fc367d1-...","assetIds":[16,17,18,19],"packageTemplateId":9101376,"price":12000,"recipientCharacterId":1,"recipientName":"Atlas"}}
```
Note `characterId == 1` **and** `recipientCharacterId == 1`.

The consumer then made **no** HTTP call at all. Contrast the `PURCHASE` event at
19:24:08 and the `RING_PURCHASED` events at 19:29:55 / 19:30:44, each of which
is immediately followed in the same log by
`GET /api/accounts/1/cash-shop/inventory/compartments/<id>/assets/<n>`. The
buy-for-self arm of `handleStatusEventPackagePurchased` is the only arm that
fetches assets; the gift arm makes no HTTP call. Its absence is the proof that
the **gift arm** ran.

The client therefore received `GIFT_PACKAGE_SUCCESS` (mode 139 on v83) — an arm
whose wire shape carries **no item blob at all**
(`libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`,
`GiftPackageDone`) — instead of `BUY_PACKAGE_SUCCESS` (mode 137), which carries
the count-prefixed `GW_CashItemInfo` list. Nothing was appended to the client's
`m_aCashItemInfo`, so the four assets only appeared on the next locker load.

### Expected
`BUY_PACKAGE_SUCCESS` with the four projected `CashInventoryItem` blobs.

### Root cause
Producer/consumer contract mismatch across the `EVENT_TOPIC_CASH_SHOP_STATUS`
seam.

- Producer: `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_package.go:139`
  sets `effectiveRecipientCharacterId := characterId` and only overwrites it
  when the *command's* `recipientCharacterId != 0`. So on the status event the
  field is **never zero**.
- The status body's own doc comment states this deliberately:
  `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:372-375`
  — "RecipientCharacterId/RecipientName echo the buyer's own identity on a
  buy-for-self purchase (RecipientCharacterId == 0 on the **command**)".
- Consumer: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:419`
  branches on `if e.Body.RecipientCharacterId != 0`, i.e. it applies the
  **command** body's zero-means-self convention to the **status** body, where it
  can never hold. The gift arm is therefore taken unconditionally.

The `handleStatusEventGiftPurchased` doc comment directly above it makes the
same convention explicit for the gift event, which is why the mistake reads as
plausible in review — the gate cannot see it, only a live run or a seam test can.

Recommended fix: discriminate on `e.Body.RecipientCharacterId != e.CharacterId`
in the consumer (the status body already tells the truth), and pin it with a
consumer test that asserts a buy-for-self `PACKAGE_PURCHASED` announces
`BUY_PACKAGE_SUCCESS` with `len(items) == len(AssetIds)`. The existing
consumer tests in `consumer_test.go` evidently only exercised the gift arm or
constructed a body with a zero recipient — the new test must use a body whose
`RecipientCharacterId == CharacterId`.

---

## Defect F — a gift recipient can never read the gift message: the channel never sends LOAD_GIFT_SUCCESS

Covers reported item **4** ("while it does award the recipient a ring in that character's cash inventory, I don't ever get to see the message sent along with it. I assume this'll also be a bug with 'gift'").

### Reproduced
Live: the gift at 19:22:20 (`Character [1] gifted item [1003050] (asset [14]) to character [2]`) carried a `giftMessage`; the ring purchases at 19:29:55 / 19:30:44 likewise create the partner's half. In no case did the recipient see the accompanying message.

### Observed
The asset is created with `GiftFrom`/`GiftMessage`
(`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift.go:141`,
`astP.CreateGift(...)(..., characterId, senderName, giftMessage)`), so the
message **is** persisted.

But `LOAD_GIFT_SUCCESS` — the only arm that carries gift text to the client
(`GiftListEntry.sText`, a 73-byte fixed field inside the 98-byte `GW_GiftList`
record; `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`) —
is **never announced by atlas-channel**. `grep -rn "LoadGiftDone\|CashShopLoadGift"
services/atlas-channel/atlas.com/channel/` returns exactly one hit, and it is a
comment in `socket/handler/note_operation.go:41`. There is no producer of that
arm anywhere in the channel.

Correspondingly, the channel also never handles the client's gift-list request
(`CCashShop::SendGiftsPacket` exists in the v83 export at
`docs/packets/ida-exports/gms_v83.json`), and the buyer-side arms that DO fire
(`GIFT_SUCCESS`, `COUPLE_SUCCESS`, `FRIENDSHIP_SUCCESS`) carry no message field
by design — `GiftDone` is `mode + recipientName + itemId + quantity +
nxCashSpent` and nothing else, confirmed against the v83 export
(`CCashShop::OnCashItemResult#GIFT_SUCCESS @ 0x47a856`).

### Expected
The recipient opens the Cash Shop's gift/locker view and sees the sender name
and the accompanying message.

### Root cause
The receive-a-gift half of the gift feature is unimplemented in atlas-channel:
no `LOAD_GIFT_SUCCESS` announcement, and no serverbound handler for the client's
gift-list request. The send half (task-240 task 13) landed without it.

This is a **missing feature, not a broken one**. Whether it belongs in this PR
or a follow-up task is a scope call for the user — see `## Not yet answered`.

---

## Defect G — couple/friendship ring pairing has no in-field effect and no partner display

Covers reported item **5** ("when I am in game, I don't see the character effect when the two stand next to each other. You also don't see who the ring is connected with").

### Reproduced
Live, 19:29:55 (FRIENDSHIP, template 1112800, pair `f5d6ccc7-…`) and 19:30:44
(COUPLE, template 1112005, pair `e549f926-…`). Both halves were created; neither
produced any in-field behaviour.

### Observed
`RING_PURCHASED` carries `pairId`, and atlas-cashshop persists the pair. But the
pairing is not consumed anywhere downstream: a case-insensitive sweep for ring
pairing across `services/atlas-channel` and `services/atlas-character` finds no
ring-pair type, no ring-pair REST surface, and no consumer of `pairId` — the
only `ring` hits are unrelated substring matches ("during", "string").

`handleStatusEventRingPurchased`'s own doc comment
(`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:461-470`)
states that the partner's half is deliberately not announced ("there is no live
session correlation for it on this event (see OQ-R1)").

### Expected (per the tester)
- The couple-ring proximity effect when both wearers are near each other.
- The partner's name surfaced where the client shows ring linkage.

### Root cause
Ring **pairing semantics** (equip-side linkage, proximity effect broadcast,
partner name in the character info / ring UI) were never in task-240's scope —
the task delivered the *purchase* of a ring pair, not its *behaviour*. `pairId`
is written and never read.

This is a **feature gap outside the PR's scope**, not a defect in what was
built. It needs its own spec (which packets carry the effect on v83, who owns
ring state, how proximity is evaluated). Do not attempt it as a bug fix — see
`## Not yet answered`.

---

## Unresolved symptom — "the gift could not be sent" error on the client

Covers reported item **1**. **Root cause NOT established. Do not guess one.**

### What the evidence rules OUT

1. **Server-side failure.** The gift fully succeeded:
   `atlas-cashshop 19:22:20.655 Character [1] gifted item [1003050] (asset [14]) to character [2] for [2500] currency.`
   No error- or warning-level line in `atlas-cashshop` in that window (the only
   two are a startup `purchaserecord/backfill.go:127` `uuid = text` warning at
   19:06 and the `STATUS_TOPIC_CASH_ITEM` env-var default notice — see below).
2. **A server-sent GIFT_FAILED.** Every `announceGiftFailure` path in
   `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_gift.go:91-140`
   logs before it fires; none did. The channel log for 19:22:20 shows only
   `[CashShopOperationHandle] read [op [4]]` → recipient lookup → sender lookup
   → `Character [1] gifting serial [10002308] to character [2]. Transaction [...]`.
3. **A mis-resolved mode byte.** `atlas-channel` consumed
   `19:22:20.843 {"characterId":1,"type":"GIFT_PURCHASED",...}` and
   `handleStatusEventGiftPurchased` announces `CashShopGiftDoneBody`. The v83
   seed template maps `GIFT_SUCCESS → 94`
   (`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`,
   `CashShopOperation` writer, opCode `0x145`), and the v83 IDA export confirms
   `CCashShop::OnCashItemResult#GIFT_SUCCESS @ 0x47a856` dispatches on
   **`0x5e` = 94**. No `Code [...] not configured in property [operations]`
   warning appears in the channel log.
4. **A wrong wire shape.** The v83 export's read order for that arm is
   `Decode1(mode) · DecodeStr(recipientName) · Decode4(itemId) ·
   Decode2(quantity) · Decode4(nxCashSpent)` — byte-for-byte what
   `GiftDone.Encode` writes for a GMS tenant
   (`giftHasNxCashSpent` is region-gated on `GMS`).
5. **A dead session.** The same session received and rendered
   `INVENTORY_CAPACITY_INCREASE_SUCCESS` at 19:23:15 while inside the Cash Shop
   (the tester saw the slots change), so `IfPresentByCharacterId(sc.Channel())`
   resolves for a cash-shop-resident character.

### What is still unknown
Why the client rendered a failure notice when the server sent a correctly-moded,
correctly-shaped `GIFT_SUCCESS`. The verbatim on-screen string is the missing
evidence — each v83 failure notice maps to a specific `NoticeFailReason` byte
under `GIFT_FAILED` (mode 95, `Decode1(mode) · Decode1(errorCode)`), and the
exact wording discriminates a client-side pre-send refusal from a server-arm
mismatch. **Ask the tester for the exact text and whether Chronicle actually
received the Bunny Ears in their cash locker** before opening this line of
investigation. Do not fix speculatively.

---

## Fix

### Defect D
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:354-360` —
  `PurchaseInventoryIncreaseByTypeAndEmit`: change the hard-coded amount `8` to
  `4`. Leave the `4000` cost literal alone.
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_test.go` (or
  the nearest existing inventory-increase test file — the round-1 fix added
  cases there) — add a case asserting the by-type path emits
  `InventoryCapacityIncreasedBody.Amount == 4` and moves capacity by 4. Assert
  parity with the by-item path.
- `services/atlas-cashshop/docs/` — update any doc that states the by-type grant
  size, if one exists (`grep -rn "8 slots" services/atlas-cashshop/`).

### Defect E
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:419`
  — `handleStatusEventPackagePurchased`: replace `e.Body.RecipientCharacterId != 0`
  with `e.Body.RecipientCharacterId != e.CharacterId`. Update the function's doc
  comment (lines 392-406), which currently asserts the ZERO convention, to state
  the status body's actual convention and cite
  `kafka/message/cashshop/kafka.go:372-375`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`
  — add a buy-for-self case whose body sets `RecipientCharacterId == CharacterId`
  and assert the announced arm decodes as `BuyPackageDone` with one item per
  `AssetId`. Keep/adjust the existing gift case so it uses a recipient id that
  differs from `CharacterId`.
- Check the same `!= 0` discriminator has not been copied into any other handler
  in that file (`grep -n "RecipientCharacterId" services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`).

### Defect F and Defect G
**Do not implement without a scope ruling** — see `## Not yet answered`.

---

## Not yet answered

- **Scope of Defect F (gift-receive UI).** Implementing it means a new serverbound
  handler for `CCashShop::SendGiftsPacket` plus a `LOAD_GIFT_SUCCESS`
  announcement backed by a gift-list read on atlas-cashshop (the assets already
  carry `GiftFrom`/`GiftMessage`). That is a task-sized change, not a bug fix.
  **User decision required: land it in this PR or spec it as a new task.**
- **Scope of Defect G (ring pairing behaviour).** Genuinely out of task-240's
  delivered scope and needs its own spec (which v83 packets carry the couple
  effect, who owns ring-pair state, proximity evaluation). **Recommend a new
  task; user decision required.**
- **The client's exact gift-failure string** (see the Unresolved symptom above).
- **`STATUS_TOPIC_CASH_ITEM` is not set in the `atlas-pr-1426` deployment.**
  `atlas-cashshop` logs `[STATUS_TOPIC_CASH_ITEM] environment variable not set.
  Defaulting to provided token.` on every cash-shop command. The token default
  happens to match what the channel consumes in this environment (all the status
  events above were delivered), so it is not the cause of any symptom here — but
  the deployment manifest should declare it. Verify against
  `services/atlas-cashshop/README.md:46` and the overlay.
- **`purchaserecord/backfill.go:127` fails on startup**:
  `ERROR: operator does not exist: uuid = text (SQLSTATE 42883)`, logged as
  `Failed to backfill purchase records from cash_assets history.` The backfill
  is comparing a `uuid` column against a `text` value. It is non-fatal (the
  service starts and purchases work), and it is a **separate defect from the
  round-1 Defect A** upsert fix. Not triaged here.
- The round-1 `UNKNOWN_ERROR` template-mapping gap remains open and untouched.

---

## Outcome

<!-- filled in after the fix lands -->
