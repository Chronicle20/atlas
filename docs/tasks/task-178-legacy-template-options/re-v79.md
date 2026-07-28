# task-178 — GMS v79 serverbound operation mode bytes (RE)

IDB: `GMS_v79_1_DEVM.exe` (ida-pro port 13340). Read-only.
Instance guard PASSED: `CField::SendAcceptFriendMsg` @0x51c85d builds
`COutPacket(&pkt, 127)` (0x7F) then `Encode1(&pkt, 2u)`.

Method: located the client send function that builds `COutPacket(<v79 opcode>)`
and read the leading `Encode1(<mode>)` literal. Modes are quoted from the actual
decompiled/disassembled line. Compared to the v83 ground-truth table in
`re-reference.md`.

---

## Buddy — opcode 0x7F (127)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RELOAD | 0 | `CWvsContext::LoadFriend` @0x98903b | (anchor) |
| ADD | 1 | `sub_51C614` @0x51c72a | `COutPacket(v21,127)`; `Encode1(v21,1u)` then EncodeStr(name), EncodeStr(group) |
| ACCEPT | 2 | `CField::SendAcceptFriendMsg` @0x51c8c7 | `COutPacket(v14,127)`; `Encode1(v14,2u)` |
| DELETE | 3 | `CField::SendDeleteFriendMsg` @0x51c7b1 | (anchor; opcode 127) |

All match v83.

## GuildBBS — opcode 0x98 (152)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| CREATE_OR_EDIT_THREAD | 0 | `CUIGuildBBS::OnRegister` @0x78690e | `COutPacket(v17,152)`; `Encode1(v17,0)` |
| DELETE_THREAD | 1 | `CUIGuildBBS::OnDelete` @0x786a32 | `COutPacket(v10,152)`; `Encode1(v10,1u)` |
| LIST_THREADS | 2 | `CUIGuildBBS::SendLoadListRequest` @0x786c93 | `COutPacket(v3,152)`; `Encode1(v3,2u)` |
| DISPLAY_THREAD | 3 | `CUIGuildBBS::SendViewEntryRequest` @0x786cf4 | `COutPacket(v3,152)`; `Encode1(v3,3u)` |
| REPLY_THREAD | 4 | `CUIGuildBBS::OnComment` @0x786b41 | `COutPacket(v13,152)`; `Encode1(v13,4u)` |
| DELETE_REPLY | 5 | `CUIGuildBBS::OnCommentDelete` @0x786bfe | `COutPacket(v11,152)`; `Encode1(v11,5u)` |

All match v83.

## NPCShop — opcode 0x3B (59)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| BUY | 0 | `CShopDlg::SendBuyRequest` @0x6d6a47 | `COutPacket(v32,59)`; `Encode1(v32,0)` |
| SELL | 1 | `CShopDlg::SendSellRequest` @0x6d6cc9 | `COutPacket(v23,59)`; `Encode1(v23,1u)` |
| RECHARGE | 2 | `CShopDlg::SendRechargeRequest` @0x6d6e55 | `COutPacket(v15,59)`; `Encode1(v15,2u)` |
| LEAVE | 3 | `sub_6D3CC4` @0x6d3cd9 | `COutPacket(v4,59)`; `Encode1(v4,3u)` |

All match v83.

## Storage — opcode 0x3C (60)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RETRIEVE_ASSET | 4 | `sub_73B5B6` @0x73b6fe | `COutPacket(v22,60)`; `Encode1(v22,4u)` |
| STORE_ASSET | 5 | `sub_73B793` @0x73b9a6 | `COutPacket(v21,60)`; `Encode1(v21,5u)` |
| ARRANGE_ASSET | 6 | `sub_73BA2B` @0x73ba49 | `COutPacket(v3,60)`; `Encode1(v3,6u)` |
| MESO | 7 | `sub_73BA8E` @0x73baf7 (deposit) / `sub_73BB45` @0x73bc00 (withdraw, negative amount) | `COutPacket(60)`; `Encode1(...,7u)` |
| CLOSE | 8 | `sub_739527` @0x73953c | `COutPacket(v4,60)`; `Encode1(v4,8u)` |

All match v83.

## Note / Memo — opcode 0x80 (128)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| SEND | UNRESOLVED | — | no `COutPacket(128); Encode1(0)` send site exists in the IDB (see below) |
| DISCARD | 1 | `CMemoListDlg::SetRet` @0x619fb7 | `COutPacket(v28,128)`; `Encode1(v28,1u)` (memo-list return: receive/delete read memos) |
| REQUEST | 2 | `CWvsContext::OnMemoNotify_Receive` @0x96f316 | `COutPacket(v3,128)`; `Encode1(v3,2u)` |

DISCARD=1 and REQUEST=2 match v83. **SEND=0 send site not found** — an
exhaustive scan of every `COutPacket` construction site (xrefs to the
`COutPacket::COutPacket(long)` ctor @0x67ad6b) yields exactly three opcode-128
callers: the two above plus `CCashShop::OnCashItemResLoadGiftDone` @0x47265c
(cash-gift locker, unrelated). None encodes mode 0. Marked UNRESOLVED (v83
parity strongly suggests 0, but not asserted).

## Messenger — opcode 0x77 (119)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| ANSWER_INVITE (self-enter) | 0 | `sub_7B8C8A` @0x7b919c | `COutPacket(v71,119)`; `Encode1(v71,0)` then Encode4(msgId) |
| CLOSE | 2 | `sub_7B91F5` @0x7b9215 | `COutPacket(v6,119)`; `Encode1(v6,2u)` |
| INVITE | 3 | `sub_7BCACF` @0x7bcb47 | `COutPacket(var_24,119)`; `Encode1(...,3)` then EncodeStr(name) (disasm-confirmed) |
| DECLINE_INVITE | 5 | `CUIMessenger::OnInvite` @0x7bc3a6 | `COutPacket(v9,119)`; `Encode1(v9,5u)` |
| CHAT | 6 | `sub_7B9807` @0x7b9893 / @0x7b9961 | `COutPacket(119)`; `Encode1(...,6u)` |

All match v83.

## Guild — opcode 0x7B (123)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| REQUEST_CREATE | 2 | `CField::InputGuildName` @0x51bc07 | `COutPacket(v12,123)`; `Encode1(v12,2u)` then EncodeStr(name) |
| INVITE | 5 | `CField::SendInviteGuildMsg` @0x51be20 | `COutPacket(v12,123)`; `Encode1(v12,5u)` |
| JOIN | 6 | `CUIFadeYesNo::OnButtonClicked` @0x50e49a (case 8) | `COutPacket(v15,123)`; `Encode1(v15,6u)` |
| WITHDRAW | 7 | `CField::SendWithdrawGuildMsg` @0x51bf94 | `COutPacket(v13,123)`; `Encode1(v13,7u)` |
| KICK | 8 | `CField::SendKickGuildMsg` @0x51c247 | `COutPacket(v14,123)`; `Encode1(v14,8u)` |
| SET_TITLE_NAMES | 13 | `CField::SendSetGradeNameMsg` @0x51c42b | `COutPacket(v6,123)`; `Encode1(v6,0xDu)` |
| SET_MEMBER_TITLE | 14 | `CField::SendSetMemberGradeMsg` @0x51c3bf | `COutPacket(v3,123)`; `Encode1(v3,0xEu)` |
| SET_EMBLEM | 15 | `CField::SendSetGuildMarkMsg` @0x51c534 | `COutPacket(v6,123)`; `Encode1(v6,0xFu)` |
| SET_NOTICE | 16 | `CField::SendSetGuildNoticeMsg` @0x51c5b2 | `COutPacket(v2,123)`; `Encode1(v2,0x10u)` |
| AGREEMENT_RESPONSE | 30 | `CField::SendCreateGuildAgreeMsg` @0x51bc72 | `COutPacket(v4,123)`; `Encode1(v4,0x1Eu)` |

All match v83.

## CashShop — opcode 0xDD (221) — **DIVERGENT ENUM, DO NOT COPY v83**

The v79 CashShop request enum is materially different from v83. Verified
per-named-send:

| v83 key | v83 mode | v79 mode | fn @addr | evidence |
|---|---|---|---|---|
| BUY | 3 | **3** ✓ | `CCashShop::OnBuy` @0x46844a; also `SendBuyAvatarPacket` @0x46676d | `Encode1(v28,3u)` |
| GIFT | 4 | **4** ✓ | `CCashShop::SendGiftsPacket` @0x469428 (via `OnGift`) | `Encode1(v28,4u)` |
| SET_WISHLIST | 5 | **5** ✓ | `CCashShop::OnSetWish` @0x46a7ac | `Encode1(v16,5u)` |
| ENABLE_EQUIP_SLOT | 9 | **6 / 7** ⚠ | `CCashShop::OnEnableEquipSlotExt` @0x46a2bf | `Encode1(v31, (itemId/1000==9110)+6)` — mode 6 normal, 7 for 9110xxx |
| BUY_FRIENDSHIP | 35 | **8** ✗ | `CCashShop::OnBuyFriendship` @0x467339 | `Encode1(v17,8u)` |
| INCREASE_CHARACTER_SLOT | 8 | **9** ✗ | `CCashShop::OnIncCharacterSlotCount` @0x4674da | `Encode1(v20,9u)` |
| BUY_COUPLE | 29 | **30 (0x1E)** ✗ | `CCashShop::OnBuyCouple` @0x46898e | `Encode1(v32,0x1Eu)` |
| BUY_OTHER_PACKAGE | 31 | **31 (0x1F)** ✓ | `CCashShop::OnGiftPackage` @0x46984f | `Encode1(v23,0x1Fu)` |
| BUY_PACKAGE | 30 | **32 (0x20)** ✗ | `CCashShop::OnBuyPackage` @0x468c21 | `Encode1(v22,0x20u)` |
| BUY_NORMAL | 20 | **35 (0x23)** ✗ | `CCashShop::OnBuyNormal` @0x46a5dc | `Encode1(v22,0x23u)` |
| GET_PURCHASE_RECORD | 40 | **40 (0x28)** ✓ | `CCashShop::RequestCashPurchaseRecord` @0x466800 | `Encode1(v3,0x28u)` |

CashShop UNRESOLVED / ABSENT keys:
- **BUY_WORLD_TRANSFER (v83=49): ABSENT from 0xDD.** `CCashShop::OnBuyTransferWorldItem`
  @0x469dd6 sends via `sub_46CF90` @0x46cfa5 which builds `COutPacket(18)` (0x12) —
  a **separate opcode**, not a mode of the cash-shop dispatcher.
- INCREASE_INVENTORY (v83=6), INCREASE_STORAGE (v83=7): no distinct v79 named send
  located (slot-expansion appears consolidated into OnEnableEquipSlotExt 6/7). UNRESOLVED.
- MOVE_FROM_CASH_INVENTORY (13), MOVE_TO_CASH_INVENTORY (14), REBATE_LOCKER_ITEM (26),
  APPLY_WISHLIST (33), BUY_NAME_CHANGE (46): no v79 named send with these modes
  located among the CCashShop::On* send functions. UNRESOLVED.

---

## DIFFS FROM V83

- **CashShop is heavily renumbered** (opcode 0xDD): ENABLE_EQUIP_SLOT 9→6/7,
  BUY_FRIENDSHIP 35→8, INCREASE_CHARACTER_SLOT 8→9, BUY_COUPLE 29→30,
  BUY_PACKAGE 30→32, BUY_NORMAL 20→35. Only BUY(3), GIFT(4), SET_WISHLIST(5),
  BUY_OTHER_PACKAGE(31), GET_PURCHASE_RECORD(40) keep their v83 mode.
  **The v79 CashShop operations table must NOT be copied from v83.**
- BUY_WORLD_TRANSFER: not a 0xDD mode in v79 (separate opcode 0x12).
- Buddy, GuildBBS, NPCShop, Storage, Messenger, Guild: **identical modes to v83.**
- Note: DISCARD/REQUEST identical to v83; SEND(0) not located.

## UNRESOLVED

- Note SEND (v83=0) — no `COutPacket(128); Encode1(0)` send site in the IDB.
- CashShop INCREASE_INVENTORY, INCREASE_STORAGE, MOVE_FROM_CASH_INVENTORY,
  MOVE_TO_CASH_INVENTORY, REBATE_LOCKER_ITEM, APPLY_WISHLIST, BUY_NAME_CHANGE —
  no v79 named send located.

## ABSENT

- CashShop BUY_WORLD_TRANSFER — routed via separate opcode 0x12, not a mode of 0xDD.
