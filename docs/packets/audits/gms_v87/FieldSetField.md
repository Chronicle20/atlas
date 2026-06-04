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
| 8 | int32 | bytes `CharacterData::Decode (@0x7c440a) — ENVELOPE BOUNDARY; inner shape audited under character domain (task-028)` | ✅ |  |
| 9 | bytes | int32 `m_bPredictQuit (OnSetLogoutGiftConfig @0x7c4412) — atlas logout-gift int #1, gated (GMS>83 \|\| JMS)` | ✅ |  |
| 10 | byte | int32 `logout gift commodity SN #1` | ❌ | width mismatch |
| 11 | byte | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 12 | byte | int32 `logout gift commodity SN #3` | ❌ | width mismatch |
| 13 | int64 | int64 `timestamp (DecodeBuffer p,8u @0x7c4558 — FILETIME); atlas WriteInt64` | ✅ |  |
| 14 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
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
| 27 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 51 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 52 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 53 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 54 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 55 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 56 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 57 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 58 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 59 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 60 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 61 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 62 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 63 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 64 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 65 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 66 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 67 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 68 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 69 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 70 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 71 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 72 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 73 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 74 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 75 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 76 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 77 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 78 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 79 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 80 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 81 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 82 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 83 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 84 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 85 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 86 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 87 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 88 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 89 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

