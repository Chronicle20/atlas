# CharacterSpawn (← `CUserPool::OnUserEnterField`)

- **IDA:** 0x7bd862
- **Atlas file:** `libs/atlas-packet/character/clientbound/spawn.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | string | string `` | ✅ |  |
| 3 | string | int16 `` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | width mismatch |
| 5 | byte | int16 `` | ❌ | width mismatch |
| 6 | int16 | byte `` | ❌ | width mismatch |
| 7 | byte | bytes `` | ✅ |  |
| 8 | byte | byte `` | 🔍 | sub-struct: v — see _substruct/ |
| 9 | byte | byte `` | ✅ |  |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | byte | int32 `` | 🔍 | sub-struct: bts — see _substruct/ |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int32 | int32 `` | ✅ |  |
| 14 | byte | int32 `` | ❌ | width mismatch |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | byte | int16 `` | ❌ | width mismatch |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int32 | int16 `` | ❌ | width mismatch |
| 20 | byte | int32 `` | ❌ | width mismatch |
| 21 | int32 | int32 `` | ✅ |  |
| 22 | int32 | int32 `` | ✅ |  |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | int32 | int32 `` | ✅ |  |
| 25 | int32 | int32 `` | ✅ |  |
| 26 | int32 | int32 `` | ✅ |  |
| 27 | int32 | int32 `` | ✅ |  |
| 28 | int32 | int32 `` | ✅ |  |
| 29 | byte | byte `` | ✅ |  |
| 30 | int16 | byte `` | ❌ | width mismatch |
| 31 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 32 | byte | int16 `` | ❌ | width mismatch |
| 33 | int32 | byte `` | ❌ | width mismatch |
| 34 | string | byte `` | ❌ | width mismatch |
| 35 | int64 | int32 `` | ❌ | width mismatch |
| 36 | int16 | byte `` | ❌ | width mismatch |
| 37 | int16 | int32 `` | ❌ | width mismatch |
| 38 | byte | byte `` | ✅ |  |
| 39 | int32 | int32 `` | ✅ |  |
| 40 | byte | byte `` | ✅ |  |
| 41 | int32 | int32 `` | ✅ |  |
| 42 | int32 | int32 `` | ✅ |  |
| 43 | int32 | bytes `` | ✅ |  |
| 44 | int32 | int32 `` | ✅ |  |
| 45 | int32 | int32 `` | ✅ |  |
| 46 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |

