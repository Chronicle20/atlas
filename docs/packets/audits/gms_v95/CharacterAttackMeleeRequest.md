# CharacterAttackMeleeRequest (← `CUserLocal::TryDoingNormalAttack`)

- **IDA:** 0x9123c0
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | int32 | int16 `` | ❌ | width mismatch |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | int32 | byte `` | ❌ | width mismatch |
| 17 | int32 | byte `` | ❌ | width mismatch |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | int16 `` | ❌ | width mismatch |
| 21 | int16 | int16 `` | ✅ |  |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 33 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

