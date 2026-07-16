# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0x91b961
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | string | string `` | ✅ |  |
| 6 | string | string `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | string | string `` | ✅ |  |
| 11 | byte | byte `` | ✅ |  |
| 12 | int16 | int16 `` | ✅ |  |
| 13 | byte | byte `` | ✅ |  |
| 14 | int16 | int16 `` | ✅ |  |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | byte | byte `` | ✅ |  |
| 17 | byte | byte `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | byte | byte `` | ✅ |  |
| 22 | byte | bytes `` | ✅ |  |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | int32 | int32 `` | ✅ |  |
| 25 | int32 | int32 `` | ✅ |  |
| 26 | int32 | int32 `` | ✅ |  |
| 27 | int32 | int32 `` | ✅ |  |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

