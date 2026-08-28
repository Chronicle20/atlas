# v95 serverbound mode tables — PartyOperationHandle / GuildOperationHandle / BuddyOperationHandle

Task 7 (W3a). IDB `ecc757f4` (`GMS_v95.0_U_DEVM.exe`, confirmed via `server_health`:
`idb_path` ends `IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`, `imagebase 0x400000`).

Method, per the "enumerated by construction" discipline in
`cash_shop_operation_handle.yaml`: the mode switch is a server-side construct —
each client UI action builds its own `COutPacket` and writes its own leading
`Encode1(mode)`. For each opcode, `find_bytes` searched the whole `.text`
image for the literal 5-byte immediate push that constructs
`COutPacket::COutPacket(&oPacket, <opcode>)` — `68 <op-lo> 00 00 00` (`<op>`
is 0x91/0x95/0x99, all >= 0x80 so never encoded as the 2-byte `6A` sign-extend
form; confirmed by inspection of every real hit below). Every match was
resolved to its owning function with `lookup_funcs`; non-owning matches (UI
`Draw`/color/coordinate constants that happen to equal the opcode value) were
excluded. Every remaining function was decompiled and its `COutPacket::COutPacket(&oPacket, N)` →
first `COutPacket::Encode1(&oPacket, mode)` pair read directly.

## PARTY_OPERATION — opCode 0x91 (145), handler `PartyOperationHandle`

`find_bytes "68 91 00 00 00"` → 22 hits. 5 resolve to real send sites (all
`CField::` party methods); the other 17 resolve to unrelated UI `Draw`/
`OnMouseButton`/`OnCreate` functions (0x91 used there as an unrelated
color/coordinate/string-id constant) or to the clientbound
`CWvsContext::OnPartyResult` receiver, and are excluded.

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 1 | CREATE | `CField::SendCreateNewPartyMsg` | 0x52ed59 | 0x52ebc0 |
| 2 | LEAVE | `CField::SendWithdrawPartyMsg` | 0x52ef3b | 0x52edb0 |
| 4 | INVITE | `CField::SendJoinPartyMsg` | 0x534549 | 0x534310 |
| 5 | EXPEL | `CField::SendKickPartyMsg` | 0x530287 | 0x530140 |
| 6 | CHANGE_LEADER | `CField::SendChangePartyBossMsg` | 0x530498 | 0x530370 |

`CField::SendJoinPartyMsg` (mode 4) is a misleading client-internal name: its
only three callers are `CTabParty::OnInvite` (0x8c6e10), `CTabFriend::OnPartyInvite`
(0x8c4390) and `CUserLocal::HandleRButtonClk` (0x911a60, right-click "invite to
party") — every call site is the *inviter* naming a target, never an accept
flow. Confirmed via `xrefs_to 0x534310` (7 xrefs, all invite-initiating UI).
The mode value (4) matches v83's INVITE=4 exactly (template_gms_83_1.json), so
the Atlas key is INVITE, not JOIN.

No mode-3 arm exists on PARTY_OPERATION in this build: accepting an invite in
v95 is routed through a *different* opcode, PARTY_RESULT (0x92, registry op
`PARTY_RESULT`, serverbound, `fname_alts: CUIFadeYesNo::OnButtonClicked`).
Decompiled `CUIFadeYesNo::OnButtonClicked` (0x529c60) in full: its `case 5`
(party-invite dialog type, `m_nType == 5`) builds `COutPacket(&oPacket, 146)`
(0x92) with mode 0x1B, not opcode 0x91. So PARTY_OPERATION mode 3 (v83's
JOIN) is not sent by this client at all — omitted per "arm whose Atlas key
is unknown is omitted, never guessed" (here: the arm itself doesn't exist,
which is the strongest form of that rule).

## GUILD_OPERATION — opCode 0x95 (149), handler `GuildOperationHandle`

`find_bytes "68 95 00 00 00"` → 24 hits. 10 resolve to real send sites; the
rest resolve to unrelated UI `Draw`/`OnCreate`/`OnMouseButton` functions,
whisper-message helpers (`SendLocationWhisper`, `SendChatMsgWhisper` — 0x95
there is an unrelated constant, not a `COutPacket(149)` build), or
`CWvsContext::OnGuildResult` (clientbound), and are excluded.

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 2 | REQUEST_CREATE | `CField::InputGuildName` | 0x534866 | 0x534866 |
| 5 | INVITE | `CField::SendInviteGuildMsg` | 0x534a46 | 0x534a46 |
| 6 | JOIN | `CUIFadeYesNo::OnButtonClicked` case 8 | 0x529e4f | 0x529c60 |
| 7 | WITHDRAW | `CField::SendWithdrawGuildMsg` | 0x534be7 | 0x534be7 |
| 8 | KICK | `CField::SendKickGuildMsg` | 0x534f26 | 0x534f26 |
| 13 | SET_TITLE_NAMES | `CField::SendSetGradeNameMsg` | 0x535003 | 0x535003 |
| 14 | SET_MEMBER_TITLE | `CField::SendSetMemberGradeMsg` | 0x52d843 | 0x52d843 |
| 15 | SET_EMBLEM | `CField::SendSetGuildMarkMsg` | 0x52d8e3 | 0x52d8e3 |
| 16 | SET_NOTICE | `CField::SendSetGuildNoticeMsg` | 0x5351a3 | 0x5351a3 |
| 32 | AGREEMENT_RESPONSE | `CField::SendCreateGuildAgreeMsg` | 0x52d7aa | 0x52d7aa |

`CUIFadeYesNo::OnButtonClicked` (0x529c60), full switch decompiled: `case 8`
(`m_nType == 8`, gated on `this->m_bGameOpt_Guild`) builds
`COutPacket(&oPacket, 149); Encode1(&oPacket, 6u);` — this is the "accept
guild invite" Yes-click path, matching v83's JOIN=6 exactly
(`template_gms_83_1.json`).

`AGREEMENT_RESPONSE` mode is 32 (0x20) here vs v83's 30 — a genuine per-version
value shift (same pattern as the documented CASHSHOP_OPERATION drift in
`cash_shop_operation_handle.yaml`), confirmed directly from
`CField::SendCreateGuildAgreeMsg(CField*, unsigned __int8 bAgree)`:
`COutPacket::COutPacket(&oPacket, 149); COutPacket::Encode1(&oPacket, 0x20u); COutPacket::Encode1(&oPacket, bAgree);`
— the `bAgree` byte written right after the mode is the accept/decline flag,
consistent with "agreement response" semantics.

All 10 v83 GUILD_OPERATION keys (REQUEST_CREATE, INVITE, JOIN, WITHDRAW,
KICK, SET_TITLE_NAMES, SET_MEMBER_TITLE, SET_EMBLEM, SET_NOTICE,
AGREEMENT_RESPONSE) are accounted for on v95; the table is complete.

## BUDDYLIST_MODIFY — opCode 0x99 (153), handler `BuddyOperationHandle`

`find_bytes "68 99 00 00 00"` → 25 hits. 4 resolve to real send sites; 2
(`0xb8795a`, `0xbabffd`) resolve to no owning function at all
(`lookup_funcs` returns `fn: null` — the byte sequence is incidental, not
code) and the remaining 19 resolve to unrelated UI `Draw`/`OnCreate`
functions where 0x99 is an unrelated constant.

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 0 | RELOAD | `CWvsContext::LoadFriend` | 0xa10263 | 0xa10263 |
| 1 | ADD | `CField::SendSetFriendMsg` | 0x5353c2 | 0x5353c2 |
| 2 | ACCEPT | `CField::SendAcceptFriendMsg` | 0x52f340 | 0x52f340 |
| 3 | DELETE | `CField::SendDeleteFriendMsg` | 0x52f219 | 0x52f219 |

All 4 v83 BUDDYLIST_MODIFY keys (RELOAD, ADD, ACCEPT, DELETE) are accounted
for on v95 with identical mode values to v83; the table is complete.

## Unresolved (Task 7)

None. Every candidate byte match was either attributed to a named,
IDA-decompiled send site or excluded with a stated reason (unrelated UI
constant, clientbound receiver, or non-code byte sequence). No arm was
guessed.

---

# Task 8 — MessengerOperationHandle / GuildBBSHandle / StorageOperationHandle

Same IDB (`ecc757f4`, `GMS_v95.0_U_DEVM.exe`, confirmed via `server_health`),
same "enumerated by construction" `find_bytes` + `lookup_funcs` +
`decompile` discipline as Task 7.

## MESSENGER — opCode 0x8F (143), handler `MessengerOperationHandle`

`find_bytes "68 8F 00 00 00"` → 17 hits (opcode 0x8F >= 0x80, always the
5-byte `68`-immediate push form). `lookup_funcs` resolved all 17. 7 unique
functions are real send sites (`CFadeWnd::SendCloseMessage`,
`CUIMessenger::OnDestroy`, `CUIMessenger::OnInvite`, `CUIMessenger::Update`
— 2 of the 17 hits, both inside this one function —, `CUIMessenger::SendInviteMsg`,
`CUIMessenger::OnCreate`, `CUIMessenger::ProcessChat`); the remaining 9
resolve to unrelated UI `Draw`/`OnCreate`/`OnMouseButton` functions
(`CMemoListDlg::DrawMemo`, `CUIAdminShopWishListCategory::OnMouseButton`,
`CUIClaim::OnCreate`, `CUIDragonBox::SetOrb`, `TabPartyAdver::Draw_Regist`,
`CUIShopScannerCategory::OnMouseButton`, `CUISkillChangeConfirm::Draw`,
`CUIUserInfo::Draw`) where 0x8F is an unrelated color/coordinate constant,
and **one** (`CField::SendChatMsgSlash` @0x5444c1) which disassembles to
`push 8Fh ; nIdx` immediately followed by `call StringPool::GetInstance` /
`StringPool::GetString` — a StringPool string-id argument (id 0x8F), NOT a
`COutPacket` construction; confirmed by `get_bytes` at 0x5444c1
(`68 8f 00 00 00 51 e8 d4`, i.e. the immediate is pushed then `StringPool::GetInstance`
is called, not `COutPacket::COutPacket`) and `disasm` (comment `; nIdx`).

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 0 | ANSWER_INVITE | `CUIMessenger::OnCreate` | 0x7f5c14 | 0x7f59d0 |
| 2 | CLOSE | `CUIMessenger::OnDestroy` | 0x7f042a | 0x7f03f0 |
| 3 | INVITE | `CUIMessenger::SendInviteMsg` | 0x7f5911 | 0x7f5820 |
| 5 | DECLINE_INVITE | `CFadeWnd::SendCloseMessage` case 0, and `CUIMessenger::OnInvite` (self-invite/blacklist auto-decline) | 0x5241cd / 0x7f2d54 | 0x524180 / 0x7f2cb0 |
| 6 | CHAT | `CUIMessenger::Update` (2 sites: send-then-clear, clear-then-focus) and `CUIMessenger::ProcessChat` | 0x7f30be / 0x7f31d9 / 0x7f62ab | 0x7f2ff0 / 0x7f2ff0 / 0x7f6140 |

All 5 v83 MESSENGER keys (`template_gms_83_1.json`: ANSWER_INVITE=0, CLOSE=2,
INVITE=3, DECLINE_INVITE=5, CHAT=6 — v83 has no CREATE key; `CREATE` is a
handler-side sub-branch of `ANSWER_INVITE` mode 0 with `MessengerId()==0`,
not a distinct wire mode) match v95 exactly — no drift. The table is
complete.

## BBS_OPERATION — opCode 0xB3 (179), handler `GuildBBSHandle`

`find_bytes "68 B3 00 00 00"` → 12 hits. 6 resolve to real send sites, all
`CUIGuildBBS::` methods (matching the registry's `BBS_OPERATION` serverbound
row `fname`/`fname_alts`); the remaining 6 (`CUIItemUpgrade::OnCreate` x2,
`TabPartySearch::OnCreate` x3, `TabPartySearch::ActivateControls`) resolve to
unrelated UI functions where 0xB3 is an unrelated constant.

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 0 | CREATE_OR_EDIT_THREAD | `CUIGuildBBS::OnRegister` | 0x7c442b | 0x7c4250 |
| 1 | DELETE_THREAD | `CUIGuildBBS::OnDelete` | 0x7c6581 | 0x7c6520 |
| 2 | LIST_THREADS | `CUIGuildBBS::SendLoadListRequest` | 0x7c36af | 0x7c3680 |
| 3 | DISPLAY_THREAD | `CUIGuildBBS::SendViewEntryRequest` | 0x7c373f | 0x7c3710 |
| 4 | REPLY_THREAD | `CUIGuildBBS::OnComment` | 0x7c461c | 0x7c4530 |
| 5 | DELETE_REPLY | `CUIGuildBBS::OnCommentDelete` | 0x7c3bd1 | 0x7c3b70 |

`OnDelete` (mode 1) is not in the registry's `fname_alts` list but is a real
send site (Yes/No-confirmed thread deletion, distinct from `OnCommentDelete`
which deletes a single reply/comment) — its Atlas key (`DELETE_THREAD`) is
unambiguous from v83's identical mode-1 key and the handler's dispatch on
`sp.ThreadId()` alone (no reply id).

All 6 v83 GuildBBS keys (`template_gms_83_1.json`: CREATE_OR_EDIT_THREAD=0,
DELETE_THREAD=1, LIST_THREADS=2, DISPLAY_THREAD=3, REPLY_THREAD=4,
DELETE_REPLY=5) match v95 exactly — no drift. The table is complete.

## STORAGE — opCode 0x43 (67), handler `StorageOperationHandle`

Opcode 0x43 = 67 < 0x80, so the compiler always uses the 2-byte sign-extend
push form (`6A 43`), not the 5-byte `68`-immediate form (`find_bytes "68 43
00 00 00"` → 0 hits, confirming this). `find_bytes "6A 43"` → 54 hits (a much
noisier search since `6A 43` also matches unrelated single-byte immediate
pushes across the whole image). Filtered to real `COutPacket(&oPacket, 67)`
build sites by requiring the 3rd byte after the match to be the `lea
this,[esp+...]; this` opcode `0x8D` that this codegen idiom always emits
between `push 43h` and `call COutPacket::COutPacket` (verified directly
against the three named send sites' disassembly first, e.g. `CTrunkDlg::SendGetItemRequest`
@0x769f90: `push 43h` / `lea ecx,[esp+...]` / `call COutPacket::COutPacket`).
This reduced 54 candidates to exactly 6 (`get_bytes` + byte-3 filter), all 6
resolving via `lookup_funcs` to real `CTrunkDlg::` send/close methods; the
other 48 were spot-checked via `lookup_funcs` and are either unrelated UI
`Draw`/`OnCreate` functions (single-byte immediate operands that happen to
equal 0x43) or non-code data (`.rdata`, `"Not a function"`).

| mode | Atlas key | send-site fn | build-site addr (push) | fn addr |
|---|---|---|---|---|
| 4 | RETRIEVE_ASSET | `CTrunkDlg::SendGetItemRequest` | 0x769f90 | 0x769e00 |
| 5 | STORE_ASSET | `CTrunkDlg::SendPutItemRequest` | 0x768831 | 0x768570 |
| 6 | ARRANGE_ASSET | `CTrunkDlg::SendSortItemRequest` | 0x767345 | 0x767310 |
| 7 | MESO | `CTrunkDlg::SendGetMoneyRequest` (withdraw) and `CTrunkDlg::SendPutMoneyRequest` (deposit, encodes `-v8`) | 0x768974 / 0x768aeb | 0x7688e0 / 0x7689e0 |
| 8 | CLOSE | `CTrunkDlg::SetRet` | 0x76727c | 0x767250 |

The three brief-named send sites (`SendGetMoneyRequest`, `SendGetItemRequest`,
`SendPutItemRequest`) decompile exactly as expected (modes 7/4/5). The
byte-search turned up the two further arms Atlas already models
(`ARRANGE_ASSET`=6 via `SendSortItemRequest`, `CLOSE`=8 via `SetRet` — the
same function the registry/template already names as the op's `fname`) plus
`SendPutMoneyRequest`, a second MESO(7) send site (deposit path) sharing the
same wire mode as the withdraw path — both write `Encode1(&oPacket, 7u)`,
differing only in the sign of the `Encode4` amount that follows, consistent
with the Go `StorageOperationHandleFunc`'s single `MESO` key covering both
directions via signed amount.

All 5 v83/v84 StorageOperationHandle keys (`template_gms_83_1.json`:
RETRIEVE_ASSET=4, STORE_ASSET=5, ARRANGE_ASSET=6, MESO=7, CLOSE=8) match v95
exactly — no drift. The table is complete.

## Unresolved (Task 8)

None. Every candidate byte match was either attributed to a named,
IDA-decompiled send site or excluded with a stated reason (unrelated UI
constant, StringPool string-id constant, or non-code byte sequence). No arm
was guessed.

---

# Task 9 — `messageType` for `NPCContinueConversationHandle` (0x41) / `NPCConversation` (0x16B)

Same IDB (`ecc757f4`, `GMS_v95.0_U_DEVM.exe`, `server_health` confirms
`idb_path` ends `IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`, `imagebase
0x400000`). Unlike Tasks 7/8, this dispatch is a genuine server-parsed
`switch (nMsgType)` inside the receiving handler, not a client-constructed
mode byte — so it was read directly from `decompile 0x6de0f0`
(`CScriptMan::OnScriptMessage`, the `fname` both template entries already
carry) rather than reconstructed via `find_bytes`.

## Method

`decompile 0x6de0f0` on `CScriptMan::OnScriptMessage` (GMS v95). The function
reads `Decode1` (speakerTypeId), `Decode4` (speakerTemplateId), `Decode1`
(`nMsgType`), `Decode1` (param), then `switch (nMsgType)`, dispatching each
arm to a named `CScriptMan::OnXxx` handler:

```
case 0:  CScriptMan::OnSay
case 1:  CScriptMan::OnSayImage
case 2:  CScriptMan::OnAskYesNo(..., 0, 0)
case 3:  CScriptMan::OnAskText
case 4:  CScriptMan::OnAskNumber
case 5:  CScriptMan::OnAskMenu
case 6:  CScriptMan::OnAskQuiz       -> thunk to CWvsContext::OnInitialQuiz @0x9ffad0
case 7:  CScriptMan::OnAskSpeedQuiz  -> thunk to CWvsContext::OnInitialSpeedQuiz @0x9f1d50
case 8:  CScriptMan::OnAskAvatar          @0x6dcff0
case 9:  CScriptMan::OnAskMembershopAvatar
case 10: CScriptMan::OnAskPet             @0x6dd6e0
case 11: CScriptMan::OnAskPetAll          @0x6ddbe0
case 13: CScriptMan::OnAskYesNo(..., 0, 1)  (same helper as case 2, quest-flavor arg=1)
case 14: CScriptMan::OnAskBoxText         @0x6dc9c0
case 15: CScriptMan::OnAskSlideMenu       @0x6dbe50
default (incl. 12): m_bProcMessage=0, no dispatch — no case 12 arm exists
```

`OnAskQuiz`/`OnAskSpeedQuiz` are one-line thunks (`decompile 0x6dbaf0` /
`0x6dbb10`) that immediately call `CWvsContext::OnInitialQuiz@0x9ffad0` /
`OnInitialSpeedQuiz@0x9f1d50` — the exact addresses the pre-existing
`gms_v95` fixture markers in `conversation_test.go` cite for `AskQuiz`/
`AskSpeedQuiz`, confirming the case-6/case-7 arms are those detail structs.
All thirteen other `gms_v95 ida=` markers listed in the brief (`AskAvatar
0x6dcff0`, `AskBoxText 0x6dc9c0`, `AskMenu 0x6dce00` — cross-checked via the
switch's callee refs list which resolves `OnAskMenu` to `0x6dce00` —,
`AskNumber 0x6dcc00`, `AskPetAll 0x6ddbe0`, `AskPet 0x6dd6e0`, `AskSlideMenu
0x6dbe50`, `AskText 0x6dc790`, `AskYesNo 0x6dc5a0`, `Say 0x6dc110`, `SayImage
0x6dc310`) match the switch's own callee addresses exactly. No detail struct
is fixtured against an arm other than the one that selects it.

## v95 `messageType` table

| value | Atlas key | dispatch target |
|---|---|---|
| 0 | SAY | `CScriptMan::OnSay` @0x6dc110 |
| 1 | SAY_IMAGE | `CScriptMan::OnSayImage` @0x6dc310 |
| 2 | ASK_YES_NO | `CScriptMan::OnAskYesNo` @0x6dc5a0 |
| 3 | ASK_TEXT | `CScriptMan::OnAskText` @0x6dc790 |
| 4 | ASK_NUMBER | `CScriptMan::OnAskNumber` @0x6dcc00 |
| 5 | ASK_MENU | `CScriptMan::OnAskMenu` @0x6dce00 |
| 6 | ASK_QUIZ | `CScriptMan::OnAskQuiz` (thunk) -> `CWvsContext::OnInitialQuiz` @0x9ffad0 |
| 7 | ASK_SPEED_QUIZ | `CScriptMan::OnAskSpeedQuiz` (thunk) -> `CWvsContext::OnInitialSpeedQuiz` @0x9f1d50 |
| 8 | ASK_AVATAR | `CScriptMan::OnAskAvatar` @0x6dcff0 |
| 9 | ASK_MEMBER_SHOP_AVATAR | `CScriptMan::OnAskMembershopAvatar` |
| 10 | ASK_PET | `CScriptMan::OnAskPet` @0x6dd6e0 |
| 11 | ASK_PET_ALL | `CScriptMan::OnAskPetAll` @0x6ddbe0 |
| (12) | *(no arm — falls to default)* | n/a |
| 13 | ASK_YES_NO_QUEST | `CScriptMan::OnAskYesNo` @0x6dc5a0 (quest-flag arg = 1) |
| 14 | ASK_BOX_TEXT | `CScriptMan::OnAskBoxText` @0x6dc9c0 |
| 15 | ASK_SLIDE_MENU | `CScriptMan::OnAskSlideMenu` @0x6dbe50 |

## Findings vs. v83

`template_gms_83_1.json`'s 14-key `messageType` block has **no `SAY_IMAGE`
key** and values `SAY=0, ASK_YES_NO=1, ASK_TEXT=2, ASK_NUMBER=3, ASK_MENU=4,
ASK_QUIZ=5, ASK_SPEED_QUIZ=6, ASK_AVATAR=7, ASK_MEMBER_SHOP_AVATAR=8,
ASK_PET=9, ASK_PET_ALL=10, ASK_YES_NO_QUEST=12, ASK_BOX_TEXT=13,
ASK_SLIDE_MENU=14` (gap at value 11, skipped exactly like v95's gap at 12).

v95 inserts a real `SAY_IMAGE` arm (case 1, `CScriptMan::OnSayImage`
@0x6dc110 area) that v83's key set lacks, shifting every subsequent value up
by exactly one relative to v83 while preserving the same internal gap
(v83 skips value 11; v95 skips value 12 — one position later, consistent
with the uniform +1 shift). Per brief instruction ("add" an arm v83 lacks),
`SAY_IMAGE: 1` is added to both v95 `options.messageType` blocks. All 14 v83
keys are otherwise present in v95, each with a v83-value+1 wire value — none
happens to equal its v83 value, so there is no accidental-copy risk to flag.

No arm was guessed: every key/value pair above traces to a `switch` case in
`decompile 0x6de0f0` or a thunk one hop from it.

## Unresolved (Task 9)

None.

## Task 10: `failedReasonCodes` for `AuthPermanentBan` / `AuthTemporaryBan`

`CLogin::OnCheckPasswordResult` (`?OnCheckPasswordResult@CLogin@@IAEXAAVCInPacket@@@Z`,
`0x5dc600`, database `ecc757f4`) is the single client-side handler for all
three `LOGIN_STATUS` writer arms — `AuthLoginFailed`, `AuthPermanentBan`,
`AuthTemporaryBan` — dispatched by the `LOGIN_STATUS` op (`0x00`) that Atlas
splits into three named writers. The function decodes the reason byte first,
unconditionally, before branching:

```
v5 = CInPacket::Decode1(iPacket);      /* 0x5dc66f — the reason code, shared by all three writer arms */
nNumOfCharacter = v5;                  /* 0x5dc674 */
sMsg._m_pStr[500] = CInPacket::Decode1(iPacket); /* 0x5dc681 — second field, unrelated to reason */
CInPacket::Decode4(iPacket);            /* 0x5dc687 — unrelated */
```

`v5 == 2` (`BANNED`) is the one value that takes the "ban" branch (decodes an
8-byte unblock-date buffer via `CInPacket::DecodeBuffer(iPacket, &dtUnblockDate, 8u)`
at `0x5dc6b8`, then formats a permanent- or temporary-ban message depending on
whether the decoded date falls beyond `Util::FTAddDay(now, 1080)`). This
confirms `AuthPermanentBan` and `AuthTemporaryBan` are both driven by the same
reason byte (`BANNED = 2`) as `AuthLoginFailed`, differentiated purely by
what follows in the packet body — exactly the v83 pattern, where
`AuthPermanentBan` and `AuthLoginFailed` already carry an identical
`failedReasonCodes` block.

For every other value, execution falls to `LABEL_42` (`0x5dcb1b`) and a
`switch ( v5 )` that enumerates the remaining reason codes:

```
case -1: case 6: case 8: case 9:  -> StringPool id 15
case 2: case 3:                    -> StringPool id 16
case 4:                            -> StringPool id 3
case 5:                            -> StringPool id 20
case 7:  CLogin::GotoTitle(...)    -> StringPool id 17
case 10:                           -> StringPool id 19
case 11:                           -> StringPool id 14
case 13:                           -> StringPool id 21
case 14:  CLoginUtilDlg::YesNo2(27) popup, no CLoginUtilDlg::Error call
case 15:  CLoginUtilDlg::YesNo2(26) popup, no CLoginUtilDlg::Error call
case 16: case 21:                  -> StringPool id 33
case 17:                           -> StringPool id 27
case 25:                           -> StringPool id 40
default: break
```

(`v5 == 27` is handled even earlier, at `0x5dcb25`, before the `goto
LABEL_42`, with its own `CUtilDlg::YesNo`/`CUtilDlg::Notice` dialog pair —
StringPool ids `0x12F5`/`0x12E7`.)

After the switch, `if ( v5 && v5 != 12 && v5 != 23 ) return;` — so `v5 == 23`
is a recognized value that intentionally falls through to further UI-reset
code without an explicit `case`, and `v5 == 0` (falsy) does the same (the
"no error" / success path — not a failure reason).

**Cross-check against the value SET already in the v95 `AuthLoginFailed`
block** (its 19 keys: `BANNED=2, DELETED_OR_BLOCKED=3, INCORRECT_PASSWORD=4,
NOT_REGISTERED=5, SYSTEM_ERROR_1=6, ALREADY_LOGGED_IN=7, SYSTEM_ERROR_2=8,
SYSTEM_ERROR_3=9, TOO_MANY_CONNECTIONS=10, AGE_LIMIT=11,
UNABLE_TO_LOG_ON_AS_MASTER_AT_IP=13, WRONG_GATEWAY=14, PROCESSING_REQUEST=15,
ACCOUNT_VERIFICATION_NEEDED=16, WRONG_PERSONAL_INFORMATION=17,
ACCOUNT_VERIFICATION_NEEDED_2=21, LICENSE_AGREEMENT=23,
MAPLE_EUROPE_NOTICE=25, FULL_CLIENT_NOTICE=27`): the switch's recognized
value set is `{2,3,4,5,6,7,8,9,10,11,13,14,15,16,17,21,25}` (17 values) plus
the pre-switch special-case `27`, plus the fall-through-without-`case`
value `23` — 19 values total, an exact 1:1 match against the 19 existing
keys. **No disagreement, and no new key.** The lone switch label not covered
by an existing key is `-1` (grouped with `6, 8, 9`); `CInPacket::Decode1`
returns a byte, so `-1` cannot be produced by a real wire value from Atlas —
it reads as a sentinel/default-initialization catch-all in the client, not a
distinct reason code Atlas would ever write, so it is not added as a key.

Per the brief, the identical `failedReasonCodes` block already present on
v95 `AuthLoginFailed` was inserted verbatim into both `AuthPermanentBan` and
`AuthTemporaryBan` (`template_gms_95_1.json`), matching the v83 precedent
where `AuthPermanentBan` and `AuthLoginFailed` already share the same block.

No arm was guessed: every value above traces to a `case` label or an
explicit comparison (`v5 == 2`, `v5 == 27`, `v5 != 12`, `v5 != 23`) in
`decompile 0x5dc600`.

## Unresolved (Task 10)

None.

## Task 11 (W3d) — the movement `types` array for four v95 writers

`options.types` is an index-addressed `[]{Name, Type}` array
(`libs/atlas-packet/model/movement.go:384-405`,
`resolveMovementPathAttr` — the attribute byte indexes the array directly).
It carried **no** array at all on the four v95 writers below, so every
fragment resolved `NOT_FOUND`/`DEFAULT` — v95 movement was misparsing.

### Handlers, and confirmation they share one table

Database `ecc757f4` (`GMS_v95.0_U_DEVM.exe`, confirmed via `server_health`,
`idb_path` ends `IDBs_v9\GMS\v95_0\GMS_v95.0_U_DEVM.exe.i64`, imagebase
`0x400000` — same session used throughout this task doc).

| Writer | fname | v95 address | decompiled |
|---|---|---|---|
| `CharacterMovement` | `CUserRemote::OnMove` | `0x948a80` | 1-line thunk: `CMovePath::OnMovePacket(&m_pInterface[142], iPacket, 0)` @`0x948a97` |
| `MoveMonster` | `CMob::OnMove` | `0x6521e0` | tail call `CMovePath::OnMovePacket(&pvc->m_path, iPacket, 0)` @`0x652620` |
| `PetMovement` | `CPet::OnMove` | `0x69fb60` | `CMovePath::OnMovePacket(&m_pInterface[142], iPacket, 0)` @`0x69fb7a` (or the `0x244`-offset fallback @`0x69fb90` when `m_pInterface` is null — same callee) |
| `NPCAction` | `CNpc::OnMove` | `0x678060` | `CMovePath::OnMovePacket(&v41[145], v3, 0)` @`0x6785df`, gated on `this->m_pTemplate->bMove` |

**All four resolve to the single shared `CMovePath::OnMovePacket`
@`0x6683f0`**, which calls `CMovePath::Decode` @`0x667920` (switch
@`0x6679ce`, element-type byte read @`0x6679b8`) — the one `CVecCtrl`/
`CMovePath` attribute switch. So the four handlers share **one** 37-entry
array, not four distinct ones (controller ruling item 1's alternative branch
does not apply here).

### The array was already independently derived and shipped for the mirror direction

Before re-deriving from IDA, `template_gms_95_1.json` was checked for an
existing v95 movement `types` table, because `model.Movement` is a single
shared codec used by both directions (encode for the clientbound writers in
this task; decode for the serverbound `CharacterMoveHandle`/
`MonsterMovementHandle`/`PetMovementHandle`/`SummonMoveHandle`/
`NPCActionHandle` handlers already present in the same template) — both
directions are governed by the identical client-side `CMovePath::Encode`/
`CMovePath::Decode` switch pair, so one table is correct for both.

`CharacterMoveHandle` (opCode `0x2C`, `fname: CVecCtrlUser::EndUpdateActive`,
`template_gms_95_1.json:308-468`) already carries a 37-entry `options.types`
array, added by task-191
(`docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md`
§3.1, commit `0412e7ffb4`, "task-191: v92/v95 movement types, v88+ header,
and a permanent movement-types guard"). That derivation cites
`CMovePath::Encode` @`0x666e20` (switch @`0x666f45`) and `CMovePath::Decode`
@`0x667920` (switch @`0x6679ce`) — the exact same `Decode` address this
task's four handlers thunk to.

This task independently re-decompiled `CMovePath::Decode` @`0x667920` (see
below) rather than trusting the citation blind, and the re-derivation
produced the identical 37-entry array with no divergence. The array below is
therefore the SAME array already shipped for `CharacterMoveHandle` /
`MonsterMovementHandle` / `PetMovementHandle` / `SummonMoveHandle` /
`NPCActionHandle`, applied to the four clientbound writers this task's brief
names. `libs/atlas-packet/test/movement_types.go`'s `MovementTypesV95()`
returns this same array so both the serverbound-handler side (template only)
and the clientbound-writer side (template + Go fixtures) stay pinned to one
literal.

### Independent re-derivation of `CMovePath::Decode` @`0x667920`

Decompiled in full (`decompile addr=0x667920`). The element-type byte is
read at `0x6679b8` and switched on at `0x6679ce`:

| Group | Decode arm (first wire-touching instruction per field) | Cases |
|---|---|---|
| NORMAL | x `0x6679de`, y `0x6679e9`, vx `0x6679f4`, vy `0x6679ff`, fh `0x667a03`; `if (nAttr == 12)` `0x667a17` → fhFallStart `0x667a20`; xOffset `0x667a2d`, yOffset `0x667a36` | 0, 5, 0xC, 0xE, 0x23, 0x24 |
| JUMP | vx `0x667ae8`, vy `0x667afa` (via shared `LABEL_16`; x/y set to the running prior values, fh left `0`) | 1, 2, 0xD, 0x10, 0x12, 0x1F, 0x20, 0x21, 0x22 |
| TELEPORT | x `0x667b2b`, y `0x667b36`, fh `0x667b3a` (vx/vy forced `0`) | 3, 4, 6, 7, 8, 0xA |
| STAT_CHANGE | bStat (int8) `0x667bb4`, then `goto LABEL_10` `0x667bde` — **skips** the shared `bMoveAction`/`tElapse` trailer (`0x667a3a`/`0x667a4b`) | 9 |
| START_FALL_DOWN | vx `0x667b73`, vy `0x667b7e`, fhFallStart `0x667b87` (x/y = running prior values, fh `0`) | 0xB |
| FLYING_BLOCK | x `0x667b99`, y `0x667ba2`, then shared `LABEL_16` vx/vy (fh left `0`) | 0x11 |
| DEFAULT | explicit no-read arm @`0x667b0d` for `0x14`-`0x1E` (copies the running prior x/y/vx/vy, no packet read); bare `default: break` for `0xF`/`0x13` (also no packet read) | 0xF, 0x13, 0x14-0x1E |

Every case value except `9` also reads the shared trailer `bMoveAction`
(int8, `0x667a3a`) + `tElapse` (int16, `0x667a4b`) after the switch. Highest
case is `0x24` (36) → array length **37**. This is a byte-for-byte match
against `libs/atlas-packet/model/movement.go`'s six element kinds:
`NormalElement` (x,y,vx,vy,fh[,fhFallStart if Name=="FALL_DOWN"][,xOffset,
yOffset on GMS v87+]), `JumpElement` (vx,vy), `TeleportElement` (x,y,fh),
`StatChangeElement` (bStat only, no trailer), `StartFallDownElement`
(vx,vy,fhFallStart), `FlyingBlockElement` (x,y,vx,vy), and the bare
`Element` fallback (no extra fields) used when `Type` is `DEFAULT` or absent
from the switch — including index 12's `nAttr == 12` special case, which
matches `NormalElement.Decode`'s own `Name == "FALL_DOWN"` check exactly.

### Naming policy

The v95 IDB carries no string/enum labels for individual `nAttr` values (no
symbol table entry resolves them; `type_query` for `CMovePath::ELEM`-adjacent
enums returned no match), so per "never invent a name," every index gets
`Name: "UNKNOWN"` **except**:

- index 0 = `NORMAL` — the pre-existing, version-invariant convention already
  used by `normalTypesOptions()` (`libs/atlas-packet/character/clientbound/
  movement_test.go`) for every client version v61 through v95; not invented
  by this task.
- index 12 = `FALL_DOWN` — functionally required: `NormalElement.Decode`
  checks `isMovementName(..., "FALL_DOWN")` to decide whether to read
  `FhFallStart`, and this is the ONLY index where the client's own switch
  special-cases the read (`if (nAttr == 12)` @`0x667a17`/`0x667a20`), so the
  Name and the IDA arm agree independently.
- index 9 = `STAT_CHANGE`, index 11 = `START_FALL_DOWN`, index 17 =
  `FLYING_BLOCK` — each is the ONLY index carrying its `Type`, so the Name
  restates the Type rather than inventing a distinct label (matching the
  existing `CharacterMoveHandle` table's naming choice for the same
  single-index groups).

No v83 index was copied (v83's `CharacterMovement` table at
`template_gms_83_1.json:3656-3749` is a 23-entry vocabulary reference only,
per the brief) — no name or index position from it was reused. Group *sizes*
happening to differ between v83 (23 entries) and v95 (37 entries) confirms
they are genuinely different tables, not a renumbering of the same one.

### 3.1 v95 per-index table (indices 0-36)

| # | `Name` | `Type` | field sequence | cite |
|---|---|---|---|---|
| 0 | `NORMAL` | `NORMAL` | x,y,vx,vy,fh,xOffset,yOffset | `0x6679de` |
| 1 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 2 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 3 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 4 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 5 | `UNKNOWN` | `NORMAL` | x,y,vx,vy,fh,xOffset,yOffset | `0x6679de` |
| 6 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 7 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 8 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 9 | `STAT_CHANGE` | `STAT_CHANGE` | bStat (int8), no trailer | `0x667bb4` |
| 10 | `UNKNOWN` | `TELEPORT` | x,y,fh | `0x667b2b` |
| 11 | `START_FALL_DOWN` | `START_FALL_DOWN` | vx,vy,fhFallStart | `0x667b73` |
| 12 | `FALL_DOWN` | `NORMAL` | x,y,vx,vy,fh,**fhFallStart**,xOffset,yOffset | `0x667a17`/`0x667a20` |
| 13 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 14 | `UNKNOWN` | `NORMAL` | x,y,vx,vy,fh,xOffset,yOffset | `0x6679de` |
| 15 | `UNKNOWN` | `DEFAULT` | (none) | `0x6679ce` default |
| 16 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 17 | `FLYING_BLOCK` | `FLYING_BLOCK` | x,y,vx,vy | `0x667b99` |
| 18 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 19 | `UNKNOWN` | `DEFAULT` | (none) | `0x6679ce` default |
| 20 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 21 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 22 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 23 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 24 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 25 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 26 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 27 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 28 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 29 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 30 | `UNKNOWN` | `DEFAULT` | (none, copies prior) | `0x667b0d` |
| 31 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 32 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 33 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 34 | `UNKNOWN` | `JUMP` | vx,vy | `0x667ae8` |
| 35 | `UNKNOWN` | `NORMAL` | x,y,vx,vy,fh,xOffset,yOffset | `0x6679de` |
| 36 | `UNKNOWN` | `NORMAL` | x,y,vx,vy,fh,xOffset,yOffset | `0x6679de` |

Group totals: NORMAL 6, JUMP 9, TELEPORT 6, STAT_CHANGE 1, START_FALL_DOWN 1,
FLYING_BLOCK 1, DEFAULT 13 = 37.

### What changed

`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:
inserted the identical 37-entry `options.types` array (between `fname` and
`services`, no `validator` on writer entries) into the four writer entries
named in the brief — `PetMovement` (`0xC9`), `CharacterMovement` (`0xD2`),
`MoveMonster` (`0x11F`), `NPCAction` (`0x13A`) — all located by `writer`/
`fname`/`opCode`, not by line number (every previously-quoted line number in
the brief was stale, as flagged). No existing binding was touched; no
binding was added (opCode/writer/fname/services all unchanged) — only an
`options` key was inserted into each of the four existing entries.

`libs/atlas-packet/test/movement_types.go` (new): exports
`MovementTypesV95()`, the single shared-source copy of this array, per Step
3 of the brief.

`libs/atlas-packet/character/clientbound/movement_test.go`: `
normalTypesOptions()` now returns `pt.MovementTypesV95()` instead of the
one-entry `{"Name":"NORMAL","Type":"NORMAL"}` stub. Added
`TestCharacterMovementHighestIndexResolves`, asserting an element with
`ElemType: 36` (the array's highest index) round-trips as `*model.
NormalElement` with the NORMAL field set intact (X/Y/Vx/Vy/Fh/XOffset/
YOffset), not the `NOT_FOUND`/`DEFAULT` bare-`Element` fallback.

`libs/atlas-packet/monster/clientbound/movement_test.go` and
`libs/atlas-packet/pet/clientbound/movement_test.go`: `
TestMonsterMovementRoundTrip`/`TestMonsterMovementRoundTripWithSkill`/
`TestPetMovementRoundTrip` now pass a one-NORMAL-element `model.Movement`
(index 0, version-invariant) plus `test.MovementTypesV95()` as options,
instead of an empty `model.Movement{}` with `nil` options — the previous
form never exercised a `types` table lookup at all (no elements to iterate).
Added `TestMonsterMovementHighestIndexResolves` and
`TestPetMovementHighestIndexResolves`, the same highest-index assertion as
the character package.

No existing v95 byte-golden expectation
(`TestCharacterMovementByteOutput`/`TestMonsterMovementBytesV79`/
`TestMonsterMovementBytesV72`/`TestPetMovementBytesV72`/
`TestPetMovementBytesV79`) changed — all pass unmodified with the real
table; none of those tests decode/encode a movement element, so feeding the
real `types` table changed no golden byte (Step 3's stop condition did not
trigger).

## Unresolved (Task 11)

None.
