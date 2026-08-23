# Bug — DUEY_ACTION RECEIVE rejects every item parcel: `itemType` is never persisted

- **Task**: task-241-duey-parcel-delivery
- **Branch**: task-241-duey-parcel-delivery (`76a8961ef` at diagnosis)
- **Environment**: `atlas-pr-1434`, tenant `a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS 83.1
- **Reported**: character `Chronicle` (id 2) cannot retrieve items from Duey; client shows
  "You have made an incorrect request." Deleting (DISCARD) a package works.

## Reproduced

Yes — from the live PR environment, not re-run by hand.

`atlas-channel-744f54ddc5-rj72f`, four RECEIVE attempts (21:46:12, 21:46:15, 21:46:16,
21:46:44):

```
[DueyActionHandle] read [mode [4]]
Issuing [GET] request to [.../api/parcels?filter[recipientId]=2&filter[worldId]=0&filter[status]=pending]
Character [2] DUEY_ACTION RECEIVE: unable to load inventory type [0].
```

The subsequent DISCARD (21:47:02, `mode [5]`) succeeds — it PATCHes atlas-parcel and never
touches a compartment, which is exactly why deleting works and receiving does not.

Live parcel state for that mailbox (`GET /api/parcels?filter[recipientId]=2&...`, run from
inside the namespace):

```json
{"itemId":2000004,"itemType":0,"quantity":5, ...}
{"itemId":2000005,"itemType":0,"quantity":10, ...}
```

## Observed

Every parcel that carries an item is stored with `itemType == 0`. `atlas-channel`'s RECEIVE
pre-flight (`duey_action_receive.go:155-161`) converts that to `inventory.Type(0)`, which is
not a valid compartment type, so `compartment.GetByType` errors and the handler answers with
`ParcelIncorrectRequestBody()` — the client's "You have made an incorrect request."

## Expected

`itemType` holds the source inventory type of the deposited item (2 / `USE` for templates
2000004 and 2000005), the compartment lookup succeeds, and the receive saga runs.

## Root cause

The source inventory type is known at send time and dropped one hop later. It is present on
`TransferToParcelPayload.SourceInventoryType` (`libs/atlas-saga/payloads.go:992`, set by
`duey_action_send.go:277`) but **`AcceptToParcelPayload` has no inventory-type field at all**
(`libs/atlas-saga/payloads.go:1017-1063`). Consequently:

- `expandTransferToParcel` (`saga/processor.go:2162`) copies the snapshot into the
  `accept_to_parcel` step without the type;
- `AcceptToParcelParams` (`saga-orchestrator/parcel/processor.go:26`), the Kafka body
  `AcceptToParcelCommandBody` (`atlas-parcel/.../kafka/message/custody/kafka.go:52`) and
  `parcel.AcceptParams` (`atlas-parcel/.../parcel/processor_custody.go:26`) all lack it;
- `AcceptCustody` (`processor_custody.go:100-133`) therefore never calls
  `SetItemType`, and the row's `item_type` column keeps its zero value.

The column, the model accessor, the REST field and the channel-side mapping all already
exist and are correct — only the write path is missing. Nothing in the repo calls
`parcel.Builder.SetItemType` on the deposit path (`grep -rn SetItemType` shows only
`entity.go:110` rehydration and `task.go:190` return-leg copy, both of which just propagate
the stored zero).

The same missing value also degrades the receive saga itself: the channel builds
`WithdrawFromParcelPayload.InventoryType` from `p.ItemType()`
(`duey_action_receive.go:202`), which `expandWithdrawFromParcel` passes to
`accept_to_character` (`saga/processor.go:2344`). So even past the pre-flight, the grant
would target inventory type 0.

## Fix

Thread the source inventory type from the send payload through to the stored row. Field
naming: `ItemType byte` on the wire/param structs (matching the persisted column and the
existing `AcceptParams` snapshot-field style); no new inventory-type derivation from the
template id anywhere — the authoritative value is what the sender's compartment was.

| File | Change |
|---|---|
| `libs/atlas-saga/payloads.go` | Add `ItemType byte \`json:"itemType"\`` to `AcceptToParcelPayload` (item-snapshot block, near `TemplateId`), documented as the source inventory type; zero when `HasItem` is false. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` | `expandTransferToParcel` (~line 2162): set `ItemType: byte(payload.SourceInventoryType)` in the `HasItem: true` accept step. Leave the meso-only branch at zero. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` | `AcceptToParcel` handler (~line 2547): map `ItemType: payload.ItemType` into `parcel.AcceptToParcelParams`. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor.go` | Add `ItemType byte` to `AcceptToParcelParams` (line 26 block). |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/producer.go` | `AcceptToParcelProvider`: copy `ItemType` into the command body. |
| `services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go` | Add `ItemType byte \`json:"itemType"\`` to `AcceptToParcelCommandBody` (line 52 block). |
| `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go` | `handleAcceptToParcel` (~line 88): map `ItemType: b.ItemType` into `parcel.AcceptParams`. |
| `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go` | Add `ItemType byte` to `AcceptParams`; in `AcceptCustody`'s `params.HasItem` branch call `b.SetItemType(params.ItemType)`. |

Tests to add/extend (Builder pattern, no `*_testhelpers.go`):

- `services/atlas-saga-orchestrator/.../saga/parcel_expansion_test.go` — assert the expanded
  `accept_to_parcel` step carries `ItemType` equal to the transfer payload's
  `SourceInventoryType` (the existing cases already use `SourceInventoryType: 1`).
- `services/atlas-parcel/.../parcel/processor_test.go` (or the custody test alongside it) —
  assert `AcceptCustody` with `HasItem: true, ItemType: 2` persists `ItemType() == 2`, and
  that a meso-only accept leaves it zero.
- A saga-orchestrator handler-level assertion that `AcceptToParcelParams.ItemType` reaches
  the producer body, if an existing test covers that mapping.

No DB migration is needed: `parcels.item_type` already exists (`atlas-parcel/.../parcel/entity.go:68`)
and is auto-migrated.

## Not yet answered

- **Existing rows are not repaired by this fix.** The two pending parcels in
  `atlas-pr-1434` (`dce1997d-…`, `3989e2e4-…`) were created before it and still hold
  `itemType 0`; they will keep failing. Re-test with a freshly sent parcel. Whether a
  backfill (derive from `itemId` via `inventory.TypeFromItemId`) is worth writing for other
  environments is a product call, deliberately **not** taken here — do not add a
  derive-from-template fallback in the read path as part of this fix.
- Not investigated: whether any other parcel field is silently zero for the same
  drop-one-hop reason (`ItemLevel`, `RingId`, `ViciousCount` are on the payload but the
  expansion does not set `ItemLevel`/`RingId`/`ViciousCount` either — worth a look, but it
  is not what breaks receive).

## Resolution

Fixed by **`d41d70d39`** — "fix(parcel): thread source inventory type through to the parcel
row". The implementer added `ItemType` to one file beyond the table above:
`services/atlas-saga-orchestrator/.../kafka/message/parcel/custody/kafka.go`, the
orchestrator's own byte-for-byte mirror of `AcceptToParcelCommandBody` and what `producer.go`
actually serializes — without it the field would have compiled but never reached the wire.

**`c3af819b2`** — "test(duey): cover ItemType mapping through parcel accept path" — closes the
review's non-blocking findings by asserting `ItemType` at the producer and consumer mapping
hops, which previously had no coverage.

- **Gate**: `tools/verify.sh --quick --base 76a8961ef` → PASS (exit 0; 91 modules, shared-lib
  fan-out).
- **Review**: `atlas-reviewer`, verdict `APPROVED_WITH_FINDINGS`, 0 blocking — see
  `reviews/review-bug-duey-receive-item-type.md`. It traced `ItemType` by hand through all 8
  mapping sites, confirmed both wire mirrors agree on `json:"itemType"`, that the value is the
  sender's actual compartment type (never re-derived), and that the meso-only branch stays zero.
- **Live re-test**: NOT yet confirmed. The two pending parcels in `atlas-pr-1434`
  (`dce1997d-…`, `3989e2e4-…`) predate the fix and still hold `itemType 0` — they will keep
  failing. Re-test requires sending a NEW parcel after the PR image rolls out.
