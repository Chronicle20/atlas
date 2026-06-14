# CharacterAttackTouchRequest (← `CUserLocal::TryDoingBodyAttack`)

- **IDA:** 0x99d42a
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int32 | byte `` | ❌ | width mismatch |
| 12 | byte | int16 `` | ❌ | width mismatch |
| 13 | int16 | byte `` | ❌ | width mismatch |
| 14 | byte | byte `` | ✅ |  |
| 15 | byte | int32 `` | ❌ | width mismatch |
| 16 | int32 | int32 `` | ✅ |  |
| 17 | int16 | byte `` | ❌ | width mismatch |
| 18 | int16 | byte `` | ❌ | width mismatch |
| 19 | byte | byte `` | ✅ |  |
| 20 | int32 | byte `` | ❌ | width mismatch |
| 21 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 22 | int16 | int16 `` | ✅ |  |
| 23 | int16 | int16 `` | ✅ |  |
| 24 | int16 | int16 `` | ✅ |  |
| 25 | int16 | int16 `` | ✅ |  |
| 26 | int32 | int32 `` | ✅ |  |
| 27 | int32 | int32 `` | ✅ |  |
| 28 | byte | int16 `` | ❌ | width mismatch |
| 29 | int16 | int16 `` | ✅ |  |
| 30 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

