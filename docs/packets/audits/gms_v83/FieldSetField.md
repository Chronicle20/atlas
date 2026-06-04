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
| 7 | byte | bytes `CharacterData::Decode — ENVELOPE BOUNDARY (character domain task-028)` | ✅ |  |
| 8 | byte | int64 `timestamp (DecodeBuffer p,8u FILETIME) — v83 has NO logout-gift block before it` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

