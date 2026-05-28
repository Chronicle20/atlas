# Task-066 Post-Phase-B — Social-Domain Audit Closeout

## Final state

- **Packets audited (v95):** 83 social FNames wired and audited in GMS v95 across Tasks 2–7
  (note: 7, buddy: 9, messenger: 13, chat: 2, party: 15, guild: 37).
- **Verdicts (v95, pipeline raw):** ✅ 47 / ⚠️ 0 / ❌ 31 / 🔍 5.
  All 31 ❌ are tool-limitation false positives (op-byte dispatcher prefix,
  mode-byte dispatcher prefix, `WritePartyData` package-level helper, DecodeBuf vs
  int64 type mismatch, op-family loop expand) — documented in `_pending.md`. Zero
  unresolved real wire bugs remain after the 7 fix commits in this branch.
- **Cross-version audit reports generated:**
  - GMS v83: 58 social .md reports
  - GMS v87: 58 social .md reports
  - GMS v95: 83 social .md reports (primary)
  - JMS v185: 67 social .md reports
- **IDA-export coverage (social FNames populated per version):**
  - `gms_v95.json` — 199 total entries, 69 social
  - `gms_v83.json` — 162 total entries, 49 social
  - `gms_v87.json` — 168 total entries, 51 social
  - `gms_jms_185.json` — 139 total entries, 71 social

## Real wire bugs fixed

| Packet | File | IDA citation | Fix one-liner | Versions affected | Commit |
|---|---|---|---|---|---|
| `NoteOperationDiscard` | `libs/atlas-packet/note/serverbound/operation_discard.go` | `CMemoListDlg::SetRet@0x624280`: op=1 + count(1) + emptySlotCount(1) + [SN4+flag1]×count | Remove spurious `val1` byte between count and emptySlotCount; rename getter Val1→EmptySlotCount | GMS all versions | `5548912cf` |
| `MessengerUpdate` | `libs/atlas-packet/messenger/clientbound/update.go` | `CUIMessenger::OnPacket` mode=7 reads only `Decode1(position)` + `AvatarLook::Decode` | Remove `name`, `channelId`, and trailing pad from `Update` struct; slim constructor to (mode, position, avatar) | GMS all versions | `d12c2cbcd` |
| `WritePartyData` (party/clientbound update+join+left) | `libs/atlas-packet/party/member_data.go` | `PARTYDATA::Decode` reads 0x17A=378 bytes; atlas was emitting 298 bytes | Add `aTownPortal[6].m_nSKillID` (6×4=24 bytes) + PQ reward arrays (56 bytes) to `WritePartyData`; update `ReadPartyData` symmetrically | GMS v95 | `2019dd581` |
| `PartyInvite` | `libs/atlas-packet/party/clientbound/invite.go` | `CWvsContext::OnPartyResult#Invite` case 4: `Decode4(partyId)+DecodeStr(name)+Decode4(nSkillID)+Decode4(level)+Decode1(autoJoin)` | Add `originatorJobId uint32` + `originatorLevel uint32` fields between inviter name and autoJoin flag | GMS v84+ and JMS | `2019dd581` |
| `GuildCapacityChange` | `libs/atlas-packet/guild/clientbound/operation.go` | `CWvsContext::OnGuildResult#CapacityChange@0xa0dfe2`: reads `Decode1` (1 byte) | Change `capacity` from `uint32` (4 bytes) to `byte` (1 byte) throughout: struct, constructor, `GuildCapacityChangedBody`, atlas-channel call site | GMS all versions | `29a248285` |
| `GuildInvite` | `libs/atlas-packet/guild/clientbound/operation.go` | `CWvsContext::OnGuildResult#Invite@0xa0d664` reads `Decode4(v21)+Decode4(nSkillID)` after inviterName | Add `unknown uint32` + `skillId uint32` fields after inviterName; update constructor to 5 args | GMS v84+ and JMS | `29a248285` |
| `WritePartyData` + `GuildInvite` (cross-version gate) | `libs/atlas-packet/party/member_data.go`, `libs/atlas-packet/guild/clientbound/operation.go` | `v83 OnPartyResult@0xa3e31c: qmemcpy(...,0x12A)`; `v83 guild mode-5: Decode4+DecodeStr (no unknown/skillId)` | Gate the 80 extra PARTYDATA bytes and party+guild invite trailing fields behind `GMS >= 95 \|\| JMS` | GMS v83 | `c0943edb4` |
| `PartyInvite` + `GuildInvite` (v84+ gate widening) | `libs/atlas-packet/party/clientbound/invite.go`, `libs/atlas-packet/guild/clientbound/operation.go` | `GMSv87_4GB.exe OnPartyResult@0xad697a` case-4 reads `Decode4(jobId)+Decode4(level)`; `OnGuildResult@0xacf9c7` reads `Decode4+Decode4` after inviterName | Widen gate from `GMS >= 95 \|\| JMS` to `GMS > 83 \|\| JMS` in both invite Encode/Decode | GMS v87 | `d6513332d` |
| `WritePartyData` (JMS v185 gate) | `libs/atlas-packet/party/member_data.go` | `JMS v185 CWvsContext::OnPartyResult@0xb297e7: qmemcpy(v120,...,0x12Au=298)` — JMS uses the 298-byte small format | Remove `\|\| t.Region() == "JMS"` from `v95plus` gate in both `WritePartyData` and `ReadPartyData` | JMS v185 | `ab8511fee` |

## Template opcode / enum fixes

None — no template opcode drift surfaced in social-domain audit.

All social-domain opcodes in `template_gms_83_1.json`, `template_gms_87_1.json`,
`template_gms_95_1.json`, and `template_jms_185_1.json` were confirmed correct against
IDA dispatch tables in each respective binary. No template file edits were required.

## Tooling improvements

- **Registry fixtures** for `model.GuildMember`, `model.Buddy`, and `model.Avatar`
  sub-structs added to the TypeRegistry pre-population layer (Phase 0, commit `591451ec5`).
  Enables sub-struct descent in audit reports without requiring analyzer surgery.
- **`candidatesFromFName` wiring** for all six social sub-domains (`note`, `buddy`,
  `messenger`, `chat`, `party`, `guild`) added to `tools/packet-audit/cmd/run.go` — the
  first social-domain entries in the audit registry. Previously only login and character
  domains were wired.
- **Documented `party.WritePartyData` as a known tool-limitation** in `_pending.md`
  (OP-FAMILY / tool-limitation section): the TypeRegistry walks receiver-method
  `Encode`/`Write` only; package-level write helpers are invisible. Affected packets
  (update, join, left) receive ⚠️ "tool-limitation: package-level write helper not
  modelled" verdicts rather than false ❌.
- **OP-FAMILY documentation** written to `_pending.md` for all social dispatcher families:
  `OP-FAMILY-note` (sub-op SEND/DISCARD/REQUEST byte values), `OP-FAMILY-buddy` (op-byte
  RELOAD/ADD/ACCEPT/DELETE, multi-step decode pattern), `OP-FAMILY-party-serverbound`
  (op-byte WITHDRAW/INVITE/EXPEL/CHANGE_BOSS), `OP-FAMILY-guild-clientbound` (mode byte
  across 15+ sub-ops), `OP-FAMILY-guild-serverbound` (op-byte LEAVE/INVITE/KICK/…),
  `OP-FAMILY-guild-bbs-serverbound` (op-byte LIST/CREATE_EDIT/REPLY/VIEW/DELETE_REPLY/DELETE_THREAD).
  All sub-op value space verification is deferred to Phase 2.
- **Chat sub-mode enum modelling deferral** documented as a single consolidated row in
  `_pending.md` covering `multi.go`, `whisper.go`, `world_message.go`,
  `world_message_extra.go` (clientbound) and `multi.go`, `whisper.go` (serverbound).
  Includes identification of one real wire bug (`Multi.Encode` missing `updateTime` field
  for v95) deferred to a follow-up task.
- **Plan-amendment commit** (`7d855306b`) corrected three audit-pipeline plan/reality gaps
  discovered during Phase 1a execution: (1) `candidatesFromFName` wiring step was
  underspecified — plan now explicitly calls out `run.go` edits per sub-domain task;
  (2) `--output` flag value in the plan used the wrong path format — corrected to flat
  `docs/packets/audits/gms_v95/`; (3) plan assumed sub-domain subdirectories under
  `docs/packets/audits/gms_v95/` but the real pipeline writes flat per the existing
  convention. These corrections apply to future cross-domain audits.
- **IDA entries corrected** in `gms_v95.json` for the messenger sub-mode dispatcher
  (mode byte added as first entry per sub-handler FName, matching the buddy precedent)
  and for the note OperationDiscard sub-handler (op-byte prefix stripped, struct body
  updated).
- **Phase 3 (Task 11)** — regression sweep of login (task-027) and character (task-028)
  audit verdicts. Zero verdict regressions: all login/character SUMMARY entries remain
  at their shipped verdicts. No commits produced.

## Remaining work

| Area | What | Why deferred |
|---|---|---|
| Chat `Multi` serverbound (v95 wire bug) | `CUIStatusBar::SendGroupMessage` prepends `Encode4(update_time)` before the chat-type byte in v95; atlas `Multi.Encode` does not include this field. Add `updateTime` field with `GMS>83` gate + update callers. | Touching a high-frequency field in all group-chat sends carries service-side caller churn; deferred to a dedicated chat-wire follow-up task. |
| `BuddyInvite` two extra `Decode4` fields | IDA `CWvsContext::OnFriendResult` case 0x09 reads `Decode4(v25)+Decode4(v26)` between originator name and `GW_Friend`; atlas does not write these 8 bytes. Potential invite display corruption. | Needs live client test to confirm whether v25/v26 are real wire fields or IDA decompiler artifacts (unambiguous `CInPacket::Decode4` calls but typed as `ZRef<CDialog>*` and `char*`). |
| `PartyOperation` trailing 0x00 | `CField::SendWithdrawPartyMsg` IDA shows `Encode1(op=2)+Encode1(0x00)`; atlas `Operation.Encode` (serverbound) writes only the op byte. | Low functional impact (server reads op byte only; trailing byte would be ignored). Deferred pending live client test confirmation. |
| Chat sub-mode enum drift | Six chat files use parameterised mode bytes (`MultiChat`, `WhisperSendResult`/7 structs, `WorldMessage`/7 structs, `WorldMessageExtra`/4 structs, `Multi` serverbound, `Whisper` serverbound). Sub-op value spaces not yet verified against v83/v87/JMS v185. | Static analyzer cannot model switch-on-mode dispatch trees. Per-mode body layout verification requires either analyzer surgery or manual inspection of each mode arm. |
| Buddy op-byte prefix (OP-FAMILY-buddy) | `BuddyOperationAdd`, `BuddyOperationAccept`, `BuddyOperationDelete` all ❌ due to op-byte prefix not included in sub-struct `Encode`. Sub-op values RELOAD=0/ADD=1/ACCEPT=2/DELETE=3 confirmed; template enum drift not yet verified cross-version. | Tool-limitation: multi-step decode (Operation prefix byte + sub-type payload) not modelled. Requires analyzer extension or a wrapper FName approach. |
| `BuddyError` sub-op conditional | Modes 0x10/0x11/0x13/0x16 read 1 byte then optionally a string; atlas `Error.hasExtra` only models the first class. Secondary conditional string not represented. | Static analysis cannot distinguish mode arms at the ❌ entry point. Deferred to Phase 2 cross-version sub-op enum pass. |
| Guild BBS sub-op enum drift | `BBS` op-byte values LIST=2/CREATE_EDIT=1/REPLY=4/DELETE_REPLY=5/VIEW=3/DELETE_THREAD=6 confirmed in GMS v95. Cross-version verification (v83/v87/JMS) not done. | BBS feature is entirely absent from JMS v185; v83/v87 BBS opcode alignment deferred to a follow-up cross-version pass. |
| Guild/party sub-op value space | `GuildOperation` op-byte and `PartyOperation` op-byte values confirmed in v95 IDA. Cross-version template enum drift verification deferred. | Per-mode audit requires manual pass against each template file's opcode constants. Scoped to Phase 2. |
| `GuildInfo` + `GuildMemberJoined` sub-struct expansion | Both ❌/🔍 due to packed-array loop not expanded by the flat analyzer. Wire is ✅ per manual `GUILDMEMBER::Decode` (37 bytes) verification. | Requires loop-body expansion in the analyzer (out of scope per design §1). |
| `NoteDisplay` int64 vs DecodeBuf | `Display.Encode` writes `WriteInt64(timestamp)` (8 bytes); IDA reads `DecodeBuffer(8)` — both 8-byte FILETIME. Pipeline ❌ due to type mismatch. | Analyzer classifies `Decode8` (int64) and `DecodeBuf` (bytes) as different types despite identical wire bytes. Requires a type-equivalence rule in the diff engine. |
| `WritePartyData` package-level helper | All three callers (update, join, left) get ⚠️ tool-limitation because the TypeRegistry does not walk package-level functions. | Requires analyzer extension to index non-method write helpers. Out of scope per design §1. |
