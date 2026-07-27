# CharacterSpawn (← `CUserPool::OnUserEnterField`)

- **IDA:** 0x87bc74
- **Atlas file:** `libs/atlas-packet/character/clientbound/spawn.go`
- **Variant:** GMS/v72
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
| 29 | int32 | int32 `` | ✅ |  |
| 30 | byte | byte `` | ✅ |  |
| 31 | int32 | byte `` | ❌ | width mismatch |
| 32 | string | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 33 | int64 | int16 `` | ❌ | width mismatch |
| 34 | int16 | byte `` | ❌ | width mismatch |
| 35 | int16 | byte `` | ❌ | width mismatch |
| 36 | int32 | int32 `` | ✅ |  |
| 37 | byte | byte `` | ✅ |  |
| 38 | byte | int32 `` | ❌ | width mismatch |
| 39 | int32 | byte `` | ❌ | width mismatch |
| 40 | int32 | int32 `` | ✅ |  |
| 41 | int32 | byte `` | ❌ | width mismatch |
| 42 | int32 | int32 `` | ✅ |  |
| 43 | int32 | int32 `` | ✅ |  |
| 44 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 45 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |

