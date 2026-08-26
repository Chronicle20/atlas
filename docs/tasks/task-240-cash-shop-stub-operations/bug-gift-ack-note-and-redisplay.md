# bug-gift-ack-note-and-redisplay

PR: atlas-pr-1426 · branch `task-240-cash-shop-stub-operations` · HEAD at triage `dd7e0bbb4`
Environment: namespace `atlas-pr-1426`, tenant `f3fc852d-555a-45b1-80d8-578ea3b9f401`, region GMS, client version **83.1** (`"ms.version":"83.1","region":"GMS"` on every session log line below).
Pods read: `atlas-channel-dc8d6668d-sv8bg`, `atlas-cashshop-fc887966c-v9wgb` (both started 21:26 UTC 2026-08-26).
Live window: 21:56–21:58 UTC on 2026-08-26.

Two reported symptoms, **two independent defects**, both root-caused. Defect G
is the direct cause of the tester's symptom 1; Defect H is symptom 2 and is
also what turns G's single failure into a repeating one.

---

## Defect G — the recipient's gift acknowledgement note is rejected, so the gifter never receives it

Covers reported item **1**.

### Reproduced

Live. Character 1 ("the gifter") gifts serial `10002321` to character 2
("Chronicle"); character 2 enters the cash shop, sees the received-gift modal,
types a message and clicks OK.

### Observed

`atlas-channel`:

```
21:56:07.893 Character [1] gifting serial [10002321] to character [2]. Transaction [bcdd8343-6070-4a1f-afa3-d06d8fbd60f1].
21:56:08.218 Message received {"characterId":1,"type":"GIFT_PURCHASED","body":{"transactionId":"bcdd8343-...","recipientName":"Chronicle","templateId":1003057,"quantity":1,"price":3100,"recipientCharacterId":2}}.
21:56:18.861 [NoteOperationHandle] read [op [0]]
21:56:18.865 Character [2] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.
21:56:22.472 Character [2] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.
21:58:15.620 Character [2] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.
21:58:28.751 Character [2] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.
21:58:32.699 Character [2] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.
```

No note is ever created; the gifter's memo list stays empty. The client
nevertheless shows `SP_2713 "The note has successfully been sent."` — it is
emitted unconditionally, before any server reply, so the failure is invisible
in-game.

### Expected

Clicking OK on the received-gift modal creates a note addressed to the
**gifter** (`GiftListEntry.BuyCharacterName`) carrying the message the
recipient typed. It costs nothing: the note is paid for by the gift purchase,
not by a Note item.

### Root cause

`CCashShop::OnCashItemResLoadGiftDone @ 0x47959e` (v83 IDB
`GMS/v83_Me/MapleStory_dump.exe.i64`, session `754107bf`) loops the gift list
and, for each entry whose modal returns `DoModal() == 1`, writes:

```
COutPacket(0x83)            // NOTE_ACTION
Encode1(0)                  // op 0 = SEND
EncodeStr(entry + 12)       // sBuyCharacterName -> the GIFTER, the note's receiver
EncodeStr(<text typed in the modal>)
Encode1(1)                  // giftFlag, hardcoded 1
Encode4(v2)                 // 0-based gift index
EncodeBuffer(&entry.liSN, 8)// the gift's cash-item SN
CClientSocket::SendPacket(...)
StringPool::GetString(SP_2713_THE_NOTE_HAS_SUCCESSFULLY_BEEN_SENT); CUtilDlg::Notice(...)
```

This is the *only* client-side writer of the NOTE_ACTION SEND arm, a fact
already recorded in
`libs/atlas-packet/note/serverbound/operation_send.go:25-37` — and the codec
already decodes `giftFlag`, `giftIndex`, `giftSN` in full. The wire is correct.

The handler is not.
`services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:40-60`
gates the arm on **Note-item ownership** (`item.ClassificationNote`) and, on
success, routes it through `handleNoteSendRequest`, whose saga
(`note_send.go:20-58`) *destroys a Note item first*. Its own comment states the
intended contract:

> "When gifting lands, gift sends must NOT route through this consume-gated
> path (the note is paid for by the gift purchase) — see design §2.2."

Gifting has now landed; the gate was never revisited. A gift recipient does not
own a Note item, so every acknowledgement is rejected at the gate.

Two further gates on the shared path are wrong for this arm:

- **Receiver-online rejection** (`note_send.go:76-82`). The gifter is very
  likely online (they just gifted). The client claims success regardless and
  the `MEMO_RESULT` `SEND_ERROR` reply has no UI while the cash-shop modal owns
  the screen, so an online gifter would silently lose the note.
- **Item consumption**, per the comment above.

**Assumption stated, not asked:** the gift acknowledgement bypasses both the
Note-item gate and the receiver-online gate, and consumes nothing. The
anti-tamper property is preserved by a different, stronger check — see the Fix.

### Fix

Add a gift-forward branch to the NOTE_ACTION SEND arm that does not consume and
does not route through `handleNoteSendRequest`.

1. `services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go`
   — in the `NoteOperationSend` arm, before the Note-item gate: if
   `sp.GiftFlag() == 1`, delegate to the new gift-forward handler and return.
   Leave the existing consume-gated path as the fallback for `giftFlag == 0`
   (no client writes it today; it remains the tamper path and keeps its gate
   and its warning log verbatim).
2. `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_gift.go`
   (or a new `note_gift_forward.go` beside it — implementer's call, keep it
   with the note handlers if that reads better) — the gift-forward handler:
   - Load the sender's own cash-shop compartment
     (`cashshop/inventory/compartment` — same `GetByAccountIdAndType(s.AccountId(), compartment.TypeExplorer)`
      call `cash_shop_entry.go:75` makes; the "TODO select correct compartment"
      there applies here identically, do not try to fix it).
   - Find the asset whose `Item().CashId()` equals `int64(sp.GiftSN())`. Reject
     (log + return, announce nothing) if absent.
   - Reject unless that asset's `GiftFrom()` is non-empty and equals
     `sp.ToName()`. **This is the anti-tamper gate that replaces the Note-item
     check**: a client can only mint a free note addressed to a character who
     actually gifted it an item it actually holds.
   - Resolve the receiver by name (`character.NewProcessor(...).GetByName`),
     reject on failure.
   - Create a **single-step** saga: `saga.NoteSend`, one `saga.CreateNote` step
     (`create_note`) with `Flag: 0`, `SenderId: s.CharacterId()`,
     `ReceiverId: <gifter>`, `Message: sp.Message()`. No `DestroyAsset` step,
     no online check. Keep the `Flag: 0` rationale comment from
     `note_send.go:44-52`.
   - Announce nothing on success: the client already showed SP_2713.
3. Tests: `services/atlas-channel/atlas.com/channel/socket/handler/note_send_test.go`
   / a new `note_gift_forward_test.go` — `giftFlag == 1` with a matching gift
   asset creates a one-step saga and destroys nothing; `giftFlag == 1` with a
   `GiftSN` the character does not own creates nothing; `giftFlag == 1` whose
   asset's `GiftFrom` differs from `ToName` creates nothing; `giftFlag == 0`
   still hits the Note-item gate unchanged.

---

## Defect H — the received-gift modal re-fires on every cash-shop entry; the gift is never marked acknowledged

Covers reported item **2**.

### Reproduced

Live, same window. Character 2 acknowledged the gift at 21:56:18, then again at
21:56:22, 21:58:15, 21:58:28 and 21:58:32 — five acknowledgements of a single
gift, across repeated cash-shop entries, the item still sitting in the cash
inventory.

### Observed

Five `[NoteOperationHandle] read [op [0]]` lines from character 2 for one
`GIFT_PURCHASED`. Each is a fresh LOAD_GIFT_SUCCESS → modal → OK cycle.

### Expected

A gift is presented once. Re-entering the cash shop with the (still unopened)
gift item in the locker must not re-announce it as a new gift, and must not
allow a second note to the gifter.

### Root cause

`buildGiftListEntries`
(`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go:148-163`)
derives the LOAD_GIFT_SUCCESS list from *every* locker asset with a non-empty
`GiftFrom()`. `GiftFrom` is permanent asset metadata
(`services/atlas-cashshop/.../inventory/asset/entity.go:48-52`) — it is the
locker's "gift from" attribution and is also encoded into every
`CashInventoryItem` blob. **There is no "this gift has been presented" state
anywhere in the system**, so the list is regenerated in full on every entry
(`cash_shop_entry.go:104-110`).

The client cannot compensate: `OnCashItemResLoadGiftDone` shows a modal for
every entry the server sends, and sends **nothing at all** when the user
cancels the modal (`DoModal() != 1` → the loop just advances). Therefore the
only place the presented/not-presented distinction can live is server-side, and
the only trigger that fires exactly once per presentation regardless of what
the user clicks is **the LOAD_GIFT_SUCCESS announce itself**. Draining on
announce, not on acknowledgement, is forced by the client's behavior.

### Fix

Persist a per-asset "gift presented" flag in atlas-cashshop; the channel filters
on it and drains it when it announces.

1. `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go`
   — add `GiftAcknowledged bool \`gorm:"not null;default:false"\`` (name is a
   suggestion; whatever is chosen must be documented as "the gift list has been
   presented to the recipient", which is NOT "the recipient clicked OK").
   `asset.Migration` is `AutoMigrate(&Entity{})`, so no hand-written migration —
   but confirm the existing-row default lands as `false`.
2. `.../inventory/asset/model.go` — field, getter, `ModelBuilder` setter,
   preserve it in the clone path (`model.go:105-106` region).
3. `.../inventory/asset/entity.go` `Transform` — carry it through.
4. `.../inventory/asset/administrator.go` — an `updateGiftAcknowledged`
   (or bulk-by-cashId) writer beside `updateQuantity` (`administrator.go:92`).
5. `.../inventory/asset/rest.go` + `rest_test.go` — expose
   `giftAcknowledged` on the JSON:API attributes, both directions.
6. `.../inventory/asset/processor.go` — an `AcknowledgeGifts`-style processor
   method taking the compartment and the cash ids.
7. `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
   — `CommandTypeAcknowledgeGifts = "ACKNOWLEDGE_GIFTS"` plus its body
   (accountId + `[]int64` cashIds; follow `RequestLockerRebateCommandBody`'s
   shape and its idempotency comment — a Kafka redelivery must be a no-op,
   which this naturally is).
8. `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`
   — register `handleCommandAcknowledgeGifts` alongside
   `handleCommandRequestLockerRebate` (`consumer.go:194`), same guard shape.
9. `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`
   — the same command constant and body struct. **Both service copies of this
   contract must be byte-identical in their JSON tags** (this is the seam that
   `review-bug-round-2-gift-notice.md` had to hand-check for
   `sceneRefreshOwned`; do the same here).
10. `services/atlas-channel/atlas.com/channel/cashshop/producer.go` +
    `processor.go` — `AcknowledgeGifts(accountId uint32, cashIds []int64) error`
    on `COMMAND_TOPIC_CASH_SHOP`, following `RequestLockerRebate`
    (`processor.go:236`).
11. `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/{model.go,builder.go,rest.go}`
    — mirror the `giftAcknowledged` attribute into the channel-side read model.
12. `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go`
    — `buildGiftListEntries` skips assets already acknowledged; after a
    successful LOAD_GIFT_SUCCESS announce, fire `AcknowledgeGifts` with exactly
    the cash ids just announced (skip the call when the list is empty). Leave
    the `loadGiftDoneConfigured` gate as-is: a tenant whose template has no
    LOAD_GIFT_SUCCESS key announces nothing and must drain nothing.
13. Tests: `cash_shop_entry_test.go` — an acknowledged gift asset is excluded
    from the list; an unacknowledged one is included and produces exactly one
    `AcknowledgeGifts` command carrying its cash id; an empty list produces no
    command. Plus the atlas-cashshop consumer/processor tests for the new
    command (`consumer_test.go` pattern) and the round-trip REST test.

### Interaction with Defect G

Defect G's gift-forward handler must **not** require the asset to be
unacknowledged — by the time the client's note arrives the asset has already
been drained by the announce. It validates ownership + `GiftFrom == ToName`
only. Land both fixes together; G alone would let one gift mint unlimited free
notes, and H alone would leave the gifter still note-less.

---

---

## Defect I — a replayed acknowledgement packet mints unlimited free notes

Raised by `review-bug-gift-ack-and-redisplay.md` (non-blocking finding 2) as a
residual property of the merged G+H design, not an implementer deviation. The
user reviewed it and **chose to close it with a second per-asset flag** rather
than accept it.

### Root cause

Defect H's `GiftAcknowledged` drains on the LOAD_GIFT_SUCCESS **announce**, and
`### Interaction with Defect G` forbids `handleNoteGiftForward` from gating on
it — by the time the client's note arrives the asset is already drained, so
such a gate would reject every legitimate note. Consequently nothing in the
gift-forward path is one-shot: a client that re-emits
`NOTE_ACTION SEND giftFlag=1` for a gift it still holds mints a fresh note
every time. Normal play cannot do this (the modal fires once per announce), so
this is a modified-client concern only — but it is a free-note primitive, and
the ordinary note path costs a Note item.

### Fix

A second, independent flag whose transition point is the note **send**, not the
announce. It does not interact with `GiftAcknowledged` and must not be merged
into it — the two answer different questions ("was this presented?" vs "has its
note been sent?") and drain at different moments.

1. atlas-cashshop asset `entity.go` / `model.go` (field, getter, builder setter,
   BOTH clone paths) / `entity.go` `Transform` / `rest.go` + `rest_test.go` —
   add `GiftNoteSent bool` alongside `GiftAcknowledged`, same gorm tag shape,
   same JSON:API round-trip. Follow exactly what `5b9c5be4e` did for
   `GiftAcknowledged`; that commit is the template for every hop.
2. atlas-cashshop `administrator.go` / `processor.go` — a writer and a processor
   method setting it for one cash id.
3. `kafka/message/cashshop/kafka.go` in **both** services —
   `CommandTypeMarkGiftNoteSent = "MARK_GIFT_NOTE_SENT"` and its body
   (accountId + a single cashId). **Byte-identical JSON tags in both copies**;
   this is the same seam the reviewer had to hand-check twice already.
4. atlas-cashshop `kafka/consumer/cashshop/consumer.go` — the handler, guard
   shape identical to `handleCommandAcknowledgeGifts`.
5. atlas-channel `cashshop/producer.go` + `processor.go` — `MarkGiftNoteSent`.
6. atlas-channel `cashshop/inventory/asset/{model.go,builder.go,rest.go}` —
   mirror the attribute into the read model.
7. atlas-channel `socket/handler/note_gift_forward.go` — after the existing
   ownership + `GiftFrom == ToName` gate, reject (log + return, announce
   nothing) when the asset already has `GiftNoteSent`. On success, fire
   `MarkGiftNoteSent` for that cash id. Keep the existing gate intact: this is
   an ADDITIONAL condition, and the handler must still NOT consult
   `GiftAcknowledged`.

### Known limitation to document in code, not to fix

The command is asynchronous, so two acknowledgement packets racing inside the
Kafka round-trip can still both pass the gate. This narrows the window from
unbounded to a single race; closing it fully would need a synchronous write on
the packet path, which no other cash-shop flow in this service does. State this
in a comment on the gate rather than inventing a new synchronous seam.

### Also in this unit — finding 1, a test that verifies nothing

`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go:100-113`
— `TestFindGiftAsset_GiftFromMismatch` never calls `handleNoteGiftForward` with
a mismatched `ToName` and never asserts that no saga is created; it passes with
the mismatch gate deleted. Rewrite it so it fails if the gate is removed. Add
the equivalent negative test for the new `GiftNoteSent` gate.

---

## Not yet answered

- Whether a gifted item that is *moved into the character's inventory* before
  the cash shop is re-entered still needs the flag. It does not re-appear today
  (it leaves the locker), so this is not load-bearing — but the flag should be
  carried on any locker→locker copy path so a rebate/return cannot resurrect
  the modal. Not verified in this triage.
- `giftIndex` (`OperationSend.GiftIndex()`) stays unused. It is a client-side
  enumeration counter over a list the server just sent; nothing needs it, and
  trusting it would be worse than trusting `GiftSN`. Documented, not fixed.
- Whether other tenants' templates (v48, v12) reach this path at all — they
  have no LOAD_GIFT_SUCCESS key, so they never announce and never drain. No
  behavior change for them; not live-tested.

## Resolution

- `5ebad82cc` — Defect G. Gift-forward branch on the NOTE_ACTION SEND arm:
  bypasses the Note-item gate and the receiver-online check, validates
  ownership + `GiftFrom == ToName` instead, creates a single-step
  `create_note` saga.
- `5b9c5be4e` — Defect H. `GiftAcknowledged` threaded through both services;
  `buildGiftListEntries` skips acknowledged assets; `ACKNOWLEDGE_GIFTS` fires
  after the LOAD_GIFT_SUCCESS announce, inside the `loadGiftDoneConfigured`
  gate.
- `a52e54f29` — formatting only. The first `--quick` gate failed the lint &
  format guard on one gofumpt key-alignment finding in
  `note_gift_forward_test.go`; this fixed it. Note that the Defect H
  implementer had reported a clean tree-wide `tools/lint.sh` shortly before
  that failure — the two claims are inconsistent, and the guard was right.
- `009bdf195` — Defect I. Second flag `GiftNoteSent`, its
  `MARK_GIFT_NOTE_SENT` command pair, and the additive one-shot gate; plus the
  rewrite of the hollow `TestFindGiftAsset_GiftFromMismatch`.

### Verdicts

- Review `review-bug-gift-ack-and-redisplay.md` (G + H) — **APPROVED_WITH_FINDINGS**,
  0 blocking, 2 non-blocking. Finding 1 (a test that passed with its gate
  deleted) and finding 2 (replayed-packet duplicate notes) were both folded
  into Defect I rather than accepted.
- Review `review-bug-gift-note-oneshot.md` (I) — **APPROVED**, 0 blocking,
  0 non-blocking. The reviewer confirmed finding 1's fix is load-bearing by
  mutation: it neutered each gate in turn and observed the corresponding test
  go RED, then restored the tree.
- Gate: `tools/verify.sh --base dd7e0bbb4` (**flagless** — docker bake and
  `-race` included) exited **0**. 34 changed paths, 2 changed Go modules, lint
  & format guard 0 issues on both.

### Still open

- **Live re-test: NOT performed.** The unit tests pin the saga shape, the
  gates, and the announce/drain ordering; none of them prove the v83 client's
  gifter actually receives the memo, or that the modal stays gone on
  re-entry. Until a tester gifts an item, acknowledges it, sees the note
  arrive in the gifter's memo list, and re-enters the cash shop with the gift
  still in the locker, these three defects are fixed-in-principle only.
- The AutoMigrate default for `GiftAcknowledged`/`GiftNoteSent` on an existing
  populated table was reviewed by convention, not observed against a live
  Postgres DB. Both reviews recorded it as not-evaluable.
