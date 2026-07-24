# CashShop deep RE — GMS v48 (`GMS_v48_1_DEVM.exe`, IDA session `ef9c0dd8`)

Opcode **0xA0 (160)**. Method: enumerated **every** `push 0A0h` (`68 A0 00 00 00`)
site in the image, mapped each to its containing function, decompiled each, and
read the first `Encode1(mode)` after `COutPacket(v,160)`. All quotes below are the
real decompiled `COutPacket(160)` + `Encode1` lines. Read-only; no renames/patches.

**v48 renumbers the whole enum — do NOT copy v83 or v79.**

## Complete 0xA0 send inventory (all sites, exhaustive)

| mode (dec/hex) | fn @addr | body after mode byte | atlas key |
|---|---|---|---|
| 2 | `SendBuyAvatarPacket` @0x44a21d; `OnBuy` @0x44b0cf | `Encode1(2u)`,`Encode1(flag = state==2)`,`Encode4(sn)` | **BUY** |
| 3 | `OnGift` @0x44ba5d | `Encode1(3u)`,`Encode4(spw)`,`Encode4(sn)`,`EncodeStr(recipient)`,`EncodeStr(msg)` | **GIFT** |
| 4 | `OnSetWish` @0x44ce9b; `OnRemoveWish` @0x44cfff | `Encode1(4u)` then 10×`Encode4` (OnSetWish) / `EncodeBuffer(buf,0x28=40)` (OnRemoveWish, rebuilds the 10-slot list) | **SET_WISHLIST** |
| 5 | `sub_44A7A7` @0x44a7a7 | `Encode1(5u)`,`Encode1(flag)`,`Encode1(0)`,`Encode1(tab 1..4)` | **INCREASE_INVENTORY** (see shared 5/6) |
| 6 | `sub_44AAD1` @0x44aad1 | `Encode1(6u)`,`Encode1(flag)`,`Encode1(0)` | **INCREASE_STORAGE** (see shared 5/6) |
| 5 or 6 | `OnBuyFriendship` @0x44c879 | `Encode1((itemId/1000==9110)+5)`,`Encode1(flag)`,`Encode1(1u)`,`Encode4(sn)` | **BUY_FRIENDSHIP / ENABLE_EQUIP_SLOT** (shared, 3rd byte=1) |
| 10 (0x0A) | `sub_44EC2C` @0x44ec2c | `Encode1(0xAu)`,`EncodeBuffer(&sn,8)`,`Encode1(invType)`,`Encode2(slot)` | **MOVE_FROM_CASH_INVENTORY** (HAS slot) |
| 11 (0x0B) | `sub_44EE3A` @0x44ee3a | `Encode1(0xBu)`,`EncodeBuffer(&sn,8)`,`Encode1(invType)` | **MOVE_TO_CASH_INVENTORY** (NO slot) |
| 22 (0x16) | `sub_44A54A` @0x44a54a | `Encode1(0x16u)`,`Encode4(spw)`,`EncodeBuffer(commodityRec,8)` | *(unmapped — buy-from-commodity/one-a-day)* |
| 25 (0x19) | `sub_44C2F4` @0x44c2f4 | `Encode1(0x19u)`,`Encode4(spw)`,`Encode4(sn)`,`EncodeStr`,`EncodeStr` | *(unmapped — couple/ring gift variant)* |
| 26 (0x1A) | `OnBuyCouple` @0x44b4c1 | `Encode1(0x1Au)` (+recipient/msg/option/birthday) | **BUY_COUPLE** |
| 27 (0x1B) | `OnGiftPackage` @0x44beef | `Encode1(0x1Bu)` | **BUY_OTHER_PACKAGE** |
| 28 (0x1C) | `OnBuyPackage` @0x44b837 | `Encode1(0x1Cu)` | **BUY_PACKAGE** |
| 29 (0x1D) | `sub_44F756` @0x44f756 | `Encode1(0x1Du)` — **mode only, no body** | **APPLY_WISHLIST** |
| 31 (0x1F) | `OnBuyNormal` @0x44cbb2 | `Encode1(0x1Fu)`,`Encode4(sn)` | **BUY_NORMAL** |
| 34 (0x22) | `sub_44F9A5` @0x44f9a5 | `Encode1(0x22u)`,`Encode4(sn)`,`EncodeStr(oldName)`,`EncodeStr(newName)` | **BUY_NAME_CHANGE** |
| 36 (0x24) | `sub_44FBFC` @0x44fbfc | `Encode1(0x24u)`,`Encode4(sn)`,`Encode4(param)` | *(unmapped — world-transfer 0xA0 variant, see below)* |
| 3/25/27/31 | `sub_44F27B` @0x44f27b | dynamic mode by itemId (locker re-buy/gift); `Encode1(mode)`,`Encode4(spw)`,`Encode4(sn)`,`EncodeStr`,`EncodeStr` | *(re-buy dispatcher, reuses GIFT/25/27/BUY_NORMAL)* |

No other `push 0A0h` sites fall inside the CCashShop code region — the enumeration
is exhaustive. (The one stray `0x44867c` is inside `CWvsContext::GetCharacterData`,
a non-sending function; a 160-byte constant, not a `COutPacket`.)

## Mode 5/6 overlap — RESOLVED

Modes 5 and 6 are a single "increase slot-count" family, disambiguated by the
**3rd byte**:

- **3rd byte = 0** → tab/storage expansion:
  - mode 5 = INCREASE_INVENTORY (`sub_44A7A7`; 4th byte = inventory tab 1..4)
  - mode 6 = INCREASE_STORAGE (`sub_44AAD1`; no 4th byte)
- **3rd byte = 1** → `OnBuyFriendship` @0x44c879, one handler for **all** item class
  `911xxxx` (`Data/10000 == 911`, subtype `Data/1000%10` in 0..4). It sends
  `Encode1((itemId/1000==9110)+5)` → **mode 6 for `9110xxx`, mode 5 otherwise**,
  then `Encode1(flag)`,`Encode1(1u)`,`Encode4(sn)`.

Consequence: **BUY_FRIENDSHIP and ENABLE_EQUIP_SLOT have NO distinct mode in v48.**
Both the friendship item (`9110xxx`) and the equip/other slot-expansion coupons
(`9111xxx`..`9114xxx`) are the *same* `OnBuyFriendship` send, funnelled through modes
5/6 with 3rd byte = 1. They are **ABSENT as distinct atlas keys** and must be
routed as the shared 5/6 send. (v83's separate ENABLE_EQUIP_SLOT=9 / BUY_FRIENDSHIP=35
do not exist here.)

## Newly-resolved ops the prior pass had marked "no send located"

- **MOVE_FROM_CASH_INVENTORY = mode 10 (0x0A)** — `sub_44EC2C`. `SN(8)+invType(1)+slot(2)`. Has the slot field (`Encode2`), matching the atlas "FROM has slot".
- **MOVE_TO_CASH_INVENTORY = mode 11 (0x0B)** — `sub_44EE3A`. `SN(8)+invType(1)`. No slot field, matching the atlas "TO has no slot".
- **APPLY_WISHLIST = mode 29 (0x1D)** — `sub_44F756`, mode-only send. Confirmed by caller `sub_47E848` @0x47e848: fires after a YesNo once the 10-slot wishlist (`this+1176..1216`) is full — "buy all wishlist items".
- **BUY_NAME_CHANGE = mode 34 (0x22)** — `sub_44F9A5`. `sn(4)+EncodeStr(old)+EncodeStr(new)`. Driven from character-rename dialogs `sub_5F64AF`/`sub_5F6709`.

## Modes 22 & 25 — identity

- **mode 22 (0x16)** `sub_44A54A`: `Encode4(spw)`,`EncodeBuffer(commodityRec,8)`. Iterates the commodity table and emits an 8-byte commodity record after an SPW (secure-PIN) auth. Best read: **buy-directly-from-commodity / one-a-day / repurchase**. No clean atlas key → **unmapped**.
- **mode 25 (0x19)** `sub_44C2F4`: `Encode4(spw)`,`Encode4(sn)`,`EncodeStr`,`EncodeStr` (recipient + message via `CUISendGifts`/`StringPool`). Shape = GIFT-with-SPW-prefix. The re-buy dispatcher `sub_44F27B` selects mode 25 specifically for item class `11120xx` (couple/marriage-ring items, excluding `1112000`). Best read: **couple/ring gift variant**. → **unmapped/uncertain** (resembles GIFT + BUY_COUPLE).

## Mode 36 (0x24) — world-transfer 0xA0 variant, NOT rebate

`sub_44FBFC` sends `Encode1(0x24u)`,`Encode4(sn)`,`Encode4(param)`. The first field is
`*(CCashShop+1248)` (offset `0x4E0`). That field is written **only** by the two
world-transfer functions:
- `OnBuyTransferWorldItem` @0x44c707 (`*((_DWORD*)this+312) = a2`)
- `sub_44C5F1` @0x44c5f1 (`*((_DWORD*)a1+312) = a3`)

So mode 36 belongs to the **world-transfer** flow (a dialog path `sub_5F7C20`
@0x5f7c20 supplies the 2nd dword from a numeric input field). It is NOT
REBATE_LOCKER_ITEM. → **unmapped** as an atlas key; feature = world transfer.

## BUY_WORLD_TRANSFER — separate opcodes (CONFIRMED) + a 0xA0 variant

The v48 world-transfer feature is spread across **three** sends, all keyed off the
pending SN in `CCashShop+1248`:
- **opcode 18 (0x12)** — `sub_44F93E` @0x44f93e: `Encode4`,`Encode4`. Called by `sub_44C5F1`←`ProcessBuy`.
- **opcode 20 (0x14)** — `sub_44FB95` @0x44fb95: `Encode4`,`Encode4`. Called by `OnBuyTransferWorldItem`.
- **0xA0 mode 36 (0x24)** — `sub_44FBFC` (dialog path), `Encode4(sn)`,`Encode4(param)`.

So the task's expectation ("BUY_WORLD_TRANSFER via separate opcode 0x14/20") is
**confirmed** — the canonical transfer sends are separate opcodes 18/20, not a 0xA0
mode. (Mode 36 is an additional 0xA0 world-transfer variant, but not the primary send.)

## ABSENT as a distinct 0xA0 mode (exhaustively searched, no send)

- **INCREASE_CHARACTER_SLOT** (v83=8) — no 0xA0 send anywhere. Feature not present in v48.
- **GET_PURCHASE_RECORD** (v83=40) — no 0xA0 send. Not present.
- **REBATE_LOCKER_ITEM** (v83=26) — no 0xA0 send (mode 36 is world-transfer, not rebate). Not present.
- **BUY_FRIENDSHIP** (v83=35) — folded into shared modes 5/6 (3rd byte=1); no distinct mode.
- **ENABLE_EQUIP_SLOT** (v83=9) — folded into shared modes 5/6 (3rd byte=1); no distinct mode.
- **BUY_WORLD_TRANSFER** (v83=49) — routed via separate opcodes 18/20 (+ 0xA0 mode 36 variant); not a canonical 0xA0 mode.

## v48 mode map (confident atlas keys)

| atlas key | v48 mode |
|---|---|
| BUY | 2 |
| GIFT | 3 |
| SET_WISHLIST | 4 |
| INCREASE_INVENTORY | 5 |
| INCREASE_STORAGE | 6 |
| MOVE_FROM_CASH_INVENTORY | 10 (0x0A) |
| MOVE_TO_CASH_INVENTORY | 11 (0x0B) |
| BUY_COUPLE | 26 (0x1A) |
| BUY_OTHER_PACKAGE | 27 (0x1B) |
| BUY_PACKAGE | 28 (0x1C) |
| APPLY_WISHLIST | 29 (0x1D) |
| BUY_NORMAL | 31 (0x1F) |
| BUY_NAME_CHANGE | 34 (0x22) |
