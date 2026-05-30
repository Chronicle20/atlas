# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x776020
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nChannelID (channel id) — v83 has NO DecodeOpt before it and NO m_dwOldDriverID after it` | ✅ |  |
| 1 | byte | byte `sNotifierMessage flag (HIBYTE v110)` | ✅ |  |
| 2 | byte | byte `bCharacterData flag (v107)` | ✅ |  |
| 3 | int16 | int16 `nNotifierCheck (v108; notifier string-loop count)` | ✅ |  |
| 4 | int32 | int32 `damage seed 1 (CalcDamage::SetSeed; bCharacterData branch)` | ✅ |  |
| 5 | int64 | int32 `damage seed 2` | ❌ | width mismatch |
| 6 | byte | int32 `damage seed 3` | ❌ | width mismatch |
| 7 | byte | bytes `CharacterData::Decode — ENVELOPE BOUNDARY (character domain task-028)` | ❌ | width mismatch |
| 8 | byte | int64 `timestamp (DecodeBuffer p,8u FILETIME) — v83 has NO logout-gift block before it` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |


## Audit notes

🔍 **envelope-only:** CharacterData inner shape audited under character domain
(task-028). The IDA model represents the `CharacterData::Decode` region
(`CStage::OnSetField` v83 @0x776182) as a single `DecodeBuf` boundary marker;
the wire-diff table validates only the envelope bytes before and after it.

⚠️ **Analyzer limitation:** the wire-diff cascade above (rows 5+) is an
alignment artifact, not independent bugs. The atlas seed loop
(`for i:=0;i<3 { WriteInt }`) collapses to one representative op and the
`WriteByteArray(characterData.Encode(...))` cross-package recurse is dropped, so
positional alignment past the seed loop is unreliable. Read the manual verdict
below, not the row-by-row table.

### Manual envelope verdict (GMS v83, `CStage::OnSetField` @0x776020)

| Field | IDA (v83) | Atlas (v83-gated) | Match |
|---|---|---|---|
| DecodeOpt | **absent** (v83 reads channelId first) | WriteShort gated `GMS>83 \|\| JMS` → **omitted for v83** | ✅ |
| channelId | Decode4 (line 125) | WriteInt | ✅ |
| **m_dwOldDriverID** | **absent** (line 130 reads sNotifier immediately after channelId) | WriteInt gated `GMS>=95` → **omitted for v83** | ✅ **(deferred bug resolved)** |
| sNotifierMessage | Decode1 (line 130) | WriteByte(1) | ✅ |
| bCharacterData | Decode1 (line 131) | WriteByte(1) | ✅ |
| nNotifierCheck | Decode2 (line 133) | WriteShort(0) gated `GMS>28 \|\| JMS` | ✅ |
| damage seeds | 3× Decode4 (lines 167-169, bCharacterData branch) | 3× WriteInt gated `GMS>28 \|\| JMS` | ✅ |
| CharacterData | CharacterData::Decode (line 173) | WriteByteArray(characterData.Encode) | 🔍 boundary (task-028) |
| logout gifts | **absent** (v83 has no OnSetLogoutGiftConfig block) | 4× WriteInt gated `GMS>83 \|\| JMS` → **omitted for v83** | ✅ |
| timestamp | DecodeBuffer(p, 8) (line 234, FILETIME) | WriteInt64 | ✅ |

### DEFERRED BUG RESOLVED — m_dwOldDriverID version-introduction (was _pending.md)

The v95 audit deferred `m_dwOldDriverID` because the version-introduction point
was unknown. **v83 IDA resolves it:** `CStage::OnSetField` v83 @0x776020 reads
`Decode4 channelId` (line 125) then **immediately** `Decode1 sNotifierMessage`
(line 130) — there is NO `Decode4` for an old-driver-id between them. The field
was therefore introduced **after v83**. The atlas fix gates the 4-byte
`m_dwOldDriverID` write on `Region()=="GMS" && MajorVersion() >= 95` (provisional
lower bound; the v87 pass will confirm whether the true bound is >=87 or >=95).
With this gate, GMS v83/v87 omit the field (✅ here) and GMS v95 emits it (the
v95 FieldSetField/FieldWarpToMap rows flip ❌→✅). See `set_field.go`,
`warp_to_map.go`.

Ack: world-audit Phase 3 v83 on 2026-05-28
