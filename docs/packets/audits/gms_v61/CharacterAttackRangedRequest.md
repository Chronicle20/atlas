# CharacterAttackRangedRequest (← `CUserLocal::TryDoingShootAttack`)

- **IDA:** 0x7a67e9
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int16 `` | ❌ | width mismatch |
| 10 | int16 | int16 `` | ✅ |  |
| 11 | int16 | byte `` | ❌ | width mismatch |
| 12 | byte | int32 `` | ❌ | width mismatch |
| 13 | int32 | int32 `` | ✅ |  |
| 14 | byte | byte `` | 🔍 | sub-struct: di — see _substruct/ |
| 15 | int16 | byte `` | ❌ | width mismatch |
| 16 | int16 | byte `` | ❌ | width mismatch |
| 17 | int16 | byte `` | ❌ | width mismatch |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int16 | int16 `` | ✅ |  |
| 20 | int16 | int16 `` | ✅ |  |
| 21 | int32 | int16 `` | ❌ | width mismatch |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

