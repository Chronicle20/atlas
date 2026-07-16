# MonsterStatSet (← `CMob::OnStatSet`)

- **IDA:** 0x63ae2b
- **Atlas file:** `libs/atlas-packet/monster/clientbound/stat.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | bytes `` | ✅ |  |
| 1 | int32 | int16 `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int16 `` | ❌ | width mismatch |
| 4 | int32 | int16 `` | ❌ | width mismatch |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int16 `` | ❌ | width mismatch |
| 7 | int32 | int16 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int16 | int16 `` | ✅ |  |
| 10 | int32 | int16 `` | ❌ | width mismatch |
| 11 | int16 | int32 `` | ❌ | width mismatch |
| 12 | int32 | int16 `` | ❌ | width mismatch |
| 13 | int32 | int16 `` | ❌ | width mismatch |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int16 | int16 `` | ✅ |  |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | byte | int32 `` | ❌ | width mismatch |
| 18 | byte | int16 `` | ❌ | width mismatch |
| 19 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 36 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 42 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 43 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 44 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 45 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 100 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 101 | byte | byte `` | ❌ | atlas: short — missing trailing field |

