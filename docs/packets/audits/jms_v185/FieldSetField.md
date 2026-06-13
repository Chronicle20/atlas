# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x7eea69
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

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
| 10 | int32 | bytes `CharacterData::Decode (@line165) — ENVELOPE BOUNDARY; inner shape under character domain` | ✅ |  |
| 11 | bytes | int32 `m_bPredictQuit (OnSetLogoutGiftConfig@0xae81c0) — logout gift int #1, gated (GMS>83 \|\| JMS)` | ✅ |  |
| 12 | byte | int32 `logout gift commodity SN #1` | ❌ | width mismatch |
| 13 | byte | int32 `logout gift commodity SN #2` | ❌ | width mismatch |
| 14 | byte | int32 `logout gift commodity SN #3` | ❌ | width mismatch |
| 15 | int64 | int64 `timestamp DecodeBuffer(p,8 @line209; FILETIME) — atlas WriteInt64` | ✅ |  |
| 16 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
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
| 28 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 51 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 52 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 53 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 54 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 55 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 56 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 57 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 58 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 59 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 60 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 61 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 62 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 63 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 64 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 65 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 66 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 67 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 68 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 69 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 70 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 71 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 72 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 73 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 74 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 75 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 76 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 77 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 78 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 79 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 80 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 81 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 82 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 83 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 84 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 85 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 86 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 87 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 88 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 89 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 90 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 91 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 92 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 93 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 94 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 95 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 96 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 97 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 98 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 99 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

