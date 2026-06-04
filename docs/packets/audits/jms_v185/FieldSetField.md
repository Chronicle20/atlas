# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x7eea69
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt (CClientOptMan::DecodeOpt @line119) — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id @line121)` | ✅ |  |
| 2 | byte | byte `JMS-only byte (@line122 → szCookie[76]) — atlas WriteByte(0) gated JMS` | ✅ |  |
| 3 | int32 | int32 `JMS-only int (@line123 → szReserved[1976]) — atlas WriteInt(0) gated JMS` | ✅ |  |
| 4 | byte | byte `sNotifierMessage flag (@line126)` | ✅ |  |
| 5 | byte | byte `bCharacterData flag (@line127; ==1 here)` | ✅ |  |
| 6 | int16 | int16 `nNotifierCheck / notifier-string count (@line128)` | ✅ |  |
| 7 | int32 | int32 `damage seed 1 (@line159)` | ✅ |  |
| 8 | int64 | int32 `damage seed 2 (@line160)` | ❌ | width mismatch |
| 9 | byte | int32 `damage seed 3 (@line161)` | ❌ | width mismatch |
| 10 | byte | bytes `CharacterData::Decode (@line165) — ENVELOPE BOUNDARY; inner shape under character domain` | ✅ |  |
| 11 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig@0xae81c0) — logout gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 12 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 13 | int32 | int32 `logout gift commodity SN #2` | ✅ |  |
| 14 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 15 | int64 | int64 `timestamp DecodeBuffer(p,8 @line209; FILETIME) — atlas WriteInt64` | ✅ |  |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

