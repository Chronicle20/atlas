# Bug — double-clicking the Quick Delivery Ticket wedges the client's exclusive-request lock

Task: task-241-duey-parcel-delivery
Environment: `atlas-pr-1434`, tenant `a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS 83.1,
world 0, channel 0. Reported 2026-08-21 during live testing.

## Reproduced

Yes — three times in the pod logs (21:15:44, 21:17:35, 21:18:33 UTC), each preceded by
a clean Duey dialog close, so no dialog was open at the time.

1. Character 1 holds item `5330000` (asset 27, qty 10, CASH compartment).
2. Double-click it in the inventory. atlas-channel logs:

   ```
   [CharacterCashItemUseHandle] read [updateTime [0], source [1], itemId [5330000]]
   Quick delivery dialog open requested.   (transaction_id …, cash_slot_type 31)
   Message received {"transactionId":"…","worldId":0,"channelId":0,"characterId":1,
                     "npcId":0,"quick":true,"type":"SHOW_PARCEL"}
   ```

3. The quick-send dialog opens, but from that point the player cannot select items in
   the inventory, cannot use skills, and cannot act — no error, no disconnect, the
   world still renders.

## Observed

After a double-click of the Quick Delivery Ticket the client is action-wedged for the
rest of the session. Nothing in the logs errors; the server considers the operation a
success.

## Expected

The quick-send dialog opens AND the client's exclusive-request lock is released, so
the player can pick the item to send out of their inventory — which is the entire
point of the dialog.

## Root cause

`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go`
— `handleDueyCouponUse` calls `enableActions()` on all three failure paths
(unsupported version, saga-create error) but **not** on the success path. Its doc
comment states the handler "consumes nothing", which is correct for FR-26, but that is
exactly what makes the omission fatal here.

`session.EnableActions`'s own contract states the rule
(`services/atlas-channel/atlas.com/channel/session/enable_actions.go:12-26`, verbatim):

> The lock it clears is `CWvsContext::m_bExclRequestSent`, which the v83 client arms
> whenever it sends an exclusive request (item use, skill use, portal entry …).
> `CWvsContext::CanSendExclRequest` refuses every subsequent request until something
> clears it, and only three things do: the leading exclRequestSent bool of
> STAT_CHANGED or INVENTORY_OPERATION, or a SET_FIELD. There is no client-side
> timeout.
>
> So an outcome that produces no inventory or stat delta and no field change MUST send
> this, or the client is wedged for the rest of the session.

The Quick Delivery Ticket double-click is an **item use**, so the client arms
`m_bExclRequestSent`. The only thing the server sends back is `PARCEL[OPEN_QUICK]`
(mode 0x1A). That is neither STAT_CHANGED, nor INVENTORY_OPERATION, nor SET_FIELD —
and the client's own OPEN_QUICK arm clears nothing either: `CParcelDlg::OnPacket`
@0x6F56EA (GMS v83 `MapleStory_dump.exe`, session 754107bf) case 26 allocates
`CParcelDlg(1)` and returns, with no unlock and no `SetCtrlEnabled` call. Deferring
the ticket consume to send time (FR-26) means no INVENTORY_OPERATION arrives either.
So nothing ever clears the lock.

This is the same defect class as task-205's escrow amendment, which
`enable_actions.go` was written to prevent.

Note the converse guard in that same doc: do NOT send EnableActions for an outcome
that warps the player. This path does not change field, so it must send it.

## Fix

- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go`
  — announce `session.EnableActions` on the success path as well, after
  `dueyCouponSagaCreateFunc` returns without error (the existing `enableActions`
  closure at the top of the handler is already the right helper). Update the handler's
  doc comment: the "consumes nothing" paragraph should state that precisely because
  the ticket is not consumed here, no INVENTORY_OPERATION follows and the
  exclusive-request lock must be released explicitly.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey_test.go`
  — add a regression test asserting the success path announces the enable-actions
  StatChanged (an empty `Update` set with `exclRequestSent = true`), alongside the
  existing saga assertion. Follow whatever announce seam the sibling tests in this
  package already use; do not introduce a new mocking style.

Scope is this handler only. Do not change the ReceivableAt/24h behavior (FR-12, works
as specified), and do not revisit the ticket-compartment gate — that was fixed in
`7c09672dd`.

## Not yet answered

- The reporter also described the client "locking up" after the
  `PARCEL[INCORRECT_REQUEST]` rejection of a quick send. Every observed instance of
  that rejection followed an already-wedged double-click in the same session, so it is
  most likely the same lock, not a second defect: `CParcelDlg::OnPacket`'s default arm
  does call `SetCtrlEnabled(v3, 1)`, which re-enables the dialog controls. Re-test
  after this fix and after `7c09672dd`; if the send-reject path still wedges, that is
  a separate bug (the DUEY_ACTION SEND rejection arms would then also need an
  EnableActions).

## Resolution

Fixed in `cd84cf08e` (atlas-channel). Module `go build ./... && go test ./...` clean;
repo gate `tools/verify.sh --quick --base d6dd596a0` covers this commit. Live re-test on
`atlas-pr-1434` still PENDING — reopen if the symptom survives the next deploy.
