# Pending IDA function exports

This list tracks IDA functions referenced by the login-domain audit matrix
(task-027) but NOT yet in `gms_v95.json`. Each row needs a future maintainer
run of `packet-audit export ...` (live IDA-MCP) or hand-derivation from a
focused spike doc to add the function's wire-layout.

## Resolved (now in gms_v95.json)

| FName | Atlas writer/handler | Verdict |
|---|---|---|
| `CLogin::OnCheckPasswordResult` (success) | AuthSuccess | ✅ (v95 field-7 width fix shipped) |
| `CLogin::OnCheckPasswordResult#AuthLoginFailed` (synthetic) | AuthLoginFailed | ✅ |
| `CLogin::OnCheckPasswordResult#AuthTemporaryBan` (synthetic) | AuthTemporaryBan | ✅ |
| `CLogin::OnCheckPasswordResult#AuthPermanentBan` (synthetic) | AuthPermanentBan | ✅ (v95 trailing-bytes fix shipped) |
| `CLogin::OnSetAccountResult` | SetAccountResult | ✅ |
| `CLogin::OnCheckPinCodeResult` | PinOperation | ✅ |
| `CLogin::OnUpdatePinCodeResult` | PinUpdate | ✅ |
| `CLogin::OnLatestConnectedWorld` | SelectWorld | ✅ |
| `CLogin::OnRecommendWorldMessage` | ServerListRecommendations | 🔍 (sub-struct loop) |
| `CLogin::OnSelectWorldResult` | CharacterList | 🔍 (sub-struct CharacterListEntry) |
| `CLogin::OnWorldInformation` | ServerListEntry | 🔍 (sub-struct ChannelLoad loop) |
| `CLogin::OnSelectCharacterResult` | ServerIP | ✅ |
| `CLogin::SendCheckPasswordPacket` | Request (LoginHandle) | ✅ |
| `CLogin::SendSelectCharPacket` | CharacterSelect | ✅ |
| `CLogin::SendCheckUserLimitPacket` | ServerStatusRequest | ✅ (v95 width fix shipped) |
| `CLogin::SendViewAllCharPacket` | AllCharacterListRequest | ✅ |
| `CLogin::OnAcceptLicense` | AcceptTos (account/serverbound) | ✅ |

**17 packets audited, 14 ✅ / 3 🔍 / 0 ❌.**

## Still pending — login domain

| FName / Symbol | Atlas writer/handler | Notes |
|---|---|---|
| `CLogin::OnViewAllCharResult` (0x5de120, size 0x521) | AllCharacterListPong | Medium-complex; involves CharacterListEntry sub-struct. Phase 2 (analyzer descent) needed for high-fidelity audit. |
| `CLogin::SendSelectCharPacketByVAC` (0x5d7550, size 0x669) | CharacterSelectWithPic / *Register? | VAC-variant of char select. Large function; needs careful branch analysis. |
| `CLogin::OnSelectCharacterByVACResult` (0x5de670, size 0x375) | PicResult? | VAC result packet. |
| `CLogin::OnDenyLicense` (0x5d45d0) | — | Client-side function; constructs an outbound deny packet. |
| `CLicenseDlg::OnButtonClicked` (0x5ff870) | (UI callback) | Drives OnAcceptLicense / OnDenyLicense; not directly a wire format. |
| `LoginAuth` (atlas writer) | — | Orphan: atlas writes `WriteAsciiString(screen)`. No IDA function found by direct search. May be a legacy v83 packet that v95 client no longer reads. |

## Out of scope for GMS v95 audit (cross-region or cross-version)

These atlas writers/handlers exist in the codebase but the GMS v95 client
doesn't exercise them. The audit pipeline correctly produces no report
because there's no v95 IDA function to compare against:

- `LoginAuth` (clientbound, writes 1 string) — **JMS v1.85 only**. Whether
  GMS ever produces it is unconfirmed. Not in the gms_95 template.
- `ServerLoad` (clientbound, writes 1 byte) — **GMS v12 (or earlier) only**.
  Not in the gms_95 template.
- `ServerSelect` (serverbound, reads 1 byte worldId) — **GMS v12 (or earlier)
  only**. v95 uses `WorldCharacterListRequest` instead. Not in the gms_95
  template; the `WorldSelectHandle` symbol is dead in v95.
- `PicResult` (clientbound, opcode 0x1C, writes 1 byte) — semantically tied
  to `CLogin::SendSelectCharPacket` (the PIC-register branch's reply).
  Opcode 0x1C is not handled by `CLogin::OnPacket` directly in v95; receipt
  is routed through a different state machine, so the audit pipeline's
  CLogin-based dispatch model can't reach it. Wire shape (1 byte) is
  trivial enough that a manual cross-check confirms ✅.

## Still pending — handlers without an IDA mapping

Atlas writers/handlers under `libs/atlas-packet/login/` whose corresponding IDA
function hasn't been identified yet. Each likely corresponds to a
`CLogin::Send*` outbound packet constructor or a `CLogin::On*` inbound result:

- `AfterLoginHandle` (opcode 0x09) — atlas decodes `byte pinMode, optional (byte opt2, string pin)`
- `RegisterPinHandle` (opcode 0x0A)
- `CheckPicHandle`, `RegisterPicHandle`, `CharacterSelectedPicHandle`, `CharacterListSelectHandle`, `CharacterListSelectWithPicHandle` (PIC family, opcodes 0x15–0x1E)
- `SetGenderHandle` (opcode 0x08) — likely `CLogin::SendSetGenderPacket`
- `WorldCharacterListRequest` (opcode 0x05) — likely `CLogin::SendSelectWorldPacket` or similar
- `ServerStatus` (clientbound) — likely sent by `CLogin::OnCheckUserLimit`?
- `ServerLoad` (clientbound)
- `ServerListEnd` (clientbound, opcode 0x0A end-of-list sentinel inside ServerListEntry) — already audited as part of ServerListEntry's dispatch byte
- `PicResult` (clientbound)

## Known false positives in current audit output

`CharacterList.md` (verdict ❌): the per-entry trailer reports a 1-byte
over-count from row 45 onward. Static analysis collects all conditional
branches' calls (viewAll byte + gm byte + world-rank-enabled byte = 3
bytes), but at runtime only 2 fire: either {viewAll=0, gm=0} → 2 bytes
total (gm path returns early) or {viewAll=0, rank-enabled=1} → 1+16 = 17
+1 = 18 bytes total. v95 reads 2 bytes (onFamily + hasRank) + optional 16
bytes — matches both runtime paths. The pipeline doesn't model
early-return blocks as exclusive, so the audit over-counts. Resolution
would require an analyzer extension that flags `return` statements inside
guarded blocks; deferred to a follow-up.

## Cosmetic / cross-version concerns (not v95-specific bugs)

- `ServerIP.codes.SERVER_UNDER_INSPECTION: 7` (template_gms_95_1.json) — in
  v95 IDA, value 7 in `OnSelectCharacterResult`'s v3 switch triggers
  `GotoTitle + Error(17)` which is the "already logged in" path, not
  server-inspection. The wire value 7 still produces the right behavior
  (kick to title), but the constant name is misleading. Renaming would
  require updating the Go constant in `services/atlas-login/atlas.com/login/socket/writer/server_ip.go`
  AND all version templates (v83/v87/v92/v95/v111/JMS) that share this
  key. Left as-is for now to avoid cross-version breakage.

## Sub-op enum drift — character domain

The following character-domain packets dispatch on a leading mode/sub-op byte
inside the packet body. The audit pipeline models a single flat sequence of
Decode calls and cannot represent a switch-on-mode dispatch tree. Each row
below was filed as ❌ by the pipeline; the real issue is sub-op enum drift
that the pipeline cannot verify.

| FName | Atlas writer structs | Notes |
|---|---|---|
| `CUser::OnEffect` | `EffectSimple`, `EffectSkillAffected`, `EffectPet`, `EffectWithId`, `EffectWithMessage`, `EffectProtectOnDie`, `EffectIncDecHP`, `EffectShowInfo`, `EffectLotteryUse`, `EffectItemMaker`, `EffectUpgradeTomb`, `EffectIncubatorUse` (all in effect.go) | 16+ sub-op modes (case 0–15+). Atlas models each mode as a separate struct. All use opcode 0xE0 (foreign) or 0xE9 (self). Pipeline can only see the outermost Decode1 (mode byte). Sub-op byte values need per-mode verification. |
| `CUser::OnEffect` | `EffectQuest`, `EffectQuestForeign` (effect_quest.go) | Mode byte = quest-effect sub-op. Same pipeline limitation. |
| `CUser::OnEffect` | `EffectSkillUse`, `EffectSkillUseForeign` (effect_skill_use.go) | Mode byte = skill-use sub-op (mode 1 in GMS). Berserk/DragonFury/MonsterMagnet branches also conditional on skill ID. |

Resolution: Phase 3 — per-mode IDA sub-function trace for each atlas Effect
struct. Each mode constant maps to a specific IDA case-arm; wire format per
arm needs to be exported and compared against the corresponding struct's
Encode method.

## Still pending — character domain

| FName | Atlas writer/handler | Notes |
|---|---|---|
| (bare-handler) | `CharacterSkillChange` (opcode 0x23) | Already in gms_v95.json. Audit reports ❌ due to tool-limitation in nested `SecondaryStat` sub-struct analysis. See CharacterSkillChange.md ack footer. Deferred to Phase 3 analyzer descent. |

## Known false positives — character misc-state bucket (Task 10)

### CharacterSitResult.md (verdict ❌)

Row 2 shows an extra byte not consumed by the client. The analyzer flattens both
branches of the `if m.sitting { WriteByte(1)+WriteShort } else { WriteByte(0) }`
into a merged call list, treating the else-branch `WriteByte(0)` as a 3rd sequential
write that appears after the if-branch writes. At runtime only one branch fires:
either `byte(1)+short(chairId)` or `byte(0)`. IDA `CUserLocal::OnSitResult`
(case 231 = 0xE7 in `CUserLocal::OnPacket`) reads `Decode1` then conditionally
`Decode2` — exactly matching the atlas encoder. The ❌ verdict is a branch-flattening
false positive; no wire bug present.

Resolution: analyzer needs to detect exclusive if/else branches and not union their writes.
Deferred to Phase 3 analyzer enhancement.

### CharacterInfo.md (verdict ❌)

Rows 9–22 show multiple width mismatches and extra bytes. `CWvsContext::OnCharacterInfo`
(case 61 = 0x3D in `CWvsContext::OnPacket`) is a complex packet with:
- A bool-terminated pet list (SetMultiPetInfo do-while loop)
- An optional taming mob block (if-guarded)
- A wishList loop (count + N × int32)
- Version-guarded monster book block (GMS < 87 only; absent in v95)
- MedalAchievementInfo sub-struct (Decode4 + Decode2 + optional loop)
- A chair list block (Decode4 count + DecodeBuffer array)

The flat analyzer cannot track loop state, conditional loops, or the version guard
producing the correct sub-sequence for v95. Cross-checking the atlas encoder against
the IDA manually confirms the encoding is correct for v95:
- No monster book block (GMS v95 ≥ 87 → guard false)
- MedalAchievementInfo: WriteInt(medalId) + WriteShort(0) = Decode4 + Decode2 ✅
- Chair list: WriteInt(0) count + no items = Decode4(0) + no buffer ✅

The ❌ verdict is a multi-cause tool limitation (loop linearization, conditional sub-struct
expansion, version guard interaction). No wire bug present.

Resolution: Phase 3 sub-struct descent + loop-aware analyzer.

## Known false positives — character spawn/list bucket (Task 9)

### AddCharacterEntry.md (verdict ❌)

Rows 42–47 show extra atlas bytes (viewAll placeholder + rankEnabled + 4 × rank int32) not
consumed by the client. `CLogin::OnCreateNewCharacterResult` reads only GW_CharacterStat +
AvatarLook; rank data is zero-filled from client state. MapleStory packets are length-prefixed;
the client silently ignores trailing bytes in standalone packets, so no wire corruption occurs.
The analyzer correctly identifies these 18 extra bytes but they are functionally harmless.
Resolution: dedicated non-rank payload type for AddCharacterEntry or context-aware CharacterListEntry
encoder — deferred to follow-up refactor.

### CharacterViewAllCharacters.md (verdict ❌)

Rows 45–50 show DecodeBuf vs 4 × int32 representation mismatch for rank fields, plus
linearization offset shifting the PIC byte. IDA reads rank as `DecodeBuffer(0x10)` (bulk 16
bytes). Atlas emits 4 × `WriteInt`. Wire bytes are identical. Resolution: diff tool DecodeBuf
expansion — deferred to Phase 3 analyzer enhancement.

## Workflow notes

Refresh procedure:
1. `mcp__ida-pro__list_functions_filter` with a partial name to find the IDA FName (mangled symbols are common; use plain prefix like "SelectChar")
2. `mcp__ida-pro__get_function_by_name` (resolve address)
3. `mcp__ida-pro__decompile_function` (extract C source)
4. Parse the `CInPacket::DecodeN` / `COutPacket::EncodeN` call sequence in lexical order (success path only; multi-branch functions need manual filtering)
5. Add the entry to `gms_v95.json` and the `candidatesFromFName` map in `tools/packet-audit/cmd/run.go`
6. Regenerate audit: `cd tools/packet-audit && go run . --csv-clientbound ... --csv-serverbound ... --template ... --atlas-packet ../../libs/atlas-packet --ida-source ../../docs/packets/ida-exports/gms_v95.json --output ../../docs/packets/audits`

The synthetic-FName scheme (e.g., `CLogin::OnCheckPasswordResult#AuthLoginFailed`)
lets one IDA function model multiple sub-branches when atlas has separate
writers for different result codes.
