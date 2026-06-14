# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x776020
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `m_nChannelID (channel id) — v83 has NO DecodeOpt before it and NO m_dwOldDriverID after it` | ❌ | width mismatch |
| 1 | int32 | byte `sNotifierMessage flag (HIBYTE v110)` | ❌ | width mismatch |
| 2 | byte | byte `bCharacterData flag (v107)` | ✅ |  |
| 3 | byte | int16 `nNotifierCheck (v108; notifier string-loop count)` | ❌ | width mismatch |
| 4 | int16 | int32 `damage seed 1 (CalcDamage::SetSeed; bCharacterData branch)` | ❌ | width mismatch |
| 5 | int32 | int32 `damage seed 2` | ✅ |  |
| 6 | int64 | int32 `damage seed 3` | ❌ | width mismatch |
| 7 | byte | bytes `CharacterData::Decode — ENVELOPE BOUNDARY (character domain task-028)` | ✅ |  |
| 8 | int32 | int64 `timestamp (DecodeBuffer p,8u FILETIME) — v83 has NO logout-gift block before it` | ❌ | width mismatch |
| 9 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
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
| 27 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
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
| 78 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 79 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 80 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 81 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 82 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 83 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 84 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 85 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 86 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 87 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 88 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 89 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 90 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

