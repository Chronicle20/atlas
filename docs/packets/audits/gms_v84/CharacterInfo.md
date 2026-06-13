# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0xa6eda8
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v84
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
| 18 | byte | int32 `` | ❌ | width mismatch |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | int32 | byte `` | ❌ | width mismatch |
| 22 | int32 | bytes `` | ✅ |  |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | int32 | int32 `` | ✅ |  |
| 25 | int32 | int32 `` | ✅ |  |
| 26 | int16 | int32 `` | ❌ | width mismatch |
| 27 | int32 | int32 `` | ✅ |  |
| 28 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

