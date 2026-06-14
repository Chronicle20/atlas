# GuildMemberJoined (← `CWvsContext::OnGuildResult#MemberJoined`)

- **IDA:** 0xa82e2b
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | string `` | ❌ | width mismatch |
| 2 | int32 | string `` | ❌ | width mismatch |
| 3 | bytes | string `` | ❌ | width mismatch |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 36 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 39 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 42 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 43 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 44 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 45 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 46 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 47 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 48 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 49 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 51 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 52 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 53 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 54 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 55 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 56 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 57 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 58 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 59 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 60 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 61 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 62 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 63 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 65 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 66 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 67 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 68 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 69 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 70 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 71 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 72 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 73 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 74 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 75 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 76 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 77 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 78 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 79 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 80 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 81 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 82 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

