# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x7c429c
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt (CClientOptMan::DecodeOpt @0x7c42c5) — atlas WriteShort(0) gated (GMS>83 \|\| JMS); present v87` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id, @0x7c42db)` | ✅ |  |
| 2 | byte | byte `sNotifierMessage flag (HIBYTE, @0x7c42fa) — read immediately after channelId; v87 has NO m_dwOldDriverID Decode4 between (unlike v95)` | ✅ |  |
| 3 | byte | byte `bCharacterData flag (@0x7c4307; atlas always 1)` | ✅ |  |
| 4 | int16 | int16 `nNotifierCheck (@0x7c431e; atlas writes 0 — no notifier title/content loop)` | ✅ |  |
| 5 | int32 | int32 `damage seed 1 (CalcDamage::SetSeed @0x7c43c2; bCharacterData branch)` | ✅ |  |
| 6 | int64 | int32 `damage seed 2 (@0x7c43cc)` | ❌ | width mismatch |
| 7 | byte | int32 `damage seed 3 (@0x7c43e1)` | ❌ | width mismatch |
| 8 | byte | bytes `CharacterData::Decode (@0x7c440a) — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ❌ | width mismatch |
| 9 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig @0x7c4412) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 10 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 11 | int16 | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 12 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 13 | int32 | int64 `timestamp (DecodeBuffer p,8u @0x7c4558 — FILETIME); atlas WriteInt64` | ❌ | width mismatch |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |


## Audit notes

🔍 **envelope-only:** CharacterData inner shape audited under character domain
(task-028). The IDA model represents `CharacterData::Decode` (line 159) as a
single `DecodeBuf` boundary; the table validates only the envelope bytes around
it.

⚠️ **Analyzer limitation:** the wire-diff cascade (rows 6+) is an alignment
artifact, not independent bugs. The atlas seed loop (`for i:=0;i<3 { WriteInt }`)
collapses to one representative op and the `WriteByteArray(characterData.Encode)`
cross-package recurse is dropped, so positional alignment past the seed loop is
unreliable. Read the manual verdict below.

### Manual envelope verdict (GMS v87, `CStage::OnSetField` @0x7c429c)

| Field | IDA (v87) | Atlas | Match |
|---|---|---|---|
| DecodeOpt | Decode2 (@0x7c42c5) | WriteShort(0) gated (GMS>83 \|\| JMS) | ✅ |
| channelId | Decode4 (@0x7c42db) | WriteInt | ✅ |
| **m_dwOldDriverID** | **ABSENT — sNotifierMessage read immediately after channelId** | **WriteInt(0) gated GMS>=95 (omitted for v87)** | ✅ **(v87 confirms v83; field introduced v87→v95)** |
| sNotifierMessage | Decode1 (@0x7c42fa) | WriteByte(1) | ✅ |
| bCharacterData | Decode1 (@0x7c4307) | WriteByte(1) | ✅ |
| nNotifierCheck | Decode2 (@0x7c431e) | WriteShort(0) gated (GMS>28 \|\| JMS) | ✅ |
| damage seeds | 3× Decode4 (@0x7c43c2-e1) | 3× WriteInt gated (GMS>28 \|\| JMS) | ✅ |
| CharacterData | CharacterData::Decode (@0x7c440a) | WriteByteArray(characterData.Encode) | 🔍 boundary (task-028) |
| logout gifts | 4× Decode4 (OnSetLogoutGiftConfig @0x7c4412) | 4× WriteInt(0) gated (GMS>83 \|\| JMS) | ✅ |
| timestamp | DecodeBuffer(p, 8) (@0x7c4558, FILETIME) | WriteInt64 | ✅ |

**oldDriverID gate CONFIRMED (task-068 Phase 3 v87):** v87 IDA
(`CStage::OnSetField` @0x7c429c) does NOT read an old-driver-id after channelId —
`sNotifierMessage` (Decode1 @0x7c42fa) is read immediately, matching v83
(@0x776020) and unlike v95 (@0x71a0a0, unconditional Decode4). The field was
introduced between v87 and v95, so the atlas gate `GMS && MajorVersion>=95`
remains correct (v83/v87 omit, v95 emits). The residual report ❌ is the
documented seed-loop/CharacterData-boundary analyzer artifact, NOT a wire bug —
every envelope field matches v87.

Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
