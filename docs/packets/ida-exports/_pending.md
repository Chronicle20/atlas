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
| `CWvsContext::OnMessage` | `StatusMessageDropPickUpInventoryFull`, `StatusMessageDropPickUpItemUnavailable`, `StatusMessageDropPickUpGameFileDamaged`, `StatusMessageDropPickUpStackableItem`, `StatusMessageDropPickUpUnStackableItem`, `StatusMessageDropLossStackableItem`, `StatusMessageDropLossUnStackableItem`, `StatusMessageDropPickUpMeso`, `StatusMessageForfeitQuestRecord`, `StatusMessageUpdateQuestRecord`, `StatusMessageCompleteQuestRecord`, `StatusMessageCashItemExpire`, `StatusMessageIncreaseExperience`, `StatusMessageIncreaseSkillPoint`, `StatusMessageIncreaseFame`, `StatusMessageIncreaseMeso`, `StatusMessageIncreaseGuildPoint`, `StatusMessageGiveBuff`, `StatusMessageGeneralItemExpire`, `StatusMessageSystemMessage`, `StatusMessageQuestRecordEx`, `StatusMessageItemProtectExpire`, `StatusMessageItemExpireReplace`, `StatusMessageSkillExpire` (all in status_message.go) | Opcode 0x26. Top-level Decode1 = mode byte (0–14); each case delegates to a sub-handler that reads mode-specific fields. Atlas has 20+ sub-op structs each writing: mode byte first, then sub-op body. Pipeline report: `StatusMessageDropPickUpInventoryFull.md` (mode=0, representative). IDA sub-handler trace per mode needed to verify sub-op body layouts. See ack footer in `StatusMessageDropPickUpInventoryFull.md`. |

Resolution: Phase 3 — per-mode IDA sub-function trace for each atlas StatusMessage
struct. Each mode constant maps to a specific IDA case-arm (OnDropPickUpMessage,
OnQuestRecordMessage, OnIncEXPMessage, etc.); wire format per arm needs to be
exported and compared against the corresponding struct's Encode method.

## Still pending — character domain

| FName | Atlas writer/handler | Notes |
|---|---|---|
| (bare-handler) | `CharacterSkillChange` (opcode 0x23) | Already in gms_v95.json. Audit reports ❌ due to tool-limitation in nested `SecondaryStat` sub-struct analysis. See CharacterSkillChange.md ack footer. Deferred to Phase 3 analyzer descent. |
| CreateCharacter (opcode 0x17 / bCharSale path) | atlas decoder absent for `m_bCharSale == true` branch in `CLogin::SendNewCharPacket@0x5d7bd0` (opcode 23, 9× AL items, no SubJob/gender). Cash Shop character creation flow not wired. | follow-up |

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

## Cross-version — character domain (v83)

Results of the GMS v83 cross-version pass (Task 15). All 44+ character FNames were
looked up in v83 IDA (base 0x400000, `MapleStory_dump.exe`).

### Missing FNames in v83 IDA

The following v95 FNames have no equivalent function in v83 IDA; the pipeline produces
no report for them. For each, the v83 behaviour is noted.

| v95 FName | v83 behaviour | Atlas struct | Notes |
|---|---|---|---|
| `CUser::OnEmotion` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xC1: reads `Decode4(emotionId)` only; calls `CAvatar::SetEmotion` directly — no separate function | `CharacterExpression` | **Fixed**: `expression.go` (clientbound) now gates `duration` + `byItemOption` on `GMS>83\|\|JMS`. v83 wire: 8 bytes (4 charId + 4 emotionId). v95 wire: 13 bytes. |
| `CUserRemote::OnSetActivePortableChair` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xC4: reads `Decode4(chairId)` directly into `RemoteUser[3567]` — no separate function | `CharacterChairShow` | Same wire shape (`characterId + chairId` = 8 bytes); no divergence. Atlas encoder correct for v83. |
| `CLogin::SendCheckDuplicateIDPacket` | In v83 this lives on `CUICharacterSaleDlg` (a UI class), not `CLogin`. Wire format `EncodeStr(name)` is identical. | `CheckName` | Audit can't match FName; no pipeline report. Wire shape unchanged — no v83 bug. |
| `CWvsContext::SendStatChangeRequest` | In v83 renamed `CWvsContext::SendStatChangeRequestByItemOption@0xa1e997`. Wire format `Encode4+Encode4+Encode2+Encode2+Encode1` is **identical** to v95. | `HealOverTime` | No divergence; audit entry added under the v95 FName key for gms_v83.json. |

### Resolved v83-only divergences (fixed in Task 15; gates updated to >87 in Task 16)

| FName | Atlas struct | v83 wire | v87 wire | v95 wire | Final gate |
|---|---|---|---|---|---|
| `CUser::ShowItemUpgradeEffect` | `ItemUpgrade` (clientbound) | `Decode1×4` (no enchantCategory, no enchantResultFlag) | `Decode1×4` (same as v83) | `Decode1×3 + Decode4 + Decode1×2` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 also has only 4 bytes |
| `CWvsContext::SendEmotionChange` | `ExpressionRequest` (serverbound) | `Encode4` (emotionId only) | `Encode4` (same as v83) | `Encode4 + Encode4 + Encode1` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 IDA@0xabbfbb |
| `CUser::OnEmotion` (absent in v83) | `CharacterExpression` (clientbound) | `Decode4` (inline in dispatcher case 0xC1) | `Decode4` (inline in case 0xCE, no separate function) | `Decode4 + Decode4 + Decode1` | `GMS>87 \|\| JMS` — widened from `>83` after Task 16 confirmed v87 IDA@0x9f7492 |

### v83 IDA structural differences not requiring encoder changes

| FName / area | Difference | Verdict |
|---|---|---|
| `CVecCtrlUser::EndUpdateActive` | v83 encodes `Encode1(fieldKey) + Encode4(crc)` only — no dr0/dr1/dr2/dr3/dwKey/crc32. v95 IDA already documented these with `GMS>83\|\|JMS` guards on dr fields. | No action — gates were already correct from v95 audit. |
| `CLogin::SendNewCharPacket` | v83 has no `Encode2(subJob)` after race index. Already gated `MajorVersion() > 83` in `create.go`. | No action — already correct. |
| `CLogin::SendDeleteCharPacket` | v83 sends `EncodeStr(deletionPwd) + Encode4(charId)` — same shape as v95. | No divergence. |
| `CFuncKeyMappedMan::OnInit` | v83 loop count is 89 entries (v95: 90). Pipeline reports ❌ for both versions (loop-count tool limitation). Atlas sends 90 × (type+id) regardless — the extra entry is harmless as the client treats it as a full keymap. | Deferred: loop-count discrepancy. No functional impact. |
| `CWvsContext::OnMessage` | v83 has 14 sub-op modes (0–0xD); v95 added mode 0xE (SkillExpire). Both versions ❌ in pipeline due to sub-op dispatch limitation. | Deferred to Phase 3 sub-op audit. |
| `GW_CharacterStat::Decode` field widths | v83: HP/MHP/MP/MMP are `Decode2` (int16); v95: widened to `Decode4` (int32). Both `CharacterList` and `CharacterViewAllCharacters` have `nSubJob` absent in v83. These are sub-struct fields inside complex packets that the flat analyzer cannot reach. | Deferred — existing `_pending.md` tool-limitation rows cover these. |

### Hard-cap gate check

No encoder/decoder in the character domain now contains more than **2 nested** `if t.Region()` / `if t.MajorVersion()` levels after this task's changes. The three fixed encoders each have a single flat gate. Hard cap not triggered.

## Cross-version — character domain (v87)

Results of the GMS v87 cross-version pass (Task 16). All 44+ character FNames were
looked up in v87 IDA (base 0x400000, `GMSv87_4GB.exe`).

### Confirmed v87 alignments (no change needed)

| FName | v87 behaviour | Notes |
|---|---|---|
| `GW_CharacterStat::Decode` HP/MHP/MP/MMP | v87: `Decode2` (int16), same as v83. Widened to `Decode4` in v95 only. Atlas currently writes int32 for all versions — this sub-struct is inside complex CharacterList packets the flat analyzer cannot reach. Deferred. | Same situation as v83; no new gate needed |
| `GW_CharacterStat::Decode` nSubJob | v87: `Decode2(nSubJob)` IS present at end of stat block. Same as v95. Gate `MajorVersion() > 83` for nSubJob already correct. | No action |
| `CFuncKeyMappedMan::OnInit` loop count | v87: loop count = **89** entries (identical to v83; v95 = 90). Deferred — pipeline cannot model loop counts; atlas always sends 90 which is harmless. | No action |
| `CWvsContext::OnMessage` sub-op modes | v87: 15 modes (0x0–0xE) including SkillExpire — same as v95. | No action |
| `CVecCtrlUser::EndUpdateActive` | v87 IDA@0xa5c937: has full dr0/dr1/fieldKey/dr2/dr3/crc/dwKey/crc32 sequence. Gate `GMS>83\|\|JMS` fires correctly for v87. | No action |
| `CLogin::OnSelectCharacterResult` | v87 success path (LABEL_48): `Decode4(ip)+Decode2(port)+Decode4(charId)+Decode1(authenCode)+Decode4(ulPremiumArgument)` — identical to v95. | No action |
| `CLogin::OnViewAllCharResult` case 0 (CharacterViewAllCharacters) | v87: reads same fields as v95 except NO `m_bLoginOpt` at end. Atlas gates `MajorVersion()>87` for this field — already correct. | No action |
| `CLogin::OnSelectWorldResult` m_nBuyCharCount | v87: absent. Atlas gates `MajorVersion()>87` for `nBuyCharCount` in `list.go` — already correct. | No action |

### Missing FNames in v87 IDA

| v95 FName | v87 behaviour | Atlas struct | Notes |
|---|---|---|---|
| `CUser::OnEmotion` | Handled inline in `CUserPool::OnUserRemotePacket@0x9f7492` case 0xCE: reads `Decode4(emotionId)` only (same as v83 case 0xC1). No duration, no byItemOption. | `CharacterExpression` | **Fixed**: gate widened to `GMS>87\|\|JMS` in Task 16. |
| `CUserRemote::OnSetActivePortableChair` | Handled inline in `CUserPool::OnUserRemotePacket` case 0xD1: reads `Decode4(chairId)` directly. Same wire shape as v95. | `CharacterChairShow` | No divergence. |

### Resolved v87-only divergences (fixed in Task 16)

| FName | Atlas struct | v87 wire | v95 wire | Fix |
|---|---|---|---|---|
| `CUser::ShowItemUpgradeEffect@0x9adb79` | `ItemUpgrade` (clientbound) | `Decode1×4` (no enchantCategory, no enchantResultFlag) | `Decode1×3+Decode4+Decode1×2` | Gate widened from `>83` to `>87` in `item_upgrade.go` |
| `CWvsContext::SendEmotionChange@0xabbfbb` | `ExpressionRequest` (serverbound) | `Encode4` (emotionId only) | `Encode4+Encode4+Encode1` | Gate widened from `>83` to `>87` in `serverbound/expression.go` |
| `CUser::OnEmotion` (inline@0x9f7492) | `CharacterExpression` (clientbound) | `Decode4` (expressionId only) | `Decode4+Decode4+Decode1` | Gate widened from `>83` to `>87` in `clientbound/expression.go` |
| `CWvsContext::OnCharacterInfo@0xabb181` | `CharacterInfo` (clientbound) | monster book block (5×int32) IS present | monster book absent (GMS≥87 guard false) | Gate changed from `< 87` to `<= 87` in `info.go` so v87 correctly includes monster book block |

### v87 IDA structural differences deferred to _pending (not fixed)

| FName | v87 difference | Atlas struct | Status |
|---|---|---|---|
| `CLogin::SendCheckPasswordPacket@0x62dfb4` | v87 appends `Encode4(PartnerCode)` after the 3×Encode1 unknowns; atlas reads only `unknown2` for `>=95` — v87 sends unknown2+PartnerCode but atlas only reads unknown1 for v87 (gate `>=95` skips unknown2 for v87). Low-severity: packet read ends cleanly since no subsequent reads follow. | `Request` | Deferred. Wire-format quirk limited to `SendCheckPasswordPacket`; functional impact is zero since atlas doesn't use PartnerCode. |
| `CLogin::SendSelectCharPacket` 0x1D/0x1E opcodes | v87 PIC-register opcode 0x1E sends `EncodeStr+Encode4+EncodeStr+EncodeStr`; v87 PIC-select opcode 0x1D sends `Encode1(1u)+Encode4+EncodeStr+EncodeStr+EncodeStr`. v95 has layouts at opcodes 0x1C/0x1D. Atlas handler–opcode mapping in v87 template assigns 0x1D→RegisterPicHandle, 0x1E→CharacterSelectedPicHandle — layouts are structurally different from the v87 wire. | `CharacterSelectRegisterPic`, `CharacterSelectWithPic` | Deferred. Requires v87-specific handler variants or opcode-keyed decode dispatch. |

### Hard-cap gate check (Task 16)

No encoder/decoder in the character domain now contains more than **2 nested** `if t.Region()` / `if t.MajorVersion()` levels after Task 16 changes. All four fixed encoders (`ItemUpgrade`, `CharacterExpression`, `ExpressionRequest`, `CharacterInfo`) have at most 2 sequential flat gates (never nested). Hard cap not triggered.

## Cross-version — character domain (JMS v185)

Results of the JMS v185 cross-version pass (Task 17). All character domain FNames were
looked up in JMS v185 IDA (base 0x400000, `MapleStory_dump_SCY.exe`, md5 af6652ff9b7c549341f35e3569d7564a).

The JMS v185 binary shares C++ mangled symbol names with GMS v95 for all character-domain
functions searched. No separate opcode space split was found for the character domain
(unlike the login domain which had distinct GMS vs JMS packet structures for
`OnCheckPasswordResult`).

### Resolved JMS divergences (fixed in Task 17; `|| JMS` clauses removed)

These gates had an incorrect `|| JMS` clause added during Task 15/16 under the assumption
that JMS v185 matched GMS v95 behaviour. JMS v185 IDA confirms it uses the older
(v83/v87-equivalent) layout for these packets.

| FName | Atlas struct | JMS v185 wire | GMS v95 wire | Fix |
|---|---|---|---|---|
| `CUser::OnEmotion@0x9f636b` | `CharacterExpression` (clientbound) | `Decode4(nEmotion)+Decode4(tDuration)` — no byItemOption | `Decode4+Decode4+Decode1` | Gate narrowed: duration emitted for JMS (Decode4), byItemOption NOT emitted for JMS. `expression.go` clientbound updated. |
| `CUser::ShowItemUpgradeEffect@0x9f1a92` | `ItemUpgrade` (clientbound) | `Decode1×5` — no Decode4(nEnchantCategory); enchantResultFlag (v6) IS present | `Decode1×3+Decode4+Decode1×2` | Gate narrowed: `|| JMS` removed from enchantCategory gate only. enchantResultFlag gate retains `|| JMS` since JMS reads Decode1(v6). `item_upgrade.go` updated. |
| `CVecCtrlUser::EndUpdateActive@0xaaa076` | `Move` (serverbound) | `Encode1(detectFlag)+[if active: Encode1(fieldKey)+Encode4(crc)+CMovePath]` — no dr0/dr1/dr2/dr3/dwKey/crc32 | Full dr-field sequence | Gate narrowed: `|| JMS` removed from all dr-field gates in `move.go`. JMS movement is GMS v83-equivalent layout. |

### Resolved JMS divergences — serverbound ExpressionRequest

| FName | Atlas struct | JMS v185 wire | GMS v95 wire | Fix |
|---|---|---|---|---|
| `CWvsContext::SendEmotionChange@0xb0b8be` | `ExpressionRequest` (serverbound) | Encodes only `Encode4(charId)` — the local user's characterId, NOT emotionId+duration+byItemOption | `Encode4(emotionId)+Encode4(duration)+Encode1(byItemOption)` | Gate narrowed: `|| JMS` removed. JMS serverbound opcode 0x2B carries only a charId. Atlas server reads the first int4 as emotionId; JMS sends charId in that slot. No duration or byItemOption for JMS. `serverbound/expression.go` updated. |

### JMS-specific structural differences (no encoder change, documented)

| FName | JMS difference | Atlas struct | Status |
|---|---|---|---|
| `CWvsContext::SendStatChangeRequestByItemOption@0xb054d6` | JMS appends `Encode4(timeGetTime())` after `Encode1(nType)` — 5 fields vs GMS v95's 5 fields (same 5 but JMS adds a 6th trailing int4). Low-severity: atlas server reads only 5 fields then stops; the trailing 4 bytes are ignored. No functional impact. | `HealOverTime` | Deferred. JMS-only trailing field; server ignores it. No encoder change needed. |
| `CWvsContext::OnCharacterInfo@0xb0aa6e` | JMS v185 INCLUDES the monster book block (`SomethingMonsterBook` call). The gate `(GMS && <=87) \|\| JMS` in `info.go` is **correct** for JMS. | `CharacterInfo` | No action — already correct. |
| `CWvsContext::SendCharacterInfoRequest@0xb0b323` | JMS wire: `Encode4(updateTime)+Encode4(dwCharacterID)+Encode1(bPetInfo)` — identical to GMS v95. | `CharacterInfoRequest` | No action — no divergence. |
| `CFuncKeyMappedMan::OnInit@0x5e79aa` | JMS function present, same structure. Loop count not easily determinable from decompile. | `FuncKeyMap` | No action — same tool-limitation as v83/v87. |
| `CUserRemote::OnAvatarModified@0xa57221` | JMS uses a *list* format for couple/friendship (Decode4(count)+loop:DecodeBuf(0x10)+Decode4(pairCharId)) vs GMS v95 which reads single-entry buffers. This is a sub-struct difference beyond the flat analyzer's scope. | `CharacterAppearanceUpdate` | Deferred to Phase 3 sub-struct descent. No wire bug in the outer packet structure. |
| `CUser::OnEmotion@0x9f636b` duration field | JMS reads Decode4(tDuration) — confirmed. Atlas now writes duration for JMS (without byItemOption). | `CharacterExpression` | Fixed — see resolved table above. |
| `CLogin::OnCheckPasswordResult@0x66e79f` | JMS v185 success path decodes differently: `Decode4(accountId)+Decode1(gender)+Decode1(gradeCode)+Decode1(combined)+2×DecodeStr(nexon IDs)+5×Decode1+DecodeBuffer(8)+DecodeStr`. Fundamentally different structure from GMS v95. Atlas server only needs the pre-shared accountId for login; login domain is tracked separately in task-027. | `AuthSuccess` (login domain) | Out of scope for character domain audit. Login domain audit (task-027) tracks this separately. |

### Deferred / known limitations — JMS v185

| Issue | Details |
|---|---|
| ExpressionRequest (sb) JMS semantic mismatch | JMS opcode 0x2B carries only charId; atlas's `Decode` reads it as `emote`. Re-broadcast CharacterExpression carries the JMS charId in the expression slot. Pre-existing on `main` — not introduced by task-028. Follow-up: dedicated JMS-aware decoder. |

### Hard-cap gate check (Task 17)

After Task 17 changes, no encoder/decoder in the character domain contains more than **2 nested** gates. The three fixed encoders each have flat sequential gates — `CharacterExpression` now has one `if GMS>87` + one `else if JMS`, `ItemUpgrade` has a single `if GMS>87`, `Move` has three sequential `if GMS>83` + one `if GMS>28`. No nested gates. Hard cap not triggered.


## Still pending — combat domain (monster)

Phase 2a (task-065) audit of 9 monster clientbound packets in GMS v95. ✅ 3 / ❌ 5 / 🔍 1.

| FName | Atlas writer | Verdict | Notes |
|---|---|---|---|
| `CMobPool::OnMobEnterField@0x6589e0` | MonsterSpawn | ❌ | **Analyzer FP (design §3).** Atlas`s `if (region/version) { if controlled then WriteByte(1) else WriteByte(5) }` if/else expands into two consecutive WriteByte entries in the flat call list, throwing off positions 2+. Plus the `m.monster.Encode` MonsterModel sub-struct cannot be fully resolved because the registry keys on unqualified struct names and there are 4 `Spawn` structs across monster/drop/reactor/pet (last-write-wins in `r.types`). Manual IDA confirms wire is ✅. Defer until registry handles qualified type names. |
| `CMobPool::OnMobLeaveField@0x658b90` | MonsterDestroy | ❌ (real) | Atlas missing optional `WriteInt(swallowCharacterId)` when destroyType == 4 (swallowed by character-eater mob like Yeti-and-Pepe). Real wire bug; narrow scope (swallow eaters only). Constructor signature change `NewMonsterDestroy` affects callers in `services/atlas-channel`. Defer to a follow-up that adds the field + updates call sites. |
| `CMobPool::OnMobChangeController@0x658d10` | MonsterControl | ❌ (real, large) | Atlas wire shape fundamentally differs from v95. Atlas writes `int8 controlType + int32 uniqueId + (if type>0: byte(5) + int32 monsterId + MonsterModel)`. v95 reads `byte controlMode + (if controlMode && opt: int32×3 seed) + int32 mobId + (if controlMode: byte aggro)`. Looks like atlas implements an older-protocol shape; v95 controllers carry a movement-seed instead of MonsterModel. Defer to follow-up — needs cross-version IDA pass (v83/v87) to understand when the shape changed. |
| `CMob::OnMove@0x6521e0` | MonsterMovement | 🔍 | Mostly analyzer FP: sub-struct expansion of `MultiTargetForBall`, `RandTimeForAreaAttack`, and `Movement` is incomplete due to registry struct-name collision. The skill block `(skillId, skillLevel)` is gated `GMS>83 || JMS` in atlas but is written as `Decode4 sEffect.m_Data` (packed) in v95 IDA, vs atlas writing `Decode2 skillId + Decode2 skillLevel` separately — same 4 bytes, different field decomposition. May be ✅ on wire bytes; defer for now. |
| `CMob::OnCtrlAck@0x640c50` | MonsterMovementAck | ✅ | Wire shape matches. uniqueId + moveId(int16) + useSkills(byte) + mp(int16) + skillId(byte) + skillLevel(byte). |
| `CMob::OnStatSet@0x652660` | MonsterStatSet | ❌ | **Analyzer FP.** Atlas writes `uniqueId + MonsterTemporaryStat.Encode(mask + per-bit data) + int16(tDelay=0) + byte(nCalcDamageStatIndex=0) + optional byte(bStat)`. v95 OnStatSet top-level reads `mobId + DecodeBuffer(0x10) mask + delegate ProcessStatSet`. The post-mask trailing fields (tDelay/calcIndex/bStat) live inside `CMob::ProcessStatSet` which the audit pipeline cannot descend into. Wire bytes likely ✅. Defer pending ProcessStatSet decompile. |
| `CMob::OnStatReset@0x652780` | MonsterStatReset | ❌ | Same analyzer FP as StatSet. |
| `CMob::OnDamaged@0x64ecb0` | MonsterDamage | ✅ | Wire shape matches. uniqueId + damageType + damage + (conditional hp/maxHp for bDamagedByMob). |
| `CMob::OnHPIndicator@0x642ef0` | MonsterHealth | ✅ | Wire shape matches. uniqueId + hpPercent. |
| `CMob::GenerateMovePath@???` | MonsterMovementHandle (sb) | (deferred) | Single packet not audited in this PR. `CMob::GenerateMovePath` is a 4 KB+ encode-side function that requires dedicated decompile + transcription. Atlas's `MonsterMovementHandle` serverbound decoder in `libs/atlas-packet/monster/serverbound/movement.go` remains unverified against v95 / v83 / v87 / JMS-v185. Follow-up: populate IDA exports for all 4 versions with `CMob::GenerateMovePath` entries. |

### Audit-tool follow-ups suggested by combat domain

- Registry should track qualified struct names (e.g. `monster/clientbound.Spawn`) so cross-sub-domain struct name collisions do not lose field-type info needed by `resolveRecurse`. The combat sub-domains all use unqualified names (Spawn/Destroy/Damage/Hit/Movement) which collide with each other and with `pet/serverbound.Spawn`.
- Analyzer could detect mutually-exclusive `if/else` writes and treat them as a single position so MonsterSpawn does not show two consecutive WriteByte entries in the flat list.
- Sub-domain pet/drop/reactor audit (Phase 2b/c/d in plan.md) is deferred; monster-only is the scope of this PR per session decision.

### Hard-cap gate check — combat domain

No combat encoder has 3+ nested region/version guards. monster/movement.go has two sequential `if (GMS>83 || JMS)` blocks (not nested). monster/spawn.go has one `(GMS>12 || JMS)` block. No hard cap triggered.


## Still pending — combat domain (pet)

Phase 2b (task-065) audit of 14 pet packets in GMS v95. ✅ 4 / ❌ 10.

Pet sub-domain shares the same analyzer-FP pattern as monster — `DecodeBuf`/`EncodeBuf` placeholders in the IDA JSON don't expand atlas's full encode call list, and `model.Movement`/`Activated` sub-struct expansion fails under the registry struct-name collision (4 `Spawn`, 4 `Destroy`, 4 `Movement` types collide across monster/drop/reactor/pet, last-write-wins in `r.types`). For most ❌ entries below, the prefix fields (characterId, slot, active, count) align ✅ — the divergence begins inside the body sub-struct.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CUserRemote::OnPetActivated@0x9547d0` | PetActivated | ❌ | Prefix (characterId+slot+active+show) ✅. Atlas writes `templateId+name+petId+x+y+stance+foothold+nameTag+chatBalloon` for active path, `despawnMode` for inactive — the IDA `DecodeBuf` placeholder for CPet::Init body doesn't expand. Wire likely ✅. |
| `CPet::OnMove@0x69fb60` | PetMovement | ❌ | Prefix (characterId+slot) ✅. Body diverges due to Movement sub-struct expansion gap. Wire likely ✅. |
| `CPet::OnAction@0x6a3860` | PetChat | ✅ | Wire matches. |
| `CPet::OnActionCommand@0x6a3930` | PetCommandResponse | ❌ | Atlas writes `petPos.x+petPos.y` (int16×2) at end, IDA OnActionCommand reads conditional bytes via reaction-table lookup. Sub-op enum drift candidate — defer pending CPet::DoAction sub-op decompile. |
| `CPet::OnLoadExceptionList@0x6a1510` | PetExcludeResponse | ❌ | Prefix + petLockerSN ✅. Atlas's loop (`for each excluded itemId: WriteInt`) vs IDA's loop body don't align in flat call list. Wire likely ✅. |
| `CWvsContext::OnCashPetFoodResult@0x9f7180` | PetCashFoodResult | ✅ | Wire matches. |
| `CWvsContext::SendActivatePetRequest@0x9f6980` | PetSpawn (sb) | ✅ | Wire matches (tick + nPos + bBossPet). |
| `CVecCtrlPet::EndUpdateActive@0x99f5a0` | PetMovementRequest (sb) | ❌ | Movement body sub-struct expansion gap (same as PetMovement clientbound). Wire likely ✅. |
| `CPet::DoAction@0x6a2340` | PetChatRequest (sb) | ❌ | Sub-op handler reachable via internal CPet logic. Wire layout: `petLockerSN(8) + actionType(1) + actionNo(1) + chatText(str)`. Atlas may write extra bytes. Defer pending atlas struct review. |
| `CPet::ParseCommand@0x6a3cc0` | PetCommand (sb) | ❌ | Similar to DoAction — internal logic. Defer. |
| `CPet::SendUpdateExceptionListRequest@0x6a0dd0` | PetExcludeItem (sb) | ❌ | Loop body expansion gap. Wire likely ✅. |
| `CWvsContext::SendPetFoodItemUseRequest@0x9d9f20` | PetFood (sb) | ✅ | Wire matches (tick + nPOS + nItemID). |
| `CWvsContext::SendStatChangeItemUseRequestByPetQ@0x9de400` | PetItemUse (sb) | ❌ | Atlas wire shape vs IDA needs cross-check. Trailing fields differ. Defer pending atlas review. |
| `CPet::SendDropPickUpRequest@0x6a0820` | PetDropPickUp (sb) | ❌ | Complex conditional encoder. Atlas may have different field order or trailing items. Defer pending detailed cross-check. |

Real wire bugs that look likely (need confirmation):
- `PetCommandResponse` trailing petPos fields may be vestigial — IDA doesn't read them on every code path.
- `PetItemUse` field order vs v95 IDA needs side-by-side.

## Still pending — combat domain (drop)

Phase 2c (task-065) audit of 3 drop packets in GMS v95. ✅ 1 / ❌ 2.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CDropPool::OnDropEnterField@0x516670` | DropSpawn | ❌ | **Analyzer FP.** Atlas's `if isMeso { WriteInt(meso) } else { WriteInt(itemId) }` if/else expands into two consecutive Encode4 entries in the flat call list, throwing off positions 4+. Wire actually matches field-for-field. Same root cause as MonsterSpawn — analyzer should model mutually-exclusive if/else writes as a single position with alternation. |
| `CDropPool::OnDropLeaveField@0x511e20` | DropDestroy | ❌ (real) | Atlas's destroy encoder for `destroyType == 4` (explode) writes `WriteInt(characterId)` + optional `WriteByte(petSlot)` but v95 reads `Decode2 (tLeaveDelay)`. Wire desync on explode. Also for `destroyType == 5` (pet pickup), v95 reads an extra `Decode4` (pet locker SN low part?) inside the case — atlas may emit petSlot byte where v95 expects 4 bytes. Defer to follow-up that adds the explode-delay field + tightens pet-pickup wire shape; needs constructor update + 4-variant test. |
| `CWvsContext::SendDropPickUpRequest@0x9d5d50` | DropPickUp (sb) | ✅ | Wire matches (fieldKey + tick + pt.x + pt.y + dropId + cliCrc). |

## Still pending — combat domain (reactor)

Phase 2d (task-065) audit of 4 reactor packets in GMS v95. ✅ 3 / ❌ 1.

| FName | Atlas writer/handler | Verdict | Notes |
|---|---|---|---|
| `CReactorPool::OnReactorEnterField@0x6cf490` | ReactorSpawn | ✅ | Wire matches (dwID + dwTemplateID + nState + ptPos + bFlip + sName). |
| `CReactorPool::OnReactorChangeState@0x6ccd60` | ReactorHit | ✅ | Wire matches (reactorId + newState + ptPos + tDelay + frameDelay + stance). |
| `CReactorPool::OnReactorLeaveField@0x6ccea0` | ReactorDestroy | ✅ | Wire matches (reactorId + finalState + ptPos). |
| `CReactorPool::FindHitReactor@0x6cd4e0` | ReactorHitRequest (sb) | ❌ | **Analyzer FP** — same if/else pattern. Atlas writes `if isSkill { WriteInt(1) } else { WriteInt(0) }` which expands to two consecutive Encode4 entries; wire bytes match v95 exactly (oid + isSkill + dwHitOption + delay + skillId = 18 bytes). |

## Phase 3 — GMS v83 cross-version pass

Phase 3 Task 8 (task-065) audit of 30 combat packets against v83 IDA. ✅ 11 / ❌ 19. Comparable verdict distribution to v95.

| Packet | v95 verdict | v83 verdict | Cross-version note |
|---|---|---|---|
| MonsterMovementAck | ✅ | ✅ | Wire matches both versions. |
| MonsterDamage | ✅ | ✅ | Wire matches both versions. |
| MonsterHealth | ✅ | ✅ | Wire matches both versions. |
| PetChat | ✅ | ✅ | Wire matches both versions. |
| PetCashFoodResult | ✅ | ✅ | Wire matches both versions. |
| PetSpawn (sb) | ✅ | (skipped) | `CWvsContext::SendActivatePetRequest` does not exist in v83 IDA. The atlas serverbound handler may target a different v83 FName; needs cross-version trace. |
| PetFood (sb) | ✅ | ✅ | Wire matches both versions. |
| DropPickUp (sb) | ✅ | ✅ | Wire matches both versions. |
| ReactorSpawn | ✅ | ✅ | Wire matches both versions. |
| ReactorHit | ✅ | ✅ | Wire matches both versions. |
| ReactorDestroy | ✅ | ✅ | Wire matches both versions. |
| MonsterMovement | 🔍 | ❌ | v83 lacks `bNotChangeAction` byte + `multiTargetForBall` + `randTimeForAreaAttack` loops. Atlas correctly gates these with `(GMS && >83) \|\| JMS` in `monster/clientbound/movement.go` so v83 wire is shorter. **No encoder fix needed** — the audit-tool's flat diff over-reports because atlas's separate `WriteInt16(skillId) + WriteInt16(skillLevel)` (4 bytes total) vs v83's packed `Decode4(sEffect.m_Data)` is the same 4 wire bytes but different field decomposition. |
| All other ❌ verdicts | ❌ | ❌ | Same analyzer FP root causes (registry struct-name collision, if/else branch double-counting, sub-struct expansion gap). No encoder change needed — wire bytes match between versions on the in-scope fields. |

**Conclusion:** v83 introduces no new wire bugs that v95's audit didn't already surface. Atlas's existing `(GMS && >83) || JMS` gate on monster movement is verified correct. No encoder commits land in this Phase 3 sub-task — verdict shifts are pure analyzer artifacts of the version delta.

## Phase 3 — GMS v87 cross-version pass

Phase 3 Task 9 (task-065) audit of 30 combat packets against v87 IDA. ✅ 12 / ❌ 18. Matches v95 verdict distribution since v87's atlas gates (`>v83 || JMS`) evaluate the same as v95.

| Difference vs v95 | Note |
|---|---|
| All FNames present | Including `CWvsContext::SendActivatePetRequest@0xabbb70` (absent in v83). |
| Same wire shape | `>v83` gate firing means v87 reads `bNotChangeAction`, `multiTargetForBall`, and `randTimeForAreaAttack` — same as v95. |
| Same verdict pattern | 11 ✅ + 1 🔍 + 18 ❌ = 30. The ❌ rows are the same analyzer FPs (registry struct-name collision, if/else branch double-counting, sub-struct expansion gap). |

**Conclusion:** v87 introduces no new wire bugs beyond v95. The atlas encoders are version-compatible across v83/v87/v95 for all packets in scope, with the documented `>v83` gates correctly narrowing v83-only wire shape differences.

## Phase 3 — JMS v185 cross-version pass

Phase 3 Task 10 (task-065) audit of 30 combat packets against JMS v185 IDA. ✅ 11 / 🔍 1 / ❌ 18. Identical distribution to GMS v95.

| Difference vs v95 | Note |
|---|---|
| All 30 FNames present | Including `CUserRemote::OnPetActivated@0xa576d3` (present in JMS like v95). |
| Atlas `\|\| JMS` gate fires | JMS v185 reads the full v95-equivalent field set including `bNotChangeAction`, `multiTargetForBall`, and `randTimeForAreaAttack`. |
| Same verdict pattern | All atlas encoders are JMS-compatible for in-scope packets. No JMS-specific opcode mapping changes needed for combat. |
| Pre-existing pipeline warnings | `DecodeSub` unknown primitive in CWvsContext::OnCharacterInfo, CLogin::OnSelectWorldResult, CLogin::OnCreateNewCharacterResult — left over from task-028's character / task-027's login work. Not introduced by combat audit. |

**Conclusion:** JMS v185 introduces no new combat-domain wire bugs beyond v95. The `(GMS && >83) || JMS` gates in atlas monster/movement and atlas's lack of JMS-specific combat divergences (no `if Region == "JMS"` paths in monster/pet/drop/reactor encoders) are verified correct.

---

## Cross-version summary (combat domain)

| Version | ✅ | 🔍 | ❌ | Notes |
|---|---|---|---|---|
| GMS v95 | 11 | 1 | 18 | Source-of-truth pass. |
| GMS v83 | 11 | 0 | 19 | PetSpawn (sb) skipped — `SendActivatePetRequest` missing in v83 binary. MonsterMovement ❌ where v95 is 🔍 (analyzer FP, wire correct per `>83` gate). |
| GMS v87 | 12 | 1 | 18 | One more ✅ than v95 (PetSpawn sb routes cleanly). Otherwise matches v95. |
| JMS v185 | 11 | 1 | 18 | Identical distribution to v95. |

**Total real wire bugs identified across all 4 versions:** 2 (MonsterDestroy swallow-id, MonsterControl shape divergence in v95) + 1 (DropDestroy explode/pet-pickup tail in v95). All deferred to follow-up tasks with constructor-signature implications.

**Total analyzer FPs:** ~16 per version. Root causes (1) registry struct-name collision across sub-domains, (2) if/else branch double-counting in flat call list, (3) sub-struct expansion gap. All have known paths to resolution in the audit-tool follow-up section.

**No encoder mutations** land in any Phase 3 sub-task — atlas's existing version gates are correct.

## Sub-op enum / sub-struct deferrals — social domain (task-066)

- **`party.WritePartyData` (package-level function)** — `libs/atlas-packet/party/member_data.go:19` flattens 6 fixed-size column slices (id, name, jobId, level, channelId, mapId) plus a leader id and 6×4 zero-padding tail. The audit pipeline's TypeRegistry walks receiver-method `Encode`/`Write` only; package-level write helpers are invisible. Affected packets: `party/clientbound/update.go`, `party/clientbound/join.go`, `party/clientbound/left.go`. Audit verdict for these three files will be ⚠️ "tool-limitation: package-level write helper not modelled; verify against IDA member-list shape".

- **OP-FAMILY-note** — `libs/atlas-packet/note/serverbound/operation.go` `Operation` struct emits only the op byte (sub-op discriminator for NOTE_ACTION opcode 0x9A/154 in GMS v95). Sub-operations audited individually via synthetic FName entries: `CWvsContext::OnMemoNotify_Receive` (op=2 REQUEST → ✅), `CMemoListDlg::SetRet` (op=1 DISCARD → ✅ after val1 fix), `CCashShop::OnCashItemResLoadGiftDone` (op=0 SEND → ✅). The sub-op value space (SEND=0, DISCARD=1, REQUEST=2) is template-configured; enum drift verification deferred to Phase 2 cross-version pass.

- **NoteDisplay tool-limitation** — `libs/atlas-packet/note/clientbound/display.go` `Display.Encode` writes `WriteInt64(model.MsTime(timestamp))` (Encode8 = 8 bytes); IDA `GW_Memo::Decode` reads `DecodeBuffer(v2, &this->dateSent, 8u)` (DecodeBuf = 8 raw bytes). Both are 8 bytes on the wire; the audit framework reports ❌ "width mismatch" because it classifies `int64` (Decode8) and `bytes` (DecodeBuf) as different types. Wire is correct: FILETIME is a 64-bit little-endian value. Verdict manually promoted to ⚠️.

- **OP-FAMILY-buddy** — `libs/atlas-packet/buddy/serverbound/{operation_add,operation_accept,operation_delete}.go` are each decoded in a two-step sequence by the atlas-channel handler (`socket/handler/buddy_operation.go`): first `buddy.Operation.Decode` reads the sub-op byte, then the sub-type `Decode` reads its payload. The audit pipeline sees only each sub-type's `Encode` method (OperationAdd: EncodeStr+EncodeStr; OperationAccept: Encode4; OperationDelete: Encode4) without the leading sub-op byte, and compares against the full IDA `Send*FriendMsg` functions which include `Encode1(sub-op)` at position 0. This mismatch causes ❌ for all three. Wire format is correct: on the wire, the `buddy.Operation` prefix byte appears first (op-byte = 1/ADD, 2/ACCEPT, 3/DELETE), followed by the sub-type payload. The audit verdict is a tool-limitation (no multi-step decoder model). Sub-op values confirmed: RELOAD=0 (`CWvsContext::LoadFriend@0xa10240`), ADD=1 (`CField::SendSetFriendMsg@0x535240`), ACCEPT=2 (`CField::SendAcceptFriendMsg@0x52f290`), DELETE=3 (`CField::SendDeleteFriendMsg@0x52f170`). Template key `operations.{RELOAD,ADD,ACCEPT,DELETE}` must map to these byte values; enum drift verification deferred to Phase 2 cross-version pass.

- **BuddyError sub-op enum** — `libs/atlas-packet/buddy/clientbound/error.go` `Error` struct has a `hasExtra bool` field that controls whether a conditional second byte is written (`if m.hasExtra { w.WriteByte(0) }`). The IDA `CWvsContext::OnFriendResult` case arms for error sub-ops (`0x0B`–`0x0F`, `0x10`–`0x13`, `0x16`, `0x17`) show varying secondary-read behaviour: mode-only arms (0x0B–0x0F, 0x17) read no additional bytes; mode+Decode1 arms (0x10, 0x11, 0x13, 0x16) read 1 byte then optionally a string. The atlas struct's `hasExtra` flag models the first class; the conditional `DecodeStr` path for modes 0x10/0x11/0x13/0x16 is not represented. Verdict: ❌ reported by pipeline (extra conditional byte). Real behaviour depends on the mode byte value at runtime; static analysis cannot distinguish the arms. Defer sub-op enum value space verification to Phase 2.

- **BuddyInvite two-extra-field investigation** — `libs/atlas-packet/buddy/clientbound/invite.go` `Invite.Encode` writes: mode + Encode4(origId) + EncodeStr(origName) + model.Buddy(39 bytes) + WriteByte(0/inShop). IDA `CWvsContext::OnFriendResult` case 0x09 reads: Decode4(origId) + DecodeStr(origName) + **Decode4(v25)** + **Decode4(v26)** + CFriend::Insert(GW_Friend 39 bytes + Decode1 inShop). The two additional Decode4 calls (v25/v26 at IDA lines 67–69) appear between the originator name and the GW_Friend insert. IDA types these as `ZRef<CDialog>*` and `char*` but they are unambiguous packet reads (`CInPacket::Decode4(v3)`). Atlas does NOT write these 8 bytes. If they are real wire fields, the client will misparse the invite packet (reading from the start of model.Buddy as v25/v26, then desynchronising). Impact: potential invite display corruption in the client. Investigation needed: (1) test invite flow in GMS v95 client against atlas server to observe client reaction; (2) attempt to identify v25/v26 semantics from context (dialog creator uses them for friend-name/icon lookup). Real wire bug candidate — deferred pending live client test confirmation.

- **Sub-op enum / sub-struct deferrals — chat sub-domain (task-066, Phase 1d)** — Six of the eight chat files use a parameterised mode byte as the first field in their `Encode` method. The audit pipeline cannot model a switch-on-mode dispatch tree and can only verify the outermost leading byte. Sub-op value spaces and per-mode body layouts are deferred. Files in scope:
  - `libs/atlas-packet/chat/clientbound/multi.go` (`MultiChat`) — `WriteByte(m.mode)` at position 0; mode values: 0=buddy, 1=party, 2=guild, 3=alliance, 6=expedition. IDA `CField::OnGroupMessage@0x535490` switch case: {0→3, 1→2, 2→4, 3→5, 6→26} for chat-log type. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/whisper.go` (all 7 structs: `WhisperSendResult`, `WhisperReceive`, `WhisperFindResultCashShop`, `WhisperFindResultMap`, `WhisperFindResultChannel`, `WhisperFindResultError`, `WhisperError`, `WhisperWeather`) — `WriteByte(m.mode)` at position 0; mode values: 5=find, 6=chat, 9=find-result-offline, 10=send-result, 18=receive, 34=blocked, 68=buddy-window-find, 72=find-status, 134=macro-notice, 146=weather-msg. IDA `CField::OnWhisper@0x5448a0` switch: {9→find-query, 10→send-result, 18→receive, 34→blocked-result, 72→find-query-type2, 146→weather}. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/world_message.go` (all 7 structs: `WorldMessageSimple`, `WorldMessageTopScroll`, `WorldMessageSuperMegaphone`, `WorldMessageBlueText`, `WorldMessageItemMegaphone`, `WorldMessageYellowMegaphone`, `WorldMessageMultiMegaphone`, `WorldMessageGachapon`) — `WriteByte(m.mode)` at position 0; mode values: 0=notice, 1=popup, 2=megaphone, 3=super-megaphone, 4=top-scroll, 5=pink-text, 6=blue-text, 7=multi-megaphone, 8=yellow-megaphone, 9=item-megaphone, 12=gachapon. IDA `CWvsContext::OnBroadcastMsg@0xa04160` dispatches on Decode1 across 12+ sub-modes. Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/clientbound/world_message_extra.go` (4 structs: `WorldMessageUnknown3`, `WorldMessageUnknown7`, `WorldMessageUnknown8`, `WorldMessageWeather`) — `WriteByte(m.mode)` at position 0; same dispatcher as world_message.go (modes 3/7/8/weather-variant). Sub-op enum drift deferred.
  - `libs/atlas-packet/chat/serverbound/multi.go` (`Multi`) — `WriteByte(m.chatType)` at position 0; chat types: 0=buddy, 1=party, 2=guild, 3=alliance, 6=expedition. IDA `CUIStatusBar::SendGroupMessage@0x87f7f0` maps nChatTarget → Encode1 value: {party→1, guild→2, alliance→3, expedition→6, buddy/friend-group→0}. The updateTime prefix (`Encode4(update_time)` before the type byte in v95) is NOT modelled in atlas `Multi.Encode` — this is a **real wire bug**: atlas writes chatType+recipientCount+recipients+text but v95 client writes updateTime+chatType+recipientCount+recipients+text. Follow-up: add `updateTime` field with `GMS>83` gate to `Multi.Encode` and update callers. Sub-op enum drift also deferred.
  - `libs/atlas-packet/chat/serverbound/whisper.go` (`Whisper`) — `WriteByte(byte(m.mode))` at position 0; `WhisperMode` enum: FIND=5, CHAT=6, BuddyWindowFind=68, MacroNotice=134. IDA `CField::SendChatMsgWhisper@0x53d3b0` for chat path encodes: `Encode1(mode) + Encode4(updateTime) + EncodeStr(targetName) + EncodeStr(msg)`. Atlas `Whisper.Encode` writes: `WriteByte(mode) + WriteInt(updateTime, GMS>=95) + WriteAsciiString(targetName) + optional WriteAsciiString(msg, mode==CHAT)` — this matches the IDA chat path wire exactly. Sub-op enum drift (non-chat modes) deferred.

  **Also deferred: `ChatGeneralChat.md` false positive** — `ChatGeneralChat` reports ❌ because the IDA entry for `CUser::OnChat@0x8e86c0` begins after the dispatcher has already consumed `Decode4(characterId)`. Atlas `GeneralChat.Encode` writes `WriteInt(characterId)` first, causing a position-0 int32 vs byte width mismatch in the diff. Wire is correct: on the wire characterId is the first 4 bytes of the CHATTEXT packet; `OnChat` is only invoked after `CUserPool::OnUserRemotePacket` strips the characterId prefix. Verdict manually promoted to ⚠️.

  **Real wire bug in `Multi` (serverbound):** `CUIStatusBar::SendGroupMessage` prepends `Encode4(update_time)` before the chat-type byte in v95. Atlas `Multi.Encode` does not include this field. Needs constructor update + `GMS>83` gate. Deferred to follow-up task.

## Sub-op enum / sub-struct deferrals — social domain (task-066, Phase 1e: party)

Party domain audit (task-066 Phase 1e) of 15 packets in GMS v95. ✅ 2 / ❌ 13.

All 13 ❌ verdicts are **tool-limitation false positives** caused by one of two structural patterns. No new real wire bugs remain after the fixes in `2019dd581`.

### Real wire bugs fixed in-branch (task-066 commits)

| Atlas struct | Bug | Fix commit |
|---|---|---|
| `party/member_data.go` `WritePartyData` | 80-byte shortfall: missing `m_nSKillID` per TOWNPORTAL (6×4=24 bytes) and all PQ reward fields (56 bytes). Client reads PARTYDATA::Decode(0x17A=378 bytes); atlas was emitting 298 bytes. | `2019dd581` |
| `party/clientbound/invite.go` `Invite` | Missing `originatorJobId` (Decode4) and `originatorLevel` (Decode4) fields between the inviter name and the autoJoin flag. IDA `OnPartyResult#Invite` case 4: `Decode4(partyId)+DecodeStr(name)+Decode4(nSkillID)+Decode4(level)+Decode1(autoJoin)`. | `2019dd581` |

### Tool-limitation pattern A — clientbound mode-byte dispatcher prefix

`CWvsContext::OnPartyResult` is a dispatcher function that reads the mode byte first, then dispatches to a sub-handler. Each sub-handler IDA entry starts at the first field AFTER the mode byte. Atlas structs encode the mode byte as their first write. The audit pipeline compares atlas position 0 (mode=byte) to IDA position 0 (first real field=int32 or larger), producing false-positive width mismatches for all subsequent fields.

Affected clientbound packets (all ❌ due to this tool-limitation):

| Report | IDA FName | Triage |
|---|---|---|
| `PartyCreated.md` | `CWvsContext::OnPartyResult#Created` | ⚠️ Tool-limitation (mode-byte prefix). Also latent width: atlas writes `int32` for portal map IDs, IDA reads `Decode2`. All fields are zeros in practice; wire is functionally correct. |
| `PartyDisband.md` | `CWvsContext::OnPartyResult#Disband` | ⚠️ Tool-limitation (mode-byte prefix). After adjusting for prefix: partyId+targetId+isForced align ✅; positions 3-4 are atlas trailing fields (partyId repetition) not read by IDA's #Disband case. Wire functionally correct. |
| `PartyError.md` | `CWvsContext::OnPartyResult#Error` | ⚠️ Tool-limitation (mode-byte prefix). IDA #Error arm reads no fields (mode-only); atlas writes mode+name. The name string is consumed by atlas server but never read by the sub-handler; sends to client harmlessly. |
| `PartyInvite.md` | `CWvsContext::OnPartyResult#Invite` | ⚠️ Tool-limitation (mode-byte prefix). Invite fields now correct after `2019dd581` fix; mode-byte misalignment causes pipeline ❌. Wire is ✅ after fix. |
| `PartyJoin.md` | `CWvsContext::OnPartyResult#Join` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. |
| `PartyLeft.md` | `CWvsContext::OnPartyResult#Left` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. |
| `PartyUpdate.md` | `CWvsContext::OnPartyResult#Update` | ⚠️ Tool-limitation (mode-byte prefix + WritePartyData). WritePartyData now 378 bytes per `2019dd581`; wire is ✅ after fix. **HOT PATH** — 4-variant byte-output test added: `TestUpdateByteOutput` (383 bytes). |
| `PartyChangeLeader.md` | `CWvsContext::OnPartyResult#ChangeLeader` | ⚠️ Tool-limitation (mode-byte prefix). After adjusting: newLeaderId(4)+disconnectedFlag(1) align ✅; position 2 extra byte is atlas trailing sentinel not read by client. Wire functionally correct. |

### Tool-limitation pattern B — serverbound op-byte dispatcher prefix

`CField::Send*PartyMsg` functions write an op byte first (`op=2/4/5/6`), then the sub-payload. Atlas serverbound structs model only the sub-payload (op byte is written by `OperationBody` helper upstream). The audit pipeline compares atlas position 0 (sub-payload first field) to IDA position 0 (op byte), producing false-positive mismatches.

Affected serverbound packets (all ❌ due to this tool-limitation):

| Report | IDA FName | Op byte | Triage |
|---|---|---|---|
| `PartyOperation.md` | `CField::SendWithdrawPartyMsg` | op=2 (LEAVE) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: nothing — Operation emits only the op byte and an unexplained trailing 0x00. Trailing 0 is not modelled in atlas but server-side it is the IDA's second byte. Minor: atlas Operation (serverbound) may be missing a trailing 0x00 byte. |
| `PartyOperationChangeLeader.md` | `CField::SendChangePartyBossMsg` | op=6 (CHANGE_BOSS) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationChangeLeader writes `targetCharacterId(4)` which aligns with IDA's Decode4(targetCharacterId). ✅ after adjustment. |
| `PartyOperationExpel.md` | `CField::SendKickPartyMsg` | op=5 (EXPEL) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationExpel writes `targetCharacterId(4)` which aligns with IDA's Decode4(targetCharacterId). ✅ after adjustment. |
| `PartyOperationInvite.md` | `CField::SendJoinPartyMsg` | op=4 (INVITE) | ⚠️ Tool-limitation (op-byte prefix). After adjusting: atlas OperationInvite writes `targetName(str)` which aligns with IDA's DecodeStr(targetName). ✅ after adjustment. |
| `PartyMemberHP.md` | `CUserRemote::OnReceiveHP` | n/a — `characterId` prefix consumed by `CUserPool::OnUserRemotePacket` | ⚠️ Tool-limitation (characterId dispatcher prefix, not op-byte). `OnReceiveHP` reads only `Decode4(hp)+Decode4(maxHp)`. Atlas `MemberHP` writes `characterId(4)+hp(4)+maxHp(4)` = 12 bytes; characterId consumed upstream. Wire is ✅. **HOT PATH** — 4-variant byte-output test added: `TestPartyMemberHPByteOutput` (12 bytes). |

### OP-FAMILY-party-serverbound

`libs/atlas-packet/party/serverbound/operation.go` `Operation` struct emits only the op byte (sub-op discriminator for PARTY_ACTION opcode in GMS v95). Sub-operations are dispatched by `CField::Send*PartyMsg` after reading the op byte:
- op=2: WITHDRAW (`CField::SendWithdrawPartyMsg`) — `Operation` only; server handles withdraw
- op=4: INVITE (`CField::SendJoinPartyMsg` invite path) — `OperationInvite` with targetName
- op=5: EXPEL (`CField::SendKickPartyMsg`) — `OperationExpel` with targetCharacterId
- op=6: CHANGE_BOSS (`CField::SendChangePartyBossMsg`) — `OperationChangeLeader` with targetCharacterId

The `OperationJoin` struct is the non-op-byte sub-type (JOIN_PARTY uses a different encoding: op is not sent by client; server handles the party creation lookup). `PartyOperationJoin` ✅.

Sub-op value space verification deferred to Phase 2 cross-version pass.

### PartyOperation trailing 0x00 — minor open question

`CField::SendWithdrawPartyMsg` IDA shows `Encode1(op=2) + Encode1(0x00)` but atlas `Operation.Encode` (serverbound) writes only `WriteByte(m.op)`. The trailing 0x00 is not written. This is a candidate real wire bug with low functional impact (server reads op byte only; trailing byte would be ignored or parsed as the next packet). Investigation deferred; no client-facing correctness issue observed in practice.

## Real wire bugs fixed in-branch (task-065 follow-up commits)

Three of the four "real wire bugs" originally deferred have been fixed in-branch after re-analysis. The fourth turned out not to be a real bug at all.

| Original deferral | Resolution | Fix commit |
|---|---|---|
| `MonsterDestroy` missing swallow-char-id | **Fixed.** Added `DestroyTypeSwallow` enum + `swallowCharacterId` field + `NewMonsterDestroyBySwallow` constructor. Wire emits `WriteInt(swallowCharacterId)` when `destroyType == 4`. Tested with 5-variant round-trip + explicit 9-byte wire-length check. v95 audit now ✅. | `ac174269b` |
| `DropDestroy` explode/pet-pickup tail | **Fixed.** Replaced `petSlot int8` field with `explodeDelay int16` (type 4) + `petPickupExtra uint32` (type 5). Encoder switches on `destroyType` to emit the correct trailing fields per case. Legacy `NewDropDestroy` constructor preserved for backwards compatibility (auto-widens petSlot to petPickupExtra for type 5; ignores params for type 4). v95 audit positions 0-3 now ✅; remaining ❌ rows (positions 4-5) are the same switch-case-flatten analyzer FP documented elsewhere — wire is correct. | `ac174269b` |
| `MonsterMovementHandle` (serverbound) deferred | **Audited.** Decompiled JMS v185 `CMob::GenerateMovePath@0x6e8892` and verified atlas's `MovementRequest` encoder matches byte-for-byte across all v95+JMS gated blocks (multiTargetForBall, randTimeForAreaAttack, hackedCodeCRC, bChasing-tail). Added IDA entries to gms_v95.json + gms_jms_185.json. Audit verdict: 🔍 (sub-struct expansion FP). Wire is correct. v83/v87 IDA entries not added — `CMob::GenerateMovePath` address lookups deferred to next IDA swap. | `e32a3d809` |
| `MonsterControl` shape divergence | **Not a real bug.** Re-analysis with JMS v185 IDA loaded showed atlas's encoder writes `byte(controlType) + int(uniqueId) + (if type>0: byte(5) + int(monsterId) + MonsterModel)`. JMS reads `byte(controlMode) + int(mobId) + (if mode != 0: byte(aggro) + int(templateId) + MonsterModel)`. Production v95 (with dev-mode `CClientOptMan::GetOpt(2)` off) reads the same shape. **The earlier ❌ verdict was a false-positive** from my initial IDA entries unconditionally listing the dev-mode `moveRandSeed` block. Atlas server never enables opt 2, so seeds never appear on wire. Fix: removed seeds from IDA entries in all 4 version files (gms_v95.json, gms_v83.json, gms_v87.json, gms_jms_185.json). The hardcoded `byte(5)` at the aggro position is a *semantic* concern (atlas always sends 5 regardless of actual aggro state) but not a wire-shape bug — width and position match. | `e32a3d809` |
