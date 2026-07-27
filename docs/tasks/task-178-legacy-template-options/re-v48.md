# task-178 — GMS v48 serverbound operation mode bytes (RE)

IDB: `GMS_v48_1_DEVM.exe` (ida-pro session `ef9c0dd8`). Read-only.

**Instance guard PASSED:** `CWvsContext::LoadFriend` @0x72ae59 builds
`COutPacket::COutPacket(v2, 100)` (0x64 = v48 buddy opcode) then
`COutPacket::Encode1(v2, 0)` ⇒ Buddy RELOAD=0. Correct session.

Method: located each client send function that builds `COutPacket(<v48 opcode>)`
and read the leading `Encode1(<mode>)` literal, quoted from the actual decompiled
line. Opcode-scoped `push <opcode>h` disassembly scans (completed, `done:true`)
were used to enumerate **every** immediate `COutPacket(<opcode>)` construction
site for the harder families (Messenger 0x5C, NPCShop 0x30, Storage 0x34,
Note 0x65). v48 is the oldest client — CashShop is renumbered and several ops
are absent. GuildBBS is not routed in v48 and is skipped per scope.

v48 serverbound opcodes: Buddy=0x64(100), NPCShop=0x30(48), Storage=0x34(52),
Note=0x65(101), Messenger=0x5C(92), Guild=0x60(96), CashShop=0xA0(160).

Shared refs: `COutPacket::COutPacket(long)` ctor @0x57b77e · `Encode1` @0x44a4b8.

---

## Buddy — opcode 0x64 (100)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RELOAD | 0 | `CWvsContext::LoadFriend` @0x72ae6d | `COutPacket(v2,100)`; `Encode1(v2,0)` |
| ADD | 1 | `sub_4C6452` @0x4c6538 | `COutPacket(v18,100)`; `Encode1(v18,1u)` then `EncodeStr(name)` |
| ACCEPT | 2 | `sub_4C6643` @0x4c66aa | `COutPacket(v11,100)`; `Encode1(v11,2u)`; `Encode4(a1)` (friendId) |
| DELETE | 3 | `sub_4C659B` @0x4c65fb | `COutPacket(v10,100)`; `Encode1(v10,3u)`; `Encode4(a1)` |

All match v83. (v48 has no `SendAcceptFriendMsg` symbol — the add/accept/delete
sends are inlined `CField`-region helpers; ACCEPT is also invoked from the
fade-yes/no handler `sub_4BE880` case 1 → `sub_4C6643`.)

## NPCShop — opcode 0x30 (48)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| BUY | 0 | `sub_5B7422` @0x5b75cc | `COutPacket(v24,48)`; `Encode1(v24,0)`; `Encode2(slot)`,`Encode4(itemId)`,`Encode2(qty)` |
| SELL | 1 | `sub_5B7693` @0x5b7849 | `COutPacket(v16,48)`; `Encode1(v16,1u)`; `Encode2(pos)`,`Encode4(itemId)`,`Encode2(qty)` |
| RECHARGE | 2 | `sub_5B78C0` @0x5b79d4 | `COutPacket(v13,48)`; `Encode1(v13,2u)`; `Encode2(pos)` |
| LEAVE | 3 | `sub_5B4B82` @0x5b4b99 | `COutPacket(v2,48)`; `Encode1(v2,3u)` |

All match v83.

## Storage — opcode 0x34 (52)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RETRIEVE_ASSET | 4 | `sub_57E987` @0x57ea2c | `COutPacket(v13,52)`; `Encode1(v13,4u)`; `Encode4(itemId)` |
| STORE_ASSET | 5 | `sub_57EAA3` @0x57eb04 | `COutPacket(v9,52)`; `Encode1(v9,5u)`; `Encode4(itemId)` |
| MESO | 7 | `sub_5832D1` @0x5832e8 | `COutPacket(v3,52)`; `Encode1(v3,7u)` |

RETRIEVE/STORE/MESO match v83. See UNRESOLVED for ARRANGE_ASSET(6) and CLOSE(8).
Extra v48-only modes on 0x34 (not atlas keys): **mode 2** = "quick delivery /
send package" (`sub_57FB3B` @0x57fc63 `Encode1(v13,2u)` service-charge string;
`sub_58135E` @0x58144a `Encode1(v12,2u)` transport-fee string); **mode 0** =
`sub_4BE880` case 8 @0x4be9fc `Encode1(v6,0)`,`Encode4(0xFFFFFFFF)`,`Encode4(2)`.

## Note / Memo — opcode 0x65 (101)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| DISCARD | 1 | `CMemoListDlg::SetRet` @0x534e45 | `COutPacket(v27,101)`; `Encode1(v27,1u)` then per-note id/type list |

DISCARD=1 matches v83. A **complete** `push 65h` scan (10 hits, `done:true`)
shows only two `COutPacket(101)` construction sites: `CMemoListDlg::SetRet`
(mode 1) and `CCashShop::OnCashItemResLoadLockerDone` @0x453c2b (cash-gift
locker, unrelated). No mode-0 (SEND) or mode-2 (REQUEST) send site exists ⇒
both UNRESOLVED. `CWvsContext::OnMemoResult` @0x71d8e2 is receive-only (Decode1),
and `CUIFadeYesNo::CreateNewMemo` @0x4bf10e is only a fade-window, not a send.

## Messenger — opcode 0x5C (92)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| ANSWER_INVITE (self-enter) | 0 | `sub_61A701` @0x61ac1b | `COutPacket(v53,92)`; `Encode1(v53,0)`; `Encode4(a2)` (roomId) |
| CLOSE | 2 | `sub_61AC75` @0x61ac97 | `COutPacket(v6,92)`; `Encode1(v6,2u)` |
| DECLINE_INVITE | 5 | `sub_61DB2C` @0x61dbb2 | `COutPacket(v10,92)`; `Encode1(v10,5u)`; `EncodeStr(name)`,`EncodeStr(myName)`,`Encode1(1)` |
| CHAT | 6 | `sub_61B27C` @0x61b309 / `sub_61E25E` @0x61e376 | `COutPacket(v25,92)`; `Encode1(v25,6u)`; `EncodeStr(msg)` |

ANSWER_INVITE/CLOSE/DECLINE_INVITE/CHAT match v83. A **complete** `push 5Ch`
scan (30 hits, `done:true`) yields `COutPacket(92)` immediate sends only at the
0x61xxxx messenger-dialog cluster (modes 0/2/5/6) plus `sub_4BCE54` @0x4bcfab
(mode 5, another decline path). **No mode-3 (INVITE) send site exists** ⇒
INVITE=3 UNRESOLVED (all other `push 5Ch` hits — `sub_631512`, `sub_64EAB7`,
`sub_733D91`, etc. — are UI-layout coordinates / NPK-driver constants, not
opcodes).

## Guild — opcode 0x60 (96)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| REQUEST_CREATE | 2 | `CField::InputGuildName` @0x4c59c6 | `COutPacket(v10,96)`; `Encode1(v10,2u)` then `EncodeStr(name)` |
| INVITE | 5 | `CField::SendInviteGuildMsg` @0x4c5c11 | `COutPacket(v13,96)`; `Encode1(v13,5u)` then `EncodeStr(name)` |
| JOIN | 6 | `sub_4BE880` case 6 @0x4be99d | `COutPacket(v8,96)`; `Encode1(v8,6u)`; `Encode4(inviterId)`,`Encode4(guildId)` |
| WITHDRAW | 7 | `sub_4C5CC4` @0x4c5d9a | `COutPacket(v11,96)`; `Encode1(v11,7u)`; `Encode4(guildId)`,`EncodeStr(charName)` |
| KICK | 8 | `CField::SendKickGuildMsg` @0x4c5ff1 | `COutPacket(v16,96)`; `Encode1(v16,8u)`; `Encode4(v5)` |
| SET_TITLE_NAMES | 13 | `sub_4C624A` @0x4c6267 | `COutPacket(v7,96)`; `Encode1(v7,0xDu)` then 5×`EncodeStr` |
| SET_MEMBER_TITLE | 14 | `sub_4C61E4` @0x4c61f8 | `COutPacket(v3,96)`; `Encode1(v3,0xEu)`; `Encode4(memberId)`,`Encode1(grade)` |
| SET_EMBLEM | 15 | `CField::SendSetGuildMarkMsg` @0x4c6370 | `COutPacket(v6,96)`; `Encode1(v6,0xFu)` |
| SET_NOTICE | 16 | `sub_4C63D8` @0x4c63f0 | `COutPacket(v3,96)`; `Encode1(v3,0x10u)` then `EncodeStr(notice)` |
| AGREEMENT_RESPONSE | 30 | `CField::SendCreateGuildAgreeMsg` @0x4c5a33 | `COutPacket(v4,96)`; `Encode1(v4,0x1Eu)` then `Encode4(guildId)`,`Encode1(a2)` |

All match v83. **Extra v48 mode (not an atlas key):** mode 9 — `sub_4C60DC`
@0x4c6187 `COutPacket(v5,96)`; `Encode1(v5,9u)`; `Encode4(guildId)`,`Encode4(a1)`
(master-only op on a member; likely promote/set-grade).

## CashShop — opcode 0xA0 (160) — **DIVERGENT ENUM, DO NOT COPY v83 OR v79**

The v48 CashShop request enum is materially renumbered. Verified per-op sends:

| atlas key | v83 mode | v48 mode | fn @addr | evidence |
|---|---|---|---|---|
| BUY | 3 | **2** ✗ | `CCashShop::OnBuy` @0x44b38a; `SendBuyAvatarPacket` @0x44a44d | `Encode1(v21,2u)`; `Encode1(flag)`,`Encode4(sn)` |
| GIFT | 4 | **3** ✗ | `CCashShop::OnGift` @0x44bd4e | `Encode1(v24,3u)`; `Encode4(spw)`,`Encode4(sn)`,`EncodeStr`,`EncodeStr` |
| SET_WISHLIST | 5 | **4** ✗ | `CCashShop::OnSetWish` @0x44cf78 | `Encode1(v14,4u)` then 10×`Encode4` |
| INCREASE_INVENTORY | 6 | **5** ✗ | `sub_44A7A7` @0x44a9d4 | `Encode1(v18,5u)`; `Encode1(flag)`,`Encode1(0)`,`Encode1(tab 1-4)` |
| INCREASE_STORAGE | 7 | **6** ✗ | `sub_44AAD1` @0x44abe6 | `Encode1(v11,6u)`; `Encode1(flag)`,`Encode1(0)` |
| BUY_COUPLE | 29 | **26 (0x1A)** ✗ | `CCashShop::OnBuyCouple` @0x44b79b | `Encode1(v26,0x1Au)` |
| BUY_OTHER_PACKAGE | 31 | **27 (0x1B)** ✗ | `CCashShop::OnGiftPackage` @0x44c206 | `Encode1(v21,0x1Bu)` |
| BUY_PACKAGE | 30 | **28 (0x1C)** ✗ | `CCashShop::OnBuyPackage` @0x44b9e1 | `Encode1(v16,0x1Cu)` |
| BUY_NORMAL | 20 | **31 (0x1F)** ✗ | `CCashShop::OnBuyNormal` @0x44cdaf | `Encode1(v21,0x1Fu)` |

**Slot-expansion overlap:** `CCashShop::OnBuyFriendship` @0x44cadb (item
911xxxx) sends `Encode1(v25,(itemId/1000==9110)+5)` = **mode 5 (6 for 9110xxx)**,
`Encode1(flag)`,`Encode1(1)`,`Encode4(sn)`. This **collides with
INCREASE_INVENTORY=5 / INCREASE_STORAGE=6** — in v48 modes 5/6 are a single
"increase slot count" family disambiguated by the 3rd byte (0 = inventory tab,
1 = friendship/equip subtype). So v48 has **no distinct** BUY_FRIENDSHIP(35) or
ENABLE_EQUIP_SLOT(9) mode — both funnel through 5/6. Flagged, not copied.

**Extra v48 modes (present, not cleanly mappable to an atlas key):**
mode 22 (0x16) `sub_44A54A` @0x44a6f2 `Encode1(v20,0x16u)`; `Encode4(spw)`,
`EncodeBuffer(commodity,8)` — buy-from-commodity/repurchase. mode 25 (0x19)
`sub_44C2F4` @0x44c505 `Encode1(v21,0x19u)`; `Encode4(spw)`,`Encode4(sn)`,
`EncodeStr`,`EncodeStr` — gift-token buy variant.

---

## DIFFS FROM V83

- **CashShop is heavily renumbered (opcode 0xA0):** BUY 3→2, GIFT 4→3,
  SET_WISHLIST 5→4, INCREASE_INVENTORY 6→5, INCREASE_STORAGE 7→6,
  BUY_COUPLE 29→26, BUY_OTHER_PACKAGE 31→27, BUY_PACKAGE 30→28, BUY_NORMAL 20→31.
  Slot/equip ops consolidated into modes 5/6. **Do NOT copy v83 (or v79) CashShop.**
- **Buddy, NPCShop, Storage, Note, Messenger, Guild: all located modes match v83.**

## UNRESOLVED (searched, no send site found — never assumed to v83)

- **Note SEND (v83=0)** — no `COutPacket(101); Encode1(0)` in the complete
  `push 65h` scan.
- **Note REQUEST (v83=2)** — no `COutPacket(101); Encode1(2)` send site.
- **Messenger INVITE (v83=3)** — no `COutPacket(92); Encode1(3)` in the complete
  `push 5Ch` scan (messenger cluster emits only 0/2/5/6).
- **Storage ARRANGE_ASSET (v83=6)** — no `COutPacket(52); Encode1(6)` in the
  complete `push 34h` scan.
- **Storage CLOSE (v83=8)** — no `COutPacket(52); Encode1(8)` send site
  (client appears not to emit an explicit trunk-close in this DEV build).
- **CashShop BUY_FRIENDSHIP (v83=35)** — send exists (`OnBuyFriendship`, mode
  5/6) but overlaps INCREASE_INVENTORY/STORAGE; no distinct mode. Ambiguous.
- **CashShop ENABLE_EQUIP_SLOT (v83=9)** — no distinct mode; consolidated into
  modes 5/6 (see slot-expansion overlap).
- **CashShop INCREASE_CHARACTER_SLOT (v83=8), GET_PURCHASE_RECORD (v83=40),
  MOVE_FROM_CASH_INVENTORY (13), MOVE_TO_CASH_INVENTORY (14),
  REBATE_LOCKER_ITEM (26), APPLY_WISHLIST (33), BUY_NAME_CHANGE (46)** — no
  corresponding `CCashShop::On*` named send / mode located.

## ABSENT (feature/op routed via a different opcode in v48)

- **CashShop BUY_WORLD_TRANSFER (v83=49)** — `CCashShop::OnBuyTransferWorldItem`
  @0x44c707 sends via `sub_44FB95` @0x44fbac which builds `COutPacket(20)`
  (0x14) `Encode4`,`Encode4` — a **separate opcode**, not a mode of 0xA0.
- **GuildBBS** — out of scope (not routed in v48).
