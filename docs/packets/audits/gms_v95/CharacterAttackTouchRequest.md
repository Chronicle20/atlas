# CharacterAttackTouchRequest (← `CUserLocal::TryDoingBodyAttack`)

- **IDA:** 0x930710
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v95
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
| 7 | byte | byte `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | byte `` | ❌ | width mismatch |
| 13 | int32 | int16 `` | ❌ | width mismatch |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int32 | byte `` | ❌ | width mismatch |
| 16 | int32 | byte `` | ❌ | width mismatch |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | byte `` | ❌ | width mismatch |
| 21 | byte | byte `` | ✅ |  |
| 22 | byte | byte `` | ✅ |  |
| 23 | int32 | byte `` | ❌ | width mismatch |
| 24 | int32 | int16 `` | ❌ | width mismatch |
| 25 | int32 | int16 `` | ❌ | width mismatch |
| 26 | int16 | int16 `` | ✅ |  |
| 27 | int16 | int16 `` | ✅ |  |
| 28 | byte | int16 `` | ❌ | width mismatch |
| 29 | int32 | int32 `` | ✅ |  |
| 30 | int32 | int32 `` | ✅ |  |
| 31 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 32 | int16 | int16 `` | ✅ |  |
| 33 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 34 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 35 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 36 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 37 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 38 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 39 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 40 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

