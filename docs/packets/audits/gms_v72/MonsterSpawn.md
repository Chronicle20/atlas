# MonsterSpawn (← `CMobPool::OnMobEnterField`)

- **IDA:** 0x6256de
- **Atlas file:** `libs/atlas-packet/monster/clientbound/spawn.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int16 `` | ❌ | width mismatch |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int16 `` | ❌ | width mismatch |
| 7 | int32 | int16 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int16 `` | ❌ | width mismatch |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | int16 `` | ❌ | width mismatch |
| 13 | int16 | int16 `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int32 | int16 `` | ❌ | width mismatch |
| 16 | int32 | int16 `` | ❌ | width mismatch |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | byte | int16 `` | ❌ | width mismatch |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | byte | int16 `` | ❌ | width mismatch |
| 22 | int32 | int16 `` | ❌ | width mismatch |
| 23 | byte | int32 `` | ❌ | width mismatch |
| 24 | int32 | int16 `` | ❌ | width mismatch |
| 25 | int32 | int16 `` | ❌ | width mismatch |
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
| 88 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 100 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 101 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 102 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 103 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 104 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 105 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 106 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 107 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 108 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 109 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 110 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 111 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 112 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 113 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 114 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 115 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 116 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 117 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 118 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 119 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 120 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 121 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 122 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 123 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 124 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 125 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 126 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 127 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 128 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 129 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 130 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 131 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 132 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 133 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 134 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 135 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 136 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 137 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 138 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 139 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 140 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 141 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 142 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 143 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 144 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 145 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 146 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 147 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 148 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 149 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 150 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 151 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 152 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 153 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 154 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 155 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 156 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 157 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 158 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 159 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 160 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 161 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 162 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 163 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 164 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 165 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 166 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 167 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 168 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 169 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 170 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 171 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 172 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 173 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 174 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 175 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 176 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 177 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 178 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 179 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 180 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 181 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 182 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 183 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 184 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 185 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 186 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 187 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 188 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 189 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 190 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 191 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 192 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 193 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

