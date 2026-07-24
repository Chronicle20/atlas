# task-178 — Legacy template `options` RE reference

## Problem
PR #971 routed the socket handlers for GMS v48/v61/v72/v79 but left 9–11 handlers
with **no `options` block**. atlas-channel handlers that resolve a serverbound
*operation mode byte* via `readerOptions["operations"]` (e.g. `isBuddyOperation`)
have **no fallback** — an absent table makes every branch log
`"Code [X] not configured for use."` and silently drop the packet. So buddy /
guild / guild-BBS / messenger / note / storage / NPC-shop / cash-shop and the
pet/monster/NPC movement handlers are all dead on these versions.

## Two option categories

### A. `types` movement tables — NO RE (copy)
`NPCActionHandle`, `PetMovementHandle`, `MonsterMovementHandle` carry a `types`
list (movement-fragment type → category). In v83/v84 these are **byte-identical
to that version's `CharacterMoveHandle.types`** (the shared CMovePath type
table). Every legacy template already has a verified `CharacterMoveHandle.types`
(23 entries). **Resolution: copy each version's own `CharacterMoveHandle.types`
into its Pet/Monster/NPCAction handlers.** No IDA work.

### B. `operations` serverbound mode tables — RE per legacy IDB
8 families. The atlas-channel handler reads a mode byte the **client sends**.
Recipe (validated on v83 buddy ACCEPT and v79 buddy ACCEPT):

> Find the client send function that builds `COutPacket(<serverbound opcode>)`
> then reads the leading `COutPacket::Encode1(<mode>)`. That literal is the mode.

v83 `CField::SendAcceptFriendMsg` → `COutPacket(0x82); Encode1(2)` ⇒ ACCEPT=2.
v79 `CField::SendAcceptFriendMsg` → `COutPacket(0x7F); Encode1(2)` ⇒ ACCEPT=2.
(Opcodes shift per version; the **mode byte** is what we need.)

Send functions keep the same mangled names across versions — use
`func_query name_regex` to locate them, then `decompile`.

## v83 ground-truth operation tables (verified; keys + mode ints)

| Family (handler) | v83 opcode | key = mode |
|---|---|---|
| BuddyOperationHandle | 0x82 | RELOAD=0, ADD=1, ACCEPT=2, DELETE=3 |
| GuildBBSHandle | 0x9B | CREATE_OR_EDIT_THREAD=0, DELETE_THREAD=1, LIST_THREADS=2, DISPLAY_THREAD=3, REPLY_THREAD=4, DELETE_REPLY=5 |
| NPCShopHandle | 0x3D | BUY=0, SELL=1, RECHARGE=2, LEAVE=3 |
| StorageOperationHandle | 0x3E | RETRIEVE_ASSET=4, STORE_ASSET=5, ARRANGE_ASSET=6, MESO=7, CLOSE=8 |
| NoteOperationHandle | 0x83 | SEND=0, DISCARD=1, REQUEST=2 |
| MessengerOperationHandle | 0x7A | ANSWER_INVITE=0, CLOSE=2, INVITE=3, DECLINE_INVITE=5, CHAT=6 |
| GuildOperationHandle | 0x7E | REQUEST_CREATE=2, INVITE=5, JOIN=6, WITHDRAW=7, KICK=8, SET_TITLE_NAMES=13, SET_MEMBER_TITLE=14, SET_EMBLEM=15, SET_NOTICE=16, AGREEMENT_RESPONSE=30 |
| CashShopOperationHandle | 0xE5 | BUY=3, GIFT=4, SET_WISHLIST=5, INCREASE_INVENTORY=6, INCREASE_STORAGE=7, INCREASE_CHARACTER_SLOT=8, ENABLE_EQUIP_SLOT=9, MOVE_FROM_CASH_INVENTORY=13, MOVE_TO_CASH_INVENTORY=14, BUY_NORMAL=20, REBATE_LOCKER_ITEM=26, BUY_COUPLE=29, BUY_PACKAGE=30, BUY_OTHER_PACKAGE=31, APPLY_WISHLIST=33, BUY_FRIENDSHIP=35, GET_PURCHASE_RECORD=40, BUY_NAME_CHANGE=46, BUY_WORLD_TRANSFER=49 |

## Per-version serverbound opcodes (from templates, already routed)

| Family | v79 | v72 | v61 | v48 |
|---|---|---|---|---|
| Buddy | 0x7F | 0x80 | 0x76 | 0x64 |
| GuildBBS | 0x98 | 0x99 | 0x86 | (not routed) |
| NPCShop | 0x3B | 0x3C | 0x3C | 0x30 |
| Storage | 0x3C | 0x3D | 0x3D | 0x34 |
| Note | 0x80 | 0x81 | 0x77 | 0x65 |
| Messenger | 0x77 | 0x78 | 0x6E | 0x5C |
| Guild | 0x7B | 0x7C | 0x72 | 0x60 |
| CashShop | 0xDD | 0xDB | 0xC4 | 0xA0 |

## IDA instances
v48=13337 (GMS_v48_1_DEVM.exe) · v61=13338 (GMS_v61.1_U_DEVM.exe) ·
v72=13339 (GMS_v72.1_U_DEVM.exe) · v79=13340 (GMS_v79_1_DEVM.exe) ·
v83=13342 (MapleStory_dump.exe, ground truth).

## Grounding rules (hard)
- Read the **actual** `Encode1` literal from the decompile; quote the line.
- A mode you cannot locate a send-site for = **UNRESOLVED**, not a v83 copy.
  List it explicitly for stop-and-ask. Never invent.
- Flag every mode that DIFFERS from the v83 value above.
- Older clients may lack some ops entirely (feature absent) — if the send
  function/opcode does not exist, mark the key **ABSENT** with evidence
  (searched names, not found), don't guess.
