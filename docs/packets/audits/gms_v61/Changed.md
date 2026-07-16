# Changed (← `CWvsContext::OnStatChanged`)

- **IDA:** 0x842d04
- **Atlas file:** `libs/atlas-packet/stat/clientbound/changed.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int16 | bytes `` | ✅ |  |
| 6 | int32 | bytes `` | ✅ |  |
| 7 | int64 | bytes `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | int16 `` | ❌ | width mismatch |
| 10 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | byte `` | ❌ | atlas: short — missing trailing field |

