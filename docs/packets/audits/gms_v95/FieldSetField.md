# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x71a0a0
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/set_field.go`
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
| 9 | byte | bytes `CharacterData::Decode — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ✅ |  |
| 10 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 11 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 12 | int16 | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 13 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 14 | int64 | int64 `timestamp (DecodeBuffer p,8u — FILETIME); atlas WriteInt64` | ✅ |  |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

