# Does GMS v83 tolerate zero `CashId`/`Expiration` in the buy-done `CashInventoryItem`?

IDA: `gms_v83` (`MapleStory_dump.exe.i64`, session `41f13e0d`). All addresses below are
from this binary unless noted. v95 was not reached (see Limitations).

## 1. Handlers located

- `CCashShop::OnCashItemNameChangeResBuyDone` — `0x47bccb`
  (`?OnCashItemNameChangeResBuyDone@CCashShop@@IAEXAAVCInPacket@@@Z`)
- `CCashShop::OnCashItemResTransferWorldDone` — `0x47bfa2`
  (`?OnCashItemResTransferWorldDone@CCashShop@@IAEXAAVCInPacket@@@Z`)

Both are dispatched from `CCashShop::OnCashItemResult` (`0x47915f`, mode-prefix switch;
mode byte not individually re-verified here since the two target handlers were already
located by name).

Both handlers are structurally identical (only the string-table ID for the notice
differs, and both are the same size, `0xd0`):

```c
void __thiscall CCashShop::OnCashItemNameChangeResBuyDone(_DWORD *this, CInPacket *a2)
{
  ...
  LODWORD(v12) = 55;                                    /*0x47bcdb*/
  v3 = sub_47CD85(this + 290, -1);                       /*0x47bce5*/
  CInPacket::DecodeBuffer(a2, v3, v12);                  /*0x47bcee*/
  CWnd::InvalidateRect(this + 617, 0);                   /*0x47bcfe*/
  CCSWnd_Locker::SetSelectedNo(this + 617, 0);            /*0x47bd06*/
  if ( this[310] ) { ... }  // append to a pending-queue array, no field access
  else { ... CUtilDlg::Notice(...) "purchase successful" ... }
  if ( dword_BEC360 ) (*(*dword_BEC360 + 52))(dword_BEC360, 2);
}
```

`OnCashItemResTransferWorldDone` is byte-for-byte the same shape at `0x47bfa2`,
using `dword_BEC364` in place of `dword_BEC360`.

## 2. What `sub_47CD85` + `DecodeBuffer(..., 55)` actually do

`sub_47CD85` (`0x47cd85`) grows the `CCashShop` locker array at `this+290`
by one 55-byte slot and returns a pointer to the new slot (append, since the
handler passes index `-1`):

```c
int __thiscall sub_47CD85(const void **this, int a2)
{
  if ( *this ) v3 = *(*this - 1); else v3 = 0;   // current element count
  if ( a2 == -1 ) a2 = v3;                        // -1 => append at end
  sub_47E803(this, v3 + 1, 2, &v6);               // grow array to v3+1 elements
  LODWORD(v5) = 55 * v3 - 55 * a2;
  memcpy_0(*this + 55*a2 + 55, *this + 55*a2, v5); // shift tail (no-op on append)
  return *this + 55 * a2;                          // -> pointer to new (uninitialized) slot
}
```

`CInPacket::DecodeBuffer(a2, v3, 55)` then does a **raw 55-byte memcpy** from the
packet buffer straight into that slot. There is no per-field parsing, branching, or
validation on any individual field — `CashId` and `Expiration` are decoded exactly
as whatever bytes the server sent, with no special-casing for zero.

55 bytes matches `CashInventoryItem.EncodeBytes` in
`libs/atlas-packet/cash/clientbound/shop_inventory.go:16-25` exactly:
`Int64(8) + Int(4) + Int(4) + Int(4) + Int(4) + Int16(2) + PaddedString(13) + Int64(8) + Int(4) + Int(4) = 55`.
This confirms the struct layout and offsets used below:
`CashId` @ offset 0 (8 bytes), `AccountId` @ 8, `CharacterId` @ 12, `TemplateId` @ 16,
`CommodityId` @ 20, `Quantity` @ 24, `GiftFrom` @ 26 (13 bytes), `Expiration` @ 39 (8 bytes).

Neither buy-done handler itself reads `CashId` or `Expiration` back out of the new
slot — they only call `InvalidateRect` (repaint) and `SetSelectedNo(0)`. No crash,
no assert, no date formatting happens **in these two functions**.

## 3. Where `CashId` gets read back out — and why zero is not merely cosmetic

The locker array these handlers append to (`this+290` on `CCashShop`) is the same
array read by `CCSWnd_Locker::OnMouseButton` (`0x4b053b`) when the player
double/right-clicks a locker slot to move the item into inventory (WM message
`515`/`WM_RBUTTONDOWN`-class per the `a2==515` arm):

```c
LockerIndex = CCSWnd_Locker::GetLockerIndex(this - 1, a4, a5);
...
v14 = 55 * v12;
v15 = *(55*v12 + v13 + 16);                 // offset 16 = TemplateId
if ( v15 == &loc_4FAE70 ) { ... gachapon path ... }
else {
  a2a = v15 / 1000000;
  EmptySlotPosition = CharacterData::FindEmptySlotPosition(v28, v15 / 1000000);
  if ( EmptySlotPosition )
    CCashShop::OnMoveCashItemLtoS(this[357], *(*v11+v14), *(*v11+v14+4), a2a, EmptySlotPosition);
```

`*(*v11+v14)` and `*(*v11+v14+4)` are offsets 0 and 4 of the 55-byte slot — i.e. the
low and high DWORDs of the slot's `CashId` (int64). These are passed straight into
`CCashShop::OnMoveCashItemLtoS(this, arg0, a3, a4, a2)` (`0x472632`) as `arg0`/`a3`.

Inside `OnMoveCashItemLtoS`, the client **re-looks-up the locker slot by matching
CashId**, not by index:

```c
for ( i = 0; ; i += 55 ) {
  v12 = *(this + 290);
  if ( !v12 || v10 >= *(v12 - 4) ) break;
  v13 = (*(this + 290) + i);
  if ( *v13 == arg0 && v13[1] == a3 ) break;   // match by CashId (low, high)
  ++v10;
}
...
v15 = v14 + 55 * v10;
CWvsContext::GetCommodityBySN(v31, &v29, *(v15 + 20));   // offset 20 = CommodityId, confirms layout
...
COutPacket::COutPacket(v23, 0xE5);
COutPacket::Encode1(v23, 0xDu);
LODWORD(v22) = 8;
COutPacket::EncodeBuffer(v23, &arg0, v22);   // raw 8 bytes = arg0/a3 = CashId, re-encoded verbatim
COutPacket::Encode1(v23, a4);
COutPacket::Encode2(v23, a2);
CClientSocket::SendPacket(TSingleton<CClientSocket>::ms_pInstance, v23);
```

So the round trip is: server sends `CashId` in the buy-done body → client stores it
verbatim in the locker array → on "move to inventory," the client re-derives the
target slot **by scanning the locker array for the first entry whose stored CashId
equals the CashId it just read off that same click** → then re-encodes that same
`CashId` into outbound opcode `0xE5` (move-locker-item-to-storage) sent to the server.

**Consequence of always encoding `CashId = 0`:** the identity carried through this
loop is not unique. If the locker array ever contains **more than one** entry with
`CashId == 0` (e.g. the player buys a name-change license and later a world-transfer
license, or any other cash purchase, before the shop's cash-inventory list is next
fully refreshed from the server), then `OnMoveCashItemLtoS`'s `*v13 == arg0 &&
v13[1] == a3` scan matches the **first** zero-`CashId` slot in the array — not
necessarily the slot the player actually clicked. The outbound `0xE5` packet would
then carry `CashId = 0` referring, from the server's point of view, to an
ambiguous/wrong record. This is traced end-to-end through decompiled code, not
inferred.

Separately — even with only one zero-`CashId` item ever present — the client sends
`CashId = 0` to the server in this outbound packet. Whatever atlas-side handler
consumes cash-inventory-item move requests by `CashId` (out of scope for this IDA
pass; not traced here) would need to tolerate/special-case `CashId == 0`, since it
is not the item's real persisted identifier.

`Expiration` (offset 39) was not observed being read anywhere in the traced paths
(`OnCashItemNameChangeResBuyDone`, `OnCashItemResTransferWorldDone`,
`CCSWnd_Locker::OnMouseButton`, `CCSWnd_Locker::OnButtonClicked`,
`CCashShop::OnMoveCashItemLtoS`). No date-formatting or FILETIME conversion call was
found reachable from this code path within the scope of this pass. `CCSWnd_Locker`
has no `OnDraw`/paint override of its own in the function list (`OnCreate`,
`SetSelectedNo`, `OnButtonClicked`, `OnMouseButton`, `GetLockerIndex`,
constructor only) — rendering of locker slot tooltips/labels, if it reads
`Expiration`, happens through a path this pass did not locate (see Limitations).

`CCSWnd_Locker::OnButtonClicked` (button id `1000`, likely "rebate"/return item)
indexes the locker array **by index** (`this[357]`, a slot index, not `CashId`), so
that path does not depend on `CashId` being non-zero.

## 4. What was traced vs. inferred

Traced end-to-end (decompiled, addresses quoted above):
- Both buy-done handlers append a raw 55-byte decoded struct to the `CCashShop`
  locker array; no field-level access to `CashId`/`Expiration` in the handlers
  themselves.
- `CCSWnd_Locker::OnMouseButton` reads the clicked slot's `CashId` and passes it to
  `CCashShop::OnMoveCashItemLtoS`.
- `OnMoveCashItemLtoS` re-resolves the slot index by scanning the locker array for a
  `CashId` match, then re-encodes that `CashId` verbatim into outbound opcode `0xE5`.

Inferred, not directly observed:
- That the atlas-side consumer of opcode `0xE5` (locker→storage move) does or does
  not already special-case `CashId == 0`. Out of scope for this IDA pass (Go source,
  not decompiled here).
- Whether a genuine multi-zero-`CashId` collision is reachable in practice (requires
  two cash items both encoded with `CashId = 0` to be present in the client-side
  locker array simultaneously — plausible any time a player makes two cash
  purchases, of any type that similarly zero-fills, before the shop UI reloads the
  full cash inventory list from the server).

Not established (UNVERIFIED):
- Whether `Expiration == 0` is read/rendered anywhere in the cash-shop locker UI.
  No reachable formatter was found in this pass, but `CCSWnd_Locker`'s draw/tooltip
  behavior was not fully traced (no `OnDraw` override was found in the function
  list under that class name; it may be inherited/composed through a base class or
  child control not enumerated by the name-based queries used here).

## 5. Limitations

- v95 (`79906a1e`) was not cross-checked — the v83 finding above (`CashId` reused as
  a lookup key for the locker→storage move, then re-encoded to the server) was
  judged sufficiently decisive on its own; a v95 divergence, if any, was not ruled
  out.
- The exact reachable formatter (if any) for `Expiration` as a displayed
  date/tooltip was not located; this remains UNVERIFIED rather than confirmed safe.
- The atlas-side (Go) consumer of the `0xE5` "move cash item to storage" request was
  not inspected as part of this IDA-only pass.

## Verdict

**UNSAFE for `CashId`.** The client does not merely store `CashId` cosmetically —
it re-derives which locker slot to act on by matching the stored `CashId`, and it
re-transmits that same `CashId` to the server as the identifier of the item to move
from the cash locker to inventory storage. Zero-filling `CashId` on every buy-done
response makes this identifier non-unique the moment a second cash item is also
present with `CashId == 0`, and in all cases sends a `CashId` to the server that is
not the item's real persisted identifier. `CashId` should be set to the item's real,
server-assigned cash-inventory-item id (whatever atlas persists for that row) — not
left as 0 — in both `CashShopNameChangeBuyDoneBody`'s and
`CashShopTransferWorldDoneBody`'s embedded `CashInventoryItem`.

**UNVERIFIED for `Expiration`.** No code path reading or formatting the zero
`Expiration` was located in this pass, so no corruption was observed, but the
locker-slot drawing/tooltip code that might consume it was not fully traced. Treat
as unresolved rather than confirmed safe.
