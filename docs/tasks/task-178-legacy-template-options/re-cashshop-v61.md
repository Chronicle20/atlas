# task-178 — CashShop (0xC4 / 196) deep RE for GMS v61

IDB: `GMS_v61.1_U_DEVM.exe` (ida-pro session `9a1bdd7a`). Read-only.
Method: enumerate every `COutPacket::COutPacket(v, 196)` construction site, read the
`Encode1(mode)` immediately following, and match the post-mode body against the
atlas `libs/atlas-packet/cash/serverbound/*` decoders. All modes below are quoted
from the actual client send; nothing is copied from v83/v79.

CashShop opcode confirmed **0xC4 (196)** at every send site (`push 0C4h` → COutPacket ctor `??0COutPacket@@QAE@J@Z` @0x5ffc4f).

## Completeness — every 0xC4 send site

A whole-binary `push 0C4h` sweep plus a scoped `0x453000–0x464000` sweep both
resolve to the **same 21 CashShop send sites**. Every 0xC4 push outside CCashShop
is either an alloc/`memcpy` `; Size` argument (0x406afb, 0x43e77f, 0x44496a,
0x5086a2, 0x5111c3) or a clientbound reader / unrelated opcode-196 consumer
(CStage::OnSetField, sub_57962E, sub_586801, SendConsumeCashItemUseRequest, …).
The entire serverbound CashShopOperation family lives in CCashShop (0x453–0x45d).

| send site | fn | Encode1 mode | resolved key |
|---|---|---|---|
| 0x456c6d | `CCashShop::SendBuyAvatarPacket` @0x456a42 | 3 | BUY |
| 0x4581e3 | `CCashShop::OnBuy` @0x457ea4 | 3 | BUY |
| 0x458b9b | `CCashShop::OnGift` @0x4588f4 | 4 | GIFT |
| 0x45a41a | `CCashShop::OnSetWish` @0x45a345 | 5 | SET_WISHLIST |
| 0x45a53a | `CCashShop::OnRemoveWish` @0x45a4a9 | 5 | SET_WISHLIST (remove) |
| 0x457206 | `sub_456FE4` @0x456fe4 | 6 | INCREASE_INVENTORY |
| 0x457439 | `sub_457316` @0x457316 | 7 | INCREASE_STORAGE |
| 0x457606 | `CCashShop::OnBuyFriendship` @0x4574b0 | 8 | BUY_FRIENDSHIP |
| 0x459bb2 | `CCashShop::OnIncCharacterSlotCount` @0x459928 | 6 or 7 (folded) | INCREASE_CHARACTER_SLOT / ENABLE_EQUIP_SLOT |
| 0x45c1d8 | `sub_45C063` @0x45c063 | 12 (0x0C) | **MOVE_FROM_CASH_INVENTORY** |
| 0x45c2d8 | `sub_45C271` @0x45c271 | 13 (0x0D) | **MOVE_TO_CASH_INVENTORY** |
| 0x456f27 | `sub_456D87` @0x456d87 | 24 (0x18) | **REBATE_LOCKER_ITEM** |
| 0x459402 | `sub_4591C0` @0x4591c0 | 27 (0x1B) | gift-wrap buy variant (unmapped) |
| 0x458620 | `CCashShop::OnBuyCouple` @0x45832d | 28 (0x1C) | BUY_COUPLE |
| 0x4590ca | `CCashShop::OnGiftPackage` @0x458e10 | 29 (0x1D) | BUY_OTHER_PACKAGE |
| 0x458870 | `CCashShop::OnBuyPackage` @0x4586ce | 30 (0x1E) | BUY_PACKAGE |
| 0x45cb26 | `sub_45CB0E` @0x45cb0e | 31 (0x1F) | **APPLY_WISHLIST** (mode-only) |
| 0x459e9e | `CCashShop::OnBuyNormal` @0x459c93 | 33 (0x21) | BUY_NORMAL |
| 0x45cdef | `sub_45CDD9` @0x45cdd9 | 39 (0x27) | **BUY_NAME_CHANGE** |
| 0x45cf76 | `sub_45CF64` @0x45cf64 | 41 (0x29) | world-transfer commit (unmapped) |
| 0x45c979 | `sub_45C607` @0x45c607 | 4 / 27 / 29 / 33 | buy dispatcher (reuses above modes) |

## Requested resolutions

### Modes 24 (0x18) and 27 (0x1B)

**Mode 24 (0x18) = REBATE_LOCKER_ITEM.** `sub_456D87` @0x456d87, send @0x456f27:
```
COutPacket::COutPacket(v20, 196)          /*0x456f2f*/
COutPacket::Encode1(v20, 0x18u)           /*0x456f3d*/  mode 24
COutPacket::Encode4(v20, v24)             /*0x456f48*/  v24 = ask_SPW(100, v9)  -> spw/birthday int
COutPacket::EncodeBuffer(v5, 8u)          /*0x456f53*/  v5 = locker item (this[290] array) -> 8-byte serial
```
The function computes a discounted/rebate price (`price * (100 - discount%) / 100`)
over a **locker** commodity (`this[290]` = the cash-inventory array, the same array
read by MOVE_FROM). Body = `ask_SPW int(4) + 8-byte serial` matches
`ShopOperationRebateLockerItem` **exactly** (`birthday uint32` [v83 ask_SPW int] +
`unk uint64` [8-byte locker serial], `shop_operation_rebate_locker_item.go`).
This upgrades the prior doc's tentative "mode 24 ≈ REBATE 26 unconfirmed" to a
confirmed match. (Note the v61 mode number is 24, not the v83 26.)

**Mode 27 (0x1B) = gift-wrap buy variant — no distinct atlas key.** `sub_4591C0`
@0x4591c0, send @0x459402:
```
COutPacket::COutPacket(v21, 196)          /*0x45940a*/
COutPacket::Encode1(v21, 0x1Bu)           /*0x459418*/  mode 27
COutPacket::Encode4(v21, v29)             /*0x459423*/  v29 = ask_SPW  -> spw
COutPacket::Encode4(v21, v35)             /*0x45942e*/  v35 = point/currency flags bitmask
COutPacket::Encode4(v21, a3)              /*0x459439*/  commodity SN
COutPacket::EncodeStr(...)                /*0x459452*/  string 1 (recipient)
COutPacket::EncodeStr(...)                /*0x45946b*/  string 2 (message)
```
Called from `CCashShop::ProcessBuy` @0x45cb70; also emitted by the `sub_45C607`
dispatcher when `itemId/100 == 11120` (and `!= 1112000`). Body =
`spw(4) + flags(4) + sn(4) + str + str`. This is a gift/friendship-wrap purchase
(2 strings = recipient + message) with an extra currency-flags int vs plain GIFT
(mode 4). There is **no 1:1 atlas key** — atlas already binds GIFT=4,
BUY_FRIENDSHIP=8, BUY_COUPLE=28 separately; mode 27 is a distinct v61 buy path.
Reported UNCERTAIN as to exact atlas key.

### INCREASE_CHARACTER_SLOT / ENABLE_EQUIP_SLOT — folded into modes 6 / 7

`CCashShop::OnIncCharacterSlotCount` @0x459928, send @0x459bb2 — a **single**
handler emits both keys, the mode chosen by item id:
```
COutPacket::COutPacket(v22, 196)                          /*0x459bba*/
v16 = TSecType<long>::GetData(v25 + 16)                   /*0x459bc9*/  itemId
COutPacket::Encode1(v22, (v16 / 1000 == 9110) + 6)        /*0x459be5*/  -> 7 if 9110xxx else 6
COutPacket::Encode1(v22, v34 == 2)                        /*0x459bf5*/  isPoints/pointType
COutPacket::Encode4(v22, v34)                             /*0x459c00*/  currency/point-type bitmask
COutPacket::Encode1(v22, 1u)                              /*0x459c0c*/  constant flag = 1
COutPacket::Encode4(v22, a2)                              /*0x459c17*/  serialNumber
```
- **ENABLE_EQUIP_SLOT** (item `9110xxx`, `itemId/1000 == 9110`) → **mode 7**.
- **INCREASE_CHARACTER_SLOT** (other slot-coupons; item guard `itemId/10000 == 911 || 543`) → **mode 6**.

So there is no dedicated 8/9 (v83) or 9 (v79) slot mode in v61; the server
disambiguates 6/7 by body. Body `pointType(1) + currency(4) + flag=1(1) + sn(4)`
aligns with the atlas `legacyGMS` branch of `shop_operation_enable_equip_slot.go`
and `shop_operation_increase_character_slot.go`.

### GET_PURCHASE_RECORD — ABSENT

No 0xC4 send with a `Encode1(mode); Encode4(sn)`-only (single-int) body exists in
v61. There is no `RequestCashPurchaseRecord` emitter. v79 had one; v61 does not.
ABSENT (no send site under any opcode among the CCashShop send functions).

### MOVE_FROM_CASH_INVENTORY (12) vs MOVE_TO_CASH_INVENTORY (13)

**MOVE_FROM_CASH_INVENTORY = mode 12 (0x0C)** — `sub_45C063` @0x45c063, send @0x45c1d8
(caller `sub_488A7A` @0x488a7a):
```
COutPacket::Encode1(v22, 0xCu)            /*0x45c1ee*/  mode 12
COutPacket::EncodeBuffer(&Src, 8u)        /*0x45c1fc*/  8-byte serial
COutPacket::Encode1(v22, a4)              /*0x45c207*/  inventoryType
COutPacket::Encode2(v22, (u16)a5)         /*0x45c212*/  slot
```
Body `serial(8) + invType(1) + slot(2)` matches `ShopOperationMoveFromCashInventory`
(`serialNumber uint64 + inventoryType byte + slot int16`) **exactly**. HAS slot (locker→inventory).

**MOVE_TO_CASH_INVENTORY = mode 13 (0x0D)** — `sub_45C271` @0x45c271, send @0x45c2d8
(caller `sub_489C99` @0x489c99):
```
COutPacket::Encode1(v6, 0xDu)             /*0x45c2ed*/  mode 13
COutPacket::EncodeBuffer(&Src, 8u)        /*0x45c2fb*/  8-byte serial
COutPacket::Encode1(v6, a4)               /*0x45c306*/  inventoryType
```
Body `serial(8) + invType(1)`, NO slot, matches `ShopOperationMoveToCashInventory`
**exactly** (inventory→locker). The v61 mode numbers are 12/13, one below the v83 13/14.

### APPLY_WISHLIST — mode 31 (0x1F), mode-only

`sub_45CB0E` @0x45cb0e, send @0x45cb26 (caller `sub_4936BF` @0x4936bf):
```
COutPacket::COutPacket(v3, 196)           /*0x45cb2e*/
COutPacket::Encode1(v3, 0x1Fu)            /*0x45cb3c*/  mode 31
CClientSocket::SendPacket(...)            /*0x45cb4b*/  <- no body
```
Only mode-only send in the family; matches the atlas APPLY_WISHLIST "mode only,
no body" shape. The caller (`sub_4936BF`) validates a ≥10-item cart
(`this[29]+1180 … +1220`, a 10-slot commodity array) and confirms before sending —
i.e. "apply/commit the wishlist cart," which the server reads from stored state.
Resolved APPLY_WISHLIST = 31 (medium-high confidence; body is definitive, semantic
caller is the buy-all-cart flow).

### BUY_NAME_CHANGE — mode 39 (0x27)

`sub_45CDD9` @0x45cdd9, send @0x45cdef (callers `sub_6A2881`/`sub_6A2AE1`, the
name-change dialog: validates 4–12 char name, `CCurseProcess::ProcessString`,
old≠new):
```
COutPacket::Encode1(v5, 0x27u)            /*0x45ce08*/  mode 39
COutPacket::Encode4(v5, a2)               /*0x45ce13*/  serialNumber
COutPacket::EncodeStr(oldName)            /*0x45ce2d*/
COutPacket::EncodeStr(newName)            /*0x45ce47*/
```
Body `sn(4) + str + str` matches `ShopOperationBuyNameChange`
(`serialNumber uint32 + oldName + newName`) **exactly**. Resolved BUY_NAME_CHANGE = 39.

### BUY_WORLD_TRANSFER — separate opcode 0x12 (18), confirmed

`CCashShop::OnBuyTransferWorldItem` @0x459737 → `sub_45CEFD` @0x45cefd:
```
COutPacket::COutPacket(v5, 18)            /*0x45cf14*/  opcode 0x12 (NOT 0xC4)
COutPacket::Encode4(v5, a2)               /*0x45cf23*/  worldNo (g_pWvsContext+8328)
COutPacket::Encode4(v5, a3)               /*0x45cf2e*/  spw (ask_SPW result)
```
Confirmed: the world-transfer purchase is opcode **0x12**, body `worldNo(4) + spw(4)`.
NOT a 0xC4 mode. This is the initiate/check step gated by
`CCashShop::CheckTransferWorldPossible` @0x45ce91.

## Newly discovered unmapped mode

### Mode 41 (0x29) — world-transfer commit (second packet)

`sub_45CF64` @0x45cf64 (adjacent to the 0x12 sender `sub_45CEFD`), send @0x45cf76,
caller `sub_6A3FF8` (world-transfer confirmation dialog, vtable @0x8ec42c):
```
COutPacket::Encode1(v5, 0x29u)            /*0x45cf8c*/  mode 41
COutPacket::Encode4(v5, a2)               /*0x45cf97*/  a2 = *(CCashShop + 1252) = stored world-transfer SN
COutPacket::Encode4(v5, a3)               /*0x45cfa2*/  a3 = dialog this[33] (likely target world index)
```
`OnBuyTransferWorldItem` stores the world-transfer commodity at
`*((_DWORD *)this + 313)` (= offset 1252) @0x459820; `sub_45CF64` re-reads exactly
that offset. So mode 41 is the **commit** step of a two-step v61 world-transfer flow
(0x12 initiate/check → server `OnCheckTransferWorldPossibleResult` @0x463ba6 → dialog
→ 0xC4 mode 41 commit). Body `SN(4) + field(4)`. No distinct atlas key (atlas models
BUY_WORLD_TRANSFER only at the initiate opcode 0x12). Reported UNCERTAIN — the second
field (`this[33]`) is most likely the chosen target-world index but was not proven.

## v61 CashShop mode map (final)

| atlas key | v61 mode | evidence |
|---|---|---|
| BUY | 3 | OnBuy @0x4581e3 / SendBuyAvatarPacket @0x456c6d |
| GIFT | 4 | OnGift @0x458b9b |
| SET_WISHLIST | 5 | OnSetWish @0x45a41a / OnRemoveWish @0x45a53a |
| INCREASE_INVENTORY | 6 | sub_456FE4 @0x457206 |
| INCREASE_STORAGE | 7 | sub_457316 @0x457439 |
| INCREASE_CHARACTER_SLOT | 6 | OnIncCharacterSlotCount @0x459be5 (`(itemId/1000==9110)+6`, else branch) |
| ENABLE_EQUIP_SLOT | 7 | OnIncCharacterSlotCount @0x459be5 (itemId 9110xxx branch) |
| BUY_FRIENDSHIP | 8 | OnBuyFriendship @0x457613 (`push 8; Encode1`) |
| MOVE_FROM_CASH_INVENTORY | 12 | sub_45C063 @0x45c1ee (`Encode1 0xCu`) |
| MOVE_TO_CASH_INVENTORY | 13 | sub_45C271 @0x45c2ed (`Encode1 0xDu`) |
| REBATE_LOCKER_ITEM | 24 | sub_456D87 @0x456f3d (`Encode1 0x18u`) |
| BUY_COUPLE | 28 | OnBuyCouple @0x458620 (`Encode1 0x1Cu`) |
| BUY_OTHER_PACKAGE | 29 | OnGiftPackage @0x4590ca (`Encode1 0x1Du`) |
| BUY_PACKAGE | 30 | OnBuyPackage @0x458870 (`Encode1 0x1Eu`) |
| APPLY_WISHLIST | 31 | sub_45CB0E @0x45cb3c (`Encode1 0x1Fu`, mode-only) |
| BUY_NORMAL | 33 | OnBuyNormal @0x459e9e (`Encode1 0x21u`) |
| BUY_NAME_CHANGE | 39 | sub_45CDD9 @0x45ce08 (`Encode1 0x27u`) |

Unmapped v61 modes (no distinct atlas key): **27** (gift-wrap buy variant),
**41** (world-transfer commit).

ABSENT from 0xC4: **GET_PURCHASE_RECORD** (no emitter). **BUY_WORLD_TRANSFER**
(routed via opcode 0x12, not a 0xC4 mode). Cash-coupon status
(`OnStatusCoupon` @0x45a6b5) uses opcode 0xC5 (197), separate.

## Corrections to `re-v61.md`

- Mode 24 (0x18) is now **confirmed REBATE_LOCKER_ITEM** (was "≈ REBATE 26, unconfirmed").
  Body `ask_SPW int(4) + 8-byte locker serial` = atlas rebate decoder exactly.
- MOVE_FROM_CASH_INVENTORY (**12**), MOVE_TO_CASH_INVENTORY (**13**), APPLY_WISHLIST
  (**31**), BUY_NAME_CHANGE (**39**) were listed "no v61 send located" — all four are
  now resolved to the unnamed send helpers sub_45C063 / sub_45C271 / sub_45CB0E /
  sub_45CDD9 (the prior pass only scanned the named `CCashShop::On*` methods).
- Mode **41 (0x29)** newly identified: world-transfer commit (second packet), which
  the prior pass did not list.
