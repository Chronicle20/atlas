# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x7c429c
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/set_field.go`
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
| 8 | byte | bytes `CharacterData::Decode (@0x7c440a) — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ✅ |  |
| 9 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig @0x7c4412) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 10 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 11 | int16 | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 12 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 13 | int64 | int64 `timestamp (DecodeBuffer p,8u @0x7c4558 — FILETIME); atlas WriteInt64` | ✅ |  |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

