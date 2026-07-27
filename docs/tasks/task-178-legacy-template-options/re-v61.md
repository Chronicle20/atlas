# task-178 — GMS v61 serverbound operation mode bytes (RE)

IDB: `GMS_v61.1_U_DEVM.exe` (ida-pro session `9a1bdd7a`). Read-only.
Instance guard PASSED: `CField::SendAcceptFriendMsg` @0x4e9df4 builds
`COutPacket((COutPacket *)v12, 118)` (0x76) then `COutPacket::Encode1((COutPacket *)v12, 2u)`.

Method: located the client send function that builds `COutPacket(<opcode>)` and
read the leading `Encode1(<mode>)` literal from the actual decompiled/disassembled
line. Compared to the v83 ground-truth table in `re-reference.md`.

The single `COutPacket(long)` opcode-ctor is `??0COutPacket@@QAE@J@Z` @0x5ffc4f;
its full xref set (391 sites, non-truncated) is the authority for "does a send
for opcode X / mode Y exist".

> **Opcode note (routing):** the reference table's per-version opcodes are correct
> for Buddy(0x76), Guild(0x72), Messenger(0x6E), GuildBBS(0x86), Note(0x77),
> CashShop(0xC4) — all confirmed against the actual client sends. **BUT NPCShop
> and Storage differ from the reference:** the v61 client builds **NPCShop=0x39
> (57)** and **Storage=0x3A (58)**, not 0x3C / 0x3D. Mode bytes below are read
> from those actual client sends and are unaffected, but the routing opcodes for
> NPCShop/Storage in the v61 template should be re-checked.

---

## Buddy — opcode 0x76 (118) — v83-identical (anchor)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RELOAD | 0 | `CWvsContext::LoadFriend` @0x859252 | `COutPacket(v2,118)`; `Encode1(v2,0)` |
| ACCEPT | 2 | `CField::SendAcceptFriendMsg` @0x4e9df4 | `COutPacket(v12,118)`; `Encode1(v12,2u)` |

Both anchors match v83 ⇒ family recorded **v83-identical** (ADD=1, DELETE=3 by parity).

## Guild — opcode 0x72 (114) — v83-identical (anchor)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| INVITE | 5 | `CField::SendInviteGuildMsg` @0x4e92d1 | `COutPacket(v12,114)`; `Encode1(v12,5u)` |
| KICK | 8 | `CField::SendKickGuildMsg` @0x4e95f7 | `COutPacket(v14,114)`; `Encode1(v14,8u)` |

Both anchors match v83 ⇒ family recorded **v83-identical**
(REQUEST_CREATE=2, JOIN=6, WITHDRAW=7, SET_TITLE_NAMES=13, SET_MEMBER_TITLE=14,
SET_EMBLEM=15, SET_NOTICE=16, AGREEMENT_RESPONSE=30 by parity).

## NPCShop — opcode 0x39 (57) *(reference said 0x3C)*

| key | mode | fn @addr | evidence |
|---|---|---|---|
| BUY | 0 | `CShopDlg::SendBuyRequest` @0x646c41 | `COutPacket(v28,57)`; `Encode1(v28,0)` |
| SELL | 1 | `CShopDlg::SendSellRequest` @0x646eae | `COutPacket(v19,57)`; `Encode1(v19,1u)` |
| RECHARGE | 2 | `CShopDlg::SendRechargeRequest` @0x6470c4 | `COutPacket(v13,57)`; `Encode1(v13,2u)` |

BUY/SELL anchors + RECHARGE all match v83 ⇒ **mode table v83-identical** (LEAVE=3 by parity).

## Storage — opcode 0x3A (58) *(reference said 0x3D)* — fully derived

| key | mode | fn @addr | evidence |
|---|---|---|---|
| RETRIEVE_ASSET | 4 | `sub_690ADB` @0x690adb | `COutPacket(v18,58)`; `Encode1(v18,4u)` then Encode1(invType),Encode1(slot) |
| STORE_ASSET | 5 | `sub_690C58` @0x690c58 | `COutPacket(v19,58)`; `Encode1(v19,5u)` then Encode2(slot),Encode4(itemId),Encode2(qty) |
| ARRANGE_ASSET | 6 | `sub_690EBB` @0x690ebb | `COutPacket(v3,58)`; `Encode1(v3,6u)` |
| MESO | 7 | `sub_690F20` @0x690f20 (deposit) / `sub_690FD6` @0x690fd6 (withdraw, negative) | `COutPacket(58)`; `Encode1(...,7u)` then Encode4(±meso) |
| CLOSE | 8 | `sub_68EA19` @0x68ea19 | `COutPacket(v2,58)`; `Encode1(v2,8u)` |

All modes match v83.

## Messenger — opcode 0x6E (110) — fully derived

| key | mode | fn @addr | evidence |
|---|---|---|---|
| ANSWER_INVITE (self-enter) | 0 | `sub_6D02EF` @0x6d02ef | `COutPacket(v53,110)`; `Encode1(v53,0)` then Encode4(msgId) |
| CLOSE | 2 | `sub_6D0863` @0x6d0863 | `COutPacket(v6,110)`; `Encode1(v6,2u)` |
| INVITE | 3 | `sub_6D3F15` @0x6d3f8d (disasm) | `push 6Eh`→`COutPacket`; `push 3`→`Encode1`; EncodeStr(name) |
| DECLINE_INVITE | 5 | `sub_6D3765` @0x6d3765 | `COutPacket(v9,110)`; `Encode1(v9,5u)` then EncodeStr(name),EncodeStr(char),Encode1(1) |
| CHAT | 6 | `sub_6D0E6A` @0x6d0e6a / `sub_6D4021` @0x6d4021 | `COutPacket(110)`; `Encode1(...,6u)` then EncodeStr(chat) |

All modes match v83. (INVITE decompiles opaquely; disasm at 0x6d3f8d–0x6d3fa1
shows `push 6Eh` → ctor, `push 3` → Encode1, EncodeStr — reached via slash-command
handler `sub_6D3DD0`.)

## GuildBBS — opcode 0x86 (134) — fully derived

| key | mode | fn @addr | evidence |
|---|---|---|---|
| CREATE_OR_EDIT_THREAD | 0 | `sub_6BB129` @0x6bb129 | `COutPacket(v15,134)`; `Encode1(v15,0)` then Encode1(edit),Encode4(id?),Encode1(notice),EncodeStr,EncodeStr |
| DELETE_THREAD | 1 | `sub_6BB30C` @0x6bb30c | `COutPacket(v8,134)`; `Encode1(v8,1u)` then Encode4(threadId) |
| LIST_THREADS | 2 | `sub_6BB596` @0x6bb596 | `COutPacket(v3,134)`; `Encode1(v3,2u)` then Encode4(start) |
| DISPLAY_THREAD | 3 | `sub_6BB5F9` @0x6bb5f9 | `COutPacket(v3,134)`; `Encode1(v3,3u)` then Encode4(threadId) |
| REPLY_THREAD | 4 | `sub_6BB3C4` @0x6bb3c4 | `COutPacket(v11,134)`; `Encode1(v11,4u)` then Encode4(threadId),EncodeStr(comment) |
| DELETE_REPLY | 5 | `sub_6BB4D6` @0x6bb4d6 | `COutPacket(v9,134)`; `Encode1(v9,5u)` then Encode4,Encode4 |

All modes match v83.

## Note / Memo — opcode 0x77 (119)

| key | mode | fn @addr | evidence |
|---|---|---|---|
| SEND | UNRESOLVED / ABSENT | — | no `COutPacket(119); Encode1(0)` site in the IDB (see below) |
| DISCARD | 1 | `CMemoListDlg::SetRet` @0x5ad50c | `COutPacket(v25,119)`; `Encode1(v25,1u)` (memo-list return: process/receive read memos) |
| REQUEST | UNRESOLVED / ABSENT | — | no `COutPacket(119); Encode1(2)` site; memos are server-pushed (see below) |

DISCARD=1 matches v83. **Only ONE opcode-119 emitter exists in the entire IDB**
(`CMemoListDlg::SetRet`, mode 1) — confirmed by enumerating every caller of the
single `COutPacket(long)` ctor @0x5ffc4f (391 sites, non-truncated) and by
`search_text "push 77h"` (0 hits). Inbound `CWvsContext::OnMemoResult` @0x8468be
reads the memo list *directly out of the server packet* (`Decode1`==2 →
`RemoveAll` + decode-loop of memos), so the v61 client never sends a REQUEST(2)
to fetch memos — a genuine protocol difference vs v83/v79. Note-compose UI exists
(`CUIFadeYesNo::CreateNewMemo` @0x4dffcb) but emits no opcode-119 packet, so
SEND(0) is likewise not present as a client send.

## CashShop — opcode 0xC4 (196) — **DIVERGENT ENUM, DO NOT COPY v83 OR v79**

Distinct from both v83 and v79. The high "buy family" modes are uniformly
**v79 minus 2** (couple 30→28, gift-pkg 31→29, package 32→30, normal 35→33), and
character-slot/equip-slot expansion is folded into the shared 6/7 slot modes
(no separate 8/9 like v83, no separate 9 like v79).

| v83 key (semantic) | v61 mode | fn @addr | v83 mode | match |
|---|---|---|---|---|
| BUY | 3 | `CCashShop::OnBuy` @0x457ea4; `SendBuyAvatarPacket` @0x456a42 | 3 | ✓ |
| GIFT | 4 | `CCashShop::OnGift` @0x4588f4 | 4 | ✓ |
| SET_WISHLIST | 5 | `CCashShop::OnSetWish` @0x45a345; `OnRemoveWish` @0x45a4a9 | 5 | ✓ |
| INCREASE_INVENTORY | 6 | `sub_456FE4` @0x456fe4 | 6 | ✓ |
| INCREASE_STORAGE | 7 | `sub_457316` @0x457316 | 7 | ✓ |
| INCREASE_CHARACTER_SLOT / ENABLE_EQUIP_SLOT | 6 or 7 | `CCashShop::OnIncCharacterSlotCount` @0x459928 | 8 / 9 | ✗ (folded into 6/7 via `Encode1((itemId/1000==9110)+6)`) |
| BUY_FRIENDSHIP | 8 | `CCashShop::OnBuyFriendship` @0x4574b0 | 35 | ✗ |
| BUY_COUPLE | 0x1C = 28 | `CCashShop::OnBuyCouple` @0x45832d | 29 | ✗ |
| BUY_OTHER_PACKAGE (gift pkg) | 0x1D = 29 | `CCashShop::OnGiftPackage` @0x458e10 | 31 | ✗ |
| BUY_PACKAGE | 0x1E = 30 | `CCashShop::OnBuyPackage` @0x4586ce | 30 | ✓ (coincidental; = v79 32 − 2) |
| BUY_NORMAL | 0x21 = 33 | `CCashShop::OnBuyNormal` @0x459c93 | 20 | ✗ |

Two additional verified buy-flow modes whose exact v83 key is uncertain:
| mode | fn @addr | evidence / semantic |
|---|---|---|
| 0x18 = 24 | `sub_456D87` @0x456d87 | `COutPacket(v20,196)`; `Encode1(v20,0x18u)` then Encode4(spw),EncodeBuffer(commodity,8) — discounted/coupon-rebate purchase (≈ v83 REBATE_LOCKER_ITEM 26, unconfirmed) |
| 0x1B = 27 | `sub_4591C0` @0x4591c0 | `COutPacket(v21,196)`; `Encode1(v21,0x1Bu)` then Encode4(spw),Encode4(flags),Encode4(item),EncodeStr,EncodeStr — gift-wrap buy variant |

### CashShop ABSENT from 0xC4
- **BUY_WORLD_TRANSFER (v83=49): ABSENT.** `CCashShop::OnBuyTransferWorldItem`
  @0x459737 sends via `sub_45CEFD` @0x45cefd which builds `COutPacket(v5,18)`
  (**0x12**, `Encode4(worldNo);Encode4(spw)`) — a separate opcode, not a 0xC4 mode.
- **Cash coupon** (`CCashShop::OnStatusCoupon` @0x45a6b5) builds `COutPacket(v10,197)`
  (**0xC5**), a separate opcode, not a 0xC4 mode.

### CashShop UNRESOLVED (no v61 send site among the CCashShop send functions)
- GET_PURCHASE_RECORD (v83=40) — no v61 emitter (v79 had `RequestCashPurchaseRecord`; absent here).
- MOVE_FROM_CASH_INVENTORY (13), MOVE_TO_CASH_INVENTORY (14) — no distinct v61 named send located.
- APPLY_WISHLIST (v83=33) — no v61 emitter (note: v61 mode 33 is **BUY_NORMAL**, not apply-wishlist).
- BUY_NAME_CHANGE (v83=46) — no v61 emitter located.

---

## DIFFS FROM V83
- **CashShop (0xC4) is heavily renumbered — must be derived, not copied.**
  BUY_FRIENDSHIP 35→8, BUY_COUPLE 29→28, BUY_OTHER_PACKAGE 31→29, BUY_NORMAL 20→33,
  slot-expansion 8/9→folded 6/7. Only BUY(3), GIFT(4), SET_WISHLIST(5),
  INCREASE_INVENTORY(6), INCREASE_STORAGE(7), BUY_PACKAGE(30, coincidental) share
  the v83 mode.
- **BUY_WORLD_TRANSFER**: separate opcode 0x12 in v61, not a 0xC4 mode.
- **NPCShop opcode = 0x39** and **Storage opcode = 0x3A** (reference expected 0x3C/0x3D);
  mode bytes for both families are v83-identical.
- **Note REQUEST(2)** not sent by v61 client — memos are server-pushed via OnMemoResult.
- Buddy, Guild, NPCShop, Storage, Messenger, GuildBBS **mode tables identical to v83.**

## UNRESOLVED
- Note SEND (v83=0), Note REQUEST (v83=2) — no opcode-119 send sites; only DISCARD(1) emitted.
- CashShop GET_PURCHASE_RECORD, MOVE_FROM/TO_CASH_INVENTORY, APPLY_WISHLIST, BUY_NAME_CHANGE — no v61 send located.
- CashShop mode 24 (0x18) and 27 (0x1B): verified modes, exact v83-key mapping uncertain.

## ABSENT
- CashShop BUY_WORLD_TRANSFER — routed via separate opcode 0x12, not a 0xC4 mode.
- CashShop cash-coupon — separate opcode 0xC5, not a 0xC4 mode.
