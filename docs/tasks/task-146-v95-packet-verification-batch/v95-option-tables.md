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

## BUDDY_OPERATION — opCode 0x99 (153), handler `BuddyOperationHandle`

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

All 4 v83 BUDDY_OPERATION keys (RELOAD, ADD, ACCEPT, DELETE) are accounted
for on v95 with identical mode values to v83; the table is complete.

## Unresolved

None. Every candidate byte match was either attributed to a named,
IDA-decompiled send site or excluded with a stated reason (unrelated UI
constant, clientbound receiver, or non-code byte sequence). No arm was
guessed.
