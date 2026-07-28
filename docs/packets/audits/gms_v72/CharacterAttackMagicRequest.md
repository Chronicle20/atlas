# CharacterAttackMagicRequest (← `CUserLocal::TryDoingMagicAttack`)

- **IDA:** 0x8625da
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v72
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
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | int16 | byte `` | ❌ | width mismatch |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int16 | byte `` | ❌ | width mismatch |
| 12 | int16 | byte `` | ❌ | width mismatch |
| 13 | byte | byte `` | ✅ |  |
| 14 | int32 | byte `` | ❌ | width mismatch |
| 15 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int16 | int16 `` | ✅ |  |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | int32 | int32 `` | ✅ |  |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int16 | int16 `` | ✅ |  |
| 24 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

