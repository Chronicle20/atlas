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
| 9 | int32 | bytes `CharacterData::Decode — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ✅ |  |
| 10 | bytes | int32 `m_bPredictQuit (OnSetLogoutGiftConfig) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ✅ |  |
| 11 | byte | int32 `logout gift commodity SN #1` | ❌ | width mismatch |
| 12 | byte | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 13 | byte | int32 `logout gift commodity SN #3` | ❌ | width mismatch |
| 14 | int64 | int64 `timestamp (DecodeBuffer p,8u — FILETIME); atlas WriteInt64` | ✅ |  |
| 15 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 51 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 52 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 53 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 54 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 55 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 56 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 57 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 58 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 59 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 60 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 61 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 62 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 63 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 64 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 65 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 66 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 67 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 68 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 69 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 70 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 71 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 72 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 73 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 74 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 75 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 76 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 77 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 78 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 79 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 80 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 81 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 82 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 83 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 84 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 85 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

