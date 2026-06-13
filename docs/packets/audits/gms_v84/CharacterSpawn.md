# CharacterSpawn (← `CUserPool::OnUserEnterField`)

- **IDA:** 0x9b20a0
- **Atlas file:** `libs/atlas-packet/character/clientbound/spawn.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | string | string `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int16 | int16 `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | bytes | bytes `` | ✅ |  |
| 9 | int16 | byte `` | ❌ | width mismatch |
| 10 | byte | byte `` | ✅ |  |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | byte | int32 `` | ❌ | width mismatch |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | byte | int32 `` | ❌ | width mismatch |
| 16 | int32 | int32 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | byte | int16 `` | ❌ | width mismatch |
| 20 | int32 | int16 `` | ❌ | width mismatch |
| 21 | int32 | int32 `` | ✅ |  |
| 22 | int32 | int32 `` | ✅ |  |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | int32 | int32 `` | ✅ |  |
| 25 | int32 | int32 `` | ✅ |  |
| 26 | int32 | int32 `` | ✅ |  |
| 27 | int32 | int32 `` | ✅ |  |
| 28 | int32 | int32 `` | ✅ |  |
| 29 | byte | int32 `` | ❌ | width mismatch |
| 30 | int32 | int32 `` | ✅ |  |
| 31 | string | int32 `` | ❌ | width mismatch |
| 32 | int64 | int32 `` | ❌ | width mismatch |
| 33 | int32 | int32 `` | ✅ |  |
| 34 | int32 | int32 `` | ✅ |  |
| 35 | byte | int32 `` | ❌ | width mismatch |
| 36 | byte | int32 `` | ❌ | width mismatch |
| 37 | int32 | byte `` | ❌ | width mismatch |
| 38 | int32 | byte `` | ❌ | width mismatch |
| 39 | int32 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 40 | int16 | int16 `` | ✅ |  |
| 41 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 42 | int32 | int32 `` | ✅ |  |
| 43 | byte | int32 `` | ❌ | width mismatch |
| 44 | byte | int32 `` | ❌ | width mismatch |
| 45 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |

