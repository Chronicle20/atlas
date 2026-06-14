# CharacterAttackTouchRequest (← `CUserLocal::TryDoingBodyAttack`)

- **IDA:** 0xa2ac53
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | int32 `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | byte | byte `` | ✅ |  |
| 11 | int32 | int16 `` | ❌ | width mismatch |
| 12 | byte | byte `` | 🔍 | sub-struct: di — see _substruct/ |
| 13 | int16 | byte `` | ❌ | width mismatch |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | int16 | int32 `` | ❌ | width mismatch |
| 17 | int32 | byte `` | ❌ | width mismatch |
| 18 | byte | byte `` | ✅ |  |
| 19 | int16 | byte `` | ❌ | width mismatch |
| 20 | int16 | byte `` | ❌ | width mismatch |
| 21 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

