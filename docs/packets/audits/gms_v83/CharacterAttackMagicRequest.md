# CharacterAttackMagicRequest (← `CUserLocal::TryDoingMagicAttack`)

- **IDA:** 0x95571f
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
| 11 | int32 | int32 `` | ✅ |  |
| 12 | byte | byte `` | ✅ |  |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | byte | byte `` | 🔍 | sub-struct: di — see _substruct/ |
| 15 | int16 | byte `` | ❌ | width mismatch |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int16 | int16 `` | ✅ |  |
| 20 | int16 | int16 `` | ✅ |  |
| 21 | int32 | int32 `` | ✅ |  |
| 22 | byte | int32 `` | ❌ | width mismatch |
| 23 | int16 | int16 `` | ✅ |  |
| 24 | int16 | int16 `` | ✅ |  |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

