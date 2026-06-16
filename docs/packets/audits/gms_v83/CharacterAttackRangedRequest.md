# CharacterAttackRangedRequest (← `CUserLocal::TryDoingShootAttack`)

- **IDA:** 0x9537d5
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v83
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
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | byte `` | ✅ |  |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int16 | int16 `` | ✅ |  |
| 12 | int16 | int16 `` | ✅ |  |
| 13 | byte | byte `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | byte | int32 `` | 🔍 | sub-struct: di — see _substruct/ |
| 16 | int16 | byte `` | ❌ | width mismatch |
| 17 | int16 | byte `` | ❌ | width mismatch |
| 18 | int16 | byte `` | ❌ | width mismatch |
| 19 | int16 | byte `` | ❌ | width mismatch |
| 20 | int16 | int16 `` | ✅ |  |
| 21 | int16 | int16 `` | ✅ |  |
| 22 | int32 | int16 `` | ❌ | width mismatch |
| 23 | byte | int16 `` | ❌ | width mismatch |
| 24 | int16 | int16 `` | ✅ |  |
| 25 | int16 | int32 `` | ❌ | width mismatch |
| 26 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

