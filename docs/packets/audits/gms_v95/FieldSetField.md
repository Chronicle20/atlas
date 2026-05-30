# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x71a0a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt (CClientOptMan::DecodeOpt) — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id)` | ✅ |  |
| 2 | int32 | int32 `m_dwOldDriverID — v95 reads Decode4 unconditionally; atlas now emits it gated GMS>=95 (RESOLVED, v83 omits)` | ✅ |  |
| 3 | byte | byte `sNotifierMessage flag (HIBYTE)` | ✅ |  |
| 4 | byte | byte `bCharacterData flag (atlas always 1)` | ✅ |  |
| 5 | int16 | int16 `nNotifierCheck (atlas writes 0 — no notifier title/content loop)` | ✅ |  |
| 6 | int32 | int32 `damage seed 1 (CalcDamage::SetSeed)` | ✅ |  |
| 7 | int64 | int32 `damage seed 2` | ❌ | width mismatch |
| 8 | byte | int32 `damage seed 3` | ❌ | width mismatch |
| 9 | byte | bytes `CharacterData::Decode — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ❌ | width mismatch |
| 10 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 11 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 12 | int16 | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 13 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 14 | int32 | int64 `timestamp (DecodeBuffer p,8u — FILETIME); atlas WriteInt64` | ❌ | width mismatch |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |


## Audit notes

🔍 **envelope-only:** CharacterData inner shape audited under character domain
(task-028). The IDA model represents `CharacterData::Decode` (line 174) as a
single `DecodeBuf` boundary; the table validates only the envelope bytes around
it.

⚠️ **Analyzer limitation:** the wire-diff cascade (rows 7+) is an alignment
artifact, not independent bugs. The atlas seed loop (`for i:=0;i<3 { WriteInt }`)
collapses to one representative op and the `WriteByteArray(characterData.Encode)`
cross-package recurse is dropped, so positional alignment past the seed loop is
unreliable. Read the manual verdict below.

### Manual envelope verdict (GMS v95, `CStage::OnSetField` @0x71a0a0)

| Field | IDA (v95) | Atlas | Match |
|---|---|---|---|
| DecodeOpt | Decode2 | WriteShort(0) gated (GMS>83 \|\| JMS) | ✅ |
| channelId | Decode4 (line 128) | WriteInt | ✅ |
| **m_dwOldDriverID** | **Decode4 (line 129, unconditional)** | **WriteInt(0) gated GMS>=95** | ✅ **(RESOLVED — was deferred)** |
| sNotifierMessage | Decode1 (line 132) | WriteByte(1) | ✅ |
| bCharacterData | Decode1 (line 133) | WriteByte(1) | ✅ |
| nNotifierCheck | Decode2 (line 134) | WriteShort(0) gated (GMS>28 \|\| JMS) | ✅ |
| damage seeds | 3× Decode4 (lines 164-166) | 3× WriteInt gated (GMS>28 \|\| JMS) | ✅ |
| CharacterData | CharacterData::Decode (line 174) | WriteByteArray(characterData.Encode) | 🔍 boundary (task-028) |
| logout gifts | 4× Decode4 (OnSetLogoutGiftConfig) | 4× WriteInt(0) gated (GMS>83 \|\| JMS) | ✅ |
| timestamp | DecodeBuffer(p, 8) (line 235, FILETIME) | WriteInt64 | ✅ |

**DEFERRED BUG RESOLVED — m_dwOldDriverID (task-068 Phase 3 v83):** v83 IDA
(`CStage::OnSetField` @0x776020) does NOT read an old-driver-id after channelId
(sNotifierMessage is read immediately), proving the field was introduced after
v83. Atlas now emits the 4-byte field gated `GMS && MajorVersion>=95` (see
`set_field.go`), aligning row 2 with v95 (✅) while omitting it for v83/v87. The
residual report ❌ is the documented seed-loop/CharacterData-boundary analyzer
artifact, NOT a wire bug — every envelope field matches v95.

Ack: world-audit Phase 3 v95-refresh on 2026-05-28
