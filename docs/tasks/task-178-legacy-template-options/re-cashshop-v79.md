# task-178 — GMS v79 CashShop (opcode 0xDD / 221) deep resolution

IDB: `GMS_v79_1_DEVM.exe` (ida-pro session `88dfa464`). Read-only.
Instance guard: `CField::SendAcceptFriendMsg` builds `COutPacket(&pkt,127)` then
`Encode1(2u)` — v79, matches prior pass.

Method: enumerated **every** `push 0DDh` (byte pattern `68 DD 00 00 00`, 35
sites). The 12 sites outside the `CCashShop` code range (0x466xxx–0x46Dxxx) were
disasm-checked and are UI/COM/StringPool params, **not** `COutPacket(221)` sends
(e.g. `0x587de4` in `CRegisterAuctionEntryDlg::OnCreate` is `push 0DDh` feeding a
`call dword ptr [edx+20h]` vtable call). Every real 0xDD send is in `CCashShop`.
Each mode below is quoted from the actual decompiled `Encode1` line that
immediately follows `COutPacket::COutPacket(v, 221)`.

---

## Resolved this pass (still-open keys)

| atlas key | v79 mode | fn @addr | evidence line |
|---|---|---|---|
| INCREASE_INVENTORY | **6** | `sub_466B13` @0x466ded | `COutPacket(v22,221)`; `Encode1(v22,6u)` then `Encode1(v31==2)`,`Encode4(v31)`,`Encode1(0)`,`Encode1(a2)` — `a2` gated `>=1 && <=4` (tab); StringPool 489/493, cost 4000/tab |
| INCREASE_STORAGE | **7** | `sub_467038` @0x467166 | `COutPacket(v13,221)`; `Encode1(v13,7u)` then `Encode1(v18==2)`,`Encode4(v18)`,`Encode1(0)` — storage limit check `+291 + 4 > 48`, StringPool 490/494 |
| MOVE_FROM_CASH_INVENTORY | **13 (0xD)** | `sub_46C026` @0x46c1a4 | `COutPacket(v24,221)`; `Encode1(v24,0xDu)` then `EncodeBuffer(&Src,8u)` (SN, 8B), `Encode1(a4)` (invType), **`Encode2(a5)` (slot)** — finds empty target slot, locker→inventory. HAS slot. |
| MOVE_TO_CASH_INVENTORY | **14 (0xE)** | `sub_46C214` @0x46c2ed | `COutPacket(v18,221)`; `Encode1(v18,0xEu)` then `EncodeBuffer(&Src,8u)` (SN, 8B), `Encode1(a3)` (invType) — **no slot field**; validates item is cash (`sub_5AC496`), inventory→locker. |
| REBATE_LOCKER_ITEM | **26 (0x1A)** | `sub_4668BE` @0x466a59 | `COutPacket(v21,221)`; `Encode1(v21,0x1Au)` then `Encode4(v25)` (rebate amount) + `EncodeBuffer(v4,8u)` (locker-item key/date). |
| APPLY_WISHLIST | **33 (0x21)** | `sub_46CB3D` @0x46cb5b | `COutPacket(v3,221)`; `Encode1(v3,0x21u)` — **mode byte only, no body**, then SendPacket. |
| BUY_NAME_CHANGE | **45 (0x2D)** | `sub_46CE23` @0x46ce42 | `COutPacket(v6,221)`; `Encode1(v6,0x2Du)` then `Encode4(a2)` (sn), `EncodeStr(oldName)`, `EncodeStr(newName)`. Called by name-change confirm dialogs `sub_760805`/`sub_760A67`. |

## Modes 6 and 7 — clarified (NOT exclusively ENABLE_EQUIP_SLOT)

Modes **6 and 7 are each multiplexed** across two atlas ops by a discriminator
byte. Body layout for both modes: `[mode][flag:1][pointType:Encode4(4)][disc:1]…`

- **disc = 1 → ENABLE_EQUIP_SLOT** (cash-equip / character slot-ext coupon).
  `CCashShop::OnEnableEquipSlotExt` @0x46a2bf:
  `Encode1(v31,(v19/1000==9110)+6)` → **mode 6** for 5430xxx & 911x items,
  **mode 7** for 9110xxx items; then `Encode1(v45==2)`,`Encode4(v45)`,
  `Encode1(1u)` (disc), `Encode4(sn)`. (Handles items `/10000==911` OR `/1000==5430`.)
- **disc = 0, mode 6 → INCREASE_INVENTORY** (`sub_466B13`, trailing `Encode1(tab 1..4)`).
- **disc = 0, mode 7 → INCREASE_STORAGE** (`sub_467038`, no trailing field).

So the "6 / 7 for 9110" of ENABLE_EQUIP_SLOT is correct, but those same two mode
bytes ALSO carry INCREASE_INVENTORY (6) and INCREASE_STORAGE (7); the server must
read the disc byte at body offset +6 to route. **atlas cannot distinguish these
three keys by mode byte alone on v79.**

## BUY_WORLD_TRANSFER — ABSENT from 0xDD (separate opcode 0x12)

`CCashShop::OnBuyTransferWorldItem` @0x469dd6, on YesNo confirm, calls
`sub_46CF90` @0x46cfa5 which builds **`COutPacket(v5,18)`** (0x12) with
`Encode4(a2)`,`Encode4(a3)` — a distinct opcode, **not** a 0xDD mode. Confirmed.

## Unmapped 0xDD modes (no atlas key)

| mode | fn @addr | body | notes / guess |
|---|---|---|---|
| **29 (0x1D)** | `sub_46993D` @0x469b7d (called by `CCashShop::ProcessBuy` @0x46cbc3); also emitted by `CCashShop::GiftWishItem` @0x46ca0d for `item/100==11120` (couple rings) | `Encode1(29)`,`Encode4(index)`,`Encode4(pointType)`,`Encode4(sn)`,`EncodeStr(recipient)`,`EncodeStr(msg)` | Gift/buy **couple-ring** variant. Distinct from BUY_COUPLE=30 (`OnBuyCouple`, body sn+recipient+msg+option+birthday). `GiftWishItem` selects mode by item class: 4=normal gift, 29=11120 couple, 31=910xxxx package, 35=11128 friendship. |
| **48 (0x30)** | `sub_46CFF5` @0x46d00d (called by char-service confirm dialog `sub_761FC8` @0x76208f) | `Encode1(48)`,`Encode4(a2)`,`Encode4(a3)` | Commodity-index + token, structurally like the world-transfer 0x12 body. Best guess: **name-change / character-service commit step** (dialog fetches `GetCharacterName`, computes token via `sub_7629BB`). No atlas key. |

## Already-verified this session (cross-check, matched task's list)

- BUY=3 (`OnBuy` @0x468442 / `SendBuyAvatarPacket` @0x466762), GIFT=4
  (`SendGiftsPacket` @0x469419), SET_WISHLIST=5 (`OnSetWish` @0x46a7a4; also
  `OnRemoveWish` @0x46a8dc re-sends mode 5 with `EncodeBuffer(Src,0x28)` = 10×sn),
  BUY_FRIENDSHIP=8 (`OnBuyFriendship` @0x467331), INCREASE_CHARACTER_SLOT=9
  (`OnIncCharacterSlotCount` @0x4674d2), BUY_COUPLE=30 (`OnBuyCouple` @0x468986),
  BUY_OTHER_PACKAGE=31 (`OnGiftPackage` @0x469847), BUY_PACKAGE=32
  (`OnBuyPackage` @0x468c19), BUY_NORMAL=35 (`OnBuyNormal` @0x46a5d4),
  GET_PURCHASE_RECORD=40 (`RequestCashPurchaseRecord` @0x4667f8).

## v79 CashShop mode map (final)

| atlas key | v79 mode | v83 mode | match? |
|---|---|---|---|
| BUY | 3 | 3 | ✓ |
| GIFT | 4 | 4 | ✓ |
| SET_WISHLIST | 5 | 5 | ✓ |
| INCREASE_INVENTORY | 6 (disc=0) | 6 | ✓ (shares byte w/ ENABLE_EQUIP_SLOT) |
| INCREASE_STORAGE | 7 (disc=0) | 7 | ✓ (shares byte w/ ENABLE_EQUIP_SLOT) |
| ENABLE_EQUIP_SLOT | 6 / 7 (disc=1) | 9 | ✗ (renumbered; multiplexed onto 6/7) |
| BUY_FRIENDSHIP | 8 | 35 | ✗ |
| INCREASE_CHARACTER_SLOT | 9 | 8 | ✗ |
| MOVE_FROM_CASH_INVENTORY | 13 | 13 | ✓ |
| MOVE_TO_CASH_INVENTORY | 14 | 14 | ✓ |
| REBATE_LOCKER_ITEM | 26 | 26 | ✓ |
| BUY_COUPLE | 30 | 29 | ✗ |
| BUY_OTHER_PACKAGE | 31 | 31 | ✓ |
| BUY_PACKAGE | 32 | 30 | ✗ |
| APPLY_WISHLIST | 33 | 33 | ✓ |
| BUY_NORMAL | 35 | 20 | ✗ |
| GET_PURCHASE_RECORD | 40 | 40 | ✓ |
| BUY_NAME_CHANGE | 45 | 46 | ✗ |
| BUY_WORLD_TRANSFER | — (opcode 0x12) | 49 | ✗ (separate opcode) |

Unmapped 0xDD modes: **29**, **48**.
