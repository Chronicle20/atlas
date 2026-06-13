# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x798987
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `` | ❌ | width mismatch |
| 1 | int32 | byte `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int16 `` | ❌ | width mismatch |
| 4 | int16 | string `` | ❌ | width mismatch |
| 5 | byte | string `` | ❌ | width mismatch |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int64 | int32 `` | ❌ | width mismatch |
| 9 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 33 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 36 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 42 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 43 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 44 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 45 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

