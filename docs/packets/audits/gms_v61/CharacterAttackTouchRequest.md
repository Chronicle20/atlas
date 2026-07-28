# CharacterAttackTouchRequest (← `CUserLocal::TryDoingBodyAttack`)

- **IDA:** 0x7b084b
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | byte | int32 `` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | int16 | byte `` | ❌ | width mismatch |
| 11 | int16 | byte `` | ❌ | width mismatch |
| 12 | byte | byte `` | ✅ |  |
| 13 | int32 | int16 `` | ❌ | width mismatch |
| 14 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 15 | int16 | int16 `` | ✅ |  |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int16 | int32 `` | ❌ | width mismatch |
| 20 | int32 | int16 `` | ❌ | width mismatch |
| 21 | byte | int16 `` | ❌ | width mismatch |
| 22 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

