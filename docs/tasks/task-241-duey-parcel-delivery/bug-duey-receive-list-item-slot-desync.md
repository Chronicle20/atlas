# bug: Duey receive list renders garbage from the 2nd parcel onward

## Reproduced

Ephemeral env `atlas-pr-1434`, tenant `237a...`, GMS v83 client, character
`Chronicle` (characterId 2). Three pending parcels, Duey NPC 9010009 → Receive
tab.

Server-side data is correct. `atlas-channel` pod
`atlas-channel-85cf7f74d7-bxkq2`, `GET /api/parcels?filter[recipientId]=2&filter[worldId]=0&filter[status]=pending`
response verbatim:

| # | senderName | mesoAmount | message | itemId | itemType | quantity | expiresAt |
|---|---|---|---|---|---|---|---|
| 1 | Atlas | 1000 | "" | 2000004 | 0 | 5 | 2026-09-20T21:09:23Z |
| 2 | Atlas | 5000 | "lulnub" | 2000004 | 0 | 5 | 2026-09-21T00:27:38Z |
| 3 | Atlas | 1234 | "lulul" | 2000005 | 0 | 10 | 2026-09-21T00:28:02Z |

## Observed (client)

| # | Name | Valid Until | Sender | Mesos | Item | Message |
|---|---|---|---|---|---|---|
| 1 | Atlas | 29 day(s) | Atlas | 1000 | (empty) | (empty) |
| 2 | (blank) | Delivering | (blank) | 2097152000 | (empty) | (empty) |
| 3 | (random ASCII) | Delivering | (blank) | 65536 | (empty) | (two random ASCII chars) |

Row 1's scalar fields are right; its **Item is missing** even though the parcel
carries 5× 2000004. Rows 2 and 3 are shifted garbage.

## Expected

All three rows render sender/mesos/expiry/message correctly and show their
attached item name.

## Root cause — established, IDA-confirmed

The attached item is encoded with a **leading inventory-slot byte that the
client never reads**, so the item block desyncs the whole list.

GMS v83 IDB (session `754107bf`, `MapleStory_dump.exe.i64`):

- `PARCEL::Decode` @0x4E4345 — `CInPacket::DecodeBuffer(a2, this, 234)`, then
  `if (CInPacket::Decode1(a2)) { GW_ItemSlotBase::Decode(v5, a2); ... }`.
  The 234-byte fixed block and the `hasItem` bool match our encoder exactly.
- `GW_ItemSlotBase::Decode` @0x4E33F9 — verbatim first two statements:
  `v2 = CInPacket::Decode1(a2); GW_ItemSlotBase::CreateItem(&v6, v2);` then
  `(*(*v7 + 112))(v7, a2)` (the type's `RawDecode`). **The very first byte it
  reads is the item TYPE.** There is no slot prefix in this call site — unlike
  the inventory/storage call sites, where the caller reads the slot itself.
  When `CreateItem` returns null (`if (v7)` false) the function stores
  `*(a1+4) = 0` and **consumes no further bytes**.

Our encoder: `libs/atlas-packet/model/asset.go` `encodeStackableInfo` opens with

```go
if !m.zeroPosition {
    w.WriteInt8(int8(m.slot))
}
w.WriteByte(2)   // <- the type byte the client actually expects first
```

and `services/atlas-channel/.../parcel/model.go:91` builds the asset as
`packetmodel.NewAsset(false, 0, *m.ItemId(), time.Time{})` — `zeroPosition =
false`, so a `0x00` slot byte is emitted ahead of the type byte.

The client therefore reads item type `0` → `CreateItem(0)` yields null → row 1
shows no item and zero item bytes are consumed → the remaining 20 bytes of
parcel 1's item block are re-interpreted as the head of parcel 2's 234-byte
block, and the desync compounds through parcel 3. That is exactly the observed
"row 1 fine except item, rows 2–3 garbage" signature.

The same slot-prefix bug applies to every branch of `Asset.Encode` reached from
a parcel (`encodeSlot` in the equip paths, the inline `WriteInt8` in
`encodeStackableInfo`), so the fix must be at the parcel codec, not only in the
stackable branch.

The committed byte fixtures encode the bug: every
`libs/atlas-packet/parcel/clientbound/v*_test.go` builds its item as
`model.NewAsset(false, 1, 1302000, time.Time{})` and asserts a slot byte on the
wire. They must be corrected along with the codec, or they will pin the wrong
layout.

## Fix

- `libs/atlas-packet/model/asset.go` — add an immutable setter that clears the
  slot prefix (e.g. `func (m Asset) SetZeroPosition(v bool) Asset`), following
  the existing `SetX` setter style in this file. Do not change any existing
  encode path's behaviour.
- `libs/atlas-packet/parcel/parcel.go` — `Parcel.SetItem` must normalize the
  asset to zero-position so the PARCEL wire form can never carry a slot,
  regardless of caller. Update the `+235.. GW_ItemSlotBase item` field comment
  to record the "no slot prefix — `GW_ItemSlotBase::Decode` @0x4E33F9 reads the
  type byte first" fact and cite the IDA address.
- `services/atlas-channel/atlas.com/channel/parcel/model.go` (~line 91) — build
  the asset with `zeroPosition = true` so the intent is explicit at the call
  site as well as enforced by the codec.
- `libs/atlas-packet/parcel/clientbound/v72_test.go`, `v79_test.go`,
  `v83_test.go`, `v84_test.go`, `v87_test.go`, `v92_test.go`, `v95_test.go`,
  `v185_test.go`, and `parcel_test.go` — drop the leading slot byte from every
  expected item block, and update the header comments that name
  `model.NewAsset(false, 1, ...)`.
- `docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json` —
  regenerate with `go run ./tools/packet-audit matrix` and commit. The PR's
  `check` job (packet-matrix) is currently failing for exactly this: "STATUS.md
  is stale — regenerate and commit" / "status.json is stale". Regenerate AFTER
  the fixture edits so one regeneration covers both.

Verification: module-local `go build ./... && go test ./...` in
`libs/atlas-packet` and `services/atlas-channel`, plus
`go run ./tools/packet-audit matrix --check` exiting 0.

## Not yet answered

- **Message rendering.** Row 1's message was legitimately empty, and rows 2–3
  were desynced, so this bug gives no evidence about whether the message at
  block offset +29 renders. `libs/atlas-packet/parcel/parcel.go` already flags
  the +29..233 message/padding boundary as a design-level inference, not
  IDA-confirmed (no v83 consumer dereferences it). If, after this fix, rows 2
  and 3 still show empty Message, that is a SEPARATE bug — the message field
  offset/encoding — and needs its own investigation. Do not pre-emptively
  change the message encoding as part of this fix.
- **`itemType` is always 0.** `atlas-parcel` never calls
  `Builder.SetItemType` on the parcel-create path, so every row persists
  `itemType = 0`, which is not a valid `inventory.Type` (Equip = 1, Use = 2).
  It is inert today: `atlas-channel`'s `ToPacket` only tests
  `!= TypeValueEquip`, and `Asset.Encode` derives the real type from the
  template id. Recorded as a data-quality defect, explicitly OUT OF SCOPE for
  this fix.

## Outcome

Fixed. `Asset.SetZeroPosition` (`libs/atlas-packet/model/asset.go`) added as
an immutable setter; `Parcel.SetItem` (`libs/atlas-packet/parcel/parcel.go`)
now normalizes every attached asset to zero-position, so the PARCEL wire
form can never carry a slot prefix regardless of caller. The
`atlas-channel` call site (`services/atlas-channel/atlas.com/channel/parcel/model.go`)
also builds the asset with `zeroPosition = true` explicitly. The nine
`libs/atlas-packet/parcel/clientbound/*_test.go` fixtures were corrected to
drop the leading slot byte from their expected item blocks, and
`docs/packets/audits/STATUS.md`/`status.json` were regenerated.
See `bug-duey-receive-list-item-slot-desync-fix-report.md` for full detail.
