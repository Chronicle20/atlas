# task-178 — GMS v72 serverbound operation mode bytes (RE)

IDB: `GMS_v72.1_U_DEVM.exe` (ida-pro session `eb2a156e`). Read-only.

Instance guard PASSED: `CField::SendAcceptFriendMsg` @0x5157c4 builds
`COutPacket::COutPacket((COutPacket *)v14, 128)` (0x80 = v72 buddy) then
`COutPacket::Encode1((COutPacket *)v14, 2u)`.

Method: located the client send function that builds `COutPacket(<v72 opcode>)`
and read the leading `Encode1(<mode>)` literal. Every mode below is quoted from
the actual decompiled line. Compared to the v83 ground truth in
`re-reference.md`.

Per the effort-allocation directive, the six families that were byte-identical
on v79 (Buddy, GuildBBS, NPCShop, Storage, Messenger, Guild) were checked with 2
anchor sends each; where both anchors matched v83 the family is recorded
v83-identical (all 6 did, and for GuildBBS/NPCShop/Storage/Messenger the full
table was read anyway and matches). CashShop was fully derived per-op.

---

## Buddy — opcode 0x80 (128) — v83-IDENTICAL

Anchors verified:

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RELOAD | 0 | `CWvsContext::LoadFriend` @0x9369bb | `COutPacket(v2,128)`; `Encode1((COutPacket *)v2, 0)` @0x9369de |
| ACCEPT | 2 | `CField::SendAcceptFriendMsg` @0x5157c4 | `COutPacket(v14,128)`; `Encode1((COutPacket *)v14, 2u)` @0x51583e |

ADD=1, DELETE=3 inferred by v83 parity (both anchors match; family stable on v79
and v72). Full keys: RELOAD=0, ADD=1, ACCEPT=2, DELETE=3.

## GuildBBS — opcode 0x99 (153) — v83-IDENTICAL (full table read)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| CREATE_OR_EDIT_THREAD | 0 | `sub_7517CA` @0x7518d0 | `COutPacket(v17,153)`; `Encode1((COutPacket *)v17, 0)` @0x7518dd |
| DELETE_THREAD | 1 | `sub_7519AF` @0x7519f4 | `COutPacket(v10,153)`; `Encode1((COutPacket *)v10, 1u)` @0x751a01 |
| LIST_THREADS | 2 | `sub_751C3D` @0x751c55 | `COutPacket(v3,153)`; `Encode1((COutPacket *)v3, 2u)` @0x751c63 |
| DISPLAY_THREAD | 3 | `sub_751C9E` @0x751cb6 | `COutPacket(v3,153)`; `Encode1((COutPacket *)v3, 3u)` @0x751cc4 |
| REPLY_THREAD | 4 | `sub_751A68` @0x751b03 | `COutPacket(v13,153)`; `Encode1((COutPacket *)v13, 4u)` @0x751b11 |
| DELETE_REPLY | 5 | `sub_751B7C` @0x751bc0 | `COutPacket(v11,153)`; `Encode1((COutPacket *)v11, 5u)` @0x751bcd |

Anchors (OnRegister=CREATE=0, OnComment=REPLY=4) both match v83. All 6 match.

## NPCShop — opcode 0x3C (60) — v83-IDENTICAL (full table read)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| BUY | 0 | `sub_6A8B15` @0x6a8cb9 | `COutPacket(v32,60)`; `Encode1((COutPacket *)v32, 0)` @0x6a8cca |
| SELL | 1 | `sub_6A8D8F` @0x6a8f3b | `COutPacket(v23,60)`; `Encode1((COutPacket *)v23, 1u)` @0x6a8f4b |
| RECHARGE | 2 | `sub_6A8FB2` @0x6a90c7 | `COutPacket(v15,60)`; `Encode1((COutPacket *)v15, 2u)` @0x6a90d5 |
| LEAVE | 3 | `sub_6A5F39` @0x6a5f4e | `COutPacket(v4,60)`; `Encode1((COutPacket *)v4, 3u)` @0x6a5f5c |

Anchors (BUY=0, SELL=1) both match v83. All 4 match.

## Storage — opcode 0x3D (61) — v83-IDENTICAL (full table read)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RETRIEVE_ASSET | 4 | `sub_703AD8` @0x703c20 | `COutPacket(v22,61)`; `Encode1((COutPacket *)v22, 4u)` @0x703c2e |
| STORE_ASSET | 5 | `sub_703CB5` @0x703ec8 | `COutPacket(v21,61)`; `Encode1((COutPacket *)v21, 5u)` @0x703ed6 |
| ARRANGE_ASSET | 6 | `sub_703F4D` @0x703f6b | `COutPacket(v3,61)`; `Encode1((COutPacket *)v3, 6u)` @0x703f79 |
| MESO | 7 | `sub_703FB0` @0x704019 (deposit) / `sub_704067` @0x704122 (withdraw, `Encode4(-v6)`) | `COutPacket(61)`; `Encode1(..., 7u)` @0x704029 / @0x704132 |
| CLOSE | 8 | `sub_701A4C` @0x701a61 | `COutPacket(v4,61)`; `Encode1((COutPacket *)v4, 8u)` @0x701a6f |

Anchors (RETRIEVE=4, CLOSE=8) both match v83. All match.

## Note / Memo — opcode 0x81 (129) — v83-IDENTICAL (DISCARD/REQUEST verified)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| SEND | 0 (parity-inferred) | — | UI-gated compose; no `COutPacket(129); Encode1(0)` send site located (matches v79). Not asserted from a byte. |
| DISCARD | 1 | `CMemoListDlg::SetRet` @0x5fb4c8 | `COutPacket(v28,129)`; `Encode1((COutPacket *)v28, 1u)` @0x5fb4d5 |
| REQUEST | 2 | `CWvsContext::OnMemoNotify_Receive` @0x91d3ce | `COutPacket(v3,129)`; `Encode1((COutPacket *)v3, 2u)` @0x91d3dc |

DISCARD=1, REQUEST=2 match v83. SEND=0 UI-gated, parity-inferred (same as v79).

## Messenger — opcode 0x78 (120) — v83-IDENTICAL

| key | mode | fn @addr | evidence |
|---|---|---|---|
| ANSWER_INVITE (self-enter) | 0 | `sub_77470A` @0x774c1c | `COutPacket(v71,120)`; `Encode1((COutPacket *)v71, 0)` @0x774c29 |
| CLOSE | 2 | `sub_774C75` @0x774c95 | `COutPacket(v6,120)`; `Encode1((COutPacket *)v6, 2u)` @0x774ca3 |
| INVITE | 3 | `sub_77854F` @0x7785cc | disasm: `push 78h` @0x7785c7, ctor @0x7785cc, `push 3` + `Encode1` @0x7785da, then EncodeStr(name) |
| DECLINE_INVITE | 5 | `CUIMessenger::OnInvite` @0x777e26 | `COutPacket(v9,120)`; `Encode1((COutPacket *)v9, 5u)` @0x777e34 |
| CHAT | 6 | `sub_775287` @0x775313/@0x7753e1 · `sub_77865C` @0x778773 | `COutPacket(120)`; `Encode1(..., 6u)` @0x775320 / @0x7753f2 / @0x778781 |

Anchors (INVITE=3, CLOSE=2) both match v83. All match.

## Guild — opcode 0x7C (124) — v83-IDENTICAL

Anchors verified:

| key | mode | fn @addr | evidence |
|---|---|---|---|
| INVITE | 5 | `CField::SendInviteGuildMsg` @0x514d81 | `COutPacket(v16,124)`; `Encode1((COutPacket *)v16, 5u)` @0x514d8f |
| KICK | 8 | `CField::SendKickGuildMsg` @0x5151a8 | `COutPacket(v14,124)`; `Encode1((COutPacket *)v14, 8u)` @0x5151b6 |

Both anchors (KICK=8, INVITE=5) match v83; opcode 124 = 0x7C confirmed. Remaining
keys inferred by v83 parity (family byte-identical on v79 and v72):
REQUEST_CREATE=2, JOIN=6, WITHDRAW=7, SET_TITLE_NAMES=13, SET_MEMBER_TITLE=14,
SET_EMBLEM=15, SET_NOTICE=16, AGREEMENT_RESPONSE=30.

---

## CashShop — opcode 0xDB (219) — **DIVERGENT ENUM, DO NOT COPY v83**

Every 0xDB (219) construction site in the CCashShop code range (0x461000–0x474000)
was enumerated via `search_text COutPacket::COutPacket` and each decompiled. The
v72 request enum differs materially from both v83 and v79.

### Modes mapped to v83 atlas keys

| v83 key | v83 mode | v72 mode | fn @addr | evidence |
|---|---|---|---|---|
| BUY | 3 | **3** ✓ | `CCashShop::OnBuy` @0x467347; `SendBuyAvatarPacket` @0x465936 | `COutPacket(v37,219)`; `Encode1((COutPacket *)v37, 3u)` @0x467355 |
| GIFT | 4 | **4** ✓ | `CCashShop::SendGiftsPacket` @0x4681df (via `OnGift`) | `COutPacket(&v30,219)`; `Encode1((COutPacket *)&v30, 4u)` @0x4681ed |
| SET_WISHLIST | 5 | **5** ✓ | `CCashShop::OnSetWish` @0x469646; `OnRemoveWish` @0x469769 | `COutPacket(v16,219)`; `Encode1((COutPacket *)v16, 5u)` @0x469654 |
| INCREASE_INVENTORY | 6 | **6** ✓ | `sub_465CDC` @0x465fb6 (inventory tab expand) | `COutPacket(v22,219)`; `Encode1((COutPacket *)v22, 6u)` @0x465fc4 |
| INCREASE_STORAGE | 7 | **7** ✓ | `sub_466201` @0x46632f (storage expand) | `COutPacket(v13,219)`; `Encode1((COutPacket *)v13, 7u)` @0x46633d |
| INCREASE_CHARACTER_SLOT | 8 | **6 / 7** ✗ | `CCashShop::OnIncCharacterSlotCount` @0x469159 | `Encode1((COutPacket *)v31, (v19 / 1000 == 9110) + 6)` @0x469184 — no distinct mode 8; folded into 6/7 |
| ENABLE_EQUIP_SLOT | 9 | **ABSENT** ✗ | — | no `Encode1(9)` on 0xDB; equip-slot-ext consolidated into OnIncCharacterSlotCount 6/7 |
| MOVE_FROM_CASH_INVENTORY | 13 | **12 / 13** ✗ | `sub_46AEC0` @0x46b03e (mode 12); `sub_46B0AE` @0x46b187 (mode 13) | `Encode1((COutPacket *)v24, 0xCu)` @0x46b04c; `Encode1((COutPacket *)v18, 0xDu)` @0x46b197 |
| MOVE_TO_CASH_INVENTORY | 14 | **12 / 13** ✗ | (same two funcs) | v72 locker-move family uses modes 12 & 13 (shifted down 1 from v83's 13/14); exact FROM/TO ↔ 12/13 direction not byte-provable |
| BUY_NORMAL | 20 | **34 (0x22)** ✗ | `CCashShop::OnBuyNormal` @0x469476 | `COutPacket(v22,219)`; `Encode1((COutPacket *)v22, 0x22u)` @0x469484 |
| REBATE_LOCKER_ITEM | 26 | **25 (0x19)** ✗ | `sub_465A87` @0x465c22 | `COutPacket(v21,219)`; `Encode1((COutPacket *)v21, 0x19u)` @0x465c30 |
| BUY_COUPLE | 29 | **29 (0x1D)** ✓ | `CCashShop::OnBuyCouple` @0x467834 | `COutPacket(v32,219)`; `Encode1((COutPacket *)v32, 0x1Du)` @0x467842 |
| BUY_PACKAGE | 30 | **31 (0x1F)** ✗ | `CCashShop::OnBuyPackage` @0x467abb | `COutPacket(v22,219)`; `Encode1((COutPacket *)v22, 0x1Fu)` @0x467ac9 |
| BUY_OTHER_PACKAGE | 31 | **30 (0x1E)** ✗ | `CCashShop::OnGiftPackage` @0x468606 | `COutPacket(v23,219)`; `Encode1((COutPacket *)v23, 0x1Eu)` @0x468614 |
| APPLY_WISHLIST | 33 | **ABSENT** ✗ | — | no distinct mode 33; all wishlist ops (set/remove) use mode 5 |
| BUY_FRIENDSHIP | 35 | **8** ✗ | `CCashShop::OnBuyFriendship` @0x466502 | `COutPacket(v17,219)`; `Encode1((COutPacket *)v17, 8u)` @0x466510 |
| GET_PURCHASE_RECORD | 40 | **39 (0x27)** ✗ | `CCashShop::RequestCashPurchaseRecord` @0x4659c9 | `COutPacket(v3,219)`; `Encode1((COutPacket *)v3, 0x27u)` @0x4659d7 |
| BUY_NAME_CHANGE | 46 | **41 (0x29)** ✗ (tentative) | `sub_46BCAC` @0x46bccb | `COutPacket(v6,219)`; `Encode1((COutPacket *)v6, 0x29u)` @0x46bcd9; encodes Encode4(sn)+EncodeStr+EncodeStr. Best-fit for name-change; atlas-key mapping not fully certain |
| BUY_WORLD_TRANSFER | 49 | **ABSENT** (separate opcode 18) | `CCashShop::OnBuyTransferWorldItem` @0x468b93 → `sub_46BE19` @0x46be2e | builds `COutPacket(v5, 18)` (0x12), not a mode of 0xDB (same as v79) |

### Additional v72 0xDB modes with no clean v83-key mapping (recorded for completeness)

| v72 mode | fn @addr | evidence | note |
|---|---|---|---|
| 28 (0x1C) | `sub_4686F4` @0x468934; `sub_46B4F5` @0x46b84c | `Encode1((COutPacket *)v22, 0x1Cu)` @0x468942 | buy/gift-package-normal variant (ask_SPW + EncodeStr×2) |
| 32 (0x20) | `sub_46B9D7` @0x46b9f5 | `Encode1((COutPacket *)v3, 0x20u)` @0x46ba03 | no-argument request (close/confirm-type) |
| 44 (0x2C) | `sub_46BE7E` @0x46be96 | `Encode1((COutPacket *)v5, 0x2Cu)` @0x46bea4 | Encode4+Encode4 request |

`sub_46B4F5` @0x46b84c selects its mode at runtime (`Encode1(v27)` @0x46b8a7)
from {4, 28, 30, 34} by item id — a shared locker/gift buy dispatcher.

### Non-0xDB cash-shop sends (separate opcodes, not dispatcher modes)

- `sub_46BB81` @0x46bbe7 → `COutPacket(21)` (name-check).
- `sub_46BC47` @0x46bc5c → `COutPacket(16)`.
- `sub_46BE19` @0x46be2e → `COutPacket(18)` (world transfer, per OnBuyTransferWorldItem).
- `CCashShop::OnStatusCoupon` @0x46997e → `COutPacket(220)` (0xDC, coupon).
- `CCashShop::TrySendQueryCashRequest`/`sub_46B1E6` @0x46b208 → `COutPacket(218)` (0xDA).
- `CCashShop::SendTransferFieldPacket` @0x473b49 → `COutPacket(37)`.

---

## DIFFS FROM V83

- **CashShop (0xDB) is heavily renumbered — must NOT copy v83.** Matching keys:
  BUY=3, GIFT=4, SET_WISHLIST=5, INCREASE_INVENTORY=6, INCREASE_STORAGE=7,
  BUY_COUPLE=29. Divergent keys:
  - BUY_FRIENDSHIP 35→**8**
  - INCREASE_CHARACTER_SLOT 8→**6/7** (item-id driven, no distinct mode)
  - ENABLE_EQUIP_SLOT 9→**absent** (folded into 6/7)
  - MOVE_FROM/TO_CASH_INVENTORY 13/14→**12/13**
  - BUY_NORMAL 20→**34 (0x22)**
  - REBATE_LOCKER_ITEM 26→**25 (0x19)**
  - BUY_PACKAGE 30→**31 (0x1F)** and BUY_OTHER_PACKAGE 31→**30 (0x1E)** (swapped vs v83)
  - GET_PURCHASE_RECORD 40→**39 (0x27)**
  - BUY_NAME_CHANGE 46→**41 (0x29)** (tentative)
  - APPLY_WISHLIST 33→**absent**; BUY_WORLD_TRANSFER 49→**absent** (opcode 18)
- Buddy, GuildBBS, NPCShop, Storage, Messenger, Guild: **identical modes to v83.**
- Note: DISCARD/REQUEST identical to v83; SEND(0) UI-gated (parity-inferred).

## UNRESOLVED

- Note SEND (v83=0) — UI-gated compose path; no `COutPacket(129); Encode1(0)`
  send site in the IDB. Recorded parity-inferred=0 (not byte-verified).
- CashShop MOVE_FROM vs MOVE_TO exact mapping to modes 12/13 — both modes exist
  (`sub_46AEC0`=12 with inv-type+slot, `sub_46B0AE`=13); direction naming is
  best-effort, not byte-provable.
- CashShop mode 41 (`sub_46BCAC`) mapped to BUY_NAME_CHANGE by shape
  (sn + 2 strings); atlas-key identity not fully certain.

## ABSENT

- CashShop ENABLE_EQUIP_SLOT (v83=9) — no distinct 0xDB mode; consolidated into
  OnIncCharacterSlotCount modes 6/7.
- CashShop APPLY_WISHLIST (v83=33) — no distinct mode; wishlist ops use mode 5.
- CashShop BUY_WORLD_TRANSFER (v83=49) — routed via separate opcode 18 (0x12),
  not a mode of 0xDB.
