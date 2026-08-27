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
