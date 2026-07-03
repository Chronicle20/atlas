# BuffGive (← `CWvsContext::OnTemporaryStatSet`)

- **IDA:** 0x71af4b
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | bytes | bytes `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | width mismatch |
| 3 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
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
| 91 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 100 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 101 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 102 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 103 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 104 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 105 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 106 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 107 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 108 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 109 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 110 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 111 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 112 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 113 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 114 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 115 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 116 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 117 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 118 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 119 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 120 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 121 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 122 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 123 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 124 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 125 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 126 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 127 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 128 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 129 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 130 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 131 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 132 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 133 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 134 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 135 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 136 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 137 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 138 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 139 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 140 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 141 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 142 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 143 | byte | byte `` | ❌ | atlas: short — missing trailing field |

