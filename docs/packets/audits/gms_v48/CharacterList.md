# CharacterList (← `sub_5013ED`)

- **IDA:** 0x5013ed
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v48
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | bytes | int32 `` | ✅ |  |
| 4 | byte | bytes `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int32 | byte `` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | int64 | byte `` | ❌ | width mismatch |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int16 | bytes `` | ✅ |  |
| 14 | int16 | byte `` | ❌ | width mismatch |
| 15 | int16 | int16 `` | ✅ |  |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int32 | int16 `` | ❌ | width mismatch |
| 19 | int16 | int16 `` | ✅ |  |
| 20 | int32 | int16 `` | ❌ | width mismatch |
| 21 | int16 | int16 `` | ✅ |  |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int32 | int16 `` | ❌ | width mismatch |
| 24 | byte | int16 `` | ❌ | width mismatch |
| 25 | int32 | int16 `` | ❌ | width mismatch |
| 26 | byte | int32 `` | ❌ | width mismatch |
| 27 | int32 | int16 `` | ❌ | width mismatch |
| 28 | byte | int32 `` | ❌ | width mismatch |
| 29 | byte | byte `` | ✅ |  |
| 30 | int32 | byte `` | ❌ | width mismatch |
| 31 | byte | byte `` | ✅ |  |
| 32 | int32 | int32 `` | ✅ |  |
| 33 | int32 | byte `` | ❌ | width mismatch |
| 34 | byte | int32 `` | ❌ | width mismatch |
| 35 | byte | byte `` | ✅ |  |
| 36 | int32 | int32 `` | ✅ |  |
| 37 | int32 | byte `` | ❌ | width mismatch |
| 38 | int32 | int32 `` | ✅ |  |
| 39 | int32 | int32 `` | ✅ |  |
| 40 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 42 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

