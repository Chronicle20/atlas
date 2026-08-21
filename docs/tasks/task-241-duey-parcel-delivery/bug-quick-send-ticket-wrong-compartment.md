# Bug — quick DUEY_ACTION SEND always rejected: ticket looked up in the wrong compartment

Task: task-241-duey-parcel-delivery
Environment: `atlas-pr-1434` (namespace `atlas-pr-1434`), tenant
`a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS 83.1, world 0, channel 0.
Reported: 2026-08-21, live testing on the PR environment.

## Reproduced

Yes — reproduced from pod logs plus the live inventory state, three times in a row
(21:16:23, 21:17:14, 21:18:57 UTC).

1. Character 1 ("Atlas") holds the Quick Delivery Ticket: asset id 27, template
   `5330000`, quantity 10, slot 1, awarded 21:15:40 via `@award me item 5330000 10`.
   `GET atlas-inventory /api/characters/1/inventory` places that asset in compartment
   `87655eb2-c20c-48c0-a021-9968f2022fec`, whose `type` is **5 (CASH)**. The
   character's ETC compartment is `9b3d4086-5e83-41ce-9130-a823b7bc80e0` and does not
   contain it.
2. Open Duey (NPC 9010009), fill in the send tab, check quick delivery, send.
3. atlas-channel logs, verbatim:

   ```
   [DueyActionHandle] read [mode [2]]
   Character [1] attempted a quick DUEY_ACTION SEND without holding a Quick Delivery Ticket [5330000].
   ```

   The channel announces `PARCEL[INCORRECT_REQUEST]` (mode 11 on gms_v83) and starts
   no saga.

## Observed

Every quick send is rejected with SP_3903 "You have made an incorrect request",
regardless of the player actually holding the ticket. No parcel is created, no meso
is charged, no ticket is consumed (inventory still reads quantity 10 after three
attempts — the server never destroyed one; any decrement the player sees in the
client is client-side only and returns on relog).

## Expected

A quick send by a character holding item 5330000 passes the ticket gate (FR-9),
builds the `parcel_send` saga with the `consume_quick_delivery_ticket`
`DestroyAsset` step, and the parcel is created without the 5,000 meso surcharge
(FR-3).

## Root cause

`services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send.go:64-71`
— the `hasTicket` dependency queries the **ETC** compartment:

```go
hasTicket: func(characterId uint32) (bool, error) {
        cp, err := compartment.NewProcessor(l, ctx).GetByType(characterId, inventory.TypeValueETC)
        ...
        _, found := cp.FindFirstByItemId(item.QuickDeliveryTicketId)
```

Item `5330000` is classification 533, a **cash** item — it lives in inventory type 5
(`inventory.TypeValueCash`), which is exactly where the live data puts it, and is why
`GetCashSlotItemType` routes it through the CASH_ITEM_USE path at all
(`character_cash_item_use_duey.go`). `FindFirstByItemId` on the ETC compartment can
therefore never find it, so `hasTicket` returns `false` for every character and the
quick arm is unreachable in production.

The unit tests did not catch this because `duey_action_send_test.go:155` stubs
`hasTicket` directly (`dueySendFixture.hasTicket`); the compartment type in the
production wiring is never asserted by any test.

The `consume_quick_delivery_ticket` step is NOT affected: `handleDestroyAsset`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:1197`)
calls `RequestDestroyItem` by `templateId`, with no inventory type, so it resolves the
right compartment on its own. Only the pre-flight gate is wrong.

## Fix

- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send.go`
  — `hasTicket` must look in the compartment the ticket actually lives in. Derive it
  rather than hard-coding a second magic constant: `inventory.TypeFromItemId(...)`
  exists at `libs/atlas-constants/inventory/constants.go:40` and returns
  `TypeValueCash` for `5330000`. Keep the existing error and not-found behavior
  (`PARCEL[INCORRECT_REQUEST]`, no saga) unchanged.
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send_test.go`
  — add a regression test that pins the **compartment type actually requested** by
  the production wiring, not the stubbed bool. A stubbed `hasTicket` cannot fail this
  bug; the assertion has to reach the `GetByType` argument (extract the lookup into a
  small named helper with a seam for the compartment fetch, in the style of
  `dueyCouponSagaCreateFunc`, and assert it asks for type 5).

No other service changes. Do not touch the 24-hour `ReceivableAt` behavior — that is
FR-12 and is working as specified.

## Not yet answered

- **The client lockup after the rejection.** The reporter says the error "locks up my
  client". The v83 client's `CParcelDlg::OnPacket` default arm
  (@0x6F56EA, GMS v83 `MapleStory_dump.exe`) runs `NoticeResult(mode)` and then
  `CParcelDlg::SetCtrlEnabled(v3, 1)`, i.e. it re-enables the dialog controls — so on
  the packet level mode 11 should not wedge the dialog, *provided* a `CParcelDlg` is
  open (if none is open, that same arm throws `CDisconnectException`). Whether the
  remaining lock is the player-action lock taken when the cash item was double-clicked
  (the handler at `character_cash_item_use_duey.go` deliberately sends no
  EnableActions on its success path) is UNCONFIRMED. Not diagnosed here, not in scope
  of this fix.
- Whether the ticket count the reporter saw decrease is purely a client-side
  optimistic decrement. The server-side quantity is unchanged at 10, verified after
  three attempts; no destroy was ever requested.

## Resolution

Fixed in `7c09672dd` (atlas-channel). Module `go build ./... && go test ./...` clean;
repo gate `tools/verify.sh --quick --base d6dd596a0` covers this commit. Live re-test on
`atlas-pr-1434` still PENDING — reopen if the symptom survives the next deploy.
